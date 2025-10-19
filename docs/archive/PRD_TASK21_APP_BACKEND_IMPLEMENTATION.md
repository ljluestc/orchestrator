# Product Requirements Document: ORCHESTRATOR: Task21 App Backend Implementation

---

## Document Information
**Project:** orchestrator
**Document:** TASK21_APP_BACKEND_IMPLEMENTATION
**Version:** 1.0.0
**Date:** 2025-10-13
**Status:** READY FOR TASK-MASTER PARSING

---

## 1. EXECUTIVE SUMMARY

### 1.1 Overview
This PRD captures the requirements and implementation details for ORCHESTRATOR: Task21 App Backend Implementation.

### 1.2 Purpose
This document provides a structured specification that can be parsed by task-master to generate actionable tasks.

### 1.3 Scope
The scope includes all requirements, features, and implementation details from the original documentation.

---

## 2. REQUIREMENTS


## 3. TASKS

The following tasks have been identified for implementation:

**TASK_001** [MEDIUM]: **Purpose**: Main server implementation with lifecycle management

**TASK_002** [MEDIUM]: HTTP server using Gin framework

**TASK_003** [MEDIUM]: Graceful start/stop with context management

**TASK_004** [MEDIUM]: Background cleanup tasks

**TASK_005** [MEDIUM]: Agent registration and tracking

**TASK_006** [MEDIUM]: CORS middleware for cross-origin requests

**TASK_007** [MEDIUM]: `NewServer(config)`: Creates new server instance

**TASK_008** [MEDIUM]: `Start(ctx)`: Starts HTTP server and background tasks

**TASK_009** [MEDIUM]: `Stop()`: Gracefully shuts down server

**TASK_010** [MEDIUM]: `GetStats()`: Returns server statistics

**TASK_011** [MEDIUM]: **Purpose**: HTTP request handlers for all REST endpoints

**TASK_012** [MEDIUM]: `POST /api/v1/agents/register` - Register new agent

**TASK_013** [MEDIUM]: `POST /api/v1/agents/heartbeat/:agent_id` - Agent heartbeat

**TASK_014** [MEDIUM]: `GET /api/v1/agents/config/:agent_id` - Get agent configuration

**TASK_015** [MEDIUM]: `GET /api/v1/agents/list` - List all registered agents

**TASK_016** [MEDIUM]: `POST /api/v1/reports` - Submit probe report

**TASK_017** [MEDIUM]: `GET /api/v1/query/topology` - Get current topology view

**TASK_018** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/latest` - Get latest report for agent

**TASK_019** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/timeseries?duration=1h` - Get time-series data

**TASK_020** [MEDIUM]: `GET /api/v1/query/stats` - Get server statistics

**TASK_021** [MEDIUM]: `GET /health` - Health check endpoint

**TASK_022** [MEDIUM]: `GET /api/v1/ping` - Ping endpoint for connectivity test

**TASK_023** [MEDIUM]: `GET /api/v1/ws` - WebSocket endpoint for real-time updates

**TASK_024** [MEDIUM]: **Purpose**: Report aggregation and topology building

**TASK_025** [MEDIUM]: Merges probe data into unified topology view

**TASK_026** [MEDIUM]: Maintains hierarchical node structure (host → container → process)

**TASK_027** [MEDIUM]: Tracks network connections between nodes

**TASK_028** [MEDIUM]: Automatic cleanup of stale nodes

**TASK_029** [MEDIUM]: `ProcessReport(report)`: Processes and aggregates report data

**TASK_030** [MEDIUM]: `GetTopology()`: Returns current topology snapshot

**TASK_031** [MEDIUM]: `GetNodesByType(type)`: Filters nodes by type

**TASK_032** [MEDIUM]: `CleanStaleNodes(maxAge)`: Removes old nodes

**TASK_033** [MEDIUM]: **Purpose**: Real-time updates to connected UI clients

**TASK_034** [MEDIUM]: Manages WebSocket client connections

**TASK_035** [MEDIUM]: Broadcasts topology updates to all clients

**TASK_036** [MEDIUM]: Broadcasts report updates

**TASK_037** [MEDIUM]: Automatic client cleanup on disconnect

**TASK_038** [MEDIUM]: Ping/pong keep-alive mechanism

**TASK_039** [MEDIUM]: `topology_update`: Full topology updates

