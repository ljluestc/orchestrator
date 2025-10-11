package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// WebUI represents the web user interface
type WebUI struct {
	ID          string
	Port        int
	TopologyURL string
	server      *http.Server
}

// NewWebUI creates a new web UI
func NewWebUI(id string, port int, topologyURL string) *WebUI {
	return &WebUI{
		ID:          id,
		Port:        port,
		TopologyURL: topologyURL,
	}
}

// Start starts the web UI
func (w *WebUI) Start() error {
	router := w.setupRoutes()

	w.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", w.Port),
		Handler: router,
	}

	log.Printf("Starting Web UI on :%d", w.Port)
	return w.server.ListenAndServe()
}

// Stop stops the web UI
func (w *WebUI) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return w.server.Shutdown(ctx)
}

// setupRoutes sets up HTTP routes
func (w *WebUI) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Main dashboard
	router.HandleFunc("/", w.handleDashboard).Methods("GET")
	router.HandleFunc("/dashboard", w.handleDashboard).Methods("GET")

	// Topology views
	router.HandleFunc("/topology", w.handleTopology).Methods("GET")
	router.HandleFunc("/topology/{view}", w.handleTopologyView).Methods("GET")

	// API proxy
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/topology", w.handleAPIProxy).Methods("GET")
	api.HandleFunc("/topology/nodes", w.handleAPIProxy).Methods("GET")
	api.HandleFunc("/topology/edges", w.handleAPIProxy).Methods("GET")
	api.HandleFunc("/topology/search", w.handleAPIProxy).Methods("GET")
	api.HandleFunc("/topology/filter", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/views", w.handleAPIProxy).Methods("GET")
	api.HandleFunc("/metrics", w.handleAPIProxy).Methods("GET")

	// Container control
	api.HandleFunc("/containers/{id}/start", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/containers/{id}/stop", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/containers/{id}/restart", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/containers/{id}/pause", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/containers/{id}/unpause", w.handleAPIProxy).Methods("POST")
	api.HandleFunc("/containers/{id}/logs", w.handleAPIProxy).Methods("GET")

	// WebSocket proxy
	router.HandleFunc("/ws", w.handleWebSocketProxy)

	// Static assets
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Health check
	router.HandleFunc("/health", w.handleHealth).Methods("GET")

	return router
}

