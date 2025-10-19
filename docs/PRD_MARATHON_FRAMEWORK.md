# Product Requirements Document: Marathon Framework & Container Management

**Project:** Orchestrator - Marathon Framework Implementation
**Document:** PRD_MARATHON_FRAMEWORK
**Version:** 1.0.0
**Date:** 2025-10-14
**Status:** ✅ READY FOR TASK-MASTER PARSING
**Dependencies:** Task 21 (App Backend Server) ✅ COMPLETED

---

## 1. EXECUTIVE SUMMARY

### 1.1 Overview
Implement the Marathon framework layer for the orchestrator platform, enabling container lifecycle management, auto-scaling, auto-healing, and zero-downtime deployments. This phase builds on the completed App Backend Server (Task 21) to provide a complete container orchestration solution.

### 1.2 Objectives
- Integrate Docker containerizer for container lifecycle management
- Implement auto-scaling based on metrics and policies
- Add auto-healing with health check monitoring
- Enable zero-downtime rolling updates with rollback
- Provide service discovery and load balancing
- Achieve 99.9% uptime for containerized applications

### 1.3 Scope
This PRD covers Tasks 5, 8, 9, 10, 11, and 12 from the MASTER_PRD:
- **Task 5:** Docker Containerizer Integration
- **Task 8:** Marathon Auto-Scaling & Auto-Healing
- **Task 9:** Marathon Rolling Updates
- **Task 10:** Marathon Health Checks
- **Task 11:** Service Discovery
- **Task 12:** Load Balancing

---

## 2. BACKGROUND & CONTEXT

### 2.1 Current State
✅ **Completed:**
- Task 21: App Backend Server with REST API and WebSocket
- Task 1: Mesos Master HA Setup
- Task 2: Zookeeper Cluster
- Task 3: Mesos Agent Deployment
- Task 7: Marathon Framework Base

### 2.2 Problem Statement
Current system can deploy containers but lacks:
1. **Automated scaling** - Manual intervention required for load changes
2. **Self-healing** - No automatic recovery from failures
3. **Safe deployments** - No rolling update mechanism
4. **Service discovery** - Applications can't find each other
5. **Load distribution** - No intelligent traffic routing

### 2.3 Success Criteria
- Containers scale automatically based on CPU/memory metrics
- Failed containers restart within 30 seconds
- Zero-downtime deployments with rollback capability
- 99.9% service availability
- < 100ms service discovery lookup time

---

## 3. TASK BREAKDOWN

## TASK 5: Docker Containerizer Integration

### Description
Build the Docker containerizer to manage container lifecycle operations (create, start, stop, restart, delete) and integrate with Mesos agents.

### Implementation Details

**File:** `pkg/containerizer/docker_containerizer.go`

**Components:**
1. **Docker Client Wrapper**
   - Initialize Docker API client
   - Handle connection pooling
   - Manage API version compatibility

2. **Container Lifecycle Operations**
   - `Create()` - Create container from image
   - `Start()` - Start stopped container
   - `Stop()` - Stop running container (graceful)
   - `Kill()` - Force stop container
   - `Restart()` - Restart container
   - `Remove()` - Delete container
   - `Inspect()` - Get container details

3. **Resource Management**
   - Set CPU limits (shares, quota, period)
   - Set memory limits (hard, soft, swap)
   - Configure network mode (bridge, host, none)
   - Mount volumes and bind mounts

4. **Health Monitoring**
   - Stream container logs
   - Get container stats (CPU, memory, network, disk)
   - Check container status
   - Handle container events

5. **Image Management**
   - Pull images from registry
   - Tag and push images
   - List and remove images
   - Handle authentication

**API Interface:**
```go
type Containerizer interface {
    Create(ctx context.Context, config ContainerConfig) (string, error)
    Start(ctx context.Context, containerID string) error
    Stop(ctx context.Context, containerID string, timeout time.Duration) error
    Kill(ctx context.Context, containerID string) error
    Restart(ctx context.Context, containerID string) error
    Remove(ctx context.Context, containerID string, force bool) error
    Inspect(ctx context.Context, containerID string) (*ContainerInfo, error)
    Logs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error)
    Stats(ctx context.Context, containerID string) (*ContainerStats, error)
    List(ctx context.Context, filters map[string]string) ([]ContainerInfo, error)
}
```

**Configuration:**
```go
type ContainerConfig struct {
    Name       string
    Image      string
    Cmd        []string
    Entrypoint []string
    Env        []string
    Labels     map[string]string
    CPUShares  int64
    CPUQuota   int64
    Memory     int64
    MemorySwap int64
    NetworkMode string
    Volumes    []VolumeMount
    Ports      []PortMapping
}
```

