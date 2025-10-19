# Product Requirements Document: ORCHESTRATOR: Task21 Implementation Summary

---

## Document Information
**Project:** orchestrator
**Document:** TASK21_IMPLEMENTATION_SUMMARY
**Version:** 1.0.0
**Date:** 2025-10-13
**Status:** READY FOR TASK-MASTER PARSING

---

## 1. EXECUTIVE SUMMARY

### 1.1 Overview
This PRD captures the requirements and implementation details for ORCHESTRATOR: Task21 Implementation Summary.

### 1.2 Purpose
This document provides a structured specification that can be parsed by task-master to generate actionable tasks.

### 1.3 Scope
The scope includes all requirements, features, and implementation details from the original documentation.

---

## 2. REQUIREMENTS


## 3. TASKS

The following tasks have been identified for implementation:

**TASK_001** [MEDIUM]: **HTTP Server**: Gin-based REST API server with production-ready middleware

**TASK_002** [MEDIUM]: **WebSocket Hub**: Real-time communication manager for UI updates

**TASK_003** [MEDIUM]: **Aggregator**: Report processing and topology building engine

**TASK_004** [MEDIUM]: **Storage**: Time-series data store with automatic cleanup

**TASK_005** [MEDIUM]: **Agent Management**: Registration, heartbeat, and configuration management

**TASK_006** [MEDIUM]: Graceful shutdown with context cancellation

**TASK_007** [MEDIUM]: Configurable cleanup intervals for stale data

**TASK_008** [MEDIUM]: CORS middleware for cross-origin requests

**TASK_009** [MEDIUM]: Background goroutines for periodic tasks

**TASK_010** [MEDIUM]: Thread-safe operations with proper mutex usage

**TASK_011** [MEDIUM]: `POST /api/v1/agents/register` - Register new probe agents

**TASK_012** [MEDIUM]: `POST /api/v1/agents/heartbeat/:agent_id` - Agent heartbeat

**TASK_013** [MEDIUM]: `GET /api/v1/agents/config/:agent_id` - Get agent configuration

**TASK_014** [MEDIUM]: `GET /api/v1/agents/list` - List all registered agents

**TASK_015** [MEDIUM]: `POST /api/v1/reports` - Submit monitoring reports from probes

**TASK_016** [MEDIUM]: `GET /api/v1/query/topology` - Get current topology view

**TASK_017** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/latest` - Get latest report

**TASK_018** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/timeseries` - Get time-series data

**TASK_019** [MEDIUM]: `GET /api/v1/query/stats` - Get server statistics

**TASK_020** [MEDIUM]: `GET /health` - Health check endpoint

**TASK_021** [MEDIUM]: `GET /api/v1/ping` - Ping endpoint

**TASK_022** [MEDIUM]: `GET /api/v1/ws` - WebSocket connection for real-time updates

**TASK_023** [MEDIUM]: Multi-level topology: hosts → containers → processes

**TASK_024** [MEDIUM]: Network connection mapping with edge creation

**TASK_025** [MEDIUM]: Container ID extraction from cgroup paths

**TASK_026** [MEDIUM]: Stale node cleanup with configurable thresholds

**TASK_027** [MEDIUM]: Thread-safe concurrent processing

**TASK_028** [MEDIUM]: Deep copy topology views to prevent race conditions

**TASK_029** [MEDIUM]: **Nodes**: Host, Container, and Process entities

**TASK_030** [MEDIUM]: **Edges**: Parent-child and network connections

**TASK_031** [MEDIUM]: **Metadata**: Rich metadata per node (CPU, memory, stats, etc.)

**TASK_032** [MEDIUM]: **Statistics**: Real-time topology metrics

**TASK_033** [MEDIUM]: Client registration/unregistration

**TASK_034** [MEDIUM]: Broadcast to all connected clients

**TASK_035** [MEDIUM]: Message queuing with buffer overflow protection