// handleDashboard handles the main dashboard
func (w *WebUI) handleDashboard(wr http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mesos-Docker Orchestration Platform</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/cytoscape@3.26.0/dist/cytoscape.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/cytoscape-dagre@2.5.0/cytoscape-dagre.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/cytoscape-cose-bilkent@4.1.0/cytoscape-cose-bilkent.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 1rem 2rem;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        
        .header h1 {
            font-size: 1.8rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
        }
        
        .header p {
            opacity: 0.9;
            font-size: 1rem;
        }
        
        .nav {
            background: white;
            padding: 0 2rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            display: flex;
            align-items: center;
            gap: 2rem;
        }
        
        .nav-item {
            padding: 1rem 0;
            text-decoration: none;
            color: #666;
            font-weight: 500;
            border-bottom: 3px solid transparent;
            transition: all 0.3s ease;
        }
        
        .nav-item:hover, .nav-item.active {
            color: #667eea;
            border-bottom-color: #667eea;
        }
        
        .main {
            display: flex;
            height: calc(100vh - 140px);
        }
        
        .sidebar {
            width: 300px;
            background: white;
            border-right: 1px solid #e0e0e0;
            padding: 1.5rem;
            overflow-y: auto;
        }
        
        .content {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .toolbar {
            background: white;
            padding: 1rem 2rem;
            border-bottom: 1px solid #e0e0e0;
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        
        .search-box {
            flex: 1;
            max-width: 400px;
        }
        
        .search-box input {
            width: 100%;
            padding: 0.75rem 1rem;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 0.9rem;
        }
        
        .view-selector {
            display: flex;
            gap: 0.5rem;
        }
        
        .view-btn {
            padding: 0.5rem 1rem;
            border: 1px solid #ddd;
            background: white;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.9rem;
            transition: all 0.2s ease;
        }
        
        .view-btn:hover, .view-btn.active {
            background: #667eea;
            color: white;
            border-color: #667eea;
        }
        
        .graph-container {
            flex: 1;
            background: white;
            margin: 1rem;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            position: relative;
            overflow: hidden;
        }
        
        #cytoscape {
            width: 100%;
            height: 100%;
        }
        
        .context-panel {
            position: absolute;
            top: 1rem;
            right: 1rem;
            width: 300px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.15);
            padding: 1.5rem;
            display: none;
        }
        
        .context-panel h3 {
            margin-bottom: 1rem;
            color: #333;
            font-size: 1.1rem;
        }
        
        .metric-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.5rem 0;
            border-bottom: 1px solid #f0f0f0;
        }
        
        .metric-item:last-child {
            border-bottom: none;
        }
        
        .metric-label {
            font-size: 0.9rem;
            color: #666;
        }
        
        .metric-value {
            font-weight: 600;
            color: #333;
        }
        
        .sparkline {
            width: 100px;
            height: 20px;
            margin-left: 1rem;
        }
        
        .status-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 0.5rem;
        }
        
        .status-healthy { background: #4CAF50; }
        .status-warning { background: #FF9800; }
        .status-critical { background: #F44336; }
        .status-unknown { background: #9E9E9E; }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        
        .stat-card {
            background: white;
            padding: 1.5rem;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            text-align: center;
        }
        
        .stat-value {
            font-size: 2rem;
            font-weight: 700;
            color: #667eea;
            margin-bottom: 0.5rem;
        }
        
        .stat-label {
            color: #666;
            font-size: 0.9rem;
        }
        
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 200px;
            color: #666;
        }
        
        .error {
            color: #F44336;
            background: #ffebee;
            padding: 1rem;
            border-radius: 6px;
            margin: 1rem;
        }
        
        @media (max-width: 768px) {
            .main { flex-direction: column; }
            .sidebar { width: 100%; height: auto; }
            .context-panel { position: relative; width: 100%; }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Mesos-Docker Orchestration Platform</h1>
        <p>Real-time topology visualization and container management</p>
    </div>
    
    <div class="nav">
        <a href="/" class="nav-item active">Dashboard</a>
        <a href="/topology" class="nav-item">Topology</a>
        <a href="/containers" class="nav-item">Containers</a>
        <a href="/hosts" class="nav-item">Hosts</a>
        <a href="/metrics" class="nav-item">Metrics</a>
    </div>
    
    <div class="main">
        <div class="sidebar">
            <div class="stats-grid" id="statsGrid">
                <div class="stat-card">
                    <div class="stat-value" id="totalNodes">-</div>
                    <div class="stat-label">Total Nodes</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="totalEdges">-</div>
                    <div class="stat-label">Connections</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="healthyNodes">-</div>
                    <div class="stat-label">Healthy</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" id="criticalNodes">-</div>
                    <div class="stat-label">Critical</div>
                </div>
            </div>
            
            <h3>Views</h3>
            <div class="view-selector">
                <button class="view-btn active" data-view="processes">Processes</button>
                <button class="view-btn" data-view="containers">Containers</button>
                <button class="view-btn" data-view="hosts">Hosts</button>
                <button class="view-btn" data-view="pods">Pods</button>
                <button class="view-btn" data-view="services">Services</button>
            </div>
            
            <h3 style="margin-top: 2rem;">Filters</h3>
            <div class="search-box">
                <input type="text" id="searchInput" placeholder="Search nodes...">
            </div>
            
            <div style="margin-top: 1rem;">
                <label>
                    <input type="checkbox" id="filterHealthy" checked> Healthy
                </label>
                <label style="margin-left: 1rem;">
                    <input type="checkbox" id="filterWarning"> Warning
                </label>
                <label style="margin-left: 1rem;">
                    <input type="checkbox" id="filterCritical"> Critical
                </label>
            </div>
        </div>
        
        <div class="content">
            <div class="toolbar">
                <div class="search-box">
                    <input type="text" id="mainSearchInput" placeholder="Search across all nodes...">
                </div>
                <button id="refreshBtn" class="view-btn">Refresh</button>
                <button id="layoutBtn" class="view-btn">Auto Layout</button>
            </div>
            
            <div class="graph-container">
                <div id="cytoscape"></div>
                <div class="context-panel" id="contextPanel">
                    <h3 id="contextTitle">Node Details</h3>
                    <div id="contextContent">
                        <div class="loading">Select a node to view details</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        class TopologyVisualizer {
            constructor() {
                this.cy = null;
                this.currentView = 'processes';
                this.ws = null;
                this.nodes = [];
                this.edges = [];
                this.metrics = {};
                this.selectedNode = null;
                
                this.init();
            }
            
            init() {
                this.setupEventListeners();
                this.initCytoscape();
                this.connectWebSocket();
                this.loadInitialData();
            }
            
            setupEventListeners() {
                // View selector
                document.querySelectorAll('.view-btn[data-view]').forEach(btn => {
                    btn.addEventListener('click', (e) => {
                        document.querySelectorAll('.view-btn').forEach(b => b.classList.remove('active'));
                        e.target.classList.add('active');
                        this.currentView = e.target.dataset.view;
                        this.loadView(this.currentView);
                    });
                });
                
                // Search
                document.getElementById('mainSearchInput').addEventListener('input', (e) => {
                    this.search(e.target.value);
                });
                
                // Refresh
                document.getElementById('refreshBtn').addEventListener('click', () => {
                    this.loadInitialData();
                });
                
                // Layout
                document.getElementById('layoutBtn').addEventListener('click', () => {
                    this.autoLayout();
                });
            }
            
            initCytoscape() {
                this.cy = cytoscape({
                    container: document.getElementById('cytoscape'),
                    elements: [],
                    style: [
                        {
                            selector: 'node',
                            style: {
                                'background-color': '#667eea',
                                'label': 'data(name)',
                                'text-valign': 'center',
                                'text-halign': 'center',
                                'color': 'white',
                                'font-size': '12px',
                                'font-weight': 'bold',
                                'width': '60px',
                                'height': '60px',
                                'border-width': 2,
                                'border-color': '#fff',
                                'border-opacity': 1
                            }
                        },
                        {
                            selector: 'node[type="host"]',
                            style: {
                                'background-color': '#4CAF50',
                                'width': '80px',
                                'height': '80px'
                            }
                        },
                        {
                            selector: 'node[type="container"]',
                            style: {
                                'background-color': '#2196F3',
                                'width': '50px',
                                'height': '50px'
                            }
                        },
                        {
                            selector: 'node[type="process"]',
                            style: {
                                'background-color': '#FF9800',
                                'width': '40px',
                                'height': '40px'
                            }
                        },
                        {
                            selector: 'node[type="pod"]',
                            style: {
                                'background-color': '#9C27B0',
                                'width': '60px',
                                'height': '60px'
                            }
                        },
                        {
                            selector: 'node[type="service"]',
                            style: {
                                'background-color': '#E91E63',
                                'width': '70px',
                                'height': '70px'
                            }
                        },
                        {
                            selector: 'node[status="healthy"]',
                            style: {
                                'border-color': '#4CAF50'
                            }
                        },
                        {
                            selector: 'node[status="warning"]',
                            style: {
                                'border-color': '#FF9800'
                            }
                        },
                        {
                            selector: 'node[status="critical"]',
                            style: {
                                'border-color': '#F44336'
                            }
                        },
                        {
                            selector: 'edge',
                            style: {
                                'width': 2,
                                'line-color': '#ccc',
                                'target-arrow-color': '#ccc',
                                'target-arrow-shape': 'triangle',
                                'curve-style': 'bezier'
                            }
                        },
                        {
                            selector: 'edge[type="network"]',
                            style: {
                                'line-color': '#2196F3',
                                'target-arrow-color': '#2196F3'
                            }
                        },
                        {
                            selector: 'edge[type="process"]',
                            style: {
                                'line-color': '#FF9800',
                                'target-arrow-color': '#FF9800'
                            }
                        }
                    ],
                    layout: {
                        name: 'cose',
                        idealEdgeLength: 100,
                        nodeOverlap: 20,
                        refresh: 20,
                        fit: true,
                        padding: 30,
                        randomize: false,
                        componentSpacing: 100,
                        nodeRepulsion: 400000,
                        edgeElasticity: 100,
                        nestingFactor: 5,
                        gravity: 80,
                        numIter: 1000,
                        initialTemp: 200,
                        coolingFactor: 0.95,
                        minTemp: 1.0
                    }
                });
                
                // Node click handler
                this.cy.on('tap', 'node', (evt) => {
                    const node = evt.target;
                    this.selectNode(node);
                });
                
                // Background click handler
                this.cy.on('tap', (evt) => {
                    if (evt.target === this.cy) {
                        this.deselectNode();
                    }
                });
            }
            
            connectWebSocket() {
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const wsUrl = protocol + '//' + window.location.host + '/ws';
                
                this.ws = new WebSocket(wsUrl);
                
                this.ws.onopen = () => {
                    console.log('WebSocket connected');
                };
                
                this.ws.onmessage = (event) => {
                    try {
                        const data = JSON.parse(event.data);
                        this.handleWebSocketMessage(data);
                    } catch (e) {
                        console.error('Error parsing WebSocket message:', e);
                    }
                };
                
                this.ws.onclose = () => {
                    console.log('WebSocket disconnected, reconnecting...');
                    setTimeout(() => this.connectWebSocket(), 5000);
                };
                
                this.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                };
            }
            
            handleWebSocketMessage(data) {
                switch (data.type) {
                    case 'add':
                        if (data.node) this.addNode(data.node);
                        if (data.edge) this.addEdge(data.edge);
                        break;
                    case 'update':
                        if (data.node) this.updateNode(data.node);
                        if (data.edge) this.updateEdge(data.edge);
                        break;
                    case 'remove':
                        if (data.node) this.removeNode(data.node.id);
                        if (data.edge) this.removeEdge(data.edge.id);
                        break;
                    case 'metrics':
                        this.updateMetrics(data.data);
                        break;
                }
            }
            
            async loadInitialData() {
                try {
                    const response = await fetch('/api/topology');
                    const data = await response.json();
                    
                    this.nodes = data.nodes || [];
                    this.edges = data.edges || [];
                    this.metrics = data.metrics || {};
                    
                    this.updateMetrics(this.metrics);
                    this.loadView(this.currentView);
                } catch (error) {
                    console.error('Error loading initial data:', error);
                    this.showError('Failed to load topology data');
                }
            }
            
            loadView(viewName) {
                const filteredNodes = this.filterNodesByView(this.nodes, viewName);
                const filteredEdges = this.filterEdgesByView(this.edges, viewName);
                
                this.cy.elements().remove();
                this.cy.add(filteredNodes.map(node => ({
                    data: {
                        id: node.id,
                        name: node.name,
                        type: node.type,
                        status: node.status,
                        metadata: node.metadata
                    }
                })));
                
                this.cy.add(filteredEdges.map(edge => ({
                    data: {
                        id: edge.id,
                        source: edge.source,
                        target: edge.target,
                        type: edge.type
                    }
                })));
                
                this.cy.layout({ name: 'cose' }).run();
            }
            
            filterNodesByView(nodes, viewName) {
                const viewFilters = {
                    processes: ['process', 'container', 'host'],
                    containers: ['container', 'host'],
                    hosts: ['host'],
                    pods: ['pod', 'container', 'host'],
                    services: ['service', 'pod', 'container']
                };
                
                const allowedTypes = viewFilters[viewName] || [];
                return nodes.filter(node => allowedTypes.includes(node.type));
            }
            
            filterEdgesByView(edges, viewName) {
                const viewFilters = {
                    processes: ['process', 'container'],
                    containers: ['network', 'container'],
                    hosts: ['network'],
                    pods: ['network', 'container'],
                    services: ['network', 'service']
                };
                
                const allowedTypes = viewFilters[viewName] || [];
                return edges.filter(edge => allowedTypes.includes(edge.type));
            }
            
            addNode(node) {
                this.nodes.push(node);
                this.cy.add({
                    data: {
                        id: node.id,
                        name: node.name,
                        type: node.type,
                        status: node.status,
                        metadata: node.metadata
                    }
                });
            }
            
            updateNode(node) {
                const index = this.nodes.findIndex(n => n.id === node.id);
                if (index !== -1) {
                    this.nodes[index] = node;
                }
                
                const cyNode = this.cy.getElementById(node.id);
                if (cyNode.length > 0) {
                    cyNode.data({
                        name: node.name,
                        type: node.type,
                        status: node.status,
                        metadata: node.metadata
                    });
                }
            }
            
            removeNode(nodeId) {
                this.nodes = this.nodes.filter(n => n.id !== nodeId);
                this.cy.getElementById(nodeId).remove();
            }
            
            addEdge(edge) {
                this.edges.push(edge);
                this.cy.add({
                    data: {
                        id: edge.id,
                        source: edge.source,
                        target: edge.target,
                        type: edge.type
                    }
                });
            }
            
            updateEdge(edge) {
                const index = this.edges.findIndex(e => e.id === edge.id);
                if (index !== -1) {
                    this.edges[index] = edge;
                }
                
                const cyEdge = this.cy.getElementById(edge.id);
                if (cyEdge.length > 0) {
                    cyEdge.data({
                        source: edge.source,
                        target: edge.target,
                        type: edge.type
                    });
                }
            }
            
            removeEdge(edgeId) {
                this.edges = this.edges.filter(e => e.id !== edgeId);
                this.cy.getElementById(edgeId).remove();
            }
            
            selectNode(node) {
                this.selectedNode = node;
                this.showContextPanel(node);
            }
            
            deselectNode() {
                this.selectedNode = null;
                this.hideContextPanel();
            }
            
            showContextPanel(node) {
                const panel = document.getElementById('contextPanel');
                const title = document.getElementById('contextTitle');
                const content = document.getElementById('contextContent');
                
                title.textContent = node.data('name');
                
                const nodeData = this.nodes.find(n => n.id === node.id());
                if (nodeData) {
                    content.innerHTML = this.renderNodeDetails(nodeData);
                }
                
                panel.style.display = 'block';
            }
            
            hideContextPanel() {
                document.getElementById('contextPanel').style.display = 'none';
            }
            
            renderNodeDetails(node) {
                let html = '<div class="metric-item"><span class="metric-label">Type:</span><span class="metric-value">' + node.type + '</span></div>';
                html += '<div class="metric-item"><span class="metric-label">Status:</span><span class="metric-value"><span class="status-indicator status-' + node.status + '"></span>' + node.status + '</span></div>';
                
                if (node.metrics) {
                    if (node.metrics.cpu_usage) {
                        html += '<div class="metric-item"><span class="metric-label">CPU:</span><span class="metric-value">' + node.metrics.cpu_usage.current.toFixed(1) + '%</span></div>';
                    }
                    if (node.metrics.memory_usage) {
                        html += '<div class="metric-item"><span class="metric-label">Memory:</span><span class="metric-value">' + node.metrics.memory_usage.current.toFixed(1) + '%</span></div>';
                    }
                    if (node.metrics.connections) {
                        html += '<div class="metric-item"><span class="metric-label">Connections:</span><span class="metric-value">' + node.metrics.connections.current + '</span></div>';
                    }
                }
                
                return html;
            }
            
            updateMetrics(metrics) {
                document.getElementById('totalNodes').textContent = metrics.total_nodes || 0;
                document.getElementById('totalEdges').textContent = metrics.total_edges || 0;
                document.getElementById('healthyNodes').textContent = metrics.healthy_nodes || 0;
                document.getElementById('criticalNodes').textContent = metrics.critical_nodes || 0;
            }
            
            search(query) {
                if (!query) {
                    this.cy.elements().style('opacity', 1);
                    return;
                }
                
                this.cy.elements().style('opacity', 0.3);
                
                const matchingNodes = this.cy.elements().filter(node => {
                    const name = node.data('name').toLowerCase();
                    return name.includes(query.toLowerCase());
                });
                
                matchingNodes.style('opacity', 1);
                matchingNodes.neighborhood().style('opacity', 1);
            }
            
            autoLayout() {
                this.cy.layout({ name: 'cose' }).run();
            }
            
            showError(message) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'error';
                errorDiv.textContent = message;
                document.querySelector('.content').appendChild(errorDiv);
                
                setTimeout(() => {
                    errorDiv.remove();
                }, 5000);
            }
        }
        
        // Initialize the visualizer when the page loads
        document.addEventListener('DOMContentLoaded', () => {
            new TopologyVisualizer();
        });
    </script>
</body>
</html>
`

	wr.Header().Set("Content-Type", "text/html")
	wr.Write([]byte(tmpl))
}

// handleTopology handles the topology page
func (w *WebUI) handleTopology(wr http.ResponseWriter, r *http.Request) {
	w.handleDashboard(wr, r)
}

// handleTopologyView handles specific topology views
func (w *WebUI) handleTopologyView(wr http.ResponseWriter, r *http.Request) {
	w.handleDashboard(wr, r)
}

// handleAPIProxy proxies API requests to the topology manager
func (w *WebUI) handleAPIProxy(wr http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would proxy requests to the topology manager
	// For now, return mock data
	wr.Header().Set("Content-Type", "application/json")
	json.NewEncoder(wr).Encode(map[string]interface{}{
		"nodes": []interface{}{},
		"edges": []interface{}{},
		"metrics": map[string]interface{}{
			"total_nodes":    0,
			"total_edges":    0,
			"healthy_nodes":  0,
			"critical_nodes": 0,
		},
	})
}

// handleWebSocketProxy proxies WebSocket connections to the topology manager
func (w *WebUI) handleWebSocketProxy(wr http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would proxy WebSocket connections
	wr.WriteHeader(http.StatusNotImplemented)
	wr.Write([]byte("WebSocket proxy not implemented"))
}

// handleHealth handles health checks
func (w *WebUI) handleHealth(wr http.ResponseWriter, r *http.Request) {
	wr.Header().Set("Content-Type", "application/json")
	json.NewEncoder(wr).Encode(map[string]string{"status": "healthy"})
}
