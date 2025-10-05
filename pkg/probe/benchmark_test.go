package probe

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// BenchmarkHostCollector_Collect benchmarks host information collection
func BenchmarkHostCollector_Collect(b *testing.B) {
	collector := NewHostCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProcessCollector_Collect benchmarks process information collection
func BenchmarkProcessCollector_Collect(b *testing.B) {
	benchmarks := []struct {
		name         string
		includeAll   bool
		maxProcesses int
	}{
		{"All_NoLimit", true, 0},
		{"All_Limit100", true, 100},
		{"All_Limit10", true, 10},
		{"ContainersOnly_NoLimit", false, 0},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			collector := NewProcessCollector(bm.includeAll, bm.maxProcesses)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := collector.Collect()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkNetworkCollector_Collect benchmarks network information collection
func BenchmarkNetworkCollector_Collect(b *testing.B) {
	benchmarks := []struct {
		name             string
		includeLocalhost bool
		maxConnections   int
		resolveProcesses bool
	}{
		{"NoLimit_NoResolve", true, 0, false},
		{"Limit100_NoResolve", true, 100, false},
		{"NoLimit_Resolve", true, 0, true},
		{"Limit100_Resolve", true, 100, true},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			collector := NewNetworkCollector(bm.includeLocalhost, bm.maxConnections, bm.resolveProcesses)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := collector.Collect()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDockerCollector_Collect benchmarks Docker container information collection
func BenchmarkDockerCollector_Collect(b *testing.B) {
	collector, err := NewDockerCollector(false)
	if err != nil {
		b.Skip("Docker not available")
	}
	defer collector.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDockerCollector_CollectWithStats benchmarks Docker collection with stats
func BenchmarkDockerCollector_CollectWithStats(b *testing.B) {
	collector, err := NewDockerCollector(true)
	if err != nil {
		b.Skip("Docker not available")
	}
	defer collector.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProbe_Collect benchmarks full probe collection cycle
func BenchmarkProbe_Collect(b *testing.B) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:9999", // Dummy URL
		AgentID:             "benchmark-agent",
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        100,
		MaxConnections:      100,
		IncludeAllProcesses: true,
		IncludeLocalhost:    true,
		ResolveProcesses:    false,
	}

	// Try to add Docker if available
	_, err := NewDockerCollector(false)
	if err == nil {
		config.CollectDocker = true
	}

	probe, err := NewProbe(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		probe.collectAndSend(ctx)
	}
}

// TestResourceUsage validates that probe stays within resource limits
func TestResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	config := ProbeConfig{
		ServerURL:           "http://localhost:9999", // Dummy URL
		AgentID:             "resource-test",
		CollectionInterval:  100 * time.Millisecond,
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        100,
		MaxConnections:      100,
		IncludeAllProcesses: true,
		IncludeLocalhost:    true,
	}

	probe, err := NewProbe(config)
	if err != nil {
		t.Fatal(err)
	}

	// Start collection without sending (to avoid network errors)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Run collections
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Collect without sending
				report := &ReportData{Hostname: "test"}

				if probe.hostCollector != nil {
					hostInfo, _ := probe.hostCollector.Collect()
					report.HostInfo = hostInfo
				}

				if probe.processCollector != nil {
					procInfo, _ := probe.processCollector.Collect()
					report.ProcessesInfo = procInfo
				}

				if probe.networkCollector != nil {
					netInfo, _ := probe.networkCollector.Collect()
					report.NetworkInfo = netInfo
				}

			case <-ctx.Done():
				done <- true
				return
			}
		}
	}()

	<-done

	// Get final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory usage
	allocMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	t.Logf("Memory allocated during test: %.2f MB", allocMB)

	// Validate memory usage is reasonable (< 100MB)
	if allocMB > 100 {
		t.Errorf("Memory usage too high: %.2f MB (limit: 100 MB)", allocMB)
	}
}

// BenchmarkMemoryAllocation measures memory allocations per collection
func BenchmarkMemoryAllocation(b *testing.B) {
	collector := NewHostCollector()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := collector.Collect()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParallelCollection tests concurrent collection performance
func BenchmarkParallelCollection(b *testing.B) {
	hostCollector := NewHostCollector()
	processCollector := NewProcessCollector(true, 100)
	networkCollector := NewNetworkCollector(true, 100, false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate parallel collection
			go func() {
				hostCollector.Collect()
			}()
			go func() {
				processCollector.Collect()
			}()
			go func() {
				networkCollector.Collect()
			}()
		}
	})
}

// BenchmarkClient_SendReport benchmarks report transmission
func BenchmarkClient_SendReport(b *testing.B) {
	// Create mock server
	server := newMockServer()
	defer server.Close()

	client := NewClient(ClientConfig{
		ServerURL: server.server.URL,
		AgentID:   "bench-agent",
	})

	report := &ReportData{
		Hostname: "test-host",
		HostInfo: &HostInfo{
			Hostname: "test-host",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.SendReport(ctx, report)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// CPU usage benchmark - measures CPU time
func BenchmarkCPUUsage(b *testing.B) {
	config := ProbeConfig{
		ServerURL:           "http://localhost:9999",
		AgentID:             "cpu-bench",
		CollectHost:         true,
		CollectProcesses:    true,
		CollectNetwork:      true,
		MaxProcesses:        50,
		MaxConnections:      50,
		IncludeAllProcesses: true,
	}

	probe, err := NewProbe(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	// Measure CPU time
	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		probe.collectAndSend(ctx)
	}

	elapsed := time.Since(start)
	cpuTimePerOp := elapsed / time.Duration(b.N)

	b.ReportMetric(float64(cpuTimePerOp.Microseconds()), "μs/op")
}
