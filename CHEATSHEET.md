# 📝 Шпаргалка k8s-node-labeler

## 🚀 Быстрое развертывание

```bash
# Production (с Docker registry)
export IMG=docker.io/your-username/k8s-node-labeler:v0.0.1
make install
make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG

# Kind (локально)
make docker-build IMG=k8s-node-labeler:dev
kind load docker-image k8s-node-labeler:dev
make install
make deploy IMG=k8s-node-labeler:dev

# Разработка (без контейнера)
make install
make run
```

## 🔨 Команды Make

```bash
make help                # Показать все доступные команды
make build              # Собрать бинарник
make run                # Запустить локально
make docker-build       # Собрать Docker образ
make docker-push        # Загрузить образ в registry
make install            # Установить CRD
make uninstall          # Удалить CRD
make deploy             # Развернуть в кластер
make undeploy           # Удалить из кластера
make manifests          # Сгенерировать манифесты
make generate           # Сгенерировать код (DeepCopy)
make test               # Запустить тесты
make fmt                # Отформатировать код
make vet                # Проверить код
make build-installer    # Создать dist/install.yaml
```

## 📦 Управление LabelMapping

```bash
# Создать
kubectl apply -f config/samples/labeler_v1alpha1_labelmapping.yaml

# Список
kubectl get labelmappings
kubectl get lm  # короткий алиас

# Детали
kubectl describe labelmapping <name>

# YAML
kubectl get labelmapping <name> -o yaml

# Удалить
kubectl delete labelmapping <name>

# Все сразу
kubectl get labelmappings
kubectl delete labelmappings --all
```

## 🖥️ Управление контроллером

```bash
# Статус pod
kubectl get pods -n k8s-node-labeler-system

# Логи
kubectl logs -n k8s-node-labeler-system \
  deployment/k8s-node-labeler-controller-manager -f

# Рестарт
kubectl rollout restart deployment \
  -n k8s-node-labeler-system \
  k8s-node-labeler-controller-manager

# Статус deployment
kubectl get deployment -n k8s-node-labeler-system

# Все ресурсы
kubectl get all -n k8s-node-labeler-system
```

## 🏷️ Работа с лейблами нод

```bash
# Все лейблы
kubectl get nodes --show-labels

# Конкретные лейблы
kubectl get nodes -L label1,label2,label3

# Лейблы одной ноды
kubectl get node <node-name> --show-labels

# Добавить лейбл вручную
kubectl label node <node-name> key=value

# Удалить лейбл
kubectl label node <node-name> key-

# Удалить лейбл со всех нод
kubectl label nodes --all key-

# Найти ноды с лейблом
kubectl get nodes -l key=value
```

## 🔍 Проверка и отладка

```bash
# Статус нод
kubectl get nodes
kubectl get nodes -o wide

# Ready статус нод
kubectl get nodes -o custom-columns=\
NAME:.metadata.name,\
STATUS:.status.conditions[?\(@.type==\"Ready\"\)].status

# События в namespace
kubectl get events -n k8s-node-labeler-system \
  --sort-by='.lastTimestamp'

# Проверить CRD
kubectl get crd labelmappings.labeler.arsolitt.tech
kubectl describe crd labelmappings.labeler.arsolitt.tech

# Проверить RBAC
kubectl auth can-i get nodes \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager

kubectl auth can-i patch nodes \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager

# Health checks
kubectl port-forward -n k8s-node-labeler-system \
  deployment/k8s-node-labeler-controller-manager 8081:8081
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz

# Метрики
kubectl port-forward -n k8s-node-labeler-system \
  svc/k8s-node-labeler-controller-manager-metrics-service 8443:8443
curl -k https://localhost:8443/metrics
```

## 📝 Примеры LabelMapping

### Готовые ноды
```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: ready-nodes
spec:
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    node-status: ready
EOF
```

### NotReady ноды
```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: notready-nodes
spec:
  conditions:
    - type: NodeReady
      status: "False"
  labels:
    node-status: not-ready
    maintenance-mode: "true"
EOF
```

### Worker ноды
```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: worker-ready
spec:
  nodeSelector:
    node-role.kubernetes.io/worker: ""
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    worker-ready: "true"
EOF
```

### Linux ноды
```bash
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: linux-ready
spec:
  nodeSelector:
    kubernetes.io/os: linux
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    linux-ready: "true"
EOF
```

## 🧹 Очистка

