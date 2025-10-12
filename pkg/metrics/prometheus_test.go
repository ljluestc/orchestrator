package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordHTTPRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		status   string
		duration time.Duration
	}{
		{
			name:     "GET request",
			method:   "GET",
			path:     "/api/v1/topology",
			status:   "200",
			duration: 100 * time.Millisecond,
		},
		{
			name:     "POST request",
			method:   "POST",
			path:     "/api/v1/reports",
			status:   "201",
			duration: 50 * time.Millisecond,
		},
		{
			name:     "Error request",
			method:   "GET",
			path:     "/api/v1/invalid",
			status:   "404",
			duration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

			// Record the request
			RecordHTTPRequest(tt.method, tt.path, tt.status, tt.duration)

			// Verify counter was incremented
			counterValue := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues(tt.method, tt.path, tt.status))
			assert.Equal(t, 1.0, counterValue)

			// Note: Histogram testing is complex with testutil.ToFloat64
			// We verify the counter was incremented, which confirms the function was called
		})
	}
}

func TestUpdateMesosMetrics(t *testing.T) {
	tests := []struct {
		name      string
		agents    int
		tasks     int
		cpuTotal  float64
		cpuUsed   float64
		memTotal  float64
		memUsed   float64
	}{
		{
			name:      "Normal metrics",
			agents:    10,
			tasks:     50,
			cpuTotal:  100.0,
			cpuUsed:   60.0,
			memTotal:  1024.0,
			memUsed:   512.0,
		},
		{
			name:      "Zero metrics",
			agents:    0,
			tasks:     0,
			cpuTotal:  0.0,
			cpuUsed:   0.0,
			memTotal:  0.0,
			memUsed:   0.0,
		},
		{
			name:      "High utilization",
			agents:    100,
			tasks:     1000,
			cpuTotal:  1000.0,
			cpuUsed:   950.0,
			memTotal:  10000.0,
			memUsed:   9000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

			// Update metrics
			UpdateMesosMetrics(tt.agents, tt.tasks, tt.cpuTotal, tt.cpuUsed, tt.memTotal, tt.memUsed)

			// Verify agent count
			agentValue := testutil.ToFloat64(MesosAgentsRegistered)
			assert.Equal(t, float64(tt.agents), agentValue)

			// Verify task count
			taskValue := testutil.ToFloat64(MesosTasksRunning)
			assert.Equal(t, float64(tt.tasks), taskValue)

			// Verify CPU metrics
			cpuTotalValue := testutil.ToFloat64(MesosResourcesCPU.WithLabelValues("total"))
			assert.Equal(t, tt.cpuTotal, cpuTotalValue)

			cpuUsedValue := testutil.ToFloat64(MesosResourcesCPU.WithLabelValues("used"))
			assert.Equal(t, tt.cpuUsed, cpuUsedValue)

			cpuAvailableValue := testutil.ToFloat64(MesosResourcesCPU.WithLabelValues("available"))
			assert.Equal(t, tt.cpuTotal-tt.cpuUsed, cpuAvailableValue)

			// Verify memory metrics
			memTotalValue := testutil.ToFloat64(MesosResourcesMemory.WithLabelValues("total"))
			assert.Equal(t, tt.memTotal, memTotalValue)

			memUsedValue := testutil.ToFloat64(MesosResourcesMemory.WithLabelValues("used"))
			assert.Equal(t, tt.memUsed, memUsedValue)

			memAvailableValue := testutil.ToFloat64(MesosResourcesMemory.WithLabelValues("available"))
			assert.Equal(t, tt.memTotal-tt.memUsed, memAvailableValue)
		})
	}
}

func TestUpdateMarathonMetrics(t *testing.T) {
	tests := []struct {
		name        string
		apps        int
		deployments int
	}{
		{
			name:        "Normal deployment",
			apps:        5,
			deployments: 2,
		},
		{
			name:        "No deployments",
			apps:        10,
			deployments: 0,
		},
		{
			name:        "Many deployments",
			apps:        50,
			deployments: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

			// Update metrics
			UpdateMarathonMetrics(tt.apps, tt.deployments)

			// Verify app count
			appValue := testutil.ToFloat64(MarathonAppsRunning)
			assert.Equal(t, float64(tt.apps), appValue)

			// Verify deployment count
			deploymentValue := testutil.ToFloat64(MarathonDeploymentsActive)
			assert.Equal(t, float64(tt.deployments), deploymentValue)
		})
	}
}

func TestUpdateZookeeperMetrics(t *testing.T) {
	tests := []struct {
		name      string
		cluster   string
		connected bool
		syncLag   time.Duration
	}{
		{
			name:      "Connected cluster",
			cluster:   "cluster-a",
			connected: true,
			syncLag:   50 * time.Millisecond,
		},
		{
			name:      "Disconnected cluster",
			cluster:   "cluster-b",
			connected: false,
			syncLag:   1000 * time.Millisecond,
		},
		{
			name:      "High sync lag",
			cluster:   "cluster-c",
			connected: true,
			syncLag:   5000 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

			// Update metrics
			UpdateZookeeperMetrics(tt.cluster, tt.connected, tt.syncLag)

			// Verify connection state
			expectedState := 0.0
			if tt.connected {
				expectedState = 1.0
			}
			stateValue := testutil.ToFloat64(ZookeeperConnectionState.WithLabelValues(tt.cluster))
			assert.Equal(t, expectedState, stateValue)

			// Verify sync lag
			lagValue := testutil.ToFloat64(ZookeeperSyncLag)
			assert.Equal(t, float64(tt.syncLag.Milliseconds()), lagValue)
		})
	}
}