### Test Strategy
- Unit tests for each lifecycle operation
- Integration tests with actual Docker daemon
- Error handling tests (daemon down, OOM, etc.)
- Resource limit validation tests
- Concurrent container operation tests

### Success Metrics
- Container start time < 5 seconds
- 100% success rate for lifecycle operations
- Proper cleanup on errors
- < 1% CPU overhead

---

## TASK 8: Marathon Auto-Scaling & Auto-Healing

### Description
Implement automatic horizontal scaling based on metrics and auto-healing to restart failed containers automatically.

### Implementation Details

**Files:**
- `pkg/marathon/autoscaler.go`
- `pkg/marathon/autohealer.go`

### 8.1 Auto-Scaler Component

**Features:**
1. **Metrics Collection**
   - CPU usage per container
   - Memory usage per container
   - Request rate (if available)
   - Custom application metrics

2. **Scaling Policies**
   - Target-based scaling (maintain 70% CPU)
   - Step scaling (add 2 instances at 80% CPU)
   - Scheduled scaling (scale up at 9am)
   - Min/max instance limits

3. **Scale-up Logic**
   - Calculate desired instance count
   - Validate against max instances
   - Gradually add instances (not all at once)
   - Wait for new instances to be healthy

4. **Scale-down Logic**
   - Calculate desired instance count
   - Validate against min instances
   - Gracefully drain connections
   - Remove excess instances

5. **Cooldown Periods**
   - Scale-up cooldown: 60 seconds
   - Scale-down cooldown: 300 seconds
   - Prevent flapping

**API:**
```go
type AutoScaler struct {
    app         *Application
    metrics     MetricsCollector
    policies    []ScalingPolicy
    minInstances int
    maxInstances int
    cooldown    time.Duration
}

type ScalingPolicy struct {
    MetricName  string
    Threshold   float64
    Comparison  ComparisonOperator
    Action      ScalingAction
    Adjustment  int
}

func (s *AutoScaler) Start(ctx context.Context) error
func (s *AutoScaler) Stop() error
func (s *AutoScaler) Evaluate() (*ScalingDecision, error)
func (s *AutoScaler) ScaleUp(count int) error
func (s *AutoScaler) ScaleDown(count int) error
```

### 8.2 Auto-Healer Component

**Features:**
1. **Health Monitoring**
   - Monitor container status
   - Track health check failures
   - Detect OOM kills
   - Detect node failures

2. **Failure Detection**
   - Container exit codes
   - Health check timeouts
   - Consecutive failure threshold
   - Grace period for startup

3. **Recovery Actions**
   - Restart failed containers
   - Reschedule on different nodes
   - Apply exponential backoff
   - Notify operators on repeated failures

4. **Recovery Policies**
   - Max restart attempts: 5
   - Backoff interval: 1s, 2s, 4s, 8s, 16s
   - Reset after: 10 minutes healthy
   - Alert threshold: 3 failures in 5 minutes

**API:**
```go
type AutoHealer struct {
    app             *Application
    healthChecker   HealthChecker
    restartPolicy   RestartPolicy
    failureHistory  map[string][]time.Time
}

type RestartPolicy struct {
    MaxAttempts     int
    BackoffMultiplier float64
    InitialInterval time.Duration
    MaxInterval     time.Duration
    ResetInterval   time.Duration
}

func (h *AutoHealer) Start(ctx context.Context) error
func (h *AutoHealer) Stop() error
func (h *AutoHealer) MonitorHealth() error
func (h *AutoHealer) HandleFailure(containerID string) error
func (h *AutoHealer) RestartContainer(containerID string) error
```

### Test Strategy
- Unit tests for scaling calculations
- Unit tests for failure detection
- Integration tests with metrics
- Cooldown period tests
- Backoff algorithm tests
- Load tests with varying metrics

### Success Metrics
- Scale-up response time < 60 seconds
- Scale-down response time < 5 minutes
- Container restart time < 30 seconds
- Zero flapping incidents
- 99.9% application availability

---

## TASK 9: Marathon Rolling Updates

### Description
Implement zero-downtime rolling updates with canary deployments, health validation, and automatic rollback on failures.

### Implementation Details

**File:** `pkg/marathon/rolling_updater.go`

**Features:**

1. **Update Strategies**
   - **RollingUpdate:** Replace instances gradually
   - **BlueGreen:** Full environment switch
   - **Canary:** Test with small percentage first

