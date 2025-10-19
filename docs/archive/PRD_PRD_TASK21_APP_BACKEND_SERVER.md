# Product Requirements Document: ORCHESTRATOR: Prd Task21 App Backend Server

---

## Document Information
**Project:** orchestrator
**Document:** PRD_TASK21_APP_BACKEND_SERVER
**Version:** 1.0.0
**Date:** 2025-10-13
**Status:** READY FOR TASK-MASTER PARSING

---

## 1. EXECUTIVE SUMMARY

### 1.1 Overview
This PRD captures the requirements and implementation details for ORCHESTRATOR: Prd Task21 App Backend Server.

### 1.2 Purpose
This document provides a structured specification that can be parsed by task-master to generate actionable tasks.

### 1.3 Scope
The scope includes all requirements, features, and implementation details from the original documentation.

---

## 2. REQUIREMENTS

### 2.1 Functional Requirements
**Priority:** HIGH

**REQ-001:** met. The App Backend Server is production-ready with:


## 3. TASKS

The following tasks have been identified for implementation:

**TASK_001** [MEDIUM]: Go HTTP server with Gin framework

**TASK_002** [MEDIUM]: REST API for probe registration and report submission

**TASK_003** [MEDIUM]: WebSocket server for real-time UI updates

**TASK_004** [MEDIUM]: Report aggregation engine

**TASK_005** [MEDIUM]: Time-series metrics storage (15-second resolution)

**TASK_006** [MEDIUM]: Background cleanup of old data