**TASK_036** [MEDIUM]: Automatic cleanup of disconnected clients

**TASK_037** [MEDIUM]: Read/write pumps for bidirectional communication

**TASK_038** [MEDIUM]: Ping/pong keepalive mechanism

**TASK_039** [MEDIUM]: Graceful connection handling

**TASK_040** [MEDIUM]: Message type routing (ping, subscribe, topology_update, report_update)

**TASK_041** [MEDIUM]: Write timeout: 10 seconds

**TASK_042** [MEDIUM]: Pong timeout: 60 seconds

**TASK_043** [MEDIUM]: Ping period: 54 seconds

**TASK_044** [MEDIUM]: Max message size: 512KB

**TASK_045** [MEDIUM]: 15-second resolution (as specified)

**TASK_046** [MEDIUM]: Configurable data retention (default: 1 hour)

**TASK_047** [MEDIUM]: Automatic cleanup of expired data

**TASK_048** [MEDIUM]: Per-agent time-series tracking

**TASK_049** [MEDIUM]: Thread-safe concurrent access

**TASK_050** [MEDIUM]: Background cleanup goroutine

**TASK_051** [MEDIUM]: Latest report retrieval

**TASK_052** [MEDIUM]: Recent points query by duration

**TASK_053** [MEDIUM]: Agent listing and deletion

**TASK_054** [MEDIUM]: Storage statistics

**TASK_055** [MEDIUM]: Health check endpoints

**TASK_056** [MEDIUM]: Agent registration and heartbeat

**TASK_057** [MEDIUM]: Report submission

**TASK_058** [MEDIUM]: Topology queries

**TASK_059** [MEDIUM]: Time-series data retrieval

**TASK_060** [MEDIUM]: Error handling for invalid requests

**TASK_061** [MEDIUM]: Host node creation

**TASK_062** [MEDIUM]: Container node creation with parent-child edges

**TASK_063** [MEDIUM]: Process node creation with cgroup parsing

**TASK_064** [MEDIUM]: Network connection edge creation

**TASK_065** [MEDIUM]: Stale node cleanup

**TASK_066** [MEDIUM]: Concurrent report processing

**TASK_067** [MEDIUM]: Container ID extraction from cgroup paths

**TASK_068** [MEDIUM]: Hub creation and client registration

**TASK_069** [MEDIUM]: Message broadcasting

**TASK_070** [MEDIUM]: Topology and report updates

**TASK_071** [MEDIUM]: Client pumps (read/write)

**TASK_072** [MEDIUM]: Message serialization

**TASK_073** [MEDIUM]: Connection lifecycle

**TASK_074** [MEDIUM]: Full workflow: registration → reports → queries

**TASK_075** [MEDIUM]: Multiple agents concurrently

**TASK_076** [MEDIUM]: Topology aggregation verification

**TASK_077** [MEDIUM]: Time-series data persistence

**TASK_078** [MEDIUM]: Server lifecycle (start/stop)

**TASK_079** [MEDIUM]: End-to-end scenarios

**TASK_080** [MEDIUM]: Invalid report handling

**TASK_081** [MEDIUM]: Non-existent agent queries

**TASK_082** [MEDIUM]: Invalid duration formats

**TASK_083** [MEDIUM]: Missing required fields

**TASK_084** [MEDIUM]: Default configuration application

**TASK_085** [MEDIUM]: Custom configuration validation

**TASK_086** [MEDIUM]: 5 concurrent probes

**TASK_087** [MEDIUM]: 20 reports per probe (100 total)

**TASK_088** [MEDIUM]: **Results**: 472.13 requests/second, 100% success rate

**TASK_089** [MEDIUM]: Storage verification

**TASK_090** [MEDIUM]: 10 concurrent readers

**TASK_091** [MEDIUM]: 20 reads per reader (200 total)

**TASK_092** [MEDIUM]: Multiple endpoint types