2. **Rolling Update Flow**
   ```
   1. Validate new version configuration
   2. Calculate batch size (e.g., 25% of instances)
   3. For each batch:
      a. Create new instances with new version
      b. Wait for instances to be healthy
      c. Remove old instances from load balancer
      d. Gracefully stop old instances
      e. Validate metrics (error rate, latency)
      f. Continue or rollback
   4. Complete deployment
   ```

3. **Canary Deployment**
   - Deploy to small percentage (e.g., 10%)
   - Monitor metrics for duration (e.g., 5 minutes)
   - Auto-rollback if error rate increases
   - Gradually increase percentage
   - Full rollout on success

4. **Health Validation**
   - HTTP health checks
   - TCP connection checks
   - Command execution checks
   - Custom validation scripts

5. **Rollback Mechanism**
   - Automatic rollback on failures
   - Manual rollback command
   - Keep previous version running
   - Instant traffic switch

6. **Deployment Configuration**
   - Max surge: 25% (extra instances during update)
   - Max unavailable: 25% (allowed downtime)
   - Min healthy percentage: 75%
   - Health check grace period: 30s
   - Update batch size: 1-50%

**API:**
```go
type RollingUpdater struct {
    app              *Application
    updateConfig     UpdateConfig
    healthChecker    HealthChecker
    loadBalancer     LoadBalancer
}

type UpdateConfig struct {
    Strategy          UpdateStrategy
    MaxSurge          int
    MaxUnavailable    int
    MinHealthyPercent int
    HealthCheckGrace  time.Duration
    BatchSize         int
    BatchInterval     time.Duration
    CanaryPercent     int
    CanaryDuration    time.Duration
    AutoRollback      bool
}

type UpdateStrategy string

const (
    StrategyRollingUpdate UpdateStrategy = "RollingUpdate"
    StrategyBlueGreen     UpdateStrategy = "BlueGreen"
    StrategyCanary        UpdateStrategy = "Canary"
)

func (u *RollingUpdater) Update(ctx context.Context, newVersion AppVersion) error
func (u *RollingUpdater) Rollback(ctx context.Context) error
func (u *RollingUpdater) GetStatus() *UpdateStatus
func (u *RollingUpdater) Pause() error
func (u *RollingUpdater) Resume() error
```

### Test Strategy
- Unit tests for batch calculations
- Integration tests for each strategy
- Rollback scenario tests
- Concurrent update tests
- Failure injection tests
- End-to-end deployment tests

### Success Metrics
- Zero downtime during updates
- Rollback time < 60 seconds
- Update success rate > 99%
- Automatic rollback on 5xx errors
- Max 1% request failures during update

---

## TASK 10: Marathon Health Checks

### Description
Implement comprehensive health checking system with HTTP, TCP, and command-based checks.

### Implementation Details

**File:** `pkg/marathon/health_checker.go`

**Health Check Types:**

1. **HTTP Health Check**
   - GET request to endpoint
   - Expected status code (200, 201, etc.)
   - Optional response body validation
   - Timeout: 10 seconds

2. **TCP Health Check**
   - Connect to port
   - Optional send/expect protocol
   - Timeout: 5 seconds

3. **Command Health Check**
   - Execute command in container
   - Check exit code (0 = healthy)
   - Timeout: 30 seconds

4. **Custom Health Check**
   - User-defined script
   - JSON response validation

**Configuration:**
```go
type HealthCheck struct {
    Type              HealthCheckType
    Protocol          string
    Path              string
    Port              int
    Command           []string
    IntervalSeconds   int
    TimeoutSeconds    int
    GracePeriodSeconds int
    MaxConsecutiveFailures int
}

const (
    HealthCheckHTTP    HealthCheckType = "HTTP"
    HealthCheckTCP     HealthCheckType = "TCP"
    HealthCheckCommand HealthCheckType = "COMMAND"
)
```

**State Machine:**
```
Starting → Healthy → Unhealthy → Dead
   ↓          ↓          ↓
Grace → Healthy ← Recovery
```

### Test Strategy
- Unit tests for each health check type
- Integration tests with real containers
- Timeout handling tests
- Grace period tests
- Failure threshold tests

### Success Metrics
- Health check overhead < 1% CPU
- False positive rate < 0.1%
- Detection time < 30 seconds

---

## TASK 11: Service Discovery

### Description
Implement service discovery mechanism for applications to find and connect to each other dynamically.

### Implementation Details

**File:** `pkg/marathon/service_discovery.go`

**Features:**

1. **Service Registration**
   - Auto-register on container start
   - Update on IP/port changes
   - Deregister on container stop
   - Metadata and tags

2. **Service Discovery Backends**
   - Consul
   - etcd
   - Zookeeper
   - DNS-based

3. **DNS Integration**
   - Create DNS records for services
   - Format: `service.marathon.domain`
   - A records for IPs
   - SRV records for ports