**TASK_007** [HIGH]: [Product Vision](#product-vision)

**TASK_008** [HIGH]: [Task Breakdown](#task-breakdown)

**TASK_009** [HIGH]: [Technical Architecture](#technical-architecture)

**TASK_010** [HIGH]: [Implementation Details](#implementation-details)

**TASK_011** [HIGH]: [API Specifications](#api-specifications)

**TASK_012** [HIGH]: [Testing Strategy](#testing-strategy)

**TASK_013** [HIGH]: [Success Criteria](#success-criteria)

**TASK_014** [HIGH]: [Dependencies](#dependencies)

**TASK_015** [HIGH]: [Timeline](#timeline)

**TASK_016** [HIGH]: [Appendix](#appendix)

**TASK_017** [MEDIUM]: Receiving and processing reports from distributed probe agents

**TASK_018** [MEDIUM]: Aggregating data into meaningful topology views

**TASK_019** [MEDIUM]: Providing real-time updates to connected UI clients

**TASK_020** [MEDIUM]: Managing time-series metrics with efficient storage

**TASK_021** [MEDIUM]: Handling graceful shutdown and error recovery

**TASK_022** [MEDIUM]: Initialize Gin router with release mode for production

**TASK_023** [MEDIUM]: Configure CORS middleware for cross-origin requests

**TASK_024** [MEDIUM]: Set up graceful shutdown with signal handling

**TASK_025** [MEDIUM]: Add request logging and recovery middleware

**TASK_026** [MEDIUM]: Configure server timeouts (read, write, idle)

**TASK_027** [MEDIUM]: `pkg/app/server.go` - Server struct and configuration

**TASK_028** [MEDIUM]: `cmd/app/main.go` - Entry point with flag parsing

**TASK_029** [MEDIUM]: Unit tests for server lifecycle (start/stop)

**TASK_030** [MEDIUM]: Configuration validation tests

**TASK_031** [MEDIUM]: Graceful shutdown tests with active connections

**TASK_032** [MEDIUM]: `POST /api/v1/agents/register` - Register new probe agents

**TASK_033** [MEDIUM]: `POST /api/v1/agents/heartbeat/:agent_id` - Heartbeat tracking

**TASK_034** [MEDIUM]: `GET /api/v1/agents/config/:agent_id` - Agent configuration

**TASK_035** [MEDIUM]: `GET /api/v1/agents/list` - List all registered agents

**TASK_036** [MEDIUM]: `POST /api/v1/reports` - Submit probe reports

**TASK_037** [MEDIUM]: Validation of report data structure

**TASK_038** [MEDIUM]: Update agent last-seen timestamps

**TASK_039** [MEDIUM]: Forward to aggregator for processing

**TASK_040** [MEDIUM]: `GET /api/v1/query/topology` - Get current topology view

**TASK_041** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/latest` - Latest agent report

**TASK_042** [MEDIUM]: `GET /api/v1/query/agents/:agent_id/timeseries?duration=1h` - Time-series data

**TASK_043** [MEDIUM]: `GET /api/v1/query/stats` - Server statistics

**TASK_044** [MEDIUM]: `GET /health` - Health check endpoint

**TASK_045** [MEDIUM]: `GET /api/v1/ping` - Simple ping endpoint

**TASK_046** [MEDIUM]: `pkg/app/handlers.go` - Handler implementations

**TASK_047** [MEDIUM]: `pkg/app/handlers_test.go` - Handler unit tests

**TASK_048** [MEDIUM]: Unit tests for each endpoint with mock data

**TASK_049** [MEDIUM]: Request validation tests

**TASK_050** [MEDIUM]: Error handling tests (400, 404, 500)

**TASK_051** [MEDIUM]: Response format validation

**TASK_052** [MEDIUM]: `TopologyNode` - Represents hosts, containers, processes

**TASK_053** [MEDIUM]: `TopologyEdge` - Represents connections between nodes

**TASK_054** [MEDIUM]: `TopologyView` - Complete topology snapshot

**TASK_055** [MEDIUM]: Process reports from multiple probes

**TASK_056** [MEDIUM]: Build multi-level topology (hosts → containers → processes)

**TASK_057** [MEDIUM]: Track network connections between entities

**TASK_058** [MEDIUM]: Extract container IDs from cgroups

**TASK_059** [MEDIUM]: Merge overlapping data from different probes

**TASK_060** [MEDIUM]: Create/update nodes based on report data

**TASK_061** [MEDIUM]: Establish parent-child relationships

**TASK_062** [MEDIUM]: Track node metadata and timestamps

**TASK_063** [MEDIUM]: Clean stale nodes based on age threshold

**TASK_064** [MEDIUM]: RWMutex for concurrent read/write access

**TASK_065** [MEDIUM]: Deep copying for topology snapshots

**TASK_066** [MEDIUM]: Atomic updates to prevent race conditions

**TASK_067** [MEDIUM]: `pkg/app/aggregator.go` - Aggregation engine implementation

**TASK_068** [MEDIUM]: `pkg/app/aggregator_test.go` - Aggregation unit tests

**TASK_069** [MEDIUM]: Topology building tests with various report types

**TASK_070** [MEDIUM]: Concurrent access tests

**TASK_071** [MEDIUM]: Stale node cleanup tests

**TASK_072** [MEDIUM]: Container ID extraction tests

**TASK_073** [MEDIUM]: Performance tests with large datasets

**TASK_074** [MEDIUM]: Client registration/unregistration

**TASK_075** [MEDIUM]: Broadcast channel for topology updates

**TASK_076** [MEDIUM]: Per-client send buffers (256 bytes)

**TASK_077** [MEDIUM]: Connection cleanup on disconnect

**TASK_078** [MEDIUM]: Client struct with connection and send channel

**TASK_079** [MEDIUM]: Read pump for incoming messages

**TASK_080** [MEDIUM]: Write pump for outgoing messages

**TASK_081** [MEDIUM]: Ping/pong for connection keep-alive

**TASK_082** [MEDIUM]: `topology_update` - Full topology broadcasts

**TASK_083** [MEDIUM]: `report_update` - Individual agent reports

**TASK_084** [MEDIUM]: `ping/pong` - Keep-alive messages

**TASK_085** [MEDIUM]: Upgrade HTTP to WebSocket

**TASK_086** [MEDIUM]: Send initial topology on connect

**TASK_087** [MEDIUM]: Handle graceful disconnection

**TASK_088** [MEDIUM]: Automatic reconnection support

**TASK_089** [MEDIUM]: `pkg/app/websocket.go` - WebSocket hub and client

**TASK_090** [MEDIUM]: `pkg/app/websocket_test.go` - WebSocket tests

**TASK_091** [MEDIUM]: Client connection/disconnection tests

**TASK_092** [MEDIUM]: Message broadcasting tests

**TASK_093** [MEDIUM]: Concurrent client tests

**TASK_094** [MEDIUM]: Connection timeout tests

**TASK_095** [MEDIUM]: Memory leak tests for long-running connections

**TASK_096** [MEDIUM]: `TimeSeriesData` - Per-agent data with ring buffer

**TASK_097** [MEDIUM]: `TimeSeriesPoint` - Single metric point with timestamp

**TASK_098** [MEDIUM]: `TimeSeriesStore` - Global store managing all agents

**TASK_099** [MEDIUM]: 15-second resolution as specified

**TASK_100** [MEDIUM]: Configurable retention period (default 1 hour)

**TASK_101** [MEDIUM]: Automatic old data cleanup

**TASK_102** [MEDIUM]: Per-agent time-series tracking

**TASK_103** [MEDIUM]: Background cleanup goroutine

**TASK_104** [MEDIUM]: `AddReport(report)` - Store new data point

**TASK_105** [MEDIUM]: `GetRecentPoints(agentID, duration)` - Query time range

**TASK_106** [MEDIUM]: `GetLatestReport(agentID)` - Get most recent data

**TASK_107** [MEDIUM]: `GetAllAgents()` - List tracked agents

**TASK_108** [MEDIUM]: `DeleteAgent(agentID)` - Remove agent data

**TASK_109** [MEDIUM]: Pre-allocated buffers for efficiency

**TASK_110** [MEDIUM]: Lock-free reads where possible

**TASK_111** [MEDIUM]: Batched cleanup operations

**TASK_112** [MEDIUM]: Memory-efficient storage format

**TASK_113** [MEDIUM]: `internal/storage/timeseries.go` - Time-series implementation

**TASK_114** [MEDIUM]: `internal/storage/storage_test.go` - Storage tests

**TASK_115** [MEDIUM]: Data retention tests

**TASK_116** [MEDIUM]: Query performance benchmarks

**TASK_117** [MEDIUM]: Concurrent access tests

**TASK_118** [MEDIUM]: Memory usage profiling

**TASK_119** [MEDIUM]: Cleanup validation tests

**TASK_120** [MEDIUM]: Remove stale agents (no heartbeat for threshold period)

**TASK_121** [MEDIUM]: Clean old topology nodes

**TASK_122** [MEDIUM]: Purge expired time-series data

**TASK_123** [MEDIUM]: Archive data for rollback window

**TASK_124** [MEDIUM]: Configurable cleanup interval (default 5 minutes)

**TASK_125** [MEDIUM]: Stale node threshold (default 5 minutes)

**TASK_126** [MEDIUM]: Data retention policies

**TASK_127** [MEDIUM]: Graceful cleanup during shutdown

**TASK_128** [MEDIUM]: Log cleanup statistics

**TASK_129** [MEDIUM]: Track cleanup duration

**TASK_130** [MEDIUM]: Alert on excessive stale agents

**TASK_131** [MEDIUM]: Metrics export for observability

**TASK_132** [MEDIUM]: Cleanup logic in `pkg/app/server.go`

**TASK_133** [MEDIUM]: Background goroutine management

**TASK_134** [MEDIUM]: Cleanup metrics and logging

**TASK_135** [MEDIUM]: Stale data detection tests

**TASK_136** [MEDIUM]: Cleanup timing tests

**TASK_137** [MEDIUM]: Graceful shutdown during cleanup

**TASK_138** [MEDIUM]: Resource leak prevention tests

**TASK_139** [MEDIUM]: `--host` - Server host address (default: 0.0.0.0)

**TASK_140** [MEDIUM]: `--port` - Server port (default: 8080)

**TASK_141** [MEDIUM]: `--max-data-age` - Data retention period (default: 1h)

**TASK_142** [MEDIUM]: `--cleanup-interval` - Cleanup frequency (default: 5m)

**TASK_143** [MEDIUM]: `--stale-node-threshold` - Stale detection time (default: 5m)

**TASK_144** [MEDIUM]: Command-line flags (highest priority)

**TASK_145** [MEDIUM]: Environment variables

**TASK_146** [MEDIUM]: Configuration file (YAML/JSON)

**TASK_147** [MEDIUM]: Sensible defaults

**TASK_148** [MEDIUM]: Range checks for numeric values

**TASK_149** [MEDIUM]: Duration format validation

**TASK_150** [MEDIUM]: Port availability checks

**TASK_151** [MEDIUM]: Path existence verification

**TASK_152** [MEDIUM]: Flag parsing in `cmd/app/main.go`

**TASK_153** [MEDIUM]: Configuration struct in `pkg/app/server.go`

**TASK_154** [MEDIUM]: Environment variable support

**TASK_155** [MEDIUM]: Configuration parsing tests

**TASK_156** [MEDIUM]: Default value tests

**TASK_157** [MEDIUM]: Validation error tests

**TASK_158** [MEDIUM]: Priority/override tests

**TASK_159** [MEDIUM]: Agent registration → Report submission → Topology query

**TASK_160** [MEDIUM]: Multiple concurrent agents

**TASK_161** [MEDIUM]: WebSocket connections with real-time updates

**TASK_162** [MEDIUM]: Time-series data accumulation

**TASK_163** [MEDIUM]: End-to-end API flow validation

**TASK_164** [MEDIUM]: All API endpoints

**TASK_165** [MEDIUM]: WebSocket functionality

**TASK_166** [MEDIUM]: Aggregation correctness

**TASK_167** [MEDIUM]: Time-series accuracy

**TASK_168** [MEDIUM]: Error handling paths

**TASK_169** [MEDIUM]: Multiple concurrent probes (5-10)

**TASK_170** [MEDIUM]: High report submission rates (50+ req/sec)

**TASK_171** [MEDIUM]: Many WebSocket clients (100+)

**TASK_172** [MEDIUM]: Large topology datasets (1000+ nodes)

**TASK_173** [MEDIUM]: `pkg/app/app_test.go` - Integration tests

**TASK_174** [MEDIUM]: `pkg/app/loadtest_test.go` - Load tests

**TASK_175** [MEDIUM]: Test utilities and helpers

**TASK_176** [MEDIUM]: Integration tests with real server instances

**TASK_177** [MEDIUM]: Load tests with concurrent goroutines

**TASK_178** [MEDIUM]: Performance benchmarks

**TASK_179** [MEDIUM]: Memory and CPU profiling

**TASK_180** [MEDIUM]: **Language**: Go 1.21+

**TASK_181** [MEDIUM]: **Web Framework**: Gin (high-performance HTTP router)

**TASK_182** [MEDIUM]: **WebSocket**: Gorilla WebSocket

**TASK_183** [MEDIUM]: **Storage**: In-memory with planned persistence options

**TASK_184** [MEDIUM]: **Testing**: Go testing package + testify

**TASK_185** [MEDIUM]: **Logging**: Structured logging with Gin middleware

**TASK_186** [MEDIUM]: **Concurrency**: Goroutines + sync primitives (RWMutex, channels)

**TASK_187** [MEDIUM]: **Handlers**: Test each endpoint with mock data

**TASK_188** [MEDIUM]: **Aggregator**: Test topology building logic

**TASK_189** [MEDIUM]: **WebSocket**: Test client lifecycle and messaging

**TASK_190** [MEDIUM]: **Storage**: Test data retention and queries

**TASK_191** [MEDIUM]: **End-to-End Workflows**: Agent registration → Reports → Queries

**TASK_192** [MEDIUM]: **Concurrent Agents**: Multiple agents submitting reports

**TASK_193** [MEDIUM]: **WebSocket Communication**: Real-time updates to clients

**TASK_194** [MEDIUM]: **5 Concurrent Probes**: Each sending 10 reports

**TASK_195** [MEDIUM]: **100 WebSocket Clients**: Receiving broadcasts

**TASK_196** [MEDIUM]: **1000+ Node Topology**: Performance validation

**TASK_197** [MEDIUM]: **Report Processing**: Target < 100ms per report

**TASK_198** [MEDIUM]: **Topology Query**: Target < 50ms

**TASK_199** [MEDIUM]: **WebSocket Broadcast**: Target < 10ms per client

**TASK_200** [MEDIUM]: All API endpoints implemented and tested

**TASK_201** [MEDIUM]: WebSocket server operational with real-time updates

**TASK_202** [MEDIUM]: Report aggregation producing correct topology

**TASK_203** [MEDIUM]: Time-series storage with 15-second resolution

**TASK_204** [MEDIUM]: Background cleanup removing stale data

**TASK_205** [MEDIUM]: Handle 50+ reports/second

**TASK_206** [MEDIUM]: Support 100+ concurrent WebSocket clients

**TASK_207** [MEDIUM]: Query response time < 100ms (P95)

**TASK_208** [MEDIUM]: Memory usage < 500MB for 1000 nodes

**TASK_209** [MEDIUM]: Graceful shutdown with zero data loss

**TASK_210** [MEDIUM]: Error handling for all failure modes

**TASK_211** [MEDIUM]: Recovery from probe disconnections

**TASK_212** [MEDIUM]: Automatic cleanup of stale resources

**TASK_213** [MEDIUM]: 50+ unit tests all passing

**TASK_214** [MEDIUM]: Integration tests covering main workflows

**TASK_215** [MEDIUM]: Load tests validating concurrency

**TASK_216** [MEDIUM]: >80% code coverage

**TASK_217** [MEDIUM]: **Go 1.21+**: Programming language

**TASK_218** [MEDIUM]: **Gin Framework**: HTTP routing

**TASK_219** [MEDIUM]: **Gorilla WebSocket**: WebSocket support

**TASK_220** [MEDIUM]: **Testify**: Testing assertions

**TASK_221** [MEDIUM]: **pkg/probe**: Probe data structures (ReportData, HostInfo, etc.)

**TASK_222** [MEDIUM]: **internal/storage**: Storage interfaces

**TASK_223** [MEDIUM]: None (standalone server)

**TASK_224** [MEDIUM]: `app_agents_total` - Number of registered agents

**TASK_225** [MEDIUM]: `app_reports_received_total` - Total reports received

**TASK_226** [MEDIUM]: `app_websocket_clients` - Active WebSocket connections

**TASK_227** [MEDIUM]: `app_topology_nodes` - Total nodes in topology

**TASK_228** [MEDIUM]: `app_http_request_duration_seconds` - Request latency histogram

**TASK_229** [HIGH]: **Port already in use**: Change with `--port` flag

**TASK_230** [HIGH]: **High memory usage**: Reduce `--max-data-age`

**TASK_231** [HIGH]: **WebSocket disconnections**: Check firewall/proxy settings

**TASK_232** [HIGH]: **Stale agents**: Verify probe heartbeats are reaching server

**TASK_233** [MEDIUM]: ✅ Complete REST API implementation

**TASK_234** [MEDIUM]: ✅ Real-time WebSocket support

**TASK_235** [MEDIUM]: ✅ Efficient report aggregation

**TASK_236** [MEDIUM]: ✅ Time-series storage with automatic cleanup

**TASK_237** [MEDIUM]: ✅ Comprehensive test coverage (100% passing)

**TASK_238** [MEDIUM]: ✅ Production-ready error handling and logging

**TASK_239** [HIGH]: Deploy to staging environment

**TASK_240** [HIGH]: Integrate with probe agents (Task #22)

**TASK_241** [HIGH]: Build UI to consume APIs (Task #23)

**TASK_242** [HIGH]: Performance tuning based on production loads

**TASK_243** [HIGH]: Add persistence layer for data durability


## 4. DETAILED SPECIFICATIONS

### 4.1 Original Content

The following sections contain the original documentation:


#### Product Requirements Document Prd 

# Product Requirements Document (PRD)

#### Task 21 App Backend Server With Report Aggregation

## Task #21: App Backend Server with Report Aggregation

**Document Version**: 1.0.0
**Last Updated**: 2025-10-13
**Status**: ✅ COMPLETED
**Priority**: HIGH

---


#### Executive Summary

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


#### Table Of Contents

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


#### Product Vision

## Product Vision

The App Backend Server is the nerve center of the monitoring platform, responsible for:
- Receiving and processing reports from distributed probe agents
- Aggregating data into meaningful topology views
- Providing real-time updates to connected UI clients
- Managing time-series metrics with efficient storage
- Handling graceful shutdown and error recovery

This component enables operators to visualize and understand their infrastructure in real-time, with sub-second latency for critical updates.

---


#### Task Breakdown

## Task Breakdown


#### Task 21 1 Http Server Setup

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


#### Task 21 2 Rest Api Handlers

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


#### Task 21 3 Report Aggregation Engine

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


#### Task 21 4 Websocket Server

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


#### Task 21 5 Time Series Storage

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


#### Task 21 6 Background Cleanup

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


#### Task 21 7 Configuration Management

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


#### Task 21 8 Integration Tests

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


#### Technical Architecture

## Technical Architecture


#### System Components

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


#### Data Flow

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


#### Technology Stack

### Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin (high-performance HTTP router)
- **WebSocket**: Gorilla WebSocket
- **Storage**: In-memory with planned persistence options
- **Testing**: Go testing package + testify
- **Logging**: Structured logging with Gin middleware
- **Concurrency**: Goroutines + sync primitives (RWMutex, channels)

---


#### Api Specifications

## API Specifications


#### Agent Management

### Agent Management


#### Register Agent

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

#### List Agents
```http
GET /api/v1/agents/list

Response: 200 OK
{
  "agents": [...],
  "count": 5
}
```


#### Report Submission

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


#### Data Queries

### Data Queries


#### Get Topology

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


#### Get Time Series Data

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


#### Websocket

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


#### Testing Strategy

## Testing Strategy


#### Unit Tests

### Unit Tests
- **Handlers**: Test each endpoint with mock data
- **Aggregator**: Test topology building logic
- **WebSocket**: Test client lifecycle and messaging
- **Storage**: Test data retention and queries

**Coverage Target**: >80%


#### Integration Tests

### Integration Tests
- **End-to-End Workflows**: Agent registration → Reports → Queries
- **Concurrent Agents**: Multiple agents submitting reports
- **WebSocket Communication**: Real-time updates to clients


#### Load Tests

### Load Tests
- **5 Concurrent Probes**: Each sending 10 reports
- **100 WebSocket Clients**: Receiving broadcasts
- **1000+ Node Topology**: Performance validation


#### Performance Benchmarks

### Performance Benchmarks
- **Report Processing**: Target < 100ms per report
- **Topology Query**: Target < 50ms
- **WebSocket Broadcast**: Target < 10ms per client

---


#### Success Criteria

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


#### Dependencies

## Dependencies


#### External Dependencies

### External Dependencies
- **Go 1.21+**: Programming language
- **Gin Framework**: HTTP routing
- **Gorilla WebSocket**: WebSocket support
- **Testify**: Testing assertions


#### Internal Dependencies

### Internal Dependencies
- **pkg/probe**: Probe data structures (ReportData, HostInfo, etc.)
- **internal/storage**: Storage interfaces


#### System Dependencies

### System Dependencies
- None (standalone server)

---


#### Timeline

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


#### Appendix

## Appendix


#### A File Structure

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


#### B Configuration Example

### B. Configuration Example

```yaml

#### Config Yaml

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


#### C Deployment

### C. Deployment


#### Build

#### Build
```bash
./go/bin/go build -o app-server ./cmd/app/
```


#### Run

#### Run
```bash
./app-server \
  --host 0.0.0.0 \
  --port 8080 \
  --max-data-age 2h \
  --cleanup-interval 10m
```


#### Docker

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


#### D Monitoring

### D. Monitoring

**Metrics Exported**:
- `app_agents_total` - Number of registered agents
- `app_reports_received_total` - Total reports received
- `app_websocket_clients` - Active WebSocket connections
- `app_topology_nodes` - Total nodes in topology
- `app_http_request_duration_seconds` - Request latency histogram


#### E Troubleshooting

### E. Troubleshooting

**Common Issues**:

1. **Port already in use**: Change with `--port` flag
2. **High memory usage**: Reduce `--max-data-age`
3. **WebSocket disconnections**: Check firewall/proxy settings
4. **Stale agents**: Verify probe heartbeats are reaching server

---


#### Conclusion

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