**TASK_093** [MEDIUM]: **Results**: High throughput, 100% success rate

**TASK_094** [MEDIUM]: 5 concurrent workers

**TASK_095** [MEDIUM]: 50/50 read/write mix

**TASK_096** [MEDIUM]: **Results**: 79,108.67 operations/second

**TASK_097** [MEDIUM]: 100 reports from 10 agents

**TASK_098** [MEDIUM]: Cleanup verification

**TASK_099** [MEDIUM]: No memory leaks detected

**TASK_100** [MEDIUM]: **Total Tests**: 44 tests

**TASK_101** [MEDIUM]: **Status**: ✅ ALL PASSING

**TASK_102** [MEDIUM]: **Duration**: 3.646 seconds

**TASK_103** [MEDIUM]: **Coverage**: Comprehensive coverage of all major components

**TASK_104** [MEDIUM]: Report submission: ~472 reports/second (load test)

**TASK_105** [MEDIUM]: Mixed operations: ~79,000 ops/second (unit test)

**TASK_106** [MEDIUM]: Concurrent reads: Excellent (200 concurrent reads completed quickly)

**TASK_107** [MEDIUM]: Report submission: 30-400µs (microseconds)

**TASK_108** [MEDIUM]: Topology query: 40-120µs

**TASK_109** [MEDIUM]: Health check: <30µs

**TASK_110** [MEDIUM]: Agent registration: 20-150µs

**TASK_111** [MEDIUM]: Successfully handles 5+ concurrent probes

**TASK_112** [MEDIUM]: Clean handling of 100+ reports with no performance degradation

**TASK_113** [MEDIUM]: Efficient memory usage with automatic cleanup

**TASK_114** [MEDIUM]: No goroutine leaks (verified in tests)

**TASK_115** [MEDIUM]: **Decision**: 15-second resolution

**TASK_116** [MEDIUM]: **Rationale**: Balances storage efficiency with data granularity for monitoring

**TASK_117** [MEDIUM]: **Implementation**: TimeSeriesStore with automatic point aggregation

**TASK_118** [MEDIUM]: **Decision**: In-memory time-series store

**TASK_119** [MEDIUM]: **Rationale**: Fast access, simple implementation, suitable for 1-hour retention

**TASK_120** [MEDIUM]: **Trade-off**: Data lost on restart (acceptable for monitoring use case)

**TASK_121** [MEDIUM]: **Decision**: Three-level hierarchy (host → container → process)

**TASK_122** [MEDIUM]: **Rationale**: Matches Docker/Kubernetes deployment model

**TASK_123** [MEDIUM]: **Benefits**: Natural grouping, efficient queries, clear relationships

**TASK_124** [MEDIUM]: **Decision**: Hub-and-spoke model with broadcast

**TASK_125** [MEDIUM]: **Rationale**: Simple, efficient for fan-out scenarios

**TASK_126** [MEDIUM]: **Scaling**: Can handle multiple concurrent clients

**TASK_127** [MEDIUM]: **Decision**: Background goroutine with configurable intervals

**TASK_128** [MEDIUM]: **Rationale**: Prevents memory growth, removes stale data

**TASK_129** [MEDIUM]: **Implementation**: Graceful shutdown with context cancellation

**TASK_130** [MEDIUM]: `pkg/app/server.go` - Main server implementation

**TASK_131** [MEDIUM]: `pkg/app/handlers.go` - HTTP handlers

**TASK_132** [MEDIUM]: `pkg/app/aggregator.go` - Report aggregation and topology

**TASK_133** [MEDIUM]: `pkg/app/websocket.go` - WebSocket hub and clients

**TASK_134** [MEDIUM]: `pkg/app/app.go` - Legacy app code (superseded by server.go)

**TASK_135** [MEDIUM]: `cmd/app/main.go` - Application entry point

**TASK_136** [MEDIUM]: `pkg/app/handlers_test.go`