**TASK_040** [MEDIUM]: `report_update`: Individual agent report updates

**TASK_041** [MEDIUM]: `ping/pong`: Keep-alive messages

**TASK_042** [MEDIUM]: `writeWait`: 10 seconds (time to write message)

**TASK_043** [MEDIUM]: `pongWait`: 60 seconds (time to receive pong)

**TASK_044** [MEDIUM]: `pingPeriod`: 54 seconds (ping interval)

**TASK_045** [MEDIUM]: `maxMessageSize`: 512 KB

**TASK_046** [MEDIUM]: **Purpose**: Time-series data storage with 15-second resolution

**TASK_047** [MEDIUM]: Generic in-memory key-value storage

**TASK_048** [MEDIUM]: Thread-safe operations with RWMutex

**TASK_049** [MEDIUM]: Context-aware operations

**TASK_050** [MEDIUM]: Methods: Store, Get, Delete, List, Exists, Clear, Size, Close

**TASK_051** [MEDIUM]: Time-series data storage specifically for probe reports

**TASK_052** [MEDIUM]: 15-second resolution as specified in requirements

**TASK_053** [MEDIUM]: Automatic data expiration based on MaxDataAge

**TASK_054** [MEDIUM]: Background cleanup goroutine

**TASK_055** [MEDIUM]: `AddReport(report)`: Adds report to time-series

**TASK_056** [MEDIUM]: `GetLatestReport(agentID)`: Gets most recent report

**TASK_057** [MEDIUM]: `GetRecentPoints(agentID, duration)`: Gets points within duration

**TASK_058** [MEDIUM]: `GetAllAgents()`: Lists all agent IDs

**TASK_059** [MEDIUM]: `GetStats()`: Returns storage statistics

**TASK_060** [MEDIUM]: **Purpose**: Application entry point with CLI flags

**TASK_061** [MEDIUM]: Signal handling (SIGINT, SIGTERM)

**TASK_062** [MEDIUM]: Graceful shutdown

**TASK_063** [MEDIUM]: Configuration display on startup

**TASK_064** [MEDIUM]: Comprehensive endpoint listing

**TASK_065** [HIGH]: Probe sends report via `POST /api/v1/reports`

**TASK_066** [HIGH]: Handler validates report structure

**TASK_067** [HIGH]: Report is stored in time-series storage

**TASK_068** [HIGH]: Aggregator processes report and updates topology

**TASK_069** [HIGH]: WebSocket hub broadcasts update to connected clients

**TASK_070** [HIGH]: Response returned to probe

**TASK_071** [HIGH]: Client requests topology via `GET /api/v1/query/topology`

**TASK_072** [HIGH]: Handler calls aggregator.GetTopology()

**TASK_073** [HIGH]: Aggregator creates thread-safe snapshot of current topology

**TASK_074** [HIGH]: Topology JSON returned to client

**TASK_075** [HIGH]: Client connects via WebSocket at `/api/v1/ws`

**TASK_076** [HIGH]: Server sends initial topology snapshot

**TASK_077** [HIGH]: As reports arrive, updates are broadcast to all clients

**TASK_078** [HIGH]: Keep-alive pings maintain connection

**TASK_079** [HIGH]: Client disconnection triggers automatic cleanup

**TASK_080** [MEDIUM]: ✅ Health check endpoint

**TASK_081** [MEDIUM]: ✅ Ping endpoint

**TASK_082** [MEDIUM]: ✅ Agent registration

**TASK_083** [MEDIUM]: ✅ Agent heartbeat

**TASK_084** [MEDIUM]: ✅ Report submission

**TASK_085** [MEDIUM]: ✅ Topology queries

**TASK_086** [MEDIUM]: ✅ Latest report retrieval

**TASK_087** [MEDIUM]: ✅ Time-series data queries

**TASK_088** [MEDIUM]: ✅ Statistics endpoint

**TASK_089** [MEDIUM]: ✅ Agent listing

**TASK_090** [MEDIUM]: ✅ Server start/stop lifecycle

**TASK_091** [MEDIUM]: ✅ Host node creation

**TASK_092** [MEDIUM]: ✅ Container node aggregation

**TASK_093** [MEDIUM]: ✅ Process node aggregation

**TASK_094** [MEDIUM]: ✅ Network connection tracking

