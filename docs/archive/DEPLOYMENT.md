# Orchestrator Platform - Deployment Guide

This guide provides detailed instructions for deploying the Orchestrator Platform using various strategies including canary deployments with ArgoCD and Argo Rollouts.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Quick Start](#quick-start)
4. [Deployment Strategies](#deployment-strategies)
5. [Canary Deployment with Argo Rollouts](#canary-deployment-with-argo-rollouts)
6. [ArgoCD GitOps Deployment](#argocd-gitops-deployment)
7. [Helm Chart Deployment](#helm-chart-deployment)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Rollback Procedures](#rollback-procedures)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before deploying the Orchestrator Platform, ensure you have the following:

### Required Tools

- **Kubernetes Cluster**: v1.24+ (EKS, GKE, AKS, or self-hosted)
- **kubectl**: v1.24+
- **Helm**: v3.10+
- **ArgoCD**: v2.8+ (for GitOps deployments)
- **Argo Rollouts**: v1.6+ (for canary deployments)
- **Docker**: v20.10+ (for building images)

### Install Required Components

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install ArgoCD CLI
curl -sSL -o argocd-linux-amd64 https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
sudo install -m 555 argocd-linux-amd64 /usr/local/bin/argocd

# Install Argo Rollouts kubectl plugin
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x kubectl-argo-rollouts-linux-amd64
sudo mv kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts
```

---

## Architecture Overview

The Orchestrator Platform consists of:

1. **Orchestrator Application**: Central application server for report aggregation and API
2. **Probe Agents**: DaemonSet collecting metrics from each node
3. **Zookeeper Cluster**: Coordination service
4. **Mesos Master**: Resource management
5. **Marathon**: Service orchestration
6. **Monitoring Stack**: Prometheus, Grafana, ELK

### Deployment Components

```
┌─────────────────────────────────────────────────────────┐
│                    ArgoCD (GitOps)                       │
│              Continuous Deployment & Sync                │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Argo Rollouts (Canary)                      │
│         Progressive Delivery with Analysis               │
└──────────────────────┬──────────────────────────────────┘
                       │
      ┌────────────────┼────────────────┐
      ▼                ▼                ▼
┌──────────┐  ┌──────────────┐  ┌──────────┐
│ Stable   │  │   Canary     │  │Analysis  │
│ Service  │  │   Service    │  │Templates │
│ (90%)    │  │   (10%)      │  │          │
└──────────┘  └──────────────┘  └──────────┘
```

---

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/orchestrator.git
cd orchestrator
```

### 2. Build Docker Images

```bash
# Build orchestrator image
docker build -t your-registry/orchestrator:latest .

# Build probe-agent image
docker build -t your-registry/probe-agent:latest -f cmd/probe-agent/Dockerfile .

# Push images to registry
docker push your-registry/orchestrator:latest
docker push your-registry/probe-agent:latest
```

### 3. Basic Kubernetes Deployment

```bash
# Create namespace
kubectl create namespace orchestrator

# Deploy using kustomize
kubectl apply -k k8s/base

# Verify deployment
kubectl get pods -n orchestrator
kubectl get svc -n orchestrator
```

---

## Deployment Strategies

### Blue/Green Deployment

Blue/Green deployment maintains two identical environments (blue and green). Traffic is switched from one to the other after validation.

**Characteristics:**
- Zero downtime
- Instant rollback capability
- Requires 2x resources
- Simple traffic switching

**When to Use:**
- Critical production deployments
- When instant rollback is required
- When you have sufficient cluster resources

### Canary Deployment

Canary deployment gradually shifts traffic from the old version to the new version, allowing real-time validation with production traffic.

**Characteristics:**
- Gradual traffic shifting (10% → 25% → 50% → 75% → 100%)
- Real-time analysis and validation
- Automatic rollback on failures
- Minimal blast radius

**When to Use:**
- Production deployments with high availability requirements
- When you need gradual validation with real traffic
- When you want to minimize risk

---

## Canary Deployment with Argo Rollouts

### 1. Install Argo Rollouts

```bash
# Install Argo Rollouts controller
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Verify installation
kubectl get pods -n argo-rollouts
```

### 2. Deploy Analysis Templates

Analysis templates define success criteria for canary validation:

```bash
# Deploy analysis templates
kubectl apply -f k8s/argo-rollouts/analysis-template.yaml

# Verify templates
kubectl get analysistemplates -n orchestrator
```

### 3. Deploy Rollout

```bash
# Deploy the rollout
kubectl apply -f k8s/argo-rollouts/rollout.yaml

# Watch rollout progress
kubectl argo rollouts get rollout orchestrator -n orchestrator --watch

# Check rollout status
kubectl argo rollouts status orchestrator -n orchestrator
```

### 4. Monitor Canary Progress

The canary deployment follows these steps:

1. **Step 1**: Deploy canary with 10% traffic (5 min pause)
2. **Step 2**: Run analysis (success rate, latency)
3. **Step 3**: Increase to 25% traffic (5 min pause)
4. **Step 4**: Increase to 50% traffic (10 min pause)
5. **Step 5**: Run analysis again
6. **Step 6**: Increase to 75% traffic (5 min pause)
7. **Step 7**: Full promotion to 100%

```bash
# Monitor rollout in real-time
kubectl argo rollouts dashboard

# View analysis runs
kubectl get analysisruns -n orchestrator

# Check analysis results
kubectl describe analysisrun <analysis-run-name> -n orchestrator
```

### 5. Manual Controls

```bash
# Promote canary manually (skip waiting)
kubectl argo rollouts promote orchestrator -n orchestrator

# Abort rollout (rollback to stable)
kubectl argo rollouts abort orchestrator -n orchestrator

# Retry a failed rollout
kubectl argo rollouts retry orchestrator -n orchestrator

# Pause rollout
kubectl argo rollouts pause orchestrator -n orchestrator

# Resume paused rollout
kubectl argo rollouts resume orchestrator -n orchestrator
```

### 6. Update Image for Canary

```bash
# Update to new image version
kubectl argo rollouts set image orchestrator \
  orchestrator=your-registry/orchestrator:v2.0.0 \
  -n orchestrator

# Or patch the rollout
kubectl patch rollout orchestrator -n orchestrator \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/image", "value":"your-registry/orchestrator:v2.0.0"}]'
```

---

## ArgoCD GitOps Deployment

### 1. Install ArgoCD

```bash
# Create ArgoCD namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=available --timeout=600s deployment/argocd-server -n argocd

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Port forward to access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Access ArgoCD UI at: https://localhost:8080
- Username: `admin`
- Password: (from previous command)

### 2. Create ArgoCD Project

```bash
# Apply ArgoCD project
kubectl apply -f k8s/argocd/appproject.yaml

# Verify project
argocd proj get orchestrator
```

### 3. Deploy Applications via ArgoCD

```bash
# Deploy base application
kubectl apply -f k8s/argocd/application.yaml

# Deploy rollout application
kubectl apply -f k8s/argocd/application-rollout.yaml

# Deploy multi-environment setup
kubectl apply -f k8s/argocd/application-set.yaml

# Verify applications
argocd app list
argocd app get orchestrator
```

### 4. Sync Applications

```bash
# Manual sync
argocd app sync orchestrator

# Enable auto-sync
argocd app set orchestrator --sync-policy automated

# View sync status
argocd app get orchestrator --refresh

# View sync history
argocd app history orchestrator
```

### 5. Configure Notifications

```bash
# Apply notification configuration
kubectl apply -f k8s/argocd/notifications.yaml

# Test Slack notification
argocd app create test-app --upsert \
  --repo https://github.com/your-org/orchestrator.git \
  --path k8s/base \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace orchestrator
```

---

## Helm Chart Deployment

### 1. Install with Default Values

```bash
# Add Argo Rollouts Helm repository
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

# Install orchestrator with default values
helm install orchestrator ./helm/orchestrator \
  --namespace orchestrator \
  --create-namespace

# Verify installation
helm list -n orchestrator
kubectl get all -n orchestrator
```

### 2. Environment-Specific Deployments

#### Development Environment

```bash
helm install orchestrator-dev ./helm/orchestrator \
  --namespace orchestrator-dev \
  --create-namespace \
  --values ./helm/orchestrator/values-dev.yaml \
  --set orchestrator.image.tag=dev
```

#### Staging Environment

```bash
helm install orchestrator-staging ./helm/orchestrator \
  --namespace orchestrator-staging \
  --create-namespace \
  --values ./helm/orchestrator/values-staging.yaml \
  --set orchestrator.image.tag=staging
```

#### Production Environment

```bash
helm install orchestrator ./helm/orchestrator \
  --namespace orchestrator \
  --create-namespace \
  --values ./helm/orchestrator/values-prod.yaml \
  --set orchestrator.image.tag=latest
```

### 3. Upgrade Existing Deployment

```bash
# Upgrade with new image version
helm upgrade orchestrator ./helm/orchestrator \
  --namespace orchestrator \
  --values ./helm/orchestrator/values-prod.yaml \
  --set orchestrator.image.tag=v2.0.0 \
  --reuse-values

# Upgrade with custom values
helm upgrade orchestrator ./helm/orchestrator \
  --namespace orchestrator \
  --set orchestrator.replicaCount=5 \
  --set orchestrator.resources.requests.cpu=1000m
```

### 4. Rollback

```bash
# View release history
helm history orchestrator -n orchestrator

# Rollback to previous version
helm rollback orchestrator -n orchestrator

# Rollback to specific revision
helm rollback orchestrator 3 -n orchestrator
```

### 5. Uninstall

```bash
# Uninstall release
helm uninstall orchestrator -n orchestrator

# Delete namespace
kubectl delete namespace orchestrator
```

---

## Monitoring and Observability

### Prometheus Metrics

The platform exposes metrics at `/metrics` endpoint:

```bash
# Port forward to Prometheus
kubectl port-forward -n orchestrator svc/prometheus 9090:9090

# Access Prometheus UI
open http://localhost:9090
```

**Key Metrics:**
- `http_requests_total` - Total HTTP requests
- `http_request_duration_milliseconds` - Request latency
- `container_cpu_usage_seconds_total` - CPU usage
- `container_memory_usage_bytes` - Memory usage

### Grafana Dashboards

```bash
# Port forward to Grafana
kubectl port-forward -n orchestrator svc/grafana 3000:3000

# Access Grafana UI (admin/admin)
open http://localhost:3000
```

### View Logs

```bash
# View orchestrator logs
kubectl logs -n orchestrator -l app.kubernetes.io/name=orchestrator --tail=100 -f

# View probe-agent logs
kubectl logs -n orchestrator -l app.kubernetes.io/name=probe-agent --tail=100 -f

# View rollout controller logs
kubectl logs -n argo-rollouts -l app.kubernetes.io/name=argo-rollouts --tail=100 -f
```

### Analysis Results

```bash
# List analysis runs
kubectl get analysisruns -n orchestrator

# View analysis details
kubectl describe analysisrun <name> -n orchestrator

# Get analysis metrics
kubectl get analysisrun <name> -n orchestrator -o jsonpath='{.status.metricResults}'
```

---

## Rollback Procedures

### Argo Rollouts Rollback

```bash
# Abort current rollout (automatic rollback)
kubectl argo rollouts abort orchestrator -n orchestrator

# The stable version will automatically take 100% traffic
# Verify rollback
kubectl argo rollouts status orchestrator -n orchestrator
```

### Helm Rollback

```bash
# View release history
helm history orchestrator -n orchestrator

# Rollback to previous version
helm rollback orchestrator -n orchestrator

# Verify rollback
helm list -n orchestrator
kubectl get pods -n orchestrator
```

### Manual Rollback

```bash
# Scale down canary pods
kubectl scale rollout orchestrator --replicas=0 -n orchestrator

# Update to previous image
kubectl argo rollouts set image orchestrator \
  orchestrator=your-registry/orchestrator:v1.0.0 \
  -n orchestrator

# Verify rollback
kubectl get pods -n orchestrator
```

---

## Troubleshooting

### Common Issues

#### 1. Rollout Stuck in Progressing State

**Problem**: Rollout doesn't progress past certain step

**Solution**:
```bash
# Check analysis status
kubectl get analysisruns -n orchestrator

# View analysis details
kubectl describe analysisrun <name> -n orchestrator

# Check Prometheus metrics
kubectl logs -n orchestrator -l app.kubernetes.io/name=orchestrator

# If analysis is failing, abort and investigate
kubectl argo rollouts abort orchestrator -n orchestrator
```

#### 2. Pods Not Starting

**Problem**: Pods stuck in `Pending` or `CrashLoopBackOff`

**Solution**:
```bash
# Check pod events
kubectl describe pod <pod-name> -n orchestrator

# Check pod logs
kubectl logs <pod-name> -n orchestrator

# Check resource availability
kubectl top nodes
kubectl describe nodes

# Check image pull issues
kubectl get events -n orchestrator --sort-by='.lastTimestamp'
```

#### 3. Service Mesh/Ingress Issues

**Problem**: Traffic not routing correctly to canary

**Solution**:
```bash
# Check services
kubectl get svc -n orchestrator

# Check ingress
kubectl get ingress -n orchestrator
kubectl describe ingress orchestrator-ingress -n orchestrator

# Test service connectivity
kubectl run -it --rm debug --image=busybox --restart=Never -- wget -O- http://orchestrator-service:8080/health
```

#### 4. ArgoCD Sync Issues

**Problem**: Application out of sync or failing to sync

**Solution**:
```bash
# View application details
argocd app get orchestrator

# Hard refresh
argocd app get orchestrator --hard-refresh

# Sync with prune
argocd app sync orchestrator --prune

# View sync logs
argocd app logs orchestrator
```

### Debug Commands

```bash
# Get all resources
kubectl get all -n orchestrator

# Describe rollout
kubectl describe rollout orchestrator -n orchestrator

# Get rollout events
kubectl get events -n orchestrator --field-selector involvedObject.name=orchestrator

# Check service endpoints
kubectl get endpoints -n orchestrator

# Test DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup orchestrator-service.orchestrator.svc.cluster.local
```

---

## Best Practices

1. **Use GitOps**: Always deploy via ArgoCD for version control and audit trail
2. **Enable Auto-Rollback**: Configure analysis templates to automatically detect failures
3. **Monitor Metrics**: Watch key metrics during canary deployments
4. **Small Steps**: Use smaller traffic increments (10%, 25%, 50%, 75%, 100%)
5. **Pause for Validation**: Include manual pause steps for critical deployments
6. **Test in Lower Environments**: Always test canary deployments in dev/staging first
7. **Document Rollback Procedures**: Ensure team knows how to rollback quickly
8. **Set Up Alerts**: Configure alerts for deployment failures
9. **Use Pod Disruption Budgets**: Ensure high availability during deployments
10. **Resource Limits**: Always set resource requests and limits

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/your-org/orchestrator/issues
- Documentation: https://github.com/your-org/orchestrator/blob/main/COMBINED_PRD.md
- Slack: #orchestrator-support

---

## References

- [Argo Rollouts Documentation](https://argoproj.github.io/argo-rollouts/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)
- [Progressive Delivery](https://www.weave.works/blog/what-is-progressive-delivery-all-about)