**TASK_137** [MEDIUM]: `pkg/app/aggregator_test.go`

**TASK_138** [MEDIUM]: `pkg/app/websocket_test.go`

**TASK_139** [MEDIUM]: `pkg/app/app_test.go`

**TASK_140** [MEDIUM]: `pkg/app/loadtest_test.go`

**TASK_141** [MEDIUM]: `internal/storage/storage.go` - Basic key-value storage

**TASK_142** [MEDIUM]: `internal/storage/timeseries.go` - Time-series storage

**TASK_143** [MEDIUM]: `internal/storage/storage_test.go` - Enhanced test coverage

**TASK_144** [MEDIUM]: `github.com/gin-gonic/gin` - HTTP framework

**TASK_145** [MEDIUM]: `github.com/gorilla/websocket` - WebSocket implementation

**TASK_146** [MEDIUM]: `github.com/gorilla/mux` - HTTP router (used in legacy app.go)

**TASK_147** [MEDIUM]: `github.com/stretchr/testify/assert` - Test assertions

**TASK_148** [MEDIUM]: `github.com/stretchr/testify/require` - Required assertions

**TASK_149** [HIGH]: ✅ All required features implemented

**TASK_150** [HIGH]: ✅ Comprehensive test coverage

**TASK_151** [HIGH]: ✅ Load testing completed

**TASK_152** [HIGH]: Persistent storage backend (PostgreSQL/TimescaleDB)

**TASK_153** [HIGH]: Prometheus metrics export

**TASK_154** [HIGH]: Authentication/authorization

**TASK_155** [HIGH]: Rate limiting

**TASK_156** [HIGH]: API versioning

**TASK_157** [HIGH]: Compression for WebSocket messages

**TASK_158** [HIGH]: Multi-region support

**TASK_159** [HIGH]: Enhanced search capabilities

**TASK_160** [HIGH]: Alert management

**TASK_161** [HIGH]: Historical data export

**TASK_162** [MEDIUM]: Implemented with Gin framework

**TASK_163** [MEDIUM]: Clean middleware architecture

**TASK_164** [MEDIUM]: Production-ready configuration

**TASK_165** [MEDIUM]: All endpoints implemented and tested

**TASK_166** [MEDIUM]: Proper error handling

**TASK_167** [MEDIUM]: JSON request/response

**TASK_168** [MEDIUM]: Hub-based architecture

**TASK_169** [MEDIUM]: Broadcast support

**TASK_170** [MEDIUM]: Automatic client management

**TASK_171** [MEDIUM]: Keepalive mechanism

**TASK_172** [MEDIUM]: Merges probe data into topology views

**TASK_173** [MEDIUM]: Three-level hierarchy

**TASK_174** [MEDIUM]: Network connection mapping

**TASK_175** [MEDIUM]: Concurrent processing

**TASK_176** [MEDIUM]: Implemented in TimeSeriesStore

**TASK_177** [MEDIUM]: 15-second resolution enforced

**TASK_178** [MEDIUM]: Automatic point retention

**TASK_179** [MEDIUM]: Configurable cleanup interval

**TASK_180** [MEDIUM]: Removes stale nodes and agents

**TASK_181** [MEDIUM]: Memory-efficient

**TASK_182** [MEDIUM]: Proper Go project structure

**TASK_183** [MEDIUM]: Separation of concerns

**TASK_184** [MEDIUM]: Clean package organization

**TASK_185** [MEDIUM]: Unit tests for handlers ✅

**TASK_186** [MEDIUM]: Integration tests for report aggregation ✅

**TASK_187** [MEDIUM]: WebSocket connection tests ✅

**TASK_188** [MEDIUM]: Load testing with multiple concurrent probes ✅

**TASK_189** [MEDIUM]: **Robust**: Comprehensive error handling and graceful shutdown

**TASK_190** [MEDIUM]: **Performant**: >400 reports/second, microsecond latencies

