# Kubernetes Manifests

This directory contains Kubernetes manifests for deploying the Orchestrator Platform.

## Directory Structure

```
k8s/
├── base/                    # Base Kubernetes resources
│   ├── namespace.yaml       # Namespace definition
│   ├── rbac.yaml           # ServiceAccount, Role, RoleBinding
│   ├── configmap.yaml      # Configuration
│   ├── service.yaml        # Services (stable, canary, main)
│   ├── deployment.yaml     # Standard deployment
│   ├── daemonset.yaml      # Probe agent DaemonSet
│   ├── ingress.yaml        # Ingress configuration
│   └── kustomization.yaml  # Kustomize base
│
├── overlays/               # Environment-specific overlays
│   ├── dev/               # Development environment
│   ├── staging/           # Staging environment
│   └── prod/              # Production environment
│
├── argo-rollouts/         # Argo Rollouts for canary deployments
│   ├── rollout.yaml       # Rollout definition with canary strategy
│   ├── analysis-template.yaml  # Analysis templates
│   ├── experiment.yaml    # Experiment definition
│   └── kustomization.yaml
│
└── argocd/                # ArgoCD GitOps resources
    ├── appproject.yaml    # ArgoCD project
    ├── application.yaml   # ArgoCD application
    ├── application-rollout.yaml  # ArgoCD application for rollouts
    ├── application-set.yaml      # ApplicationSet for multi-env
    ├── notifications.yaml # Notification configuration
    └── kustomization.yaml
```

## Usage

### Deploy with Kubectl

```bash
# Deploy base resources
kubectl apply -k k8s/base

# Deploy to specific environment
kubectl apply -k k8s/overlays/dev
kubectl apply -k k8s/overlays/staging
kubectl apply -k k8s/overlays/prod
```

### Deploy with Argo Rollouts

```bash
# Install Argo Rollouts controller
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Deploy rollout
kubectl apply -k k8s/argo-rollouts

# Monitor rollout
kubectl argo rollouts get rollout orchestrator -n orchestrator --watch
```

### Deploy with ArgoCD

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Deploy ArgoCD applications
kubectl apply -f k8s/argocd/appproject.yaml
kubectl apply -f k8s/argocd/application.yaml
kubectl apply -f k8s/argocd/application-set.yaml
```

## Configuration

### Image Registry

Update the image registry in kustomization.yaml files:

```yaml
images:
  - name: orchestrator
    newName: your-registry/orchestrator
    newTag: latest
  - name: probe-agent
    newName: your-registry/probe-agent
    newTag: latest
```

### Environment Variables

Update environment variables in configmap.yaml:

```yaml
data:
  ORCHESTRATOR_PORT: "8080"
  ZOOKEEPER_URL: "zookeeper-1:2181,zookeeper-2:2181,zookeeper-3:2181"
  LOG_LEVEL: "info"
```

### Resource Limits

Adjust resource limits based on your needs:

```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

## Canary Deployment Strategy

The canary deployment follows these steps:

1. Deploy canary with 10% traffic → Pause 5 minutes
2. Run analysis (success rate, latency)
3. Increase to 25% → Pause 5 minutes
4. Increase to 50% → Pause 10 minutes
5. Run analysis again
6. Increase to 75% → Pause 5 minutes
7. Full promotion to 100%

### Manual Controls

```bash
# Promote canary
kubectl argo rollouts promote orchestrator -n orchestrator

# Abort rollout
kubectl argo rollouts abort orchestrator -n orchestrator

# Pause/Resume
kubectl argo rollouts pause orchestrator -n orchestrator
kubectl argo rollouts resume orchestrator -n orchestrator
```

## Troubleshooting

### Check Rollout Status

```bash
kubectl argo rollouts status orchestrator -n orchestrator
kubectl argo rollouts get rollout orchestrator -n orchestrator
```

### View Analysis Results

```bash
kubectl get analysisruns -n orchestrator
kubectl describe analysisrun <name> -n orchestrator
```

### Check Logs

```bash
kubectl logs -n orchestrator -l app.kubernetes.io/name=orchestrator
kubectl logs -n orchestrator -l app.kubernetes.io/name=probe-agent
```

## References

- [Kustomize Documentation](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
- [Argo Rollouts Documentation](https://argoproj.github.io/argo-rollouts/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
