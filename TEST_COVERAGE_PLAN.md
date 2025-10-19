# Test Coverage & Quality Improvement Plan

**Project:** Orchestrator - Mesos/Docker Platform
**Date:** 2025-10-16
**Goal:** Achieve 85%+ test coverage across all systems

---

## Current Status

### Test Files Statistics
- **Test Files:** 30
- **Source Files:** 33
- **Test Ratio:** 0.91 (excellent - almost 1:1)

### Coverage by Package (as of 2025-10-16)

| Package | Coverage | Status | Priority |
|---------|----------|--------|----------|
| pkg/app | ~90% | ✅ Excellent | Maintain |
| internal/storage | ~80% | ✅ Good | Maintain |
| pkg/marathon | ~85% | ✅ Good | Maintain |
| pkg/containerizer | ~75% | ⚠️  Needs improvement | High |
| cmd/app | ~50% | ⚠️  Low (expected for main) | Medium |
| cmd/probe | 0% | ❌ Missing | High |
| pkg/probe | 0% | ❌ Missing | High |
| pkg/migration | 0% | ❌ Missing | Medium |
| pkg/scheduler | 0% | ❌ Missing | Medium |

---

## Phase 1: Fix Build Issues ✅

### Issue: go/test directory causing module conflicts
**Solution:**
1. Add `go/test/` to `.gitignore`
2. Exclude from test runs
3. Focus on `./pkg/... ./cmd/... ./internal/...`

---

## Phase 2: Increase Coverage for Existing Code

### Priority 1: cmd/probe (CRITICAL - 0% coverage)
**Current:** No tests
**Target:** 60% coverage
**Files to test:**
- cmd/probe/main.go

**Test plan:**
```go
// cmd/probe/main_test.go
- TestParseFlags
- TestSignalHandling
- TestGracefulShutdown
- TestProbeInitialization
```

### Priority 2: pkg/probe (CRITICAL - 0% coverage)
**Current:** No tests
**Target:** 80% coverage
**Files to test:**
- pkg/probe/probe.go
- pkg/probe/client.go
- pkg/probe/docker.go
- pkg/probe/host.go
- pkg/probe/network.go
- pkg/probe/process.go

**Test plan:**
```go
// pkg/probe/probe_test.go
- TestNewProbe
- TestStart/Stop
- TestReportSubmission
- TestErrorHandling

// pkg/probe/docker_test.go
- TestDockerCollector
- TestContainerDiscovery
- TestDockerStats

// pkg/probe/host_test.go
- TestHostInfoCollection
- TestCPUInfo
- TestMemoryInfo

// pkg/probe/network_test.go
- TestNetworkConnections
- TestConnectionTracking

// pkg/probe/process_test.go
- TestProcessDiscovery
- TestProcessMetrics
```

### Priority 3: pkg/migration (MEDIUM - 0% coverage)
**Current:** No tests
**Target:** 75% coverage
**Files to test:**
- pkg/migration/sync_engine.go

**Test plan:**
```go
// pkg/migration/sync_engine_test.go
- TestSyncEngineInit
- TestBidirectionalSync
- TestConflictResolution
- TestChecksumValidation
- TestSyncLag
```

### Priority 4: pkg/scheduler (MEDIUM - 0% coverage)
**Current:** No tests
**Target:** 75% coverage
**Files to test:**
- pkg/scheduler/drf.go
- pkg/scheduler/quota_enforcer.go

**Test plan:**
```go
// pkg/scheduler/drf_test.go
- TestDRFAllocation
- TestFairnessCalculation
- TestResourceShares

// pkg/scheduler/quota_enforcer_test.go
- TestQuotaEnforcement
- TestQuotaValidation
- TestQuotaExceeded
```

---

## Phase 3: Race Condition Fixes

### Known Issues
1. **WebSocket Hub** (pkg/app/websocket.go)
   - Concurrent map access in client management
   - **Fix:** Already using mutex, verify with `-race` flag

2. **Storage Layer** (internal/storage/timeseries.go)
   - Time-series map access
   - **Fix:** Already using RWMutex, verify

3. **Aggregator** (pkg/app/aggregator.go)
   - Topology building concurrent access
   - **Fix:** Already using mutex, verify

**Verification:**
```bash
./go/bin/go test ./pkg/... ./cmd/... ./internal/... -race
```

---

## Phase 4: CI/CD Pipeline Implementation

### GitHub Actions Workflow

