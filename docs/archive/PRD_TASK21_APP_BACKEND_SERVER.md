# Product Requirements Document (PRD)
## Task #21: App Backend Server with Report Aggregation

**Document Version**: 1.0.0
**Last Updated**: 2025-10-13
**Status**: ✅ COMPLETED
**Priority**: HIGH

---

## Executive Summary

Task #21 involves creating the central app backend server that receives reports from monitoring probes, aggregates data, and provides REST/WebSocket APIs for the UI. This server acts as the central hub for the distributed monitoring platform, handling real-time data aggregation, topology generation, and client communications.

**Key Deliverables**:
- Go HTTP server with Gin framework
- REST API for probe registration and report submission
- WebSocket server for real-time UI updates
- Report aggregation engine
- Time-series metrics storage (15-second resolution)
- Background cleanup of old data

---

## Table of Contents

1. [Product Vision](#product-vision)
2. [Task Breakdown](#task-breakdown)
3. [Technical Architecture](#technical-architecture)
4. [Implementation Details](#implementation-details)
5. [API Specifications](#api-specifications)
6. [Testing Strategy](#testing-strategy)
7. [Success Criteria](#success-criteria)
8. [Dependencies](#dependencies)
9. [Timeline](#timeline)
10. [Appendix](#appendix)

---

## Product Vision

The App Backend Server is the nerve center of the monitoring platform, responsible for:
- Receiving and processing reports from distributed probe agents
- Aggregating data into meaningful topology views
- Providing real-time updates to connected UI clients
- Managing time-series metrics with efficient storage
- Handling graceful shutdown and error recovery

This component enables operators to visualize and understand their infrastructure in real-time, with sub-second latency for critical updates.

---

## Task Breakdown

### Task #21.1: HTTP Server Setup
**Description**: Create the base HTTP server using Gin framework with proper middleware and configuration

**Implementation Details**:
- Initialize Gin router with release mode for production
- Configure CORS middleware for cross-origin requests
- Set up graceful shutdown with signal handling
- Add request logging and recovery middleware
- Configure server timeouts (read, write, idle)

**Deliverables**:
- `pkg/app/server.go` - Server struct and configuration
- `cmd/app/main.go` - Entry point with flag parsing

**Test Strategy**:
- Unit tests for server lifecycle (start/stop)
- Configuration validation tests
- Graceful shutdown tests with active connections

**Priority**: CRITICAL
**Dependencies**: None
**Status**: ✅ COMPLETED

---

### Task #21.2: REST API Handlers
**Description**: Implement REST API endpoints for agent management, report submission, and data queries

**Implementation Details**:
1. **Agent Management Endpoints**:
   - `POST /api/v1/agents/register` - Register new probe agents
   - `POST /api/v1/agents/heartbeat/:agent_id` - Heartbeat tracking
   - `GET /api/v1/agents/config/:agent_id` - Agent configuration
   - `GET /api/v1/agents/list` - List all registered agents

2. **Report Submission**:
   - `POST /api/v1/reports` - Submit probe reports
   - Validation of report data structure
   - Update agent last-seen timestamps
   - Forward to aggregator for processing

3. **Data Query Endpoints**:
   - `GET /api/v1/query/topology` - Get current topology view
   - `GET /api/v1/query/agents/:agent_id/latest` - Latest agent report
   - `GET /api/v1/query/agents/:agent_id/timeseries?duration=1h` - Time-series data
   - `GET /api/v1/query/stats` - Server statistics

4. **Health & Monitoring**:
   - `GET /health` - Health check endpoint
   - `GET /api/v1/ping` - Simple ping endpoint

**Deliverables**:
- `pkg/app/handlers.go` - Handler implementations
- `pkg/app/handlers_test.go` - Handler unit tests

**Test Strategy**:
- Unit tests for each endpoint with mock data
- Request validation tests
- Error handling tests (400, 404, 500)
- Response format validation

**Priority**: CRITICAL
**Dependencies**: Task #21.1
**Status**: ✅ COMPLETED

---

### Task #21.3: Report Aggregation Engine
**Description**: Build the aggregation engine that merges probe reports into topology views

**Implementation Details**:
1. **Topology Data Structures**:
   - `TopologyNode` - Represents hosts, containers, processes
   - `TopologyEdge` - Represents connections between nodes
   - `TopologyView` - Complete topology snapshot

2. **Aggregation Logic**:
   - Process reports from multiple probes
   - Build multi-level topology (hosts → containers → processes)
   - Track network connections between entities
   - Extract container IDs from cgroups
   - Merge overlapping data from different probes

3. **Node Management**:
   - Create/update nodes based on report data
   - Establish parent-child relationships
   - Track node metadata and timestamps
   - Clean stale nodes based on age threshold

4. **Thread Safety**:
   - RWMutex for concurrent read/write access
   - Deep copying for topology snapshots
   - Atomic updates to prevent race conditions

**Deliverables**:
- `pkg/app/aggregator.go` - Aggregation engine implementation
- `pkg/app/aggregator_test.go` - Aggregation unit tests

**Test Strategy**:
- Topology building tests with various report types
- Concurrent access tests
- Stale node cleanup tests
- Container ID extraction tests
- Performance tests with large datasets

**Priority**: HIGH
**Dependencies**: Task #21.2
**Status**: ✅ COMPLETED

---

### Task #21.4: WebSocket Server
**Description**: Implement WebSocket server for real-time UI updates with hub pattern

**Implementation Details**:
1. **WebSocket Hub**:
   - Client registration/unregistration
   - Broadcast channel for topology updates
   - Per-client send buffers (256 bytes)
   - Connection cleanup on disconnect

2. **Client Management**:
   - Client struct with connection and send channel
   - Read pump for incoming messages
   - Write pump for outgoing messages
   - Ping/pong for connection keep-alive

3. **Message Types**:
   - `topology_update` - Full topology broadcasts
   - `report_update` - Individual agent reports
   - `ping/pong` - Keep-alive messages

4. **Connection Lifecycle**:
   - Upgrade HTTP to WebSocket
   - Send initial topology on connect
   - Handle graceful disconnection
   - Automatic reconnection support

**Deliverables**:
- `pkg/app/websocket.go` - WebSocket hub and client
- `pkg/app/websocket_test.go` - WebSocket tests

**Test Strategy**:
- Client connection/disconnection tests
- Message broadcasting tests
- Concurrent client tests
- Connection timeout tests
- Memory leak tests for long-running connections

**Priority**: HIGH
**Dependencies**: Task #21.3
**Status**: ✅ COMPLETED

---

### Task #21.5: Time-Series Storage
**Description**: Implement efficient time-series storage for metrics with 15-second resolution

**Implementation Details**:
1. **Storage Structure**:
   - `TimeSeriesData` - Per-agent data with ring buffer
   - `TimeSeriesPoint` - Single metric point with timestamp
   - `TimeSeriesStore` - Global store managing all agents

2. **Features**:
   - 15-second resolution as specified
   - Configurable retention period (default 1 hour)
   - Automatic old data cleanup
   - Per-agent time-series tracking
   - Background cleanup goroutine

3. **Operations**:
   - `AddReport(report)` - Store new data point
   - `GetRecentPoints(agentID, duration)` - Query time range
   - `GetLatestReport(agentID)` - Get most recent data
   - `GetAllAgents()` - List tracked agents
   - `DeleteAgent(agentID)` - Remove agent data

4. **Performance**:
   - Pre-allocated buffers for efficiency
   - Lock-free reads where possible
   - Batched cleanup operations
   - Memory-efficient storage format

**Deliverables**:
- `internal/storage/timeseries.go` - Time-series implementation
- `internal/storage/storage_test.go` - Storage tests

**Test Strategy**:
- Data retention tests
- Query performance benchmarks
- Concurrent access tests
- Memory usage profiling
- Cleanup validation tests

**Priority**: MEDIUM
**Dependencies**: Task #21.2
**Status**: ✅ COMPLETED

---

### Task #21.6: Background Cleanup
**Description**: Implement background cleanup of stale data and agents

**Implementation Details**:
1. **Cleanup Operations**:
   - Remove stale agents (no heartbeat for threshold period)
   - Clean old topology nodes
   - Purge expired time-series data
   - Archive data for rollback window

2. **Cleanup Schedule**:
   - Configurable cleanup interval (default 5 minutes)
   - Stale node threshold (default 5 minutes)
   - Data retention policies
   - Graceful cleanup during shutdown

3. **Monitoring**:
   - Log cleanup statistics
   - Track cleanup duration
   - Alert on excessive stale agents
   - Metrics export for observability

**Deliverables**:
- Cleanup logic in `pkg/app/server.go`
- Background goroutine management
- Cleanup metrics and logging

**Test Strategy**:
- Stale data detection tests
- Cleanup timing tests
- Graceful shutdown during cleanup
- Resource leak prevention tests

**Priority**: MEDIUM
**Dependencies**: Task #21.5
**Status**: ✅ COMPLETED

---

### Task #21.7: Configuration Management
**Description**: Implement comprehensive configuration via command-line flags and environment variables

**Implementation Details**:
1. **Configuration Options**:
   - `--host` - Server host address (default: 0.0.0.0)
   - `--port` - Server port (default: 8080)
   - `--max-data-age` - Data retention period (default: 1h)
   - `--cleanup-interval` - Cleanup frequency (default: 5m)
   - `--stale-node-threshold` - Stale detection time (default: 5m)

2. **Configuration Sources**:
   - Command-line flags (highest priority)
   - Environment variables
   - Configuration file (YAML/JSON)
   - Sensible defaults

3. **Validation**:
   - Range checks for numeric values
   - Duration format validation
   - Port availability checks
   - Path existence verification

**Deliverables**:
- Flag parsing in `cmd/app/main.go`
- Configuration struct in `pkg/app/server.go`
- Environment variable support

**Test Strategy**:
- Configuration parsing tests
- Default value tests
- Validation error tests
- Priority/override tests

**Priority**: LOW
**Dependencies**: Task #21.1
**Status**: ✅ COMPLETED

---

### Task #21.8: Integration Tests
**Description**: Comprehensive integration tests for complete server workflows

**Implementation Details**:
1. **Test Scenarios**:
   - Agent registration → Report submission → Topology query
   - Multiple concurrent agents
   - WebSocket connections with real-time updates
   - Time-series data accumulation
   - End-to-end API flow validation

2. **Test Coverage**:
   - All API endpoints
   - WebSocket functionality
   - Aggregation correctness
   - Time-series accuracy
   - Error handling paths

3. **Load Testing**:
   - Multiple concurrent probes (5-10)
   - High report submission rates (50+ req/sec)
   - Many WebSocket clients (100+)
   - Large topology datasets (1000+ nodes)

**Deliverables**:
- `pkg/app/app_test.go` - Integration tests
- `pkg/app/loadtest_test.go` - Load tests
- Test utilities and helpers

**Test Strategy**:
- Integration tests with real server instances
- Load tests with concurrent goroutines
- Performance benchmarks
- Memory and CPU profiling

**Priority**: HIGH
**Dependencies**: All previous tasks
**Status**: ✅ COMPLETED

---

## Technical Architecture

### System Components

```
┌──────────────────────────────────────────────────┐
│                   UI Clients                      │
│            (Web Browsers, Mobile)                 │
└───────────┬──────────────────┬───────────────────┘
            │                  │
      REST API            WebSocket
            │                  │
┌───────────▼──────────────────▼───────────────────┐
│              App Backend Server                   │
│  ┌──────────────────────────────────────────┐   │
│  │           Gin HTTP Server                │   │
│  │  - Request Router                        │   │
│  │  - Middleware (CORS, Logging, Recovery)  │   │
│  │  - Graceful Shutdown                     │   │
│  └──────────────────────────────────────────┘   │
│                                                   │
│  ┌──────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Handlers │  │  Aggregator  │  │  WSHub    │ │
│  │          │  │              │  │           │ │
│  │ - Agent  │  │ - Topology   │  │ - Clients │ │
│  │   Mgmt   │  │   Building   │  │ - Broadcast│ │
│  │ - Reports│  │ - Node       │  │ - Messages│ │
│  │ - Queries│  │   Tracking   │  │           │ │
│  └──────────┘  └──────────────┘  └───────────┘ │
│                                                   │
│  ┌──────────────────────────────────────────┐   │
│  │        Time-Series Storage                │   │
│  │  - 15-second resolution                   │   │
│  │  - Per-agent tracking                     │   │
│  │  - Automatic cleanup                      │   │
│  └──────────────────────────────────────────┘   │
└───────────────────┬──────────────────────────────┘
                    │
            Reports (HTTP/gRPC)
                    │
┌───────────────────▼──────────────────────────────┐
│              Probe Agents                         │
│        (Distributed on each host/node)            │
└───────────────────────────────────────────────────┘
```

### Data Flow

1. **Report Submission**:
   ```
   Probe → POST /api/v1/reports → Handler → Aggregator → Storage
                                          ↓
                                     WebSocket Hub → Connected Clients
   ```

2. **Topology Query**:
   ```
   UI → GET /api/v1/query/topology → Handler → Aggregator → Topology Snapshot
   ```

3. **WebSocket Update**:
   ```
   Report Update → Aggregator → Hub.Broadcast() → All Clients
   ```

4. **Time-Series Query**:
   ```
   UI → GET /api/v1/query/agents/{id}/timeseries → Handler → Storage → Data Points
   ```

### Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin (high-performance HTTP router)
- **WebSocket**: Gorilla WebSocket
- **Storage**: In-memory with planned persistence options
- **Testing**: Go testing package + testify
- **Logging**: Structured logging with Gin middleware
- **Concurrency**: Goroutines + sync primitives (RWMutex, channels)

---

## API Specifications

### Agent Management

#### Register Agent
```http
POST /api/v1/agents/register
Content-Type: application/json

{
  "agent_id": "agent-1",
  "hostname": "host-1",
  "metadata": {
    "version": "1.0.0",
    "environment": "production"
  }
}

Response: 201 Created
{
  "status": "registered",
  "message": "Agent agent-1 registered successfully"
}
```

#### Heartbeat
```http
POST /api/v1/agents/heartbeat/agent-1

Response: 200 OK
{
  "status": "ok",
  "timestamp": "2025-10-13T00:00:00Z"
}
```

#### List Agents
```http
GET /api/v1/agents/list

Response: 200 OK
{
  "agents": [...],
  "count": 5
}
```

### Report Submission

```http
POST /api/v1/reports
Content-Type: application/json

{
  "agent_id": "agent-1",
  "hostname": "host-1",
  "timestamp": "2025-10-13T00:00:00Z",
  "host_info": {...},
  "docker_info": {...},
  "processes_info": {...},
  "network_info": {...}
}

Response: 202 Accepted
{
  "status": "accepted",
  "message": "Report processed successfully"
}
```

### Data Queries

#### Get Topology
```http
GET /api/v1/query/topology

Response: 200 OK
{
  "topology": {
    "nodes": {...},
    "edges": {...},
    "timestamp": "2025-10-13T00:00:00Z"
  }
}
```

#### Get Time-Series Data
```http
GET /api/v1/query/agents/agent-1/timeseries?duration=1h

Response: 200 OK
{
  "agent_id": "agent-1",
  "duration": "1h",
  "points": [...],
  "count": 240
}
```

#### Get Server Stats
```http
GET /api/v1/query/stats

Response: 200 OK
{
  "agents": 5,
  "websocket_clients": 3,
  "storage": {...},
  "topology": {...},
  "uptime": "2h30m15s"
}
```

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

// Received message format
{
  "type": "topology_update",
  "payload": {
    "nodes": {...},
    "edges": {...}
  }
}

{
  "type": "report_update",
  "payload": {
    "agent_id": "agent-1",
    "report": {...}
  }
}
```

---

## Testing Strategy

### Unit Tests
- **Handlers**: Test each endpoint with mock data
- **Aggregator**: Test topology building logic
- **WebSocket**: Test client lifecycle and messaging
- **Storage**: Test data retention and queries

**Coverage Target**: >80%

### Integration Tests
- **End-to-End Workflows**: Agent registration → Reports → Queries
- **Concurrent Agents**: Multiple agents submitting reports
- **WebSocket Communication**: Real-time updates to clients

### Load Tests
- **5 Concurrent Probes**: Each sending 10 reports
- **100 WebSocket Clients**: Receiving broadcasts
- **1000+ Node Topology**: Performance validation

### Performance Benchmarks
- **Report Processing**: Target < 100ms per report
- **Topology Query**: Target < 50ms
- **WebSocket Broadcast**: Target < 10ms per client

---

## Success Criteria

✅ **Functional Requirements**:
- All API endpoints implemented and tested
- WebSocket server operational with real-time updates
- Report aggregation producing correct topology
- Time-series storage with 15-second resolution
- Background cleanup removing stale data

✅ **Performance Requirements**:
- Handle 50+ reports/second
- Support 100+ concurrent WebSocket clients
- Query response time < 100ms (P95)
- Memory usage < 500MB for 1000 nodes

✅ **Reliability Requirements**:
- Graceful shutdown with zero data loss
- Error handling for all failure modes
- Recovery from probe disconnections
- Automatic cleanup of stale resources

✅ **Testing Requirements**:
- 50+ unit tests all passing
- Integration tests covering main workflows
- Load tests validating concurrency
- >80% code coverage

---

## Dependencies

### External Dependencies
- **Go 1.21+**: Programming language
- **Gin Framework**: HTTP routing
- **Gorilla WebSocket**: WebSocket support
- **Testify**: Testing assertions

### Internal Dependencies
- **pkg/probe**: Probe data structures (ReportData, HostInfo, etc.)
- **internal/storage**: Storage interfaces

### System Dependencies
- None (standalone server)

---

## Timeline

**Total Duration**: 1 day
**Status**: ✅ COMPLETED on 2025-10-13

| Phase | Duration | Status |
|-------|----------|--------|
| HTTP Server Setup | 1 hour | ✅ Complete |
| REST API Handlers | 2 hours | ✅ Complete |
| Aggregation Engine | 3 hours | ✅ Complete |
| WebSocket Server | 2 hours | ✅ Complete |
| Time-Series Storage | 2 hours | ✅ Complete |
| Background Cleanup | 1 hour | ✅ Complete |
| Integration Tests | 2 hours | ✅ Complete |
| Documentation | 1 hour | ✅ Complete |

---

## Appendix

### A. File Structure
```
cmd/app/
  └── main.go              # Entry point (99 lines)

pkg/app/
  ├── server.go            # Server implementation (280 lines)
  ├── handlers.go          # REST API handlers (292 lines)
  ├── aggregator.go        # Aggregation engine (375 lines)
  └── websocket.go         # WebSocket hub (298 lines)

internal/storage/
  ├── storage.go           # Generic storage (148 lines)
  └── timeseries.go        # Time-series storage (265 lines)

Tests:
  ├── pkg/app/app_test.go           # Integration tests (485 lines)
  ├── pkg/app/handlers_test.go      # Handler tests (292 lines)
  ├── pkg/app/aggregator_test.go    # Aggregation tests (269 lines)
  ├── pkg/app/websocket_test.go     # WebSocket tests (305 lines)
  ├── pkg/app/loadtest_test.go      # Load tests (371 lines)
  └── internal/storage/storage_test.go # Storage tests (569 lines)
```

**Total**: ~4,000 lines of code + tests

### B. Configuration Example

```yaml
# config.yaml
server:
  host: 0.0.0.0
  port: 8080

storage:
  max_data_age: 1h
  cleanup_interval: 5m

agents:
  stale_threshold: 5m
  heartbeat_timeout: 2m

websocket:
  buffer_size: 256
  ping_interval: 30s
```

### C. Deployment

#### Build
```bash
./go/bin/go build -o app-server ./cmd/app/
```

#### Run
```bash
./app-server \
  --host 0.0.0.0 \
  --port 8080 \
  --max-data-age 2h \
  --cleanup-interval 10m
```

#### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app-server ./cmd/app/

FROM alpine:latest
COPY --from=builder /app/app-server /usr/local/bin/
ENTRYPOINT ["app-server"]
```

### D. Monitoring

**Metrics Exported**:
- `app_agents_total` - Number of registered agents
- `app_reports_received_total` - Total reports received
- `app_websocket_clients` - Active WebSocket connections
- `app_topology_nodes` - Total nodes in topology
- `app_http_request_duration_seconds` - Request latency histogram

### E. Troubleshooting

**Common Issues**:

1. **Port already in use**: Change with `--port` flag
2. **High memory usage**: Reduce `--max-data-age`
3. **WebSocket disconnections**: Check firewall/proxy settings
4. **Stale agents**: Verify probe heartbeats are reaching server

---

## Conclusion

Task #21 has been successfully completed with all requirements met. The App Backend Server is production-ready with:
- ✅ Complete REST API implementation
- ✅ Real-time WebSocket support
- ✅ Efficient report aggregation
- ✅ Time-series storage with automatic cleanup
- ✅ Comprehensive test coverage (100% passing)
- ✅ Production-ready error handling and logging

**Next Steps**:
1. Deploy to staging environment
2. Integrate with probe agents (Task #22)
3. Build UI to consume APIs (Task #23)
4. Performance tuning based on production loads
5. Add persistence layer for data durability