**TASK_095** [MEDIUM]: ✅ Node filtering by type

**TASK_096** [MEDIUM]: ✅ Node retrieval by ID

**TASK_097** [MEDIUM]: ✅ Stale node cleanup

**TASK_098** [MEDIUM]: ✅ Statistics generation

**TASK_099** [MEDIUM]: ✅ Container ID extraction from cgroups

**TASK_100** [MEDIUM]: ✅ Concurrent report processing

**TASK_101** [MEDIUM]: ✅ Hub creation and initialization

**TASK_102** [MEDIUM]: ✅ Client registration/unregistration

**TASK_103** [MEDIUM]: ✅ Message broadcasting

**TASK_104** [MEDIUM]: ✅ Topology update broadcasting

**TASK_105** [MEDIUM]: ✅ Report update broadcasting

**TASK_106** [MEDIUM]: ✅ Client message serialization

**TASK_107** [MEDIUM]: ✅ Client connection lifecycle

**TASK_108** [MEDIUM]: ✅ Read/write pump integration

**TASK_109** [MEDIUM]: ✅ Ping/pong keep-alive

**TASK_110** [MEDIUM]: ✅ Basic CRUD operations

**TASK_111** [MEDIUM]: ✅ Context handling

**TASK_112** [MEDIUM]: ✅ Concurrent access

**TASK_113** [MEDIUM]: ✅ Error handling

**TASK_114** [MEDIUM]: ✅ Time-series data points

**TASK_115** [MEDIUM]: ✅ Recent points retrieval

**TASK_116** [MEDIUM]: ✅ Latest report retrieval

**TASK_117** [MEDIUM]: ✅ Agent management

**TASK_118** [MEDIUM]: **Scenario**: 5 probes sending 20 reports each concurrently

**TASK_119** [MEDIUM]: Total requests processed

**TASK_120** [MEDIUM]: Success/failure rate

**TASK_121** [MEDIUM]: Requests per second

**TASK_122** [MEDIUM]: Topology node count

**TASK_123** [MEDIUM]: Storage statistics

**TASK_124** [MEDIUM]: **Success Criteria**: >95% success rate

**TASK_125** [MEDIUM]: **Scenario**: 10 readers performing 20 read operations each

**TASK_126** [MEDIUM]: Topology queries

**TASK_127** [MEDIUM]: Statistics queries

**TASK_128** [MEDIUM]: Agent list queries

**TASK_129** [MEDIUM]: Health checks

**TASK_130** [MEDIUM]: **Success Criteria**: 100% success rate

**TASK_131** [MEDIUM]: **Scenario**: 5 workers performing mixed read/write operations

**TASK_132** [MEDIUM]: **Distribution**: 50% writes, 50% reads

**TASK_133** [MEDIUM]: Write throughput

**TASK_134** [MEDIUM]: Read throughput

**TASK_135** [MEDIUM]: Operations per second

**TASK_136** [MEDIUM]: **Scenario**: 100 reports from 10 agents

**TASK_137** [MEDIUM]: Memory doesn't leak

**TASK_138** [MEDIUM]: Cleanup works correctly

**TASK_139** [MEDIUM]: Data structures remain stable

**TASK_140** [MEDIUM]: `BenchmarkReportSubmission`: Report submission throughput

**TASK_141** [MEDIUM]: `BenchmarkTopologyQuery`: Topology query performance

**TASK_142** [MEDIUM]: Complete workflow from agent registration to data aggregation

**TASK_143** [MEDIUM]: Tests all endpoints in sequence

**TASK_144** [MEDIUM]: Verifies data consistency across components

**TASK_145** [MEDIUM]: Tests: registration → reports → queries → stats → cleanup

**TASK_146** [MEDIUM]: Simulates real probe behavior over time

**TASK_147** [MEDIUM]: Multiple report submissions with time intervals

**TASK_148** [MEDIUM]: Verifies time-series data accumulation

**TASK_149** [MEDIUM]: Validates topology updates

**TASK_150** [MEDIUM]: Invalid report handling

**TASK_151** [MEDIUM]: Non-existent agent queries

**TASK_152** [MEDIUM]: Invalid parameter handling

**TASK_153** [MEDIUM]: Missing required fields

**TASK_154** [MEDIUM]: Default configuration application

