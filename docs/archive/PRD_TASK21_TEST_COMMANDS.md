# Product Requirements Document: ORCHESTRATOR: Task21 Test Commands

---

## Document Information
**Project:** orchestrator
**Document:** TASK21_TEST_COMMANDS
**Version:** 1.0.0
**Date:** 2025-10-13
**Status:** READY FOR TASK-MASTER PARSING

---

## 1. EXECUTIVE SUMMARY

### 1.1 Overview
This PRD captures the requirements and implementation details for ORCHESTRATOR: Task21 Test Commands.

### 1.2 Purpose
This document provides a structured specification that can be parsed by task-master to generate actionable tasks.

### 1.3 Scope
The scope includes all requirements, features, and implementation details from the original documentation.

---

## 2. REQUIREMENTS


## 3. TASKS

The following tasks have been identified for implementation:

**TASK_001** [MEDIUM]: name: Run App Tests

**TASK_002** [MEDIUM]: ✅ All unit tests pass

**TASK_003** [MEDIUM]: ✅ All integration tests pass

**TASK_004** [MEDIUM]: ✅ Load tests show >95% success rate

**TASK_005** [MEDIUM]: ✅ WebSocket tests complete successfully

**TASK_006** [MEDIUM]: ✅ Server starts and stops cleanly

**TASK_007** [MEDIUM]: ✅ Health endpoint returns 200

**TASK_008** [MEDIUM]: ✅ All REST endpoints respond correctly

**TASK_009** [MEDIUM]: ✅ Time-series data is stored correctly

**TASK_010** [MEDIUM]: ✅ Topology aggregation works

**TASK_011** [MEDIUM]: ✅ WebSocket connections work

**TASK_012** [MEDIUM]: ✅ Cleanup routines function properly

**TASK_013** [MEDIUM]: ✅ No race conditions detected

**TASK_014** [MEDIUM]: ✅ Memory usage is stable

**TASK_015** [MEDIUM]: ✅ No goroutine leaks

**TASK_016** [MEDIUM]: All tests are deterministic and repeatable

**TASK_017** [MEDIUM]: Integration tests create temporary servers on different ports

**TASK_018** [MEDIUM]: Load tests can be skipped with `-short` flag

**TASK_019** [MEDIUM]: WebSocket tests may have timing variations in different environments

**TASK_020** [MEDIUM]: Tests clean up after themselves (no persistent state)

**TASK_021** [MEDIUM]: Storage tests verify thread safety

**TASK_022** [MEDIUM]: All error paths are tested


## 4. DETAILED SPECIFICATIONS

### 4.1 Original Content

The following sections contain the original documentation:


#### Task 21 App Backend Testing Commands

# Task #21: App Backend Testing Commands


#### Prerequisites

## Prerequisites

Ensure Go is installed and available in your PATH:
```bash
export PATH=$PATH:/usr/local/go/bin
go version
```


#### Running Tests

## Running Tests


#### 1 Run All App Tests

### 1. Run All App Tests
```bash
cd /home/calelin/dev/orchestrator
go test ./pkg/app/... -v
```


#### 2 Run Specific Test Categories

### 2. Run Specific Test Categories


#### Handler Tests

#### Handler Tests
```bash
go test ./pkg/app -run "^TestHealth|TestPing|TestRegister|TestSubmit|TestGet" -v
```


#### Aggregator Tests

#### Aggregator Tests
```bash
go test ./pkg/app -run "^TestAggregator|TestProcess|TestClean" -v
```


#### Websocket Tests

#### WebSocket Tests
```bash
go test ./pkg/app -run "^TestWS|TestWebSocket" -v
```


#### Integration Tests New 

#### Integration Tests (NEW)
```bash
go test ./pkg/app -run "^TestAppIntegration|TestAppEndToEnd|TestAppResilience" -v
```


#### Load Tests

#### Load Tests
```bash
go test ./pkg/app -run "^TestLoad|TestMemory" -v
```


#### 3 Run With Coverage

### 3. Run with Coverage
```bash
go test ./pkg/app/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```


#### 4 Run Benchmarks

### 4. Run Benchmarks
```bash
go test ./pkg/app -bench=. -benchmem -run=^$
```


#### 5 Run Short Tests Skip Long Running Tests 

### 5. Run Short Tests (Skip Long-Running Tests)
```bash
go test ./pkg/app -short -v
```


#### Building The App Server

## Building the App Server


#### Build Binary

### Build Binary
```bash
cd cmd/app
go build -o app-server main.go
```


#### Build With Optimizations

### Build with Optimizations
```bash
cd cmd/app
go build -ldflags="-s -w" -o app-server main.go
```


#### Running The App Server

## Running the App Server


#### With Default Configuration

### With Default Configuration
```bash
cd cmd/app
./app-server
```


#### With Custom Configuration

