# 🚀 Руководство по развертыванию k8s-node-labeler

## 📋 Содержание

1. [Быстрое развертывание](#быстрое-развертывание)
2. [Что развертывается](#что-развертывается)
3. [Варианты развертывания](#варианты-развертывания)
4. [Проверка развертывания](#проверка-развертывания)
5. [Удаление](#удаление)

---

## ⚡ Быстрое развертывание

### Полный цикл за 4 команды:

```bash
# 1. Установить CRD
make install

# 2. Собрать образ (укажите свой registry)
export IMG=docker.io/your-username/k8s-node-labeler:v0.0.1
make docker-build IMG=$IMG
make docker-push IMG=$IMG

# 3. Развернуть контроллер
make deploy IMG=$IMG

# 4. Проверить
kubectl get pods -n k8s-node-labeler-system
```

---

## 📦 Что развертывается

При выполнении `make deploy` в кластер будут установлены следующие ресурсы:

### 1. **Namespace**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: k8s-node-labeler-system
```
Изолированное пространство имен для всех ресурсов оператора.

### 2. **ServiceAccount**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-node-labeler-controller-manager
  namespace: k8s-node-labeler-system
```
Идентификатор, под которым работает контроллер.

### 3. **RBAC - Leader Election Role** (в namespace)
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-node-labeler-leader-election-role
  namespace: k8s-node-labeler-system
```
Права для механизма Leader Election (нужно для HA режима):
- Управление ConfigMaps
- Управление Leases
- Создание Events

### 4. **RBAC - Manager ClusterRole** (cluster-wide)
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-node-labeler-manager-role
rules:
- apiGroups: [""]
  resources: [nodes]
  verbs: [get, list, watch, update, patch]
- apiGroups: [labeler.arsolitt.tech]
  resources: [labelmappings, labelmappings/status, labelmappings/finalizers]
  verbs: [create, delete, get, list, patch, update, watch]
```
Основные права контроллера для работы с нодами и LabelMapping.

### 5. **RBAC - Metrics Reader ClusterRole**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-node-labeler-metrics-reader
rules:
- nonResourceURLs: [/metrics]
  verbs: [get]
```
Права для чтения метрик (опционально для мониторинга).

### 6. **RBAC - Bindings**
- **RoleBinding** для leader election
- **ClusterRoleBinding** для manager role
- **ClusterRoleBinding** для metrics auth

Связывают роли с ServiceAccount.

### 7. **Deployment**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-node-labeler-controller-manager
  namespace: k8s-node-labeler-system
spec:
  replicas: 1
  template:
    spec:
      serviceAccountName: k8s-node-labeler-controller-manager
      containers:
      - name: manager
        image: controller:latest  # Заменяется на ваш образ
        args:
        - --leader-elect
        - --health-probe-bind-address=:8081
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
```
Сам контроллер в виде Deployment.

### 8. **Service для метрик**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: k8s-node-labeler-controller-manager-metrics-service
  namespace: k8s-node-labeler-system
spec:
  ports:
  - name: https
    port: 8443
    protocol: TCP
    targetPort: 8443
```
Экспонирует метрики контроллера.

---

## 🎯 Варианты развертывания

### Вариант 1: Production (публичный registry)

```bash
# Настройка
export IMG=docker.io/your-username/k8s-node-labeler:v0.0.1

# Сборка и публикация
make docker-build IMG=$IMG
make docker-push IMG=$IMG

# Развертывание
make install
make deploy IMG=$IMG
```

### Вариант 2: Kind (локальная разработка)

```bash
# Создать kind кластер
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# Собрать и загрузить в kind
make docker-build IMG=k8s-node-labeler:dev
kind load docker-image k8s-node-labeler:dev

# Развернуть
make install
make deploy IMG=k8s-node-labeler:dev
```

### Вариант 3: Minikube

```bash
# Запустить minikube
minikube start

# Использовать Docker daemon minikube
eval $(minikube docker-env)

# Собрать образ (попадет в minikube)
make docker-build IMG=k8s-node-labeler:dev

# Развернуть
make install
make deploy IMG=k8s-node-labeler:dev
```

### Вариант 4: Локальный запуск (без контейнера)

Самый простой способ для разработки:

```bash
# Установить только CRD
make install

# Запустить контроллер локально
make run

# В другом терминале тестировать
kubectl apply -f config/samples/labeler_v1alpha1_labelmapping.yaml
```

### Вариант 5: Единый манифест

```bash
# Создать единый install.yaml
make build-installer IMG=docker.io/your-username/k8s-node-labeler:v0.0.1

# Применить
kubectl apply -f dist/install.yaml
```

Этот файл можно распространять пользователям:
```bash
kubectl apply -f https://raw.githubusercontent.com/username/k8s-node-labeler/main/dist/install.yaml
```

### Вариант 6: Helm (будущее расширение)

```bash
# TODO: Создать Helm chart
helm install k8s-node-labeler ./charts/k8s-node-labeler
```

---

## ✅ Проверка развертывания

### 1. Проверить все ресурсы

```bash
# Все ресурсы в namespace
kubectl get all -n k8s-node-labeler-system

# Вывод:
# NAME                                                         READY   STATUS    RESTARTS   AGE
# pod/k8s-node-labeler-controller-manager-xxxxxxxxxx-xxxxx     1/1     Running   0          1m
#
# NAME                                                            TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
# service/k8s-node-labeler-controller-manager-metrics-service     ClusterIP   10.96.xxx.xxx   <none>        8443/TCP   1m
#
# NAME                                                      READY   UP-TO-DATE   AVAILABLE   AGE
# deployment.apps/k8s-node-labeler-controller-manager       1/1     1            1           1m
```

### 2. Проверить логи

```bash
# Логи контроллера
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager

# Следить за логами в реальном времени
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager -f
```

Ожидаемый вывод при старте:
```
INFO	setup	starting manager
INFO	controller-runtime.metrics	Metrics server is starting to listen	{"addr": "0"}
INFO	Starting server	{"kind": "health probe", "addr": "[::]:8081"}
INFO	Starting EventSource	{"controller": "labelmapping", "source": "kind source: *v1alpha1.LabelMapping"}
INFO	Starting EventSource	{"controller": "labelmapping", "source": "kind source: *v1.Node"}
INFO	Starting Controller	{"controller": "labelmapping"}
INFO	Starting workers	{"controller": "labelmapping", "worker count": 1}
```

### 3. Проверить CRD

```bash
# Проверить установку CRD
kubectl get crd labelmappings.labeler.arsolitt.tech

# Детальная информация
kubectl describe crd labelmappings.labeler.arsolitt.tech
```

### 4. Проверить RBAC

```bash
# Проверить права ServiceAccount
kubectl auth can-i get nodes \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager

kubectl auth can-i patch nodes \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager

kubectl auth can-i create labelmappings.labeler.arsolitt.tech \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager
```

Все команды должны вернуть `yes`.

### 5. Проверить health endpoints

```bash
# Port-forward к pod
kubectl port-forward -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager 8081:8081

# В другом терминале проверить
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

Оба должны вернуть `ok`.

### 6. Проверить метрики

```bash
# Port-forward к metrics service
kubectl port-forward -n k8s-node-labeler-system svc/k8s-node-labeler-controller-manager-metrics-service 8443:8443

# Получить метрики (требуется авторизация)
curl -k https://localhost:8443/metrics
```

---

## 🧪 Тестирование после развертывания

### 1. Создать простой LabelMapping

```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: test-ready-nodes
spec:
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    test-label: "true"
    auto-labeled: "ready"
EOF
```

### 2. Проверить статус

```bash
# Список LabelMapping
kubectl get labelmappings

# Должны увидеть:
# NAME               MATCHED   LABELED   AGE
# test-ready-nodes   3         3         10s

# Детали
kubectl describe labelmapping test-ready-nodes
```

### 3. Проверить лейблы на нодах

```bash
# Все лейблы
kubectl get nodes --show-labels

# Только нужные лейблы
kubectl get nodes -L test-label,auto-labeled

# Вывод:
# NAME           STATUS   ROLES           AGE   VERSION   TEST-LABEL   AUTO-LABELED
# control-plane  Ready    control-plane   10m   v1.27.3   true         ready
# worker         Ready    <none>          10m   v1.27.3   true         ready
# worker2        Ready    <none>          10m   v1.27.3   true         ready
```

### 4. Проверить логи контроллера

```bash
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager --tail=20

# Должны увидеть записи о добавлении лейблов:
# INFO	Добавлены лейблы на ноду	{"node": "control-plane", "labels": {"test-label":"true","auto-labeled":"ready"}}
# INFO	Reconciliation completed	{"matchedNodes": 3, "labeledNodes": 3}
```

---

## 🗑️ Удаление

### Удалить тестовый LabelMapping

```bash
kubectl delete labelmapping test-ready-nodes

# Внимание: Лейблы останутся на нодах!
# Удалить их вручную:
kubectl label nodes --all test-label- auto-labeled-
```

### Удалить контроллер

```bash
# Удалить deployment и все ресурсы
make undeploy
```

### Удалить CRD

```bash
# Внимание: Это удалит все LabelMapping!
make uninstall
```

### Полная очистка

```bash
# Удалить все
make undeploy
make uninstall

# Удалить namespace (если нужно)
kubectl delete namespace k8s-node-labeler-system
```

---

## 🔧 Настройка после развертывания

### Изменить ресурсы контроллера

Отредактируйте `config/manager/manager.yaml`:

```yaml
resources:
  limits:
    cpu: 1000m      # Увеличить CPU
    memory: 256Mi   # Увеличить память
  requests:
    cpu: 50m
    memory: 128Mi
```

Примените изменения:
```bash
make deploy IMG=$IMG
```

### Включить multiple replicas

Отредактируйте `config/manager/manager.yaml`:

```yaml
spec:
  replicas: 3  # Вместо 1
```

Leader Election уже включен, поэтому только одна реплика будет активна.

### Добавить NodeSelector для контроллера

Отредактируйте `config/manager/manager.yaml`:

```yaml
spec:
  template:
    spec:
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
```

---

## 🐛 Troubleshooting

### Pod не запускается

```bash
# Проверить события
kubectl get events -n k8s-node-labeler-system --sort-by='.lastTimestamp'

# Проверить описание pod
kubectl describe pod -n k8s-node-labeler-system -l control-plane=controller-manager
```

Частые проблемы:
- **ImagePullBackOff**: Образ недоступен
  - Решение: Проверить `IMG` переменную, убедиться что образ загружен
- **CrashLoopBackOff**: Контроллер падает при старте
  - Решение: Проверить логи `kubectl logs ...`

### Лейблы не применяются

```bash
# Проверить статус LabelMapping
kubectl describe labelmapping <name>

# Проверить логи
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager | grep ERROR

# Проверить RBAC
kubectl auth can-i patch nodes --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager
```

### Высокое потребление ресурсов

```bash
# Проверить использование ресурсов
kubectl top pod -n k8s-node-labeler-system

# Увеличить limits в config/manager/manager.yaml
# Уменьшить частоту синхронизации в контроллере (RequeueAfter)
```

---

## 📊 Мониторинг

### Prometheus

Если у вас установлен Prometheus Operator, раскомментируйте в `config/default/kustomization.yaml`:

```yaml
resources:
- ../prometheus
```

Затем:
```bash
make deploy IMG=$IMG
```

Это создаст ServiceMonitor для scraping метрик.

### Grafana Dashboard

Можно создать дашборд на основе метрик controller-runtime:
- `controller_runtime_reconcile_total`
- `controller_runtime_reconcile_errors_total`
- `controller_runtime_reconcile_time_seconds`

---

## 🔒 Безопасность

### NetworkPolicy

Для защиты metrics endpoint раскомментируйте в `config/default/kustomization.yaml`:

```yaml
resources:
- ../network-policy
```

### PodSecurityPolicy / PodSecurity

Контроллер совместим с `restricted` PSS:
```yaml
securityContext:
  runAsNonRoot: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
```

---

## 📝 Checklist развертывания

- [ ] Собран Docker образ
- [ ] Образ загружен в registry (или kind/minikube)
- [ ] Установлен CRD (`make install`)
- [ ] Развернут контроллер (`make deploy`)
- [ ] Pod контроллера Running
- [ ] Логи контроллера чистые (нет ERROR)
- [ ] RBAC права проверены
- [ ] Health checks работают
- [ ] Создан тестовый LabelMapping
- [ ] Лейблы применились на ноды
- [ ] Статус LabelMapping обновляется

---

## 🎓 Дополнительные ресурсы

- [QUICKSTART.md](QUICKSTART.md) - Быстрый старт
- [USAGE.md](USAGE.md) - Примеры использования
- [README.md](README.md) - Общая информация
- [Kubebuilder Book](https://book.kubebuilder.io/) - Документация по Kubebuilder

---

**Версия**: v0.0.1  
**Последнее обновление**: 2025-11-10