**TASK_155** [MEDIUM]: Custom configuration handling

**TASK_156** [MEDIUM]: Configuration validation

**TASK_157** [MEDIUM]: **Concurrent Probes**: Tested with 5+ concurrent probes

**TASK_158** [MEDIUM]: **Report Rate**: 100+ reports per second

**TASK_159** [MEDIUM]: **Read Throughput**: 200+ queries per second

**TASK_160** [MEDIUM]: **WebSocket Clients**: Supports multiple simultaneous connections

**TASK_161** [MEDIUM]: **Memory**: Automatic cleanup prevents unbounded growth

**TASK_162** [MEDIUM]: **Storage**: Time-series data expires after MaxDataAge

**TASK_163** [MEDIUM]: **Connections**: WebSocket cleanup on disconnect

**TASK_164** [MEDIUM]: **Goroutines**: Proper cleanup on shutdown

**TASK_165** [MEDIUM]: **Time-Series**: Configurable (default: 1 hour)

**TASK_166** [MEDIUM]: **Resolution**: 15 seconds as specified

**TASK_167** [MEDIUM]: **Topology**: Real-time with stale node removal

**TASK_168** [MEDIUM]: **Agent Registry**: Active agents only

**TASK_169** [MEDIUM]: **Endpoint**: `GET /health`

**TASK_170** [MEDIUM]: **Response**: `{"status": "healthy", "timestamp": "..."}`

**TASK_171** [MEDIUM]: **Endpoint**: `GET /api/v1/query/stats`

**TASK_172** [MEDIUM]: Connected agents count

**TASK_173** [MEDIUM]: WebSocket client count

**TASK_174** [MEDIUM]: Storage statistics (total points, agents)

**TASK_175** [MEDIUM]: Topology statistics (nodes, edges by type)

**TASK_176** [MEDIUM]: Server uptime

**TASK_177** [MEDIUM]: Structured logging for all major operations

**TASK_178** [MEDIUM]: Agent registration/disconnection events

**TASK_179** [MEDIUM]: Report reception confirmations

**TASK_180** [MEDIUM]: WebSocket connection events

**TASK_181** [MEDIUM]: Error conditions with context

**TASK_182** [MEDIUM]: `github.com/gin-gonic/gin` - HTTP framework

**TASK_183** [MEDIUM]: `github.com/gorilla/websocket` - WebSocket support

**TASK_184** [MEDIUM]: `github.com/gorilla/mux` - HTTP routing (legacy)

**TASK_185** [MEDIUM]: `github.com/stretchr/testify` - Test assertions and utilities

**TASK_186** [HIGH]: **Persistence Layer**: Add database backend (PostgreSQL, TimescaleDB)

**TASK_187** [HIGH]: **Authentication**: Implement API key validation

**TASK_188** [HIGH]: **Authorization**: Add RBAC for multi-tenancy

**TASK_189** [HIGH]: **Metrics Export**: Prometheus exporter for server metrics

**TASK_190** [HIGH]: **Distributed Tracing**: OpenTelemetry integration

**TASK_191** [HIGH]: **Rate Limiting**: Protect against abuse

**TASK_192** [HIGH]: **Compression**: WebSocket message compression

**TASK_193** [HIGH]: **Filtering**: Advanced topology filtering by tags

**TASK_194** [HIGH]: **Alerting**: Built-in alerting based on thresholds

**TASK_195** [HIGH]: **Data Aggregation**: Time-series downsampling for long-term storage


## 4. DETAILED SPECIFICATIONS

### 4.1 Original Content

The following sections contain the original documentation:


#### Task 21 App Backend Server With Report Aggregation Implementation Summary

# Task #21: App Backend Server with Report Aggregation - Implementation Summary


#### Overview

## Overview
The app backend server is fully implemented as a central hub that receives reports from probes, aggregates data, and provides REST/WebSocket APIs for the UI. The implementation follows modern Go best practices and provides a robust, scalable solution.


#### Architecture

## Architecture


#### Core Components

### Core Components


#### 1 Server Pkg App Server Go 

#### 1. Server (`pkg/app/server.go`)
- **Purpose**: Main server implementation with lifecycle management
- **Key Features**:
  - HTTP server using Gin framework
  - Graceful start/stop with context management
  - Background cleanup tasks
  - Agent registration and tracking
  - CORS middleware for cross-origin requests

