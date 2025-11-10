# Руководство по использованию k8s-node-labeler

## Обзор

k8s-node-labeler автоматически управляет лейблами на нодах Kubernetes на основе определенных условий.

## Структура LabelMapping

Кастомный ресурс `LabelMapping` состоит из следующих полей:

### Spec

- **nodeSelector** (опционально): Map лейблов для выбора нод, к которым применяется правило
- **conditions**: Массив условий, которым должна соответствовать нода
  - **type**: Тип условия (`NodeReady`)
  - **status**: Ожидаемый статус (`True` или `False`)
- **labels**: Map лейблов, которые будут добавлены/удалены

### Status

- **matchedNodes**: Количество нод, соответствующих условиям
- **labeledNodes**: Количество нод, на которых были изменены лейблы
- **lastSyncTime**: Время последней синхронизации
- **conditions**: Статус ресурса

## Примеры

### 1. Базовый пример: Маркировка готовых нод

```yaml
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: ready-nodes-labeler
spec:
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    node-status: ready
```

Этот маппинг добавит лейбл `node-status=ready` на все ноды со статусом Ready.

### 2. Селективная маркировка: Только Linux ноды

```yaml
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: linux-ready-nodes
spec:
  nodeSelector:
    kubernetes.io/os: linux
  conditions:
    - type: NodeReady
      status: "True"
  labels:
    node-status: ready
    workload-ready: "true"
```

Применяется только к Linux нодам, которые находятся в статусе Ready.

### 3. Маркировка нод в режиме обслуживания

```yaml
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: maintenance-mode
spec:
  conditions:
    - type: NodeReady
      status: "False"
  labels:
    node-status: not-ready
    maintenance-mode: "true"
    workload-ready: "false"
```

Добавляет лейблы на ноды, которые не готовы (NotReady).

### 4. Worker ноды

```yaml
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
    worker-ready: "true"
    can-schedule-workloads: "true"
```

Маркирует только worker ноды, которые готовы к работе.

## Работа оператора

### Логика работы

1. Оператор получает все ноды кластера
2. Для каждой ноды проверяет соответствие nodeSelector (если указан)
3. Проверяет выполнение всех условий (логика AND)
4. Если условия выполнены - добавляет лейблы
5. Если условия не выполнены - удаляет лейблы
6. Обновляет статус LabelMapping

### Синхронизация

Оператор автоматически пересогласовывает состояние каждые 30 секунд, а также при:
- Создании/изменении LabelMapping ресурса
- Изменении состояния нод

## Команды управления

### Применить LabelMapping

```bash
kubectl apply -f labelmapping.yaml
```

### Просмотр статуса

```bash
kubectl get labelmappings
```

Вывод:
```
NAME                  MATCHED   LABELED   AGE
ready-nodes-labeler   3         3         5m
```

### Подробная информация

```bash
kubectl describe labelmapping ready-nodes-labeler
```

### Просмотр лейблов на нодах

```bash
kubectl get nodes --show-labels
```

### Удаление LabelMapping

```bash
kubectl delete labelmapping ready-nodes-labeler
```

**Важно:** При удалении LabelMapping лейблы с нод НЕ удаляются автоматически.

## Лучшие практики

### 1. Используйте осмысленные имена лейблов

```yaml
# Хорошо
labels:
  node-health-status: ready
  workload-schedulable: "true"

# Плохо
labels:
  status: ok
  flag: "1"
```

### 2. Избегайте конфликтов

Убедитесь, что разные LabelMapping не управляют одними и теми же лейблами на пересекающихся множествах нод.

### 3. Используйте nodeSelector

Ограничивайте область применения правил с помощью nodeSelector для повышения производительности:

```yaml
nodeSelector:
  environment: production
  node-role.kubernetes.io/worker: ""
```

### 4. Мониторинг статуса

Регулярно проверяйте статус LabelMapping:

```bash
kubectl get labelmappings -w
```

## Расширение функциональности

В будущих версиях планируется поддержка дополнительных условий:
- Использование CPU/Memory
- Наличие определенных тaint
- Версия Kubernetes
- Пользовательские метрики

## Troubleshooting

### Лейблы не применяются

1. Проверьте статус LabelMapping:
   ```bash
   kubectl describe labelmapping <name>
   ```

2. Проверьте логи оператора:
   ```bash
   kubectl logs -n k8s-node-labeler-system deployment/k8s-node-labeler-controller-manager
   ```

3. Убедитесь, что nodeSelector корректен:
   ```bash
   kubectl get nodes -l <your-selector>
   ```

### Ноды не соответствуют условиям

Проверьте статус нод:
```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[?\(@.type==\"Ready\"\)].status
```

## Примеры интеграции

### Использование с PodNodeSelector

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  annotations:
    scheduler.alpha.kubernetes.io/node-selector: "workload-ready=true"
```

Все поды в namespace `production` будут планироваться только на ноды с лейблом `workload-ready=true`.

### Использование с Node Affinity

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-status
                operator: In
                values:
                - ready
```

## Безопасность

Оператор требует следующих RBAC прав:
- Чтение всех нод кластера
- Изменение лейблов на нодах
- Управление LabelMapping ресурсами

Все права настроены в `config/rbac/role.yaml`.