```yaml
# .github/workflows/comprehensive-ci.yml
name: Comprehensive CI/CD

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test and Coverage
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run tests with coverage
        run: |
          go test ./pkg/... ./cmd/... ./internal/... \
            -coverprofile=coverage.out \
            -covermode=atomic \
            -race

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests
          fail_ci_if_error: true

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 85" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 85% threshold"
            exit 1
          fi

  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          args: --timeout=5m

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Build binaries
        run: |
          go build -o bin/app-server ./cmd/app/
          go build -o bin/probe-agent ./cmd/probe/

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: bin/
```

---

## Phase 5: Pre-commit Hooks

### Installation
```bash
# Install pre-commit
pip install pre-commit

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Configuration (.pre-commit-config.yaml)
```yaml
repos:
  - repo: https://github.com/tekwizely/pre-commit-golang
    rev: v1.0.0-rc.1
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-imports
      - id: go-critic
      - id: go-build
      - id: go-mod-tidy

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.54.2
    hooks:
      - id: golangci-lint
        args: [--timeout=5m]

  - repo: local
    hooks:
      - id: go-test
        name: go test
        entry: go test ./pkg/... ./cmd/... ./internal/...
        language: system
        pass_filenames: false
        always_run: true

      - id: go-test-race
        name: go test -race
        entry: go test -race ./pkg/... ./cmd/... ./internal/...
        language: system
        pass_filenames: false
        stages: [push]
```

---

## Phase 6: Integration Test Coverage

### E2E Test Scenarios

1. **Full Stack Integration**
   ```go
   // test/e2e/full_stack_test.go
   - TestProbeToAppBackendIntegration
   - TestWebSocketRealTimeUpdates
   - TestTopologyBuildingE2E
   - TestContainerLifecycleE2E
   ```

2. **Marathon Integration**
   ```go
   // test/e2e/marathon_test.go
   - TestMarathonApplicationDeployment
   - TestAutoScalingE2E
   - TestAutoHealingE2E
   - TestRollingUpdateE2E
   ```

3. **Migration Integration**
   ```go
   // test/e2e/migration_test.go
   - TestZookeeperMigrationE2E
   - TestZeroDowntimeCutover
   - TestDataConsistency
   ```

---

## Phase 7: Performance & Load Testing

### Load Test Targets
```go
// test/load/load_test.go
- Test 1,000+ concurrent probe agents
- Test 10,000+ containers reporting
- Test WebSocket with 100+ clients
- Test API endpoints under load
- Measure memory usage over time
- Detect memory leaks
```

### Benchmarks
```go
// pkg/app/aggregator_benchmark_test.go
func BenchmarkTopologyBuilding(b *testing.B)
func BenchmarkReportProcessing(b *testing.B)
func BenchmarkWebSocketBroadcast(b *testing.B)
```

---

## Implementation Timeline

### Week 1 (Current)
- ✅ Fix build issues
- ⏳ Create comprehensive test plan (this document)
- 🔴 Implement cmd/probe tests
- 🔴 Implement pkg/probe tests

### Week 2
- Implement pkg/migration tests
- Implement pkg/scheduler tests
- Fix all race conditions
- Achieve 85%+ coverage

### Week 3
- Implement CI/CD pipeline
- Set up pre-commit hooks
- Add integration tests
- Add load tests

### Week 4
- Documentation completion
- Performance optimization
- Final testing & validation
- Production readiness review

---

## Coverage Targets

### Minimum Targets
| Component | Minimum | Stretch Goal |
|-----------|---------|--------------|
| pkg/app | 85% | 95% |
| pkg/marathon | 80% | 90% |
| pkg/probe | 75% | 85% |
| pkg/containerizer | 80% | 90% |
| internal/storage | 85% | 95% |
| pkg/migration | 70% | 80% |
| pkg/scheduler | 70% | 80% |
| **Overall** | **85%** | **90%** |

---

## Success Metrics

### Code Quality
- ✅ 85%+ test coverage
- ✅ Zero race conditions
- ✅ All linters passing
- ✅ CI/CD pipeline green
- ✅ Pre-commit hooks installed

### Performance
- ✅ 1,000+ nodes supported
- ✅ <2s UI rendering
- ✅ <5% CPU per probe
- ✅ <100MB memory per probe

### Reliability
- ✅ Zero known bugs
- ✅ All E2E tests passing
- ✅ Load tests passing
- ✅ Production-ready

---

**Status:** 📝 Plan Created
**Next Action:** Implement cmd/probe tests
**Owner:** Development Team
**Due Date:** 2025-10-23