- **Configuration**:
  ```go
  type ServerConfig struct {
      Host               string        // Server host address (default: 0.0.0.0)
      Port               int           // Server port (default: 8080)
      MaxDataAge         time.Duration // Maximum age for stored data (default: 1h)
      CleanupInterval    time.Duration // Cleanup interval (default: 5m)
      StaleNodeThreshold time.Duration // Stale node threshold (default: 5m)
  }
  ```

- **Key Methods**:
  - `NewServer(config)`: Creates new server instance
  - `Start(ctx)`: Starts HTTP server and background tasks
  - `Stop()`: Gracefully shuts down server
  - `GetStats()`: Returns server statistics


#### 2 Handlers Pkg App Handlers Go 

#### 2. Handlers (`pkg/app/handlers.go`)
- **Purpose**: HTTP request handlers for all REST endpoints
- **Endpoints Implemented**:


#### Agent Management

##### Agent Management
- `POST /api/v1/agents/register` - Register new agent
- `POST /api/v1/agents/heartbeat/:agent_id` - Agent heartbeat
- `GET /api/v1/agents/config/:agent_id` - Get agent configuration
- `GET /api/v1/agents/list` - List all registered agents


#### Report Submission

##### Report Submission
- `POST /api/v1/reports` - Submit probe report


#### Data Queries

##### Data Queries
- `GET /api/v1/query/topology` - Get current topology view
- `GET /api/v1/query/agents/:agent_id/latest` - Get latest report for agent
- `GET /api/v1/query/agents/:agent_id/timeseries?duration=1h` - Get time-series data
- `GET /api/v1/query/stats` - Get server statistics


#### System Endpoints

##### System Endpoints
- `GET /health` - Health check endpoint
- `GET /api/v1/ping` - Ping endpoint for connectivity test


#### Websocket

##### WebSocket
- `GET /api/v1/ws` - WebSocket endpoint for real-time updates


#### 3 Aggregator Pkg App Aggregator Go 

#### 3. Aggregator (`pkg/app/aggregator.go`)
- **Purpose**: Report aggregation and topology building
- **Key Features**:
  - Merges probe data into unified topology view
  - Maintains hierarchical node structure (host → container → process)
  - Tracks network connections between nodes
  - Automatic cleanup of stale nodes

- **Data Structures**:
  ```go
  type TopologyNode struct {
      ID       string                 // Unique node identifier
      Type     string                 // "host", "container", or "process"
      Name     string                 // Human-readable name
      Metadata map[string]interface{} // Node metadata
      ParentID string                 // Parent node ID (for hierarchy)
  }

  type TopologyEdge struct {
      Source      string                 // Source node ID
      Target      string                 // Target node ID
      Type        string                 // "network" or "parent-child"
      Protocol    string                 // Network protocol (if applicable)
      Metadata    map[string]interface{} // Edge metadata
      Connections int                    // Connection count
  }
  ```

- **Key Methods**:
  - `ProcessReport(report)`: Processes and aggregates report data
  - `GetTopology()`: Returns current topology snapshot
  - `GetNodesByType(type)`: Filters nodes by type
  - `CleanStaleNodes(maxAge)`: Removes old nodes


#### 4 Websocket Hub Pkg App Websocket Go 

#### 4. WebSocket Hub (`pkg/app/websocket.go`)
- **Purpose**: Real-time updates to connected UI clients
- **Key Features**:
  - Manages WebSocket client connections
  - Broadcasts topology updates to all clients
  - Broadcasts report updates
  - Automatic client cleanup on disconnect
  - Ping/pong keep-alive mechanism

- **Message Types**:
  ```go
  type WSMessage struct {
      Type    string      // Message type
      Payload interface{} // Message payload
  }
  ```

- **Supported Message Types**:
  - `topology_update`: Full topology updates
  - `report_update`: Individual agent report updates
  - `ping/pong`: Keep-alive messages

- **Constants**:
  - `writeWait`: 10 seconds (time to write message)
  - `pongWait`: 60 seconds (time to receive pong)
  - `pingPeriod`: 54 seconds (ping interval)
  - `maxMessageSize`: 512 KB


#### 5 Storage Layer Internal Storage 