4. **HTTP API**
   - List services: `GET /v1/services`
   - Get service: `GET /v1/services/{name}`
   - Service instances: `GET /v1/services/{name}/instances`

**API:**
```go
type ServiceDiscovery struct {
    backend      DiscoveryBackend
    dnsServer    DNSServer
    ttl          time.Duration
}

type Service struct {
    Name      string
    ID        string
    Address   string
    Port      int
    Tags      []string
    Metadata  map[string]string
    Health    HealthStatus
}

func (sd *ServiceDiscovery) Register(service *Service) error
func (sd *ServiceDiscovery) Deregister(serviceID string) error
func (sd *ServiceDiscovery) Discover(serviceName string) ([]*Service, error)
func (sd *ServiceDiscovery) Watch(serviceName string) (<-chan []*Service, error)
```

### Test Strategy
- Registration/deregistration tests
- DNS resolution tests
- Service watch tests
- TTL expiration tests
- Multi-instance tests

### Success Metrics
- Lookup latency < 100ms
- Registration latency < 500ms
- 99.99% availability
- Correct results 100% of time

---

## TASK 12: Load Balancing

### Description
Implement intelligent load balancing with multiple algorithms and health-aware routing.

### Implementation Details

**File:** `pkg/marathon/load_balancer.go`

**Load Balancing Algorithms:**

1. **Round Robin**
   - Distribute requests evenly
   - Simple and fast

2. **Least Connections**
   - Route to instance with fewest connections
   - Good for long-lived connections

3. **Weighted Round Robin**
   - Distribute based on instance weights
   - Higher weight = more traffic

4. **IP Hash**
   - Consistent hashing based on client IP
   - Session affinity

5. **Least Response Time**
   - Route to fastest instance
   - Dynamic based on latency

**Features:**

1. **Health-Aware Routing**
   - Exclude unhealthy instances
   - Gradual warm-up for new instances
   - Slow start (gradually increase traffic)

2. **Connection Draining**
   - Gracefully remove instances
   - Wait for connections to complete
   - Timeout after 5 minutes

3. **Sticky Sessions**
   - Cookie-based affinity
   - Client IP affinity
   - Configurable TTL

4. **Traffic Management**
   - Rate limiting
   - Circuit breaker
   - Retry logic

**API:**
```go
type LoadBalancer struct {
    algorithm    BalancingAlgorithm
    backends     []*Backend
    healthChecker HealthChecker
    sticky       bool
}

type Backend struct {
    ID       string
    Address  string
    Port     int
    Weight   int
    Healthy  bool
    Conns    int
}

type BalancingAlgorithm interface {
    Select(backends []*Backend, req *Request) (*Backend, error)
}

func (lb *LoadBalancer) AddBackend(backend *Backend) error
func (lb *LoadBalancer) RemoveBackend(backendID string) error
func (lb *LoadBalancer) SelectBackend(req *Request) (*Backend, error)
func (lb *LoadBalancer) MarkUnhealthy(backendID string) error
func (lb *LoadBalancer) DrainBackend(backendID string) error
```

### Test Strategy
- Algorithm correctness tests
- Health-aware routing tests
- Connection draining tests
- Sticky session tests
- Load distribution tests
- Concurrent request tests

### Success Metrics
- Request distribution accuracy: ±5%
- Failover time < 1 second
- Zero requests to unhealthy backends
- Session persistence: 99.9%

---

## 4. TECHNICAL SPECIFICATIONS

### 4.1 Technology Stack
- **Language:** Go 1.23
- **Container Runtime:** Docker API v1.43+
- **Service Discovery:** Consul/etcd
- **Load Balancing:** Custom implementation
- **Metrics:** Prometheus
- **Logging:** Structured logging (logrus/zap)

### 4.2 Dependencies
- `github.com/docker/docker` - Docker API client
- `github.com/hashicorp/consul/api` - Consul client
- `go.etcd.io/etcd/client/v3` - etcd client
- `github.com/prometheus/client_golang` - Metrics

### 4.3 Performance Requirements
- Container start time: < 5 seconds
- Scale-up response: < 60 seconds
- Scale-down response: < 5 minutes
- Health check interval: 10-30 seconds
- Service discovery lookup: < 100ms
- Load balancer overhead: < 1ms per request

### 4.4 Reliability Requirements
- Auto-healing recovery: < 30 seconds
- Zero-downtime deployments
- Automatic rollback on errors
- 99.9% service availability
- Max 1% request failures during updates

---

## 5. TESTING STRATEGY