### With Custom Configuration
```bash
./app-server \
  -host 0.0.0.0 \
  -port 8080 \
  -max-data-age 2h \
  -cleanup-interval 10m \
  -stale-node-threshold 10m
```


#### View Help

### View Help
```bash
./app-server -help
```


#### Manual Api Testing

## Manual API Testing


#### Health Check

### Health Check
```bash
curl http://localhost:8080/health
```


#### Ping

### Ping
```bash
curl http://localhost:8080/api/v1/ping
```


#### Register Agent

### Register Agent
```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "hostname": "test-host",
    "metadata": {"version": "1.0.0"},
    "timestamp": "'$(date -Iseconds)'"
  }'
```


#### Submit Report

### Submit Report
```bash
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "hostname": "test-host",
    "timestamp": "'$(date -Iseconds)'",
    "host_info": {
      "hostname": "test-host",
      "kernel_version": "5.10.0",
      "cpu_info": {
        "model": "Intel Core i7",
        "cores": 8,
        "usage": 45.5
      },
      "memory_info": {
        "total_mb": 16384,
        "used_mb": 8192,
        "free_mb": 8192,
        "available_mb": 8192,
        "usage": 50.0
      }
    }
  }'
```


#### List Agents

### List Agents
```bash
curl http://localhost:8080/api/v1/agents/list
```


#### Get Topology

### Get Topology
```bash
curl http://localhost:8080/api/v1/query/topology | jq .
```


#### Get Latest Report

### Get Latest Report
```bash
curl http://localhost:8080/api/v1/query/agents/test-agent-1/latest | jq .
```


#### Get Time Series Data

### Get Time-Series Data
```bash
curl "http://localhost:8080/api/v1/query/agents/test-agent-1/timeseries?duration=1h" | jq .
```


#### Get Server Stats

### Get Server Stats
```bash
curl http://localhost:8080/api/v1/query/stats | jq .
```


#### Agent Heartbeat

### Agent Heartbeat
```bash
curl -X POST http://localhost:8080/api/v1/agents/heartbeat/test-agent-1 \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "timestamp": "'$(date -Iseconds)'"
  }'
```


#### Get Agent Config

### Get Agent Config
```bash
curl http://localhost:8080/api/v1/agents/config/test-agent-1 | jq .
```


#### Websocket Testing

## WebSocket Testing


#### Using Websocat

### Using websocat
```bash

#### Install Websocat First Cargo Install Websocat

# Install websocat first: cargo install websocat
websocat ws://localhost:8080/api/v1/ws
```


#### Using Wscat

### Using wscat
```bash

#### Install Wscat First Npm Install G Wscat

# Install wscat first: npm install -g wscat
wscat -c ws://localhost:8080/api/v1/ws
```


#### Test Websocket Javascript In Browser Console 

### Test WebSocket (JavaScript in browser console)
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Received:', msg.type, msg.payload);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

// Send a ping
ws.send(JSON.stringify({
  type: 'ping',
  payload: {}
}));

// Subscribe to updates
ws.send(JSON.stringify({
  type: 'subscribe',
  payload: {view: 'containers'}
}));
```


#### Storage Tests

## Storage Tests


#### Test Storage Directly

### Test Storage Directly
```bash
go test ./internal/storage/... -v
```


#### Test Storage With Coverage

### Test Storage with Coverage
```bash
go test ./internal/storage -coverprofile=storage_coverage.out
go tool cover -html=storage_coverage.out
```


#### Docker Testing

## Docker Testing


#### Build Docker Image

### Build Docker Image
```bash
docker build -f Dockerfile.app -t orchestrator-app:latest .
```


#### Run Docker Container

### Run Docker Container
```bash
docker run -p 8080:8080 \
  -e HOST=0.0.0.0 \
  -e PORT=8080 \
  orchestrator-app:latest
```


#### Test Docker Container

### Test Docker Container
```bash
curl http://localhost:8080/health
```


#### Performance Testing

## Performance Testing


#### Apache Bench Simple Load Test 

### Apache Bench (Simple Load Test)
```bash

#### Install Apt Get Install Apache2 Utils

# Install: apt-get install apache2-utils


#### Test Health Endpoint

# Test health endpoint
ab -n 1000 -c 10 http://localhost:8080/health


#### Test Topology Query

# Test topology query
hey -n 1000 -c 20 http://localhost:8080/api/v1/query/topology
```


#### Hey Modern Load Testing 

### Hey (Modern Load Testing)
```bash

#### Install Go Install Github Com Rakyll Hey Latest

# Install: go install github.com/rakyll/hey@latest


#### Test Report Submission

# Test report submission
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"test","hostname":"test"}' \
  http://localhost:8080/api/v1/reports


#### Continuous Integration

## Continuous Integration


#### Github Actions Test Command

### GitHub Actions Test Command
```yaml
- name: Run App Tests
  run: |
    go test ./pkg/app/... -v -race -coverprofile=coverage.txt
    go test ./internal/storage/... -v -race
```