#### 5. Storage Layer (`internal/storage/`)
- **Purpose**: Time-series data storage with 15-second resolution
- **Components**:


####  Storage Go 

##### `storage.go`
- Generic in-memory key-value storage
- Thread-safe operations with RWMutex
- Context-aware operations
- Methods: Store, Get, Delete, List, Exists, Clear, Size, Close


####  Timeseries Go 

##### `timeseries.go`
- Time-series data storage specifically for probe reports
- 15-second resolution as specified in requirements
- Automatic data expiration based on MaxDataAge
- Background cleanup goroutine

```go
type TimeSeriesStore struct {
    maxAge     time.Duration // Data retention period
    resolution time.Duration // 15-second resolution
    // ... other fields
}
```

- **Key Methods**:
  - `AddReport(report)`: Adds report to time-series
  - `GetLatestReport(agentID)`: Gets most recent report
  - `GetRecentPoints(agentID, duration)`: Gets points within duration
  - `GetAllAgents()`: Lists all agent IDs
  - `GetStats()`: Returns storage statistics


#### 6 Command Line Entry Point Cmd App Main Go 

#### 6. Command Line Entry Point (`cmd/app/main.go`)
- **Purpose**: Application entry point with CLI flags
- **Flags**:
  ```
  -host string
      Server host address (default "0.0.0.0")
  -port int
      Server port (default 8080)
  -max-data-age duration
      Maximum age for stored data (default 1h)
  -cleanup-interval duration
      Cleanup interval for stale data (default 5m)
  -stale-node-threshold duration
      Threshold for considering nodes stale (default 5m)
  ```

- **Features**:
  - Signal handling (SIGINT, SIGTERM)
  - Graceful shutdown
  - Configuration display on startup
  - Comprehensive endpoint listing


#### Data Flow

## Data Flow


#### Report Submission Flow

### Report Submission Flow
1. Probe sends report via `POST /api/v1/reports`
2. Handler validates report structure
3. Report is stored in time-series storage
4. Aggregator processes report and updates topology
5. WebSocket hub broadcasts update to connected clients
6. Response returned to probe


#### Topology Query Flow

### Topology Query Flow
1. Client requests topology via `GET /api/v1/query/topology`
2. Handler calls aggregator.GetTopology()
3. Aggregator creates thread-safe snapshot of current topology
4. Topology JSON returned to client


#### Websocket Update Flow

### WebSocket Update Flow
1. Client connects via WebSocket at `/api/v1/ws`
2. Server sends initial topology snapshot
3. As reports arrive, updates are broadcast to all clients
4. Keep-alive pings maintain connection
5. Client disconnection triggers automatic cleanup


#### Testing Strategy

## Testing Strategy


#### Unit Tests

### Unit Tests


#### Handler Tests Handlers Test Go 

#### Handler Tests (`handlers_test.go`)
- ✅ Health check endpoint
- ✅ Ping endpoint
- ✅ Agent registration
- ✅ Agent heartbeat
- ✅ Report submission
- ✅ Topology queries
- ✅ Latest report retrieval
- ✅ Time-series data queries
- ✅ Statistics endpoint
- ✅ Agent listing
- ✅ Server start/stop lifecycle


#### Aggregator Tests Aggregator Test Go 

#### Aggregator Tests (`aggregator_test.go`)
- ✅ Host node creation
- ✅ Container node aggregation
- ✅ Process node aggregation
- ✅ Network connection tracking
- ✅ Node filtering by type
- ✅ Node retrieval by ID
- ✅ Stale node cleanup
- ✅ Statistics generation
- ✅ Container ID extraction from cgroups
- ✅ Concurrent report processing


#### Websocket Tests Websocket Test Go 

#### WebSocket Tests (`websocket_test.go`)
- ✅ Hub creation and initialization
- ✅ Client registration/unregistration
- ✅ Message broadcasting
- ✅ Topology update broadcasting
- ✅ Report update broadcasting
- ✅ Client message serialization
- ✅ Client connection lifecycle
- ✅ Read/write pump integration
- ✅ Ping/pong keep-alive


#### Storage Tests Internal Storage Storage Test Go 

#### Storage Tests (`internal/storage/storage_test.go`)
- ✅ Basic CRUD operations
- ✅ Context handling
- ✅ Concurrent access
- ✅ Error handling
- ✅ Time-series data points
- ✅ Recent points retrieval
- ✅ Latest report retrieval
- ✅ Agent management


