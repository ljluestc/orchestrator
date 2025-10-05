package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
	"github.com/stretchr/testify/assert"
)

// TestLoadMultipleProbes tests the server's ability to handle multiple concurrent probes
func TestLoadMultipleProbes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := setupTestServer(t)

	numProbes := 10
	reportsPerProbe := 100
	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	startTime := time.Now()

	// Simulate multiple probes sending reports concurrently
	for i := 0; i < numProbes; i++ {
		wg.Add(1)
		go func(probeID int) {
			defer wg.Done()

			agentID := fmt.Sprintf("load-test-agent-%d", probeID)

			// Register agent
			registration := map[string]interface{}{
				"agent_id":  agentID,
				"hostname":  fmt.Sprintf("load-test-host-%d", probeID),
				"metadata":  map[string]string{"test": "load"},
				"timestamp": time.Now(),
			}
			body, _ := json.Marshal(registration)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/agents/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			// Send multiple reports
			for j := 0; j < reportsPerProbe; j++ {
				report := probe.ReportData{
					AgentID:   agentID,
					Hostname:  fmt.Sprintf("load-test-host-%d", probeID),
					Timestamp: time.Now(),
					HostInfo: &probe.HostInfo{
						Hostname: fmt.Sprintf("load-test-host-%d", probeID),
						CPUInfo: probe.CPUInfo{
							Cores: 4,
							Usage: float64(j % 100),
						},
						MemoryInfo: probe.MemoryInfo{
							TotalMB: 8192,
							UsedMB:  uint64(j % 4096),
							Usage:   float64(j%100) * 0.5,
						},
					},
				}

				body, _ := json.Marshal(report)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				server.router.ServeHTTP(w, req)

				if w.Code == http.StatusAccepted {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}

				// Small delay to simulate real-world reporting interval
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Load test completed in %v", duration)
	t.Logf("Total requests: %d", numProbes*reportsPerProbe)
	t.Logf("Successful reports: %d", successCount)
	t.Logf("Failed reports: %d", errorCount)
	t.Logf("Requests per second: %.2f", float64(successCount)/duration.Seconds())

	// Verify that most reports succeeded (allow for some failures)
	successRate := float64(successCount) / float64(numProbes*reportsPerProbe)
	assert.Greater(t, successRate, 0.95, "Success rate should be above 95%%")

	// Verify topology was built
	topology := server.aggregator.GetTopology()
	assert.GreaterOrEqual(t, len(topology.Nodes), numProbes, "Should have at least one node per probe")

	// Verify storage
	stats := server.storage.GetStats()
	t.Logf("Storage stats: %+v", stats)
}

// TestLoadConcurrentReads tests concurrent read operations
func TestLoadConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := setupTestServer(t)

	// Add some test data
	for i := 0; i < 5; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		report := &probe.ReportData{
			AgentID:   agentID,
			Hostname:  fmt.Sprintf("host-%d", i),
			Timestamp: time.Now(),
		}
		server.storage.AddReport(report)
		server.aggregator.ProcessReport(report)
	}

	numReaders := 50
	readsPerReader := 100
	var successCount int64
	var wg sync.WaitGroup

	startTime := time.Now()

	// Simulate concurrent reads
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < readsPerReader; j++ {
				// Random read operation
				switch j % 4 {
				case 0:
					// Get topology
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/api/v1/query/topology", nil)
					server.router.ServeHTTP(w, req)
					if w.Code == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
				case 1:
					// Get stats
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/api/v1/query/stats", nil)
					server.router.ServeHTTP(w, req)
					if w.Code == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
				case 2:
					// List agents
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/api/v1/agents/list", nil)
					server.router.ServeHTTP(w, req)
					if w.Code == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
				case 3:
					// Health check
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/health", nil)
					server.router.ServeHTTP(w, req)
					if w.Code == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent read test completed in %v", duration)
	t.Logf("Total reads: %d", numReaders*readsPerReader)
	t.Logf("Successful reads: %d", successCount)
	t.Logf("Reads per second: %.2f", float64(successCount)/duration.Seconds())

	// All reads should succeed
	assert.Equal(t, int64(numReaders*readsPerReader), successCount)
}

// TestLoadMixedWorkload tests a mix of reads and writes
func TestLoadMixedWorkload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := setupTestServer(t)

	numWorkers := 20
	operationsPerWorker := 50
	var writeCount int64
	var readCount int64
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			agentID := fmt.Sprintf("worker-agent-%d", workerID)

			for j := 0; j < operationsPerWorker; j++ {
				if j%2 == 0 {
					// Write operation - submit report
					report := probe.ReportData{
						AgentID:   agentID,
						Hostname:  fmt.Sprintf("worker-host-%d", workerID),
						Timestamp: time.Now(),
						HostInfo: &probe.HostInfo{
							Hostname: fmt.Sprintf("worker-host-%d", workerID),
							CPUInfo: probe.CPUInfo{
								Cores: 4,
							},
						},
					}

					body, _ := json.Marshal(report)
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
					req.Header.Set("Content-Type", "application/json")
					server.router.ServeHTTP(w, req)

					if w.Code == http.StatusAccepted {
						atomic.AddInt64(&writeCount, 1)
					}
				} else {
					// Read operation - get topology
					w := httptest.NewRecorder()
					req, _ := http.NewRequest("GET", "/api/v1/query/topology", nil)
					server.router.ServeHTTP(w, req)

					if w.Code == http.StatusOK {
						atomic.AddInt64(&readCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Mixed workload test completed in %v", duration)
	t.Logf("Total operations: %d", numWorkers*operationsPerWorker)
	t.Logf("Successful writes: %d", writeCount)
	t.Logf("Successful reads: %d", readCount)
	t.Logf("Operations per second: %.2f", float64(writeCount+readCount)/duration.Seconds())

	// Verify results
	expectedWrites := int64(numWorkers * operationsPerWorker / 2)
	expectedReads := int64(numWorkers * operationsPerWorker / 2)

	assert.Equal(t, expectedWrites, writeCount)
	assert.Equal(t, expectedReads, readCount)
}

// BenchmarkReportSubmission benchmarks report submission performance
func BenchmarkReportSubmission(b *testing.B) {
	server := setupTestServer(&testing.T{})

	report := probe.ReportData{
		AgentID:   "bench-agent",
		Hostname:  "bench-host",
		Timestamp: time.Now(),
		HostInfo: &probe.HostInfo{
			Hostname: "bench-host",
			CPUInfo: probe.CPUInfo{
				Cores: 8,
				Usage: 50.0,
			},
			MemoryInfo: probe.MemoryInfo{
				TotalMB: 16384,
				UsedMB:  8192,
				Usage:   50.0,
			},
		},
	}

	body, _ := json.Marshal(report)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/reports", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		server.router.ServeHTTP(w, req)
	}
}

// BenchmarkTopologyQuery benchmarks topology query performance
func BenchmarkTopologyQuery(b *testing.B) {
	server := setupTestServer(&testing.T{})

	// Add some test data
	for i := 0; i < 10; i++ {
		report := &probe.ReportData{
			AgentID:   fmt.Sprintf("bench-agent-%d", i),
			Hostname:  fmt.Sprintf("bench-host-%d", i),
			Timestamp: time.Now(),
		}
		server.aggregator.ProcessReport(report)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/query/topology", nil)
		server.router.ServeHTTP(w, req)
	}
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	server := setupTestServer(t)
	ctx := context.Background()
	err := server.Start(ctx)
	assert.NoError(t, err)
	defer server.Stop()

	// Add a large number of reports
	numReports := 1000
	for i := 0; i < numReports; i++ {
		report := &probe.ReportData{
			AgentID:   fmt.Sprintf("mem-test-agent-%d", i%10), // 10 agents
			Hostname:  fmt.Sprintf("mem-test-host-%d", i%10),
			Timestamp: time.Now(),
			HostInfo: &probe.HostInfo{
				Hostname: fmt.Sprintf("mem-test-host-%d", i%10),
				CPUInfo: probe.CPUInfo{
					Cores: 8,
					Usage: float64(i % 100),
				},
			},
		}
		server.storage.AddReport(report)
		server.aggregator.ProcessReport(report)
	}

	// Verify data structures
	stats := server.GetStats()
	t.Logf("Server stats after %d reports: %+v", numReports, stats)

	// Cleanup should work
	server.cleanup()

	statsAfterCleanup := server.GetStats()
	t.Logf("Server stats after cleanup: %+v", statsAfterCleanup)
}