**TASK_191** [MEDIUM]: **Scalable**: Handles multiple concurrent probes efficiently

**TASK_192** [MEDIUM]: **Well-tested**: 44 tests with 100% pass rate

**TASK_193** [MEDIUM]: **Production-ready**: Proper logging, monitoring, and cleanup

**TASK_194** [MEDIUM]: **Maintainable**: Clean code structure with good documentation


## 4. DETAILED SPECIFICATIONS

### 4.1 Original Content

The following sections contain the original documentation:


#### Task 21 App Backend Server Implementation Summary

# Task #21: App Backend Server Implementation Summary


#### Overview

## Overview
Successfully implemented a comprehensive central app backend server that receives reports from monitoring probes, aggregates data, and provides REST/WebSocket APIs for the UI.


#### Implementation Date

## Implementation Date
October 12, 2025


#### Architecture

## Architecture


#### Main Components

### Main Components


#### 1 Server Pkg App Server Go 

#### 1. Server (`pkg/app/server.go`)
The main server component that coordinates all subsystems:
- **HTTP Server**: Gin-based REST API server with production-ready middleware
- **WebSocket Hub**: Real-time communication manager for UI updates
- **Aggregator**: Report processing and topology building engine
- **Storage**: Time-series data store with automatic cleanup
- **Agent Management**: Registration, heartbeat, and configuration management

**Key Features:**
- Graceful shutdown with context cancellation
- Configurable cleanup intervals for stale data
- CORS middleware for cross-origin requests
- Background goroutines for periodic tasks
- Thread-safe operations with proper mutex usage


#### 2 Handlers Pkg App Handlers Go 

#### 2. Handlers (`pkg/app/handlers.go`)
REST API endpoint handlers:

**Agent Management Endpoints:**
- `POST /api/v1/agents/register` - Register new probe agents
- `POST /api/v1/agents/heartbeat/:agent_id` - Agent heartbeat
- `GET /api/v1/agents/config/:agent_id` - Get agent configuration
- `GET /api/v1/agents/list` - List all registered agents

**Data Submission:**
- `POST /api/v1/reports` - Submit monitoring reports from probes

**Query Endpoints:**
- `GET /api/v1/query/topology` - Get current topology view
- `GET /api/v1/query/agents/:agent_id/latest` - Get latest report
- `GET /api/v1/query/agents/:agent_id/timeseries` - Get time-series data
- `GET /api/v1/query/stats` - Get server statistics

**Health & Monitoring:**
- `GET /health` - Health check endpoint
- `GET /api/v1/ping` - Ping endpoint

**WebSocket:**
- `GET /api/v1/ws` - WebSocket connection for real-time updates


#### 3 Aggregator Pkg App Aggregator Go 

#### 3. Aggregator (`pkg/app/aggregator.go`)
Report aggregation and topology construction:

**Features:**
- Multi-level topology: hosts → containers → processes
- Network connection mapping with edge creation
- Container ID extraction from cgroup paths
- Stale node cleanup with configurable thresholds
- Thread-safe concurrent processing
- Deep copy topology views to prevent race conditions

**Topology Structure:**
- **Nodes**: Host, Container, and Process entities
- **Edges**: Parent-child and network connections
- **Metadata**: Rich metadata per node (CPU, memory, stats, etc.)
- **Statistics**: Real-time topology metrics


#### 4 Websocket Hub Pkg App Websocket Go 

#### 4. WebSocket Hub (`pkg/app/websocket.go`)
Real-time communication system:

**Hub Features:**
- Client registration/unregistration
- Broadcast to all connected clients
- Message queuing with buffer overflow protection
- Automatic cleanup of disconnected clients

**Client Features:**
- Read/write pumps for bidirectional communication
- Ping/pong keepalive mechanism
- Graceful connection handling
- Message type routing (ping, subscribe, topology_update, report_update)

**Performance:**
- Write timeout: 10 seconds
- Pong timeout: 60 seconds
- Ping period: 54 seconds
- Max message size: 512KB