#### Load Tests Loadtest Test Go 

### Load Tests (`loadtest_test.go`)


#### Multiple Concurrent Probes Test

#### Multiple Concurrent Probes Test
- **Scenario**: 5 probes sending 20 reports each concurrently
- **Metrics**:
  - Total requests processed
  - Success/failure rate
  - Requests per second
  - Topology node count
  - Storage statistics
- **Success Criteria**: >95% success rate


#### Concurrent Reads Test

#### Concurrent Reads Test
- **Scenario**: 10 readers performing 20 read operations each
- **Operations**:
  - Topology queries
  - Statistics queries
  - Agent list queries
  - Health checks
- **Success Criteria**: 100% success rate


#### Mixed Workload Test

#### Mixed Workload Test
- **Scenario**: 5 workers performing mixed read/write operations
- **Distribution**: 50% writes, 50% reads
- **Metrics**:
  - Write throughput
  - Read throughput
  - Operations per second


#### Memory Usage Test

#### Memory Usage Test
- **Scenario**: 100 reports from 10 agents
- **Verification**:
  - Memory doesn't leak
  - Cleanup works correctly
  - Data structures remain stable


#### Benchmark Tests

### Benchmark Tests
- `BenchmarkReportSubmission`: Report submission throughput
- `BenchmarkTopologyQuery`: Topology query performance


#### Integration Tests App Test Go 

### Integration Tests (`app_test.go`)


#### Full Integration Test

#### Full Integration Test
- Complete workflow from agent registration to data aggregation
- Tests all endpoints in sequence
- Verifies data consistency across components
- Tests: registration → reports → queries → stats → cleanup


#### End To End Test

#### End-to-End Test
- Simulates real probe behavior over time
- Multiple report submissions with time intervals
- Verifies time-series data accumulation
- Validates topology updates


#### Resilience Test

#### Resilience Test
- Invalid report handling
- Non-existent agent queries
- Invalid parameter handling
- Missing required fields


#### Configuration Test

#### Configuration Test
- Default configuration application
- Custom configuration handling
- Configuration validation


#### Performance Characteristics

## Performance Characteristics


#### Scalability

### Scalability
- **Concurrent Probes**: Tested with 5+ concurrent probes
- **Report Rate**: 100+ reports per second
- **Read Throughput**: 200+ queries per second
- **WebSocket Clients**: Supports multiple simultaneous connections


#### Resource Management

### Resource Management
- **Memory**: Automatic cleanup prevents unbounded growth
- **Storage**: Time-series data expires after MaxDataAge
- **Connections**: WebSocket cleanup on disconnect
- **Goroutines**: Proper cleanup on shutdown


#### Data Retention

### Data Retention
- **Time-Series**: Configurable (default: 1 hour)
- **Resolution**: 15 seconds as specified
- **Topology**: Real-time with stale node removal
- **Agent Registry**: Active agents only


#### Api Examples

## API Examples


#### Register Agent

### Register Agent
```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-1",
    "hostname": "web-server-01",
    "metadata": {"version": "1.0.0"}
  }'
```


#### Submit Report

### Submit Report
```bash
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-1",
    "hostname": "web-server-01",
    "timestamp": "2024-01-01T12:00:00Z",
    "host_info": {
      "cpu_info": {"cores": 8, "usage": 45.5},
      "memory_info": {"total_mb": 16384, "used_mb": 8192}
    }
  }'
```


#### Get Topology

### Get Topology
```bash
curl http://localhost:8080/api/v1/query/topology
```


#### Get Time Series Data

### Get Time-Series Data
```bash
curl http://localhost:8080/api/v1/query/agents/agent-1/timeseries?duration=1h
```


#### Websocket Connection Javascript 

### WebSocket Connection (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'topology_update') {
    // Handle topology update
    console.log('Topology:', message.payload);
  }
};

ws.send(JSON.stringify({
  type: 'subscribe',
  payload: {view: 'containers'}
}));
```


#### Deployment

## Deployment


#### Building

### Building
```bash
cd cmd/app
go build -o app-server .
```


#### Running

### Running
```bash
./app-server \
  -host 0.0.0.0 \
  -port 8080 \
  -max-data-age 2h \
  -cleanup-interval 10m \
  -stale-node-threshold 10m
