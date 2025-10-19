# Task 21: App Backend Server - Quick Start Guide

This guide will help you quickly get the App Backend Server up and running.

---

## Prerequisites

- Go 1.23 or higher
- Linux/macOS environment
- curl (for testing)

---

## Build the Server

```bash
# Using the custom Go binary
./go/bin/go build -o app-server ./cmd/app/

# Or using system Go
go build -o app-server ./cmd/app/
```

---

## Run the Server

### Default Configuration

```bash
./app-server
```

This starts the server with defaults:
- Host: 0.0.0.0
- Port: 8080
- Max data age: 1 hour
- Cleanup interval: 5 minutes
- Stale node threshold: 5 minutes

### Custom Configuration

```bash
./app-server \
  --host=127.0.0.1 \
  --port=9000 \
  --max-data-age=2h \
  --cleanup-interval=10m \
  --stale-node-threshold=10m
```

### Show Help

```bash
./app-server --help
```

---

## Test the Server

### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-10-14T..."
}
```

### 2. Ping

```bash
curl http://localhost:8080/api/v1/ping
```

Expected response:
```json
{
  "pong": true,
  "timestamp": "2025-10-14T..."
}
```

### 3. Register an Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "hostname": "test-host-1",
    "metadata": {
      "region": "us-west",
      "env": "dev"
    }
  }'
```

Expected response:
```json
{
  "status": "registered",
  "message": "Agent test-agent-1 registered successfully"
}
```

### 4. List Agents

```bash
curl http://localhost:8080/api/v1/agents/list
```

Expected response:
```json
{
  "agents": [
    {
      "agent_id": "test-agent-1",
      "hostname": "test-host-1",
      "metadata": {"region": "us-west", "env": "dev"},
      "registered_at": "2025-10-14T...",
      "last_seen": "2025-10-14T..."
    }
  ],
  "count": 1
}
```

### 5. Submit a Report

```bash
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "test-agent-1",
    "hostname": "test-host-1",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "host_info": {
      "hostname": "test-host-1",
      "kernel_version": "5.15.0",
      "cpu_info": {
        "model": "Intel Core i7",
        "cores": 8,
        "usage": 25.5
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

Expected response:
```json
{
  "status": "accepted",
  "message": "Report processed successfully"
}
```

### 6. Query Topology

```bash
curl http://localhost:8080/api/v1/query/topology
```

Expected response:
```json
{
  "topology": {
    "nodes": {
      "test-agent-1": {
        "id": "test-agent-1",
        "type": "host",
        "name": "test-host-1",
        "metadata": {
          "agent_id": "test-agent-1",
          "hostname": "test-host-1",
          "cpu_cores": 8,
          "cpu_usage": 25.5,
          ...
        }
      }
    },
    "edges": {},
    "timestamp": "2025-10-14T..."
  },
  "timestamp": "2025-10-14T..."
}
```

### 7. Get Latest Report for Agent

```bash
curl http://localhost:8080/api/v1/query/agents/test-agent-1/latest
```

### 8. Get Time-Series Data

```bash
curl "http://localhost:8080/api/v1/query/agents/test-agent-1/timeseries?duration=1h"
```

### 9. Get Server Stats

```bash
curl http://localhost:8080/api/v1/query/stats
```

Expected response:
```json
{
  "agents": 1,
  "websocket_clients": 0,
  "storage": {
    "max_age": "1h0m0s",
    "resolution": "15s",
    "total_agents": 1,
    "total_points": 1
  },
  "topology": {
    "total_nodes": 1,
    "total_edges": 0,
    "nodes_by_type": {"host": 1},
    "edges_by_type": {},
    "last_update": "2025-10-14T..."
  },
  "uptime": "5m23s"
}
```

---

## WebSocket Connection

### Using websocat (WebSocket CLI tool)

```bash
# Install websocat
cargo install websocat

# Connect to WebSocket
websocat ws://localhost:8080/api/v1/ws
```

You'll receive real-time updates when:
- New reports are submitted
- Topology changes occur

### Using JavaScript

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onopen = () => {
  console.log('Connected to App Backend Server');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);

  if (message.type === 'topology_update') {
    console.log('Topology updated:', message.payload);
  } else if (message.type === 'report_update') {
    console.log('Report update:', message.payload);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from server');
};
```

---

## Run Tests

### Unit Tests

```bash
./go/bin/go test ./pkg/app/... -v
```

### With Coverage

```bash
./go/bin/go test ./pkg/app/... -coverprofile=coverage.out
./go/bin/go tool cover -func=coverage.out
```

### E2E Tests Only

```bash
./go/bin/go test ./pkg/app/... -v -run TestE2E
```

### Load Tests

```bash
./go/bin/go test ./pkg/app/... -v -run TestLoad
```

---

## Graceful Shutdown

The server handles graceful shutdown on SIGINT (Ctrl+C) or SIGTERM:

```bash
# Start server
./app-server

# Press Ctrl+C to stop
^C
```

Output:
```
Received signal: interrupt
Stopping app server...
App server stopped successfully
```

---

## Troubleshooting

### Port Already in Use

```
Error: listen tcp :8080: bind: address already in use
```

Solution: Use a different port
```bash
./app-server --port=9000
```

### Connection Refused

```
Error: curl: (7) Failed to connect to localhost port 8080
```

Solution: Verify server is running
```bash
ps aux | grep app-server
```

### High Memory Usage

Solution: Reduce data retention
```bash
./app-server --max-data-age=30m
```

---

## Production Deployment

### Systemd Service

Create `/etc/systemd/system/app-server.service`:

```ini
[Unit]
Description=App Backend Server
After=network.target

[Service]
Type=simple
User=orchestrator
Group=orchestrator
WorkingDirectory=/opt/orchestrator
ExecStart=/opt/orchestrator/app-server \
  --host=0.0.0.0 \
  --port=8080 \
  --max-data-age=1h \
  --cleanup-interval=5m
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable app-server
sudo systemctl start app-server
sudo systemctl status app-server
```

### Docker

```bash
docker run -d \
  --name app-server \
  -p 8080:8080 \
  app-server:latest \
  --host=0.0.0.0 \
  --port=8080
```

### Kubernetes

```bash
kubectl apply -f k8s/base/monitoring-app/
kubectl get pods -n orchestrator -l app=monitoring-app
kubectl port-forward -n orchestrator svc/monitoring-app 8080:8080
```

---

## Next Steps

1. **Deploy Probe Agents** - Deploy probe agents to collect data
2. **Configure UI** - Connect UI frontend to this backend
3. **Setup Monitoring** - Add Prometheus metrics export
4. **Enable TLS** - Add HTTPS support for production
5. **Add Authentication** - Implement API key or JWT auth

---

## Support

For issues or questions:
- Check logs for error messages
- Review test files for usage examples
- See `docs/TASK21_COMPLETION_REPORT.md` for detailed documentation

---

**Last Updated:** 2025-10-14