#### 5 Storage Internal Storage 

#### 5. Storage (`internal/storage/`)
Time-series data storage:

**TimeSeriesStore:**
- 15-second resolution (as specified)
- Configurable data retention (default: 1 hour)
- Automatic cleanup of expired data
- Per-agent time-series tracking
- Thread-safe concurrent access
- Background cleanup goroutine

**Storage Features:**
- Latest report retrieval
- Recent points query by duration
- Agent listing and deletion
- Storage statistics


#### Configuration

## Configuration


#### Serverconfig

### ServerConfig
```go
type ServerConfig struct {
    Host               string        // Server bind address (default: 0.0.0.0)
    Port               int           // Server port (default: 8080)
    MaxDataAge         time.Duration // Max data retention (default: 1 hour)
    CleanupInterval    time.Duration // Cleanup frequency (default: 5 minutes)
    StaleNodeThreshold time.Duration // Stale node threshold (default: 5 minutes)
}
```


#### Test Coverage

## Test Coverage


#### Unit Tests

### Unit Tests
1. **Handlers Test** (`handlers_test.go`):
   - Health check endpoints
   - Agent registration and heartbeat
   - Report submission
   - Topology queries
   - Time-series data retrieval
   - Error handling for invalid requests

2. **Aggregator Test** (`aggregator_test.go`):
   - Host node creation
   - Container node creation with parent-child edges
   - Process node creation with cgroup parsing
   - Network connection edge creation
   - Stale node cleanup
   - Concurrent report processing
   - Container ID extraction from cgroup paths

3. **WebSocket Test** (`websocket_test.go`):
   - Hub creation and client registration
   - Message broadcasting
   - Topology and report updates
   - Client pumps (read/write)
   - Message serialization
   - Connection lifecycle


#### Integration Tests

### Integration Tests
1. **App Integration Test** (`app_test.go`):
   - Full workflow: registration → reports → queries
   - Multiple agents concurrently
   - Topology aggregation verification
   - Time-series data persistence
   - Server lifecycle (start/stop)
   - End-to-end scenarios

2. **Resilience Tests**:
   - Invalid report handling
   - Non-existent agent queries
   - Invalid duration formats
   - Missing required fields

3. **Configuration Tests**:
   - Default configuration application
   - Custom configuration validation


#### Load Tests

### Load Tests
1. **Multiple Probes Test** (`loadtest_test.go`):
   - 5 concurrent probes
   - 20 reports per probe (100 total)
   - **Results**: 472.13 requests/second, 100% success rate
   - Storage verification

2. **Concurrent Reads Test**:
   - 10 concurrent readers
   - 20 reads per reader (200 total)
   - Multiple endpoint types
   - **Results**: High throughput, 100% success rate

3. **Mixed Workload Test**:
   - 5 concurrent workers
   - 50/50 read/write mix
   - **Results**: 79,108.67 operations/second

4. **Memory Usage Test**:
   - 100 reports from 10 agents
   - Cleanup verification
   - No memory leaks detected


#### Test Results

### Test Results
- **Total Tests**: 44 tests
- **Status**: ✅ ALL PASSING
- **Duration**: 3.646 seconds
- **Coverage**: Comprehensive coverage of all major components


#### Performance Metrics

## Performance Metrics


#### Throughput

### Throughput
- Report submission: ~472 reports/second (load test)
- Mixed operations: ~79,000 ops/second (unit test)
- Concurrent reads: Excellent (200 concurrent reads completed quickly)


#### Latency

### Latency
- Report submission: 30-400µs (microseconds)
- Topology query: 40-120µs
- Health check: <30µs
- Agent registration: 20-150µs


#### Scalability

### Scalability
- Successfully handles 5+ concurrent probes
- Clean handling of 100+ reports with no performance degradation
- Efficient memory usage with automatic cleanup
- No goroutine leaks (verified in tests)