```


#### Docker Support

### Docker Support
See `Dockerfile.app` for containerized deployment.


#### Kubernetes Support

### Kubernetes Support
See `k8s/base/monitoring-app/` for Kubernetes manifests.


#### Monitoring And Observability

## Monitoring and Observability


#### Health Checks

### Health Checks
- **Endpoint**: `GET /health`
- **Response**: `{"status": "healthy", "timestamp": "..."}`


#### Metrics

### Metrics
- **Endpoint**: `GET /api/v1/query/stats`
- **Metrics**:
  - Connected agents count
  - WebSocket client count
  - Storage statistics (total points, agents)
  - Topology statistics (nodes, edges by type)
  - Server uptime


#### Logging

### Logging
- Structured logging for all major operations
- Agent registration/disconnection events
- Report reception confirmations
- WebSocket connection events
- Error conditions with context


#### Files Structure

## Files Structure

```
orchestrator/
├── cmd/app/
│   └── main.go                      # Entry point
├── pkg/app/
│   ├── app.go                       # Legacy app implementation
│   ├── server.go                    # Main server implementation
│   ├── handlers.go                  # HTTP handlers
│   ├── aggregator.go                # Report aggregation
│   ├── websocket.go                 # WebSocket hub
│   ├── app_test.go                  # Integration tests (NEW)
│   ├── handlers_test.go             # Handler unit tests
│   ├── aggregator_test.go           # Aggregator unit tests
│   ├── websocket_test.go            # WebSocket unit tests
│   └── loadtest_test.go             # Load testing
├── internal/storage/
│   ├── storage.go                   # Generic storage
│   ├── timeseries.go                # Time-series storage
│   └── storage_test.go              # Storage tests
└── k8s/base/monitoring-app/         # Kubernetes manifests
```


#### Dependencies

## Dependencies


#### Core Dependencies

### Core Dependencies
- `github.com/gin-gonic/gin` - HTTP framework
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/gorilla/mux` - HTTP routing (legacy)


#### Testing Dependencies

### Testing Dependencies
- `github.com/stretchr/testify` - Test assertions and utilities


#### Future Enhancements

## Future Enhancements


#### Potential Improvements

### Potential Improvements
1. **Persistence Layer**: Add database backend (PostgreSQL, TimescaleDB)
2. **Authentication**: Implement API key validation
3. **Authorization**: Add RBAC for multi-tenancy
4. **Metrics Export**: Prometheus exporter for server metrics
5. **Distributed Tracing**: OpenTelemetry integration
6. **Rate Limiting**: Protect against abuse
7. **Compression**: WebSocket message compression
8. **Filtering**: Advanced topology filtering by tags
9. **Alerting**: Built-in alerting based on thresholds
10. **Data Aggregation**: Time-series downsampling for long-term storage


#### Testing Instructions

## Testing Instructions


#### Run All Tests

### Run All Tests
```bash
go test ./pkg/app/... -v
```


#### Run Specific Test Suite

### Run Specific Test Suite
```bash
go test ./pkg/app -run TestHandlers -v
go test ./pkg/app -run TestAggregator -v
go test ./pkg/app -run TestWebSocket -v
go test ./pkg/app -run TestIntegration -v
```


#### Run Load Tests

### Run Load Tests
```bash
go test ./pkg/app -run TestLoad -v
```


#### Run Benchmarks

### Run Benchmarks
```bash
go test ./pkg/app -bench=. -benchmem
```


#### Skip Long Running Tests

### Skip Long-Running Tests
```bash
go test ./pkg/app -short -v
```


#### Conclusion

## Conclusion

The app backend server implementation is **complete and production-ready**. It provides:

✅ **Full REST API** for agent management and data queries
✅ **WebSocket support** for real-time UI updates
✅ **Report aggregation** with intelligent topology building
✅ **Time-series storage** with 15-second resolution
✅ **Comprehensive testing** (unit, integration, load, benchmarks)
✅ **Graceful lifecycle management** (start, stop, cleanup)
✅ **Production features** (logging, metrics, health checks)
✅ **Scalability** (concurrent probes, high throughput)
✅ **Documentation** (code comments, API examples, deployment)

The implementation follows Go best practices, is well-tested, and ready for deployment in a production environment.


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
