# Быстрый старт k8s-node-labeler

## Предварительные требования

- Go 1.22+
- Docker или Podman
- kubectl
- Доступ к Kubernetes кластеру (можно использовать kind, minikube или полноценный кластер)

## Шаг 1: Локальная разработка и тестирование

### Запуск оператора локально (вне кластера)

```bash
# Установить CRD в кластер
make install

# Запустить оператор локально
make run
```

Оператор будет подключаться к кластеру, указанному в `~/.kube/config`, и работать на вашей машине.

### Создание тестового LabelMapping

В другом терминале:

```bash
# Применить пример
kubectl apply -f config/samples/labeler_v1alpha1_labelmapping.yaml

# Проверить статус
kubectl get labelmappings
kubectl describe labelmapping ready-nodes-labeler

# Проверить лейблы на нодах
kubectl get nodes --show-labels
```

## Шаг 2: Сборка и развертывание в кластере

### Сборка Docker образа

```bash
# Установите переменную с адресом вашего registry
export IMG=docker.io/your-username/k8s-node-labeler:v0.0.1

# Соберите образ
make docker-build IMG=$IMG

# Загрузите в registry
make docker-push IMG=$IMG
```

Или используйте локальный registry для kind:

```bash
# Создать kind кластер с registry
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# Собрать и загрузить в kind
make docker-build IMG=k8s-node-labeler:v0.0.1
kind load docker-image k8s-node-labeler:v0.0.1
```

### Развертывание в кластере

```bash
# Установить CRD
make install

# Развернуть оператор
make deploy IMG=$IMG

# Проверить статус
kubectl get pods -n k8s-node-labeler-system
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager -f
```

## Шаг 3: Создание LabelMapping ресурсов

### Пример 1: Маркировка всех готовых нод

```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: mark-ready-nodes
spec:
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    auto-labeled: "true"
    node-health: ready
EOF
```

Проверка:

```bash
# Проверить LabelMapping
kubectl get labelmappings mark-ready-nodes

# Должны увидеть что-то вроде:
# NAME               MATCHED   LABELED   AGE
# mark-ready-nodes   3         3         10s

# Проверить лейблы на нодах
kubectl get nodes --show-labels | grep auto-labeled
```

### Пример 2: Маркировка только worker нод

```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: worker-nodes-ready
spec:
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    worker-status: ready
    can-schedule: "true"
EOF
```

### Пример 3: Маркировка нод в режиме обслуживания

```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: maintenance-mode
spec:
  conditions:
    - type: NodeReady
      status: "False"
  labels:
    maintenance: "true"
    workload-ready: "false"
EOF
```

## Шаг 4: Проверка работы

### Просмотр всех LabelMapping

```bash
kubectl get labelmappings

# Вывод:
# NAME               MATCHED   LABELED   AGE
# mark-ready-nodes   3         3         2m
# worker-nodes-ready 2         2         1m
# maintenance-mode   0         0         30s
```

### Детальная информация

```bash
kubectl describe labelmapping mark-ready-nodes
```

### Логи оператора

```bash
# Если запущен локально - смотрите в терминале
# Если в кластере:
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager -f
```

## Шаг 5: Тестирование динамического управления

### Имитация падения ноды

```bash
# Получить список нод
kubectl get nodes

# Пометить ноду как unschedulable (это не изменит Ready статус)
kubectl cordon <node-name>

# Для реального теста можно остановить kubelet на ноде
# (требует SSH доступ к ноде):
# ssh <node> sudo systemctl stop kubelet

# Через ~40 секунд нода перейдет в NotReady
kubectl get nodes -w

# Проверить, что лейблы изменились
kubectl get node <node-name> --show-labels
```

### Восстановление ноды

```bash
# Запустить kubelet обратно
# ssh <node> sudo systemctl start kubelet

# Проверить изменение лейблов
kubectl get nodes --show-labels
```

## Шаг 6: Использование лейблов в приложениях

### NodeSelector в Pod

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  nodeSelector:
    node-health: ready
  containers:
  - name: nginx
    image: nginx:latest
EOF
```

### Node Affinity

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: production-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: production
  template:
    metadata:
      labels:
        app: production
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: worker-status
                operator: In
                values:
                - ready
      containers:
      - name: app
        image: nginx:latest
EOF
```

## Шаг 7: Очистка

### Удаление LabelMapping ресурсов

```bash
kubectl delete labelmapping mark-ready-nodes
kubectl delete labelmapping worker-nodes-ready
kubectl delete labelmapping maintenance-mode

# Или все сразу
kubectl delete labelmappings --all
```

**Важно:** Лейблы останутся на нодах! Для их удаления:

```bash
# Удалить конкретный лейбл
kubectl label node <node-name> auto-labeled-

# Или все лейблы, добавленные оператором
kubectl get nodes -o name | xargs -I {} kubectl label {} auto-labeled- node-health- worker-status- can-schedule- maintenance- workload-ready-
```

### Удаление оператора из кластера

```bash
# Удалить deployment
make undeploy

# Удалить CRD (удалит все LabelMapping!)
make uninstall
```

## Troubleshooting

### Оператор не запускается

```bash
# Проверить логи
kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager

# Проверить RBAC
kubectl auth can-i get nodes --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager
kubectl auth can-i update nodes --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager
```

### Лейблы не применяются

```bash
# Проверить статус LabelMapping
kubectl describe labelmapping <name>

# Проверить соответствие nodeSelector
kubectl get nodes -l <your-selector>

# Проверить статус нод
kubectl get nodes
```

### CRD не устанавливается

```bash
# Проверить манифест CRD
kubectl get crd labelmappings.labeler.arsolitt.tech

# Если нет - установить вручную
kubectl apply -f config/crd/bases/labeler.arsolitt.tech_labelmappings.yaml
```

## Следующие шаги

1. Прочитайте [USAGE.md](USAGE.md) для детальных примеров использования
2. Изучите [ARCHITECTURE.md](ARCHITECTURE.md) для понимания внутреннего устройства
3. Адаптируйте примеры под свои нужды
4. Настройте мониторинг и алерты

## Полезные команды

```bash
# Следить за изменениями
kubectl get labelmappings -w
kubectl get nodes -w

# Экспорт всех LabelMapping
kubectl get labelmappings -o yaml > labelmappings-backup.yaml

# Проверка YAML перед применением
kubectl apply -f config/samples/labeler_v1alpha1_labelmapping.yaml --dry-run=client

# Метрики оператора (если включены)
kubectl port-forward -n k8s-node-labeler-system svc/k8s-node-labeler-controller-manager-metrics-service 8443:8443
curl -k https://localhost:8443/metrics
```

## Дополнительные ресурсы

- [Kubernetes Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
- [Node Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

---

Если у вас возникли вопросы или проблемы, проверьте логи оператора и документацию.

