# Kubernetes Node Labeler

A Kubernetes controller that automatically manages node labels based on node conditions. This controller watches for node state changes and applies or removes labels according to defined rules.

## Features

- **Condition-based labeling**: Automatically add/remove labels based on node conditions (e.g., NodeReady)
- **Node selector support**: Target specific nodes using label selectors
- **Automatic cleanup**: Labels are removed when conditions no longer match
- **Real-time synchronization**: Watches for node changes and reconciles every 30 seconds
- **Status tracking**: Monitor matched and labeled nodes through CR status

## Use Cases

- Label nodes as `workload-ready` only when they are in Ready state
- Automatically mark nodes for specific workloads based on their health
- Implement custom scheduling constraints based on node conditions
- Separate healthy and unhealthy nodes for different workload types

## Installation

### Prerequisites

- Kubernetes cluster 1.19+
- kubectl configured to access your cluster
- Appropriate RBAC permissions to create CRDs, ClusterRoles, and Deployments

### Deploy the Controller

```bash
# Install all resources (CRD, RBAC, Controller)
kubectl apply -f https://raw.githubusercontent.com/Arsolitt/k8s-node-labeler/main/dist/install.yaml -n node-labeler

# Verify the controller is running
kubectl get pods -n node-labeler -l app.kubernetes.io/name=k8s-node-labeler
```

### Uninstall

```bash
kubectl delete -f https://raw.githubusercontent.com/Arsolitt/k8s-node-labeler/main/dist/install.yaml -n node-labeler
```

## Usage

### Basic Example

Create a `LabelMapping` resource to label all ready nodes:

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
    workload-ready: "true"
```

Apply the resource:

```bash
kubectl apply -f labelmapping.yaml
```

### Example with Node Selector

Label only Linux nodes that are ready:

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
    linux-ready: "true"
```

### Example: Label NotReady Nodes

Mark nodes that are not ready:

```yaml
apiVersion: labeler.arsolitt.tech/v1alpha1
kind: LabelMapping
metadata:
  name: notready-nodes-labeler
spec:
  conditions:
    - type: NodeReady
      status: "False"
  labels:
    node-status: notready
    workload-ready: "false"
```

## Custom Resource Specification

### LabelMapping Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `nodeSelector` | map[string]string | No | Label selector to filter nodes. If empty, applies to all nodes |
| `conditions` | []NodeCondition | Yes | List of conditions that must all match (AND logic) |
| `labels` | map[string]string | Yes | Labels to add when conditions match, remove when they don't |

### NodeCondition

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Condition type to check. Currently supported: `NodeReady` |
| `status` | string | Yes | Expected condition status: `True` or `False` |

### Status Fields

The controller updates the status of each `LabelMapping`:

```yaml
status:
  matchedNodes: 3      # Number of nodes matching all conditions
  labeledNodes: 3      # Number of nodes where labels were applied/removed
  lastSyncTime: "2025-11-17T10:30:00Z"
  conditions:
    - type: Ready
      status: "True"
      reason: ReconciliationSucceeded
      message: "Successfully reconciled 3 nodes"
```

## Monitoring

### View LabelMapping Status

```bash
# List all label mappings with status
kubectl get labelmappings

# Get detailed information
kubectl describe labelmapping ready-nodes-labeler
```

### Check Controller Logs

```bash
kubectl logs -n node-labeler -l app.kubernetes.io/name=k8s-node-labeler -f
```

### View Applied Labels on Nodes

```bash
# Show all node labels
kubectl get nodes --show-labels

# Show specific labels
kubectl get nodes -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type==\"Ready\")].status,LABELS:.metadata.labels
```

## How It Works

1. **Watch**: Controller watches for changes in `LabelMapping` resources and Node objects
2. **Match**: For each node, checks if it matches the `nodeSelector` (if specified)
3. **Evaluate**: Evaluates all conditions defined in the `LabelMapping`
4. **Apply**: 
   - If **all conditions match**: adds the specified labels to the node
   - If **any condition doesn't match**: removes the specified labels from the node
5. **Reconcile**: Repeats the process every 30 seconds and on any node/LabelMapping changes

## RBAC Permissions

The controller requires the following permissions:

- **LabelMapping resources**: Full CRUD access
- **Nodes**: Read (get, list, watch) and Update (update, patch)

All required RBAC resources are included in the installation manifest.

## Configuration

### Controller Arguments

The controller supports the following command-line arguments:

- `--leader-elect`: Enable leader election (default: enabled)
- `--health-probe-bind-address`: Health probe endpoint address (default: `:8081`)
- `--metrics-bind-address`: Metrics endpoint address (default: `:8443`)

### Resource Limits

Default resource limits (can be adjusted in `config/manager/manager.yaml`):

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

## Development

### Build

```bash
# Build binary
make build

# Build Docker image
make docker-build IMG=ghcr.io/arsolitt/k8s-node-labeler:latest

# Push Docker image
make docker-push IMG=ghcr.io/arsolitt/k8s-node-labeler:latest
```

### Run Locally

```bash
# Install CRDs
make install

# Run controller locally
make run
```

### Generate Manifests

```bash
# Generate CRDs and RBAC
make manifests

# Generate installation YAML
kustomize build config/default > dist/install.yaml
```

## Troubleshooting

### Labels Not Applied

1. Check if the LabelMapping resource exists and is valid:
   ```bash
   kubectl get labelmappings
   kubectl describe labelmapping <name>
   ```

2. Verify the controller is running:
   ```bash
   kubectl get pods -n node-labeler -l app.kubernetes.io/name=k8s-node-labeler
   ```

3. Check controller logs for errors:
   ```bash
   kubectl logs -n node-labeler -l app.kubernetes.io/name=k8s-node-labeler
   ```

4. Ensure nodes match the `nodeSelector` (if specified):
   ```bash
   kubectl get nodes --show-labels
   ```

### Controller Crashes

Check the logs and events:

```bash
kubectl logs -n node-labeler -l app.kubernetes.io/name=k8s-node-labeler --previous
kubectl get events -n node-labeler --sort-by='.lastTimestamp'
```

### Permission Issues

Verify RBAC resources are created:

```bash
kubectl get clusterrole,clusterrolebinding,serviceaccount -n node-labeler | grep node-labeler
```

## License

Apache License 2.0

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