func TestUpdateProbeMetrics(t *testing.T) {
	tests := []struct {
		name       string
		nodes      int
		containers int
	}{
		{
			name:       "Small deployment",
			nodes:      5,
			containers: 20,
		},
		{
			name:       "Large deployment",
			nodes:      100,
			containers: 1000,
		},
		{
			name:       "No containers",
			nodes:      10,
			containers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

			// Update metrics
			UpdateProbeMetrics(tt.nodes, tt.containers)

			// Verify node count
			nodeValue := testutil.ToFloat64(ProbeNodesDiscovered)
			assert.Equal(t, float64(tt.nodes), nodeValue)

			// Verify container count
			containerValue := testutil.ToFloat64(ProbeContainersDiscovered)
			assert.Equal(t, float64(tt.containers), containerValue)
		})
	}
}

func TestMetricsRegistration(t *testing.T) {
	// Test that all metrics are properly registered
	registry := prometheus.NewRegistry()
	
	// Register all metrics
	registry.MustRegister(HTTPRequestsTotal)
	registry.MustRegister(HTTPRequestDuration)
	registry.MustRegister(MesosMasterLeaderElections)
	registry.MustRegister(MesosAgentsRegistered)
	registry.MustRegister(MesosTasksRunning)
	registry.MustRegister(MesosResourcesCPU)
	registry.MustRegister(MesosResourcesMemory)
	registry.MustRegister(MarathonAppsRunning)
	registry.MustRegister(MarathonDeploymentsActive)
	registry.MustRegister(ZookeeperSyncLag)
	registry.MustRegister(ZookeeperConnectionState)
	registry.MustRegister(ProbeReportsReceived)
	registry.MustRegister(ProbeNodesDiscovered)
	registry.MustRegister(ProbeContainersDiscovered)

	// Verify metrics are registered
	metrics, err := registry.Gather()
	require.NoError(t, err)
	assert.Greater(t, len(metrics), 0)

	// Check specific metric names
	metricNames := make(map[string]bool)
	for _, metric := range metrics {
		metricNames[*metric.Name] = true
	}

	expectedMetrics := []string{
		"http_requests_total",
		"http_request_duration_milliseconds",
		"mesos_master_leader_elections_total",
		"mesos_agents_registered",
		"mesos_tasks_running",
		"mesos_resources_cpus",
		"mesos_resources_memory_bytes",
		"marathon_apps_running",
		"marathon_deployments_active",
		"zookeeper_sync_lag_milliseconds",
		"zookeeper_connection_state",
		"probe_reports_received_total",
		"probe_nodes_discovered",
		"probe_containers_discovered",
	}

	for _, expectedMetric := range expectedMetrics {
		assert.True(t, metricNames[expectedMetric], "Metric %s should be registered", expectedMetric)
	}
}

func TestMetricsConcurrency(t *testing.T) {
	// Test concurrent access to metrics
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				RecordHTTPRequest("GET", "/test", "200", 10*time.Millisecond)
				UpdateMesosMetrics(1, 1, 1.0, 0.5, 1024.0, 512.0)
				UpdateMarathonMetrics(1, 1)
				UpdateZookeeperMetrics("test-cluster", true, 10*time.Millisecond)
				UpdateProbeMetrics(1, 1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final values
	totalRequests := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
	assert.Equal(t, float64(numGoroutines*numOperations), totalRequests)
}

func TestMetricsEdgeCases(t *testing.T) {
		t.Run("Negative values", func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

		// Test with negative values
		UpdateMesosMetrics(-1, -1, -100.0, -50.0, -1024.0, -512.0)

		// Metrics should handle negative values gracefully
		agentValue := testutil.ToFloat64(MesosAgentsRegistered)
		assert.Equal(t, float64(-1), agentValue)

		taskValue := testutil.ToFloat64(MesosTasksRunning)
		assert.Equal(t, float64(-1), taskValue)
	})

		t.Run("Very large values", func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

		// Test with very large values
		UpdateMesosMetrics(1000000, 10000000, 1000000.0, 999999.0, 1000000000.0, 999999999.0)

		// Metrics should handle large values
		agentValue := testutil.ToFloat64(MesosAgentsRegistered)
		assert.Equal(t, float64(1000000), agentValue)

		taskValue := testutil.ToFloat64(MesosTasksRunning)
		assert.Equal(t, float64(10000000), taskValue)
	})

		t.Run("Zero duration", func(t *testing.T) {
			// Note: Prometheus metrics don't have Reset() methods

		// Test with zero duration
		RecordHTTPRequest("GET", "/zero-duration-test", "200", 0)

			// Should handle zero duration
			// Note: Histogram testing is complex with testutil.ToFloat64
			// We verify the function was called by checking the counter
			counterValue := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/zero-duration-test", "200"))
			assert.Equal(t, 1.0, counterValue)
	})
}

func BenchmarkRecordHTTPRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordHTTPRequest("GET", "/api/v1/topology", "200", 10*time.Millisecond)
	}
}

func BenchmarkUpdateMesosMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateMesosMetrics(100, 1000, 1000.0, 500.0, 1000000.0, 500000.0)
	}
}

func BenchmarkUpdateMarathonMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateMarathonMetrics(50, 25)
	}
}

func BenchmarkUpdateZookeeperMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateZookeeperMetrics("test-cluster", true, 50*time.Millisecond)
	}
}

func BenchmarkUpdateProbeMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UpdateProbeMetrics(100, 1000)
	}
}