### 5.1 Unit Tests
- Each component independently tested
- Mock external dependencies
- Edge case coverage
- Error handling validation

### 5.2 Integration Tests
- End-to-end workflow tests
- Real Docker daemon integration
- Multi-component interaction
- Failure scenario testing

### 5.3 Load Tests
- 1,000+ concurrent containers
- High-frequency scaling events
- Stress test auto-healing
- Update under load

### 5.4 Chaos Testing
- Random container kills
- Network partitions
- Resource exhaustion
- Dependency failures

---

## 6. DEPLOYMENT PLAN

### 6.1 Phase 1: Docker Containerizer (Week 1)
- Implement Docker API wrapper
- Add lifecycle operations
- Integration tests
- Deploy to staging

### 6.2 Phase 2: Auto-Scaling & Healing (Week 2)
- Implement auto-scaler
- Implement auto-healer
- Add metrics collection
- Integration tests

### 6.3 Phase 3: Rolling Updates (Week 3)
- Implement rolling updater
- Add rollback mechanism
- Canary deployment support
- End-to-end tests

### 6.4 Phase 4: Service Discovery & LB (Week 4)
- Implement service discovery
- Implement load balancer
- DNS integration
- Complete system tests

---

## 7. SUCCESS CRITERIA

### 7.1 Functional Requirements
✅ Containers can be created, started, stopped, restarted
✅ Applications scale automatically based on metrics
✅ Failed containers restart within 30 seconds
✅ Zero-downtime rolling updates
✅ Services can discover each other
✅ Traffic distributed across healthy instances

### 7.2 Performance Requirements
✅ Container start time < 5 seconds
✅ Scale-up response < 60 seconds
✅ Service discovery < 100ms
✅ Load balancer overhead < 1ms

### 7.3 Reliability Requirements
✅ 99.9% application availability
✅ Automatic rollback on errors
✅ Zero requests to unhealthy backends
✅ Max 1% failures during updates

---

## 8. RISKS & MITIGATION

### 8.1 Technical Risks
**Risk:** Docker daemon failures
**Mitigation:** Implement retry logic, health monitoring, alerting

**Risk:** Scaling flapping
**Mitigation:** Cooldown periods, hysteresis in scaling logic

**Risk:** Failed deployments
**Mitigation:** Automatic rollback, canary testing, validation

### 8.2 Operational Risks
**Risk:** Resource exhaustion during scale-up
**Mitigation:** Max instance limits, resource quotas, monitoring

**Risk:** Cascading failures
**Mitigation:** Circuit breakers, rate limiting, isolation

---

## 9. METRICS & MONITORING

### 9.1 Key Metrics
- Container lifecycle operation success rate
- Scaling events per hour
- Auto-healing actions per hour
- Deployment success rate
- Service discovery latency
- Load balancer request distribution

### 9.2 Alerts
- Container restart rate > 10/minute
- Scaling failures
- Deployment failures
- Service discovery errors
- Load balancer backend failures

---

## 10. DOCUMENTATION

### 10.1 Required Documentation
- API documentation for each component
- Configuration reference
- Deployment guide
- Troubleshooting guide
- Runbook for operations

### 10.2 Code Documentation
- GoDoc comments for all public APIs
- Examples for common use cases
- Architecture diagrams
- Sequence diagrams for complex flows

---

## APPENDIX A: API EXAMPLES

### Docker Containerizer
```go
// Create and start a container
config := ContainerConfig{
    Name:  "web-server",
    Image: "nginx:latest",
    Ports: []PortMapping{{HostPort: 8080, ContainerPort: 80}},
    Memory: 512 * 1024 * 1024, // 512MB
}

containerID, err := containerizer.Create(ctx, config)
if err != nil {
    return err
}

err = containerizer.Start(ctx, containerID)
```

### Auto-Scaler
```go
// Configure auto-scaling
scaler := NewAutoScaler(app, metrics)
scaler.SetMinInstances(2)
scaler.SetMaxInstances(10)
scaler.AddPolicy(ScalingPolicy{
    MetricName: "cpu_usage",
    Threshold:  70.0,
    Comparison: GreaterThan,
    Action:     ScaleUp,
    Adjustment: 2,
})

scaler.Start(ctx)
```

### Rolling Update
```go
// Perform rolling update
updater := NewRollingUpdater(app, healthChecker, loadBalancer)
config := UpdateConfig{
    Strategy:     StrategyRollingUpdate,
    BatchSize:    25,
    AutoRollback: true,
}

err := updater.Update(ctx, newVersion, config)
```

---

**END OF PRD**

**Status:** ✅ READY FOR IMPLEMENTATION
**Next Steps:** Parse with task-master and begin Task 5
