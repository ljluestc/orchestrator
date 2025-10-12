# Orchestrator Platform - Quick Start Deployment

## Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- 8GB+ RAM available
- 4+ CPU cores

## Step 1: Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

## Step 2: Install Argo Rollouts

```bash
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Wait for Argo Rollouts to be ready
kubectl wait --for=condition=available --timeout=300s deployment/argo-rollouts -n argo-rollouts
```

## Step 3: Create Orchestrator Namespace

```bash
kubectl create namespace orchestrator
```

## Step 4: Deploy Platform via ArgoCD

```bash
# Apply ArgoCD Application
kubectl apply -f k8s/argocd/applications/orchestrator-application.yaml

# Watch deployment progress
watch kubectl get applications -n argocd
```

## Step 5: Monitor Deployment

```bash
# Check all pods
kubectl get pods -n orchestrator

# Watch Argo Rollout progress
kubectl argo rollouts get rollout monitoring-app -n orchestrator --watch

# Check ArgoCD application status
kubectl get application orchestrator-platform -n argocd -o yaml
```

## Step 6: Access Services

### ArgoCD UI
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443 &
# https://localhost:8080
# Username: admin
# Password: (from step 1)
```

### Mesos Master
```bash
kubectl port-forward svc/mesos-master-lb -n orchestrator 5050:5050 &
# http://localhost:5050
```

### Marathon
```bash
kubectl port-forward svc/marathon -n orchestrator 8081:8080 &
# http://localhost:8081
```

### Monitoring App
```bash
kubectl port-forward svc/monitoring-app -n orchestrator 8082:8080 &
# http://localhost:8082/api/v1/topology
```

## Step 7: Verify Deployment

```bash
# Check all components are running
kubectl get statefulsets,deployments,daemonsets -n orchestrator

# Expected output:
# NAME                                READY   AGE
# statefulset.apps/mesos-master       3/3     5m
# statefulset.apps/zookeeper          3/3     5m
#
# NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
# deployment.apps/marathon        3/3     3            3           5m
# deployment.apps/monitoring-app  3/3     3            3           5m
#
# NAME                                  DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   AGE
# daemonset.apps/mesos-agent            3         3         3       3            3           5m
# daemonset.apps/monitoring-probe       3         3         3       3            3           5m

# Check Mesos cluster health
curl http://localhost:5050/health
# Expected: {"status": "OK"}

# Check monitoring app health
curl http://localhost:8082/health
# Expected: {"status": "healthy", ...}
```

## Troubleshooting

### Pods not starting
```bash
# Check pod logs
kubectl logs -n orchestrator <pod-name>

# Check events
kubectl get events -n orchestrator --sort-by='.lastTimestamp'
```

### ArgoCD sync issues
```bash
# Check application status
kubectl describe application orchestrator-platform -n argocd

# Force sync
kubectl patch application orchestrator-platform -n argocd --type merge -p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{"syncStrategy":{"hook":{}}}}}'
```

### Resource constraints
```bash
# Check node resources
kubectl top nodes

# Reduce replicas if needed
kubectl scale statefulset mesos-master -n orchestrator --replicas=1
kubectl scale deployment marathon -n orchestrator --replicas=1
```

## Next Steps

1. Deploy a test Marathon application
2. View topology in monitoring UI
3. Configure Prometheus and Grafana
4. Set up alerting
5. Review MASTER_PRD.txt for complete feature set

## Clean Up

```bash
# Delete application
kubectl delete application orchestrator-platform -n argocd

# Delete namespace
kubectl delete namespace orchestrator

# Delete ArgoCD (optional)
kubectl delete namespace argocd

# Delete Argo Rollouts (optional)
kubectl delete namespace argo-rollouts
```