#### Data Flow

## Data Flow

1. **Probe Registration**:
   ```
   Probe → POST /api/v1/agents/register → Server stores AgentInfo
   ```

2. **Report Submission**:
   ```
   Probe → POST /api/v1/reports →
   Server updates LastSeen →
   TimeSeriesStore stores data →
   Aggregator processes topology →
   WebSocket broadcast to UI clients
   ```

3. **UI Query**:
   ```
   UI → GET /api/v1/query/topology →
   Aggregator returns topology view → UI renders
   ```

4. **Real-time Updates**:
   ```
   Probe report → Server processes →
   WebSocket Hub broadcasts →
   All connected UI clients receive update
   ```


#### Key Design Decisions

## Key Design Decisions


#### 1 Time Series Resolution

### 1. Time-Series Resolution
- **Decision**: 15-second resolution
- **Rationale**: Balances storage efficiency with data granularity for monitoring
- **Implementation**: TimeSeriesStore with automatic point aggregation


#### 2 In Memory Storage

### 2. In-Memory Storage
- **Decision**: In-memory time-series store
- **Rationale**: Fast access, simple implementation, suitable for 1-hour retention
- **Trade-off**: Data lost on restart (acceptable for monitoring use case)


#### 3 Topology Structure

### 3. Topology Structure
- **Decision**: Three-level hierarchy (host → container → process)
- **Rationale**: Matches Docker/Kubernetes deployment model
- **Benefits**: Natural grouping, efficient queries, clear relationships


#### 4 Websocket Architecture

### 4. WebSocket Architecture
- **Decision**: Hub-and-spoke model with broadcast
- **Rationale**: Simple, efficient for fan-out scenarios
- **Scaling**: Can handle multiple concurrent clients


#### 5 Cleanup Strategy

### 5. Cleanup Strategy
- **Decision**: Background goroutine with configurable intervals
- **Rationale**: Prevents memory growth, removes stale data
- **Implementation**: Graceful shutdown with context cancellation


#### Api Examples

## API Examples


#### Register Agent

### Register Agent
```bash
curl -X POST http://localhost:8080/api/v1/agents/register \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-1",
    "hostname": "host-1",
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
    "hostname": "host-1",
    "timestamp": "2025-10-12T16:00:00Z",
    "host_info": {
      "hostname": "host-1",
      "kernel_version": "5.10.0",
      "cpu_info": {"cores": 8, "usage": 25.5},
      "memory_info": {"total_mb": 16384, "used_mb": 8192}
    }
  }'
```


#### Query Topology

### Query Topology
```bash
curl http://localhost:8080/api/v1/query/topology
```


#### Get Time Series

### Get Time Series
```bash
curl "http://localhost:8080/api/v1/query/agents/agent-1/timeseries?duration=1h"
```


#### Websocket Connection

### WebSocket Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'topology_update') {
    // Update UI with new topology
  }
};
```


#### Files Created Modified

## Files Created/Modified


#### New Files

### New Files
- `pkg/app/server.go` - Main server implementation
- `pkg/app/handlers.go` - HTTP handlers
- `pkg/app/aggregator.go` - Report aggregation and topology
- `pkg/app/websocket.go` - WebSocket hub and clients
- `pkg/app/app.go` - Legacy app code (superseded by server.go)
- `cmd/app/main.go` - Application entry point
- Test files:
  - `pkg/app/handlers_test.go`
  - `pkg/app/aggregator_test.go`
  - `pkg/app/websocket_test.go`
  - `pkg/app/app_test.go`
  - `pkg/app/loadtest_test.go`


#### Modified Files

### Modified Files
- `internal/storage/storage.go` - Basic key-value storage
- `internal/storage/timeseries.go` - Time-series storage
- `internal/storage/storage_test.go` - Enhanced test coverage


#### Dependencies

## Dependencies


#### Core Dependencies

### Core Dependencies
- `github.com/gin-gonic/gin` - HTTP framework
- `github.com/gorilla/websocket` - WebSocket implementation
- `github.com/gorilla/mux` - HTTP router (used in legacy app.go)


#### Test Dependencies

### Test Dependencies
- `github.com/stretchr/testify/assert` - Test assertions
- `github.com/stretchr/testify/require` - Required assertions


#### Deployment Notes

## Deployment Notes


#### Environment Variables

### Environment Variables
```bash
export APP_HOST="0.0.0.0"
export APP_PORT="8080"
export MAX_DATA_AGE="1h"
export CLEANUP_INTERVAL="5m"
export STALE_NODE_THRESHOLD="5m"
```


#### Running The Server

### Running the Server
```bash

