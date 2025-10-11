// Package metrics provides Prometheus metrics collection (Task 16)
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_milliseconds",
			Help:    "HTTP request duration in milliseconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Mesos metrics
	MesosMasterLeaderElections = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "mesos_master_leader_elections_total",
			Help: "Total number of leader elections",
		},
	)

	MesosAgentsRegistered = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mesos_agents_registered",
			Help: "Number of registered Mesos agents",
		},
	)

	MesosTasksRunning = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mesos_tasks_running",
			Help: "Number of running tasks",
		},
	)

	MesosResourcesCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mesos_resources_cpus",
			Help: "CPU resources (total/used/available)",
		},
		[]string{"type"},
	)

	MesosResourcesMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mesos_resources_memory_bytes",
			Help: "Memory resources in bytes",
		},
		[]string{"type"},
	)

	// Marathon metrics
	MarathonAppsRunning = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "marathon_apps_running",
			Help: "Number of running Marathon applications",
		},
	)

	MarathonDeploymentsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "marathon_deployments_active",
			Help: "Number of active deployments",
		},
	)

	// Zookeeper metrics
	ZookeeperSyncLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "zookeeper_sync_lag_milliseconds",
			Help: "Zookeeper sync lag in milliseconds",
		},
	)

	ZookeeperConnectionState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zookeeper_connection_state",
			Help: "Zookeeper connection state (1=connected, 0=disconnected)",
		},
		[]string{"cluster"},
	)

	// Monitoring probe metrics
	ProbeReportsReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "probe_reports_received_total",
			Help: "Total number of probe reports received",
		},
	)

	ProbeNodesDiscovered = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "probe_nodes_discovered",
			Help: "Number of nodes discovered by probes",
		},
	)

	ProbeContainersDiscovered = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "probe_containers_discovered",
			Help: "Number of containers discovered",
		},
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, path, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(float64(duration.Milliseconds()))
}

// UpdateMesosMetrics updates Mesos-related metrics
func UpdateMesosMetrics(agents, tasks int, cpuTotal, cpuUsed, memTotal, memUsed float64) {
	MesosAgentsRegistered.Set(float64(agents))
	MesosTasksRunning.Set(float64(tasks))
	MesosResourcesCPU.WithLabelValues("total").Set(cpuTotal)
	MesosResourcesCPU.WithLabelValues("used").Set(cpuUsed)
	MesosResourcesCPU.WithLabelValues("available").Set(cpuTotal - cpuUsed)
	MesosResourcesMemory.WithLabelValues("total").Set(memTotal)
	MesosResourcesMemory.WithLabelValues("used").Set(memUsed)
	MesosResourcesMemory.WithLabelValues("available").Set(memTotal - memUsed)
}

// UpdateMarathonMetrics updates Marathon metrics
func UpdateMarathonMetrics(apps, deployments int) {
	MarathonAppsRunning.Set(float64(apps))
	MarathonDeploymentsActive.Set(float64(deployments))
}

// UpdateZookeeperMetrics updates Zookeeper metrics
func UpdateZookeeperMetrics(cluster string, connected bool, syncLag time.Duration) {
	state := 0.0
	if connected {
		state = 1.0
	}
	ZookeeperConnectionState.WithLabelValues(cluster).Set(state)
	ZookeeperSyncLag.Set(float64(syncLag.Milliseconds()))
}

// UpdateProbeMetrics updates probe discovery metrics
func UpdateProbeMetrics(nodes, containers int) {
	ProbeNodesDiscovered.Set(float64(nodes))
	ProbeContainersDiscovered.Set(float64(containers))
}
