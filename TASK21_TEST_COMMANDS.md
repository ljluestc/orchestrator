# Task #21: App Backend Testing Commands

## Prerequisites

Ensure Go is installed and available in your PATH:
```bash
export PATH=$PATH:/usr/local/go/bin
go version
```

## Running Tests

### 1. Run All App Tests
```bash
cd /home/calelin/dev/orchestrator
go test ./pkg/app/... -v
```

### 2. Run Specific Test Categories

#### Handler Tests
```bash
go test ./pkg/app -run "^TestHealth|TestPing|TestRegister|TestSubmit|TestGet" -v
```

#### Aggregator Tests
```bash
go test ./pkg/app -run "^TestAggregator|TestProcess|TestClean" -v
```

#### WebSocket Tests
```bash
go test ./pkg/app -run "^TestWS|TestWebSocket" -v
```

#### Integration Tests (NEW)
```bash
go test ./pkg/app -run "^TestAppIntegration|TestAppEndToEnd|TestAppResilience" -v
```

#### Load Tests
```bash
go test ./pkg/app -run "^TestLoad|TestMemory" -v
```

### 3. Run with Coverage
```bash
go test ./pkg/app/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### 4. Run Benchmarks
```bash
go test ./pkg/app -bench=. -benchmem -run=^$
```

### 5. Run Short Tests (Skip Long-Running Tests)
```bash
go test ./pkg/app -short -v
```

## Building the App Server

### Build Binary
```bash
cd cmd/app
go build -o app-server main.go
```

### Build with Optimizations
```bash
cd cmd/app
go build -ldflags="-s -w" -o app-server main.go
```

## Running the App Server

### With Default Configuration
```bash
cd cmd/app
./app-server
```

### With Custom Configuration
```bash
./app-server \
  -host 0.0.0.0 \
  -port 8080 \
  -max-data-age 2h \
  -cleanup-interval 10m \
  -stale-node-threshold 10m
```

### View Help
```bash
./app-server -help
```

## Manual API Testing

### Health Check
```bash
curl http://localhost:8080/health
```

### Ping
```bash
curl http://localhost:8080/api/v1/ping
```

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

### List Agents
```bash
curl http://localhost:8080/api/v1/agents/list
```

### Get Topology
```bash
curl http://localhost:8080/api/v1/query/topology | jq .
```

### Get Latest Report
```bash
curl http://localhost:8080/api/v1/query/agents/test-agent-1/latest | jq .
```

### Get Time-Series Data
```bash
curl "http://localhost:8080/api/v1/query/agents/test-agent-1/timeseries?duration=1h" | jq .
```

### Get Server Stats
```bash
curl http://localhost:8080/api/v1/query/stats | jq .
```

### Agent Heartbeat
```bash
curl -X POST http://localhost:8080/api/v1/agents/heartbeat/test-agent-1 \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "timestamp": "'$(date -Iseconds)'"
  }'
```

### Get Agent Config
```bash
curl http://localhost:8080/api/v1/agents/config/test-agent-1 | jq .
```

## WebSocket Testing

### Using websocat
```bash
# Install websocat first: cargo install websocat
websocat ws://localhost:8080/api/v1/ws
```

### Using wscat
```bash
# Install wscat first: npm install -g wscat
wscat -c ws://localhost:8080/api/v1/ws
```

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

## Storage Tests

### Test Storage Directly
```bash
go test ./internal/storage/... -v
```

### Test Storage with Coverage
```bash
go test ./internal/storage -coverprofile=storage_coverage.out
go tool cover -html=storage_coverage.out
```

## Docker Testing

### Build Docker Image
```bash
docker build -f Dockerfile.app -t orchestrator-app:latest .
```

### Run Docker Container
```bash
docker run -p 8080:8080 \
  -e HOST=0.0.0.0 \
  -e PORT=8080 \
  orchestrator-app:latest
```

### Test Docker Container
```bash
curl http://localhost:8080/health
```

## Performance Testing

### Apache Bench (Simple Load Test)
```bash
# Install: apt-get install apache2-utils

# Test health endpoint
ab -n 1000 -c 10 http://localhost:8080/health

# Test topology query
ab -n 500 -c 5 http://localhost:8080/api/v1/query/topology
```

### Hey (Modern Load Testing)
```bash
# Install: go install github.com/rakyll/hey@latest

# Test report submission
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"test","hostname":"test"}' \
  http://localhost:8080/api/v1/reports

# Test topology query
hey -n 1000 -c 20 http://localhost:8080/api/v1/query/topology
```

## Continuous Integration

### GitHub Actions Test Command
```yaml
- name: Run App Tests
  run: |
    go test ./pkg/app/... -v -race -coverprofile=coverage.txt
    go test ./internal/storage/... -v -race
```

### Test with Race Detector
```bash
go test ./pkg/app/... -race -v
```

### Test with Memory Sanitizer
```bash
go test ./pkg/app/... -msan -v
```

## Troubleshooting

### Check if Server is Running
```bash
lsof -i :8080
# or
netstat -tuln | grep 8080
```

### View Server Logs
```bash
./app-server 2>&1 | tee app-server.log
```

### Test with Verbose Output
```bash
go test ./pkg/app -v -count=1
```

### Clear Test Cache
```bash
go clean -testcache
go test ./pkg/app/... -v
```

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

## Expected Test Output

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

### Successful Integration Test
```
=== RUN   TestAppIntegration
    app_test.go:134: Integration test completed successfully!
--- PASS: TestAppIntegration (0.50s)
PASS
```

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

## Notes

- All tests are deterministic and repeatable
- Integration tests create temporary servers on different ports
- Load tests can be skipped with `-short` flag
- WebSocket tests may have timing variations in different environments
- Tests clean up after themselves (no persistent state)
- Storage tests verify thread safety
- All error paths are tested

## Quick Start

To verify everything works:

```bash
# 1. Build the server
cd cmd/app && go build -o app-server

# 2. Run all tests
cd ../.. && go test ./pkg/app/... -v -short

# 3. Start the server
cd cmd/app && ./app-server &

# 4. Test health
curl http://localhost:8080/health

# 5. Stop the server
pkill app-server
```

Success! ✅