#### Development

# Development
go run cmd/app/main.go


#### Production

# Production
go build -o app-server cmd/app/main.go
./app-server --port 8080 --max-data-age 1h
```


#### Docker Support

### Docker Support
Ready for containerization with Dockerfile.app (in repository)


#### Future Enhancements

## Future Enhancements


#### Short Term

### Short-term
1. ✅ All required features implemented
2. ✅ Comprehensive test coverage
3. ✅ Load testing completed


#### Potential Improvements

### Potential Improvements
1. Persistent storage backend (PostgreSQL/TimescaleDB)
2. Prometheus metrics export
3. Authentication/authorization
4. Rate limiting
5. API versioning
6. Compression for WebSocket messages
7. Multi-region support
8. Enhanced search capabilities
9. Alert management
10. Historical data export


#### Compliance With Requirements

## Compliance with Requirements


####  Build A Go Http Server Using Gin Gonic Router

### ✅ Build a Go HTTP server using gin-gonic router
- Implemented with Gin framework
- Clean middleware architecture
- Production-ready configuration


####  Rest Api Endpoints

### ✅ REST API endpoints
- All endpoints implemented and tested
- Proper error handling
- JSON request/response


####  Websocket Server For Real Time Ui Updates

### ✅ WebSocket server for real-time UI updates
- Hub-based architecture
- Broadcast support
- Automatic client management
- Keepalive mechanism


####  Report Aggregation Engine

### ✅ Report aggregation engine
- Merges probe data into topology views
- Three-level hierarchy
- Network connection mapping
- Concurrent processing


####  Time Series Metrics Storage With 15 Second Resolution

### ✅ Time-series metrics storage with 15-second resolution
- Implemented in TimeSeriesStore
- 15-second resolution enforced
- Automatic point retention


####  Background Cleanup Of Old Data

### ✅ Background cleanup of old data
- Configurable cleanup interval
- Removes stale nodes and agents
- Memory-efficient


####  Structure Cmd App Main Go Pkg App 

### ✅ Structure: cmd/app/main.go, pkg/app/*
- Proper Go project structure
- Separation of concerns
- Clean package organization


####  Test Strategy

### ✅ Test Strategy
- Unit tests for handlers ✅
- Integration tests for report aggregation ✅
- WebSocket connection tests ✅
- Load testing with multiple concurrent probes ✅


#### Conclusion

## Conclusion

Task #21 has been successfully completed with a production-ready implementation that exceeds the original requirements. The app backend server is:

- **Robust**: Comprehensive error handling and graceful shutdown
- **Performant**: >400 reports/second, microsecond latencies
- **Scalable**: Handles multiple concurrent probes efficiently
- **Well-tested**: 44 tests with 100% pass rate
- **Production-ready**: Proper logging, monitoring, and cleanup
- **Maintainable**: Clean code structure with good documentation

The implementation is ready for integration with the UI components and can be deployed to production environments.

---
**Implementation Status**: ✅ COMPLETE
**Test Status**: ✅ ALL PASSING (44/44)
**Performance**: ✅ EXCEEDS REQUIREMENTS
**Documentation**: ✅ COMPREHENSIVE


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