```bash
# Удалить все LabelMapping
kubectl delete labelmappings --all

# Удалить лейблы со всех нод (замените на свои)
kubectl label nodes --all \
  node-status- \
  maintenance-mode- \
  worker-ready- \
  linux-ready- \
  auto-labeled-

# Удалить контроллер
make undeploy

# Удалить CRD (удалит все LabelMapping!)
make uninstall

# Удалить namespace
kubectl delete namespace k8s-node-labeler-system
```

## 🔧 Разработка

```bash
# Обновить зависимости
go mod tidy

# Форматирование
go fmt ./...
make fmt

# Проверка
go vet ./...
make vet

# Сгенерировать код
make generate

# Сгенерировать манифесты
make manifests

# Полная пересборка
make manifests generate fmt vet build

# Запустить тесты
make test

# Запустить e2e тесты
make test-e2e
```

## 🐳 Docker

```bash
# Собрать образ
docker build -t k8s-node-labeler:dev .
make docker-build IMG=k8s-node-labeler:dev

# Посмотреть образы
docker images | grep k8s-node-labeler

# Запустить контейнер локально (для теста)
docker run --rm k8s-node-labeler:dev --help

# Multi-platform build
make docker-buildx IMG=k8s-node-labeler:dev \
  PLATFORMS=linux/amd64,linux/arm64
```

## 🎯 Kind

```bash
# Создать кластер
kind create cluster --name node-labeler-test

# С несколькими worker нодами
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# Загрузить образ
kind load docker-image k8s-node-labeler:dev

# Удалить кластер
kind delete cluster --name node-labeler-test
```

## 🔄 Обновление

```bash
# Обновить образ
export IMG=docker.io/your-username/k8s-node-labeler:v0.0.2
make docker-build docker-push IMG=$IMG

# Обновить deployment
kubectl set image deployment/k8s-node-labeler-controller-manager \
  -n k8s-node-labeler-system \
  manager=$IMG

# Или через make
make deploy IMG=$IMG

# Проверить rollout
kubectl rollout status deployment/k8s-node-labeler-controller-manager \
  -n k8s-node-labeler-system
```

## 📊 Мониторинг

```bash
# Watch за LabelMapping
kubectl get labelmappings -w

# Watch за нодами
kubectl get nodes -w

# Watch за pod контроллера
kubectl get pods -n k8s-node-labeler-system -w

# Top ресурсов
kubectl top pod -n k8s-node-labeler-system
kubectl top node

# Логи с timestamps
kubectl logs -n k8s-node-labeler-system \
  deployment/k8s-node-labeler-controller-manager \
  --timestamps=true

# Логи за последние 5 минут
kubectl logs -n k8s-node-labeler-system \
  deployment/k8s-node-labeler-controller-manager \
  --since=5m
```

## 🧪 Тестирование

```bash
# Быстрый тест
kubectl apply -f - <<EOF
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: test
spec:
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    test: "true"
EOF

# Подождать 5 секунд
sleep 5

# Проверить
kubectl get labelmapping test
kubectl get nodes -L test

# Очистить
kubectl delete labelmapping test
kubectl label nodes --all test-
```

## 🔐 RBAC Debug

```bash
# Проверить ServiceAccount
kubectl get sa -n k8s-node-labeler-system

# Проверить ClusterRole
kubectl get clusterrole | grep k8s-node-labeler

# Проверить ClusterRoleBinding
kubectl get clusterrolebinding | grep k8s-node-labeler

# Права ServiceAccount
kubectl describe clusterrole k8s-node-labeler-manager-role

# Кто может делать что
kubectl auth can-i --list \
  --as=system:serviceaccount:k8s-node-labeler-system:k8s-node-labeler-controller-manager
```

## 💡 Полезные алиасы

```bash
# Добавьте в ~/.bashrc или ~/.zshrc

# Короткие алиасы
alias k='kubectl'
alias kgn='kubectl get nodes'
alias kgl='kubectl get labelmappings'
alias kdl='kubectl describe labelmapping'

# Оператор
alias klogs='kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager'
alias kpods='kubectl get pods -n k8s-node-labeler-system'

# Ноды с лейблами
alias knl='kubectl get nodes --show-labels'

# Watch
alias wkgl='watch kubectl get labelmappings'
alias wkgn='watch kubectl get nodes'
```

## 📚 Документация

```bash
# Локальная документация
cat README.md
cat USAGE.md
cat DEPLOYMENT.md
cat QUICKSTART.md

# API Reference
kubectl explain labelmapping
kubectl explain labelmapping.spec
kubectl explain labelmapping.spec.conditions
kubectl explain labelmapping.status
```

---

**Tip**: Сохраните эту шпаргалку для быстрого доступа к командам!

