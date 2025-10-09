package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ljluestc/orchestrator/pkg/marathon"
	"github.com/ljluestc/orchestrator/pkg/mesos"
	"github.com/ljluestc/orchestrator/pkg/migration"
	"github.com/ljluestc/orchestrator/pkg/topology"
	"github.com/ljluestc/orchestrator/pkg/ui"
)

var (
	// Command line flags
	mode           = flag.String("mode", "orchestrator", "Mode: orchestrator, mesos-master, mesos-agent, marathon, migration, topology, web-ui")
	hostname       = flag.String("hostname", "localhost", "Hostname")
	port           = flag.Int("port", 8080, "Port")
	masterURL      = flag.String("master", "http://localhost:5050", "Mesos master URL")
	zookeeperURL   = flag.String("zookeeper", "localhost:2181", "Zookeeper URL")
	agentID        = flag.String("agent-id", "", "Agent ID")
	frameworkID    = flag.String("framework-id", "", "Framework ID")
	sourceCluster  = flag.String("source-cluster", "cluster-a", "Source Zookeeper cluster")
	targetCluster  = flag.String("target-cluster", "cluster-b", "Target Zookeeper cluster")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	switch *mode {
	case "orchestrator":
		runOrchestrator(ctx)
	case "mesos-master":
		runMesosMaster(ctx)
	case "mesos-agent":
		runMesosAgent(ctx)
	case "marathon":
		runMarathon(ctx)
	case "migration":
		runMigration(ctx)
	case "topology":
		runTopology(ctx)
	case "web-ui":
		runWebUI(ctx)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runOrchestrator(ctx context.Context) {
	log.Println("Starting Mesos-Docker Orchestration Platform")
	
	// Start Mesos Master
	master := mesos.NewMaster("master-1", *hostname, 5050, *zookeeperURL)
	go func() {
		if err := master.Start(); err != nil {
			log.Printf("Mesos master error: %v", err)
		}
	}()

	// Start Marathon
	marathon := marathon.NewMarathon("marathon-1", *hostname, 8080, "http://localhost:5050")
	go func() {
		if err := marathon.Start(); err != nil {
			log.Printf("Marathon error: %v", err)
		}
	}()

	// Start topology manager
	topologyManager := topology.NewManager("topology-1")
	go func() {
		if err := topologyManager.Start(); err != nil {
			log.Printf("Topology manager error: %v", err)
		}
	}()

	// Start topology collector to bridge app server and topology manager
	collector := topology.NewCollector("collector-1", "http://localhost:8082", nil)
	collector.Manager = topologyManager
	go func() {
		if err := collector.Start(); err != nil {
			log.Printf("Topology collector error: %v", err)
		}
	}()

	// Start Web UI
	webUI := ui.NewWebUI("web-ui-1", 9090, "http://localhost:8082")
	go func() {
		if err := webUI.Start(); err != nil {
			log.Printf("Web UI error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down orchestrator...")
}

func runMesosMaster(ctx context.Context) {
	log.Printf("Starting Mesos Master on %s:%d", *hostname, *port)
	
	master := mesos.NewMaster("master-1", *hostname, *port, *zookeeperURL)
	
	if err := master.Start(); err != nil {
		log.Fatalf("Failed to start Mesos master: %v", err)
	}
}

func runMesosAgent(ctx context.Context) {
	if *agentID == "" {
		*agentID = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	
	log.Printf("Starting Mesos Agent %s on %s:%d", *agentID, *hostname, *port)
	
	agent := mesos.NewAgent(*agentID, *hostname, *port, *masterURL)
	
	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start Mesos agent: %v", err)
	}
}

func runMarathon(ctx context.Context) {
	if *frameworkID == "" {
		*frameworkID = "marathon-1"
	}
	
	log.Printf("Starting Marathon Framework %s on %s:%d", *frameworkID, *hostname, *port)
	
	marathon := marathon.NewMarathon(*frameworkID, *hostname, *port, *masterURL)
	
	if err := marathon.Start(); err != nil {
		log.Fatalf("Failed to start Marathon: %v", err)
	}
}

func runMigration(ctx context.Context) {
	log.Printf("Starting Zookeeper Migration Manager")
	
	source := &migration.ZookeeperCluster{
		ID:     *sourceCluster,
		Hosts:  []string{"localhost:2181"},
		Port:   2181,
		Status: "active",
	}
	
	target := &migration.ZookeeperCluster{
		ID:     *targetCluster,
		Hosts:  []string{"localhost:2182"},
		Port:   2182,
		Status: "inactive",
	}
	
	migrationManager := migration.NewMigrationManager("migration-1", source, target)
	
	if err := migrationManager.Start(); err != nil {
		log.Fatalf("Failed to start migration manager: %v", err)
	}
}

func startWebUI(ctx context.Context) {
	// Simple web UI for the orchestration platform
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Mesos-Docker Orchestration Platform</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .status { display: inline-block; padding: 5px 10px; border-radius: 3px; color: white; }
        .status.healthy { background: #27ae60; }
        .status.unhealthy { background: #e74c3c; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { border: 1px solid #ddd; padding: 15px; border-radius: 5px; }
        .card h3 { margin-top: 0; color: #2c3e50; }
        .metrics { display: flex; justify-content: space-around; text-align: center; }
        .metric { padding: 10px; }
        .metric-value { font-size: 24px; font-weight: bold; color: #3498db; }
        .metric-label { font-size: 14px; color: #7f8c8d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Mesos-Docker Orchestration Platform</h1>
            <p>Unified datacenter resource management and container orchestration</p>
        </div>
        
        <div class="section">
            <h2>Platform Status</h2>
            <div class="grid">
                <div class="card">
                    <h3>Mesos Master</h3>
                    <p>Status: <span class="status healthy">Healthy</span></p>
                    <p>URL: <a href="http://localhost:5050">http://localhost:5050</a></p>
                </div>
                <div class="card">
                    <h3>Marathon Framework</h3>
                    <p>Status: <span class="status healthy">Healthy</span></p>
                    <p>URL: <a href="http://localhost:8080">http://localhost:8080</a></p>
                </div>
                <div class="card">
                    <h3>Zookeeper Migration</h3>
                    <p>Status: <span class="status healthy">Ready</span></p>
                    <p>URL: <a href="http://localhost:8080/api/v1/migration/status">Migration Status</a></p>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>Cluster Metrics</h2>
            <div class="metrics">
                <div class="metric">
                    <div class="metric-value">5</div>
                    <div class="metric-label">Active Agents</div>
                </div>
                <div class="metric">
                    <div class="metric-value">12</div>
                    <div class="metric-label">Running Tasks</div>
                </div>
                <div class="metric">
                    <div class="metric-value">3</div>
                    <div class="metric-label">Frameworks</div>
                </div>
                <div class="metric">
                    <div class="metric-value">85%</div>
                    <div class="metric-label">Resource Usage</div>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>Quick Actions</h2>
            <div class="grid">
                <div class="card">
                    <h3>Deploy Application</h3>
                    <p>Deploy a new application via Marathon</p>
                    <button onclick="window.open('http://localhost:8080/v2/apps', '_blank')">Marathon UI</button>
                </div>
                <div class="card">
                    <h3>View Cluster State</h3>
                    <p>Monitor cluster resources and tasks</p>
                    <button onclick="window.open('http://localhost:5050/api/v1/master/state', '_blank')">Cluster State</button>
                </div>
                <div class="card">
                    <h3>Migration Status</h3>
                    <p>Monitor Zookeeper migration progress</p>
                    <button onclick="window.open('http://localhost:8080/api/v1/migration/status', '_blank')">Migration Status</button>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>API Endpoints</h2>
            <ul>
                <li><strong>Mesos Master:</strong> <a href="http://localhost:5050/api/v1/master/info">http://localhost:5050/api/v1/master/info</a></li>
                <li><strong>Marathon Apps:</strong> <a href="http://localhost:8080/v2/apps">http://localhost:8080/v2/apps</a></li>
                <li><strong>Migration Status:</strong> <a href="http://localhost:8080/api/v1/migration/status">http://localhost:8080/api/v1/migration/status</a></li>
                <li><strong>Health Check:</strong> <a href="http://localhost:8080/health">http://localhost:8080/health</a></li>
            </ul>
        </div>
    </div>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
	
	go func() {
		log.Println("Starting web UI on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Web UI error: %v", err)
		}
	}()
}

func runTopology(ctx context.Context) {
	log.Printf("Starting Topology Manager on %s:%d", *hostname, *port)
	
	topologyManager := topology.NewManager("topology-1")
	
	if err := topologyManager.Start(); err != nil {
		log.Fatalf("Failed to start topology manager: %v", err)
	}
}

func runWebUI(ctx context.Context) {
	log.Printf("Starting Web UI on %s:%d", *hostname, *port)
	
	webUI := ui.NewWebUI("web-ui-1", *port, "http://localhost:8082")
	
	if err := webUI.Start(); err != nil {
		log.Fatalf("Failed to start web UI: %v", err)
	}
}