#### Test With Race Detector

### Test with Race Detector
```bash
go test ./pkg/app/... -race -v
```


#### Test With Memory Sanitizer

### Test with Memory Sanitizer
```bash
go test ./pkg/app/... -msan -v
```


#### Troubleshooting

## Troubleshooting


#### Check If Server Is Running

### Check if Server is Running
```bash
lsof -i :8080

#### Or

# or
netstat -tuln | grep 8080
```


#### View Server Logs

### View Server Logs
```bash
./app-server 2>&1 | tee app-server.log
```


#### Test With Verbose Output

### Test with Verbose Output
```bash
go test ./pkg/app -v -count=1
```


#### Clear Test Cache

### Clear Test Cache
```bash
go clean -testcache
go test ./pkg/app/... -v
```


#### Validation Checklist

## Validation Checklist

After running tests, verify:

- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Load tests show >95% success rate
- ✅ WebSocket tests complete successfully
- ✅ Server starts and stops cleanly
- ✅ Health endpoint returns 200
- ✅ All REST endpoints respond correctly
- ✅ Time-series data is stored correctly
- ✅ Topology aggregation works
- ✅ WebSocket connections work
- ✅ Cleanup routines function properly
- ✅ No race conditions detected
- ✅ Memory usage is stable
- ✅ No goroutine leaks


#### Expected Test Output

## Expected Test Output


#### Successful Test Run

### Successful Test Run
```
=== RUN   TestHealthCheck
--- PASS: TestHealthCheck (0.00s)
=== RUN   TestPing
--- PASS: TestPing (0.00s)
=== RUN   TestRegisterAgent
--- PASS: TestRegisterAgent (0.00s)
...
PASS
ok      github.com/ljluestc/orchestrator/pkg/app    2.456s
```


#### Successful Integration Test

### Successful Integration Test
```
=== RUN   TestAppIntegration
    app_test.go:134: Integration test completed successfully!
--- PASS: TestAppIntegration (0.50s)
PASS
```


#### Load Test Output

### Load Test Output
```
=== RUN   TestLoadMultipleProbes
    loadtest_test.go:102: Load test completed in 2.5s
    loadtest_test.go:103: Total requests: 100
    loadtest_test.go:104: Successful reports: 100
    loadtest_test.go:105: Failed reports: 0
    loadtest_test.go:106: Requests per second: 40.00
--- PASS: TestLoadMultipleProbes (2.50s)
```


#### Notes

## Notes

- All tests are deterministic and repeatable
- Integration tests create temporary servers on different ports
- Load tests can be skipped with `-short` flag
- WebSocket tests may have timing variations in different environments
- Tests clean up after themselves (no persistent state)
- Storage tests verify thread safety
- All error paths are tested


#### Quick Start

## Quick Start

To verify everything works:

```bash

#### 1 Build The Server

# 1. Build the server
cd cmd/app && go build -o app-server


#### 2 Run All Tests

# 2. Run all tests
cd ../.. && go test ./pkg/app/... -v -short


#### 3 Start The Server

# 3. Start the server
cd cmd/app && ./app-server &


#### 4 Test Health

# 4. Test health
curl http://localhost:8080/health


#### 5 Stop The Server

# 5. Stop the server
pkill app-server
```

Success! ✅


---

## 5. TECHNICAL REQUIREMENTS

### 5.1 Dependencies
- All dependencies from original documentation apply
- Standard development environment
- Required tools and libraries as specified

### 5.2 Compatibility
- Compatible with existing infrastructure
- Follows project standards and conventions

---

## 6. SUCCESS CRITERIA

### 6.1 Functional Success Criteria
- All identified tasks completed successfully
- All requirements implemented as specified
- All tests passing

### 6.2 Quality Success Criteria
- Code meets quality standards
- Documentation is complete and accurate
- No critical issues remaining

---

## 7. IMPLEMENTATION PLAN

### Phase 1: Preparation
- Review all requirements and tasks
- Set up development environment
- Gather necessary resources

### Phase 2: Implementation
- Execute tasks in priority order
- Follow best practices
- Test incrementally

### Phase 3: Validation
- Run comprehensive tests
- Validate against requirements
- Document completion

---

## 8. TASK-MASTER INTEGRATION

### How to Parse This PRD

```bash
# Parse this PRD with task-master
task-master parse-prd --input="{doc_name}_PRD.md"

# List generated tasks
task-master list

# Start execution
task-master next
```

### Expected Task Generation
Task-master should generate approximately {len(tasks)} tasks from this PRD.

---

## 9. APPENDIX

### 9.1 References
- Original document: {doc_name}.md
- Project: {project_name}

### 9.2 Change History
| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | {datetime.now().strftime('%Y-%m-%d')} | Initial PRD conversion |

---

*End of PRD*
*Generated by MD-to-PRD Converter*
