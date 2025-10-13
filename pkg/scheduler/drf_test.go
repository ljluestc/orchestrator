package scheduler

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDRFScheduler(t *testing.T) {
	clusterTotal := Resources{
		CPU:    100.0,
		Memory: 1000.0,
		GPU:    10.0,
		Disk:   5000.0,
	}

	scheduler := NewDRFScheduler(clusterTotal)

	assert.NotNil(t, scheduler)
	assert.Equal(t, clusterTotal, scheduler.clusterTotal)
	assert.NotNil(t, scheduler.tenants)
	assert.Empty(t, scheduler.tenants)
}

func TestDRFScheduler_RegisterTenant(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})

	tests := []struct {
		name   string
		id     string
		tenantName string
		quota  Resources
		weight float64
	}{
		{
			name:   "Valid tenant with default weight",
			id:     "tenant-1",
			tenantName: "Tenant One",
			quota:  Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000},
			weight: 1.0,
		},
		{
			name:   "Valid tenant with custom weight",
			id:     "tenant-2",
			tenantName: "Tenant Two",
			quota:  Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500},
			weight: 2.0,
		},
		{
			name:   "Tenant with zero weight (should default to 1.0)",
			id:     "tenant-3",
			tenantName: "Tenant Three",
			quota:  Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			weight: 0.0,
		},
		{
			name:   "Tenant with negative weight (should default to 1.0)",
			id:     "tenant-4",
			tenantName: "Tenant Four",
			quota:  Resources{CPU: 15, Memory: 150, GPU: 1, Disk: 750},
			weight: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler.RegisterTenant(tt.id, tt.tenantName, tt.quota, tt.weight)

			tenant, exists := scheduler.tenants[tt.id]
			assert.True(t, exists)
			assert.Equal(t, tt.id, tenant.ID)
			assert.Equal(t, tt.tenantName, tenant.Name)
			assert.Equal(t, tt.quota, tenant.Quota)
			
			expectedWeight := tt.weight
			if expectedWeight <= 0 {
				expectedWeight = 1.0
			}
			assert.Equal(t, expectedWeight, tenant.Weight)
			assert.Equal(t, 0.0, tenant.AllocatedCPU)
			assert.Equal(t, 0.0, tenant.AllocatedMemory)
			assert.Equal(t, 0.0, tenant.AllocatedGPU)
			assert.Equal(t, 0.0, tenant.DominantShare)
			assert.Equal(t, 0.0, tenant.WeightedShare)
		})
	}
}

func TestDRFScheduler_ScheduleTask(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	// Register tenants
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)
	scheduler.RegisterTenant("tenant-2", "Tenant Two", Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500}, 2.0)

	tests := []struct {
		name     string
		request  TaskRequest
		expected bool
		reason   string
	}{
		{
			name: "Valid task request",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-1",
				Resources: Resources{CPU: 5, Memory: 50, GPU: 0.5, Disk: 100},
			},
			expected: true,
			reason:   "scheduled",
		},
		{
			name: "Task exceeds tenant quota",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-2",
				Resources: Resources{CPU: 25, Memory: 50, GPU: 0.5, Disk: 100}, // CPU exceeds quota
			},
			expected: false,
			reason:   "quota exceeded",
		},
		{
			name: "Task exceeds cluster capacity",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-3",
				Resources: Resources{CPU: 25, Memory: 50, GPU: 0.5, Disk: 100}, // CPU exceeds tenant quota
			},
			expected: false,
			reason:   "quota exceeded",
		},
		{
			name: "Non-existent tenant",
			request: TaskRequest{
				TenantID: "non-existent",
				TaskID:   "task-4",
				Resources: Resources{CPU: 5, Memory: 50, GPU: 0.5, Disk: 100},
			},
			expected: false,
			reason:   "tenant not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := scheduler.ScheduleTask(tt.request)

			assert.Equal(t, tt.expected, allowed)
			assert.Equal(t, tt.reason, reason)

			if allowed {
				tenant := scheduler.tenants[tt.request.TenantID]
				assert.Equal(t, tt.request.Resources.CPU, tenant.AllocatedCPU)
				assert.Equal(t, tt.request.Resources.Memory, tenant.AllocatedMemory)
				assert.Equal(t, tt.request.Resources.GPU, tenant.AllocatedGPU)
				assert.Equal(t, tt.request.Resources.CPU, tenant.Usage.CPU)
				assert.Equal(t, tt.request.Resources.Memory, tenant.Usage.Memory)
				assert.Equal(t, tt.request.Resources.GPU, tenant.Usage.GPU)
				assert.Equal(t, tt.request.Resources.Disk, tenant.Usage.Disk)
			}
		})
	}
}

func TestDRFScheduler_checkQuota(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	
	tenant := &Tenant{
		ID:    "test-tenant",
		Name:  "Test Tenant",
		Quota: Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000},
		Usage: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
	}

	tests := []struct {
		name      string
		resources Resources
		expected  bool
	}{
		{
			name:      "Within quota",
			resources: Resources{CPU: 5, Memory: 50, GPU: 0.5, Disk: 200},
			expected:  true,
		},
		{
			name:      "At quota limit",
			resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			expected:  true,
		},
		{
			name:      "Exceeds CPU quota",
			resources: Resources{CPU: 15, Memory: 50, GPU: 0.5, Disk: 200},
			expected:  false,
		},
		{
			name:      "Exceeds memory quota",
			resources: Resources{CPU: 5, Memory: 150, GPU: 0.5, Disk: 200},
			expected:  false,
		},
		{
			name:      "Exceeds GPU quota",
			resources: Resources{CPU: 5, Memory: 50, GPU: 1.5, Disk: 200},
			expected:  false,
		},
		{
			name:      "Exceeds disk quota",
			resources: Resources{CPU: 5, Memory: 50, GPU: 0.5, Disk: 600},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.checkQuota(tenant, tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDRFScheduler_checkClusterCapacity(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	// Add some tenants with allocated resources
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)
	scheduler.RegisterTenant("tenant-2", "Tenant Two", Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500}, 1.0)

	// Allocate some resources
	scheduler.tenants["tenant-1"].AllocatedCPU = 40
	scheduler.tenants["tenant-1"].AllocatedMemory = 400
	scheduler.tenants["tenant-1"].AllocatedGPU = 4

	scheduler.tenants["tenant-2"].AllocatedCPU = 20
	scheduler.tenants["tenant-2"].AllocatedMemory = 200
	scheduler.tenants["tenant-2"].AllocatedGPU = 2

	tests := []struct {
		name      string
		resources Resources
		expected  bool
	}{
		{
			name:      "Within cluster capacity",
			resources: Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1000},
			expected:  true,
		},
		{
			name:      "At cluster capacity",
			resources: Resources{CPU: 40, Memory: 400, GPU: 4, Disk: 1000},
			expected:  true,
		},
		{
			name:      "Exceeds CPU capacity",
			resources: Resources{CPU: 45, Memory: 300, GPU: 3, Disk: 1000},
			expected:  false,
		},
		{
			name:      "Exceeds memory capacity",
			resources: Resources{CPU: 30, Memory: 450, GPU: 3, Disk: 1000},
			expected:  false,
		},
		{
			name:      "Exceeds GPU capacity",
			resources: Resources{CPU: 30, Memory: 300, GPU: 5, Disk: 1000},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.checkClusterCapacity(tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDRFScheduler_updateDominantShare(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	tests := []struct {
		name           string
		allocatedCPU   float64
		allocatedMem   float64
		allocatedGPU   float64
		weight         float64
		expectedDominant float64
		expectedWeighted float64
	}{
		{
			name:           "CPU dominant",
			allocatedCPU:   50,
			allocatedMem:   200,
			allocatedGPU:   2,
			weight:         1.0,
			expectedDominant: 0.5, // 50/100
			expectedWeighted: 0.5, // 0.5/1.0
		},
		{
			name:           "Memory dominant",
			allocatedCPU:   20,
			allocatedMem:   800,
			allocatedGPU:   2,
			weight:         1.0,
			expectedDominant: 0.8, // 800/1000
			expectedWeighted: 0.8, // 0.8/1.0
		},
		{
			name:           "GPU dominant",
			allocatedCPU:   20,
			allocatedMem:   200,
			allocatedGPU:   8,
			weight:         1.0,
			expectedDominant: 0.8, // 8/10
			expectedWeighted: 0.8, // 0.8/1.0
		},
		{
			name:           "With custom weight",
			allocatedCPU:   50,
			allocatedMem:   200,
			allocatedGPU:   2,
			weight:         2.0,
			expectedDominant: 0.5, // 50/100
			expectedWeighted: 0.25, // 0.5/2.0
		},
		{
			name:           "Zero GPU allocation",
			allocatedCPU:   30,
			allocatedMem:   300,
			allocatedGPU:   0,
			weight:         1.0,
			expectedDominant: 0.3, // max(0.3, 0.3, 0)
			expectedWeighted: 0.3, // 0.3/1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{
				ID:             "test-tenant",
				Name:           "Test Tenant",
				Weight:         tt.weight,
				AllocatedCPU:   tt.allocatedCPU,
				AllocatedMemory: tt.allocatedMem,
				AllocatedGPU:   tt.allocatedGPU,
			}

			scheduler.updateDominantShare(tenant)

			assert.InDelta(t, tt.expectedDominant, tenant.DominantShare, 0.001)
			assert.InDelta(t, tt.expectedWeighted, tenant.WeightedShare, 0.001)
		})
	}
}

func TestDRFScheduler_GetSchedulingOrder(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	// Register tenants with different weights and allocations
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)
	scheduler.RegisterTenant("tenant-2", "Tenant Two", Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500}, 2.0)
	scheduler.RegisterTenant("tenant-3", "Tenant Three", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)

	// Allocate different amounts to create different weighted shares
	scheduler.tenants["tenant-1"].AllocatedCPU = 20  // dominant share: 0.2, weighted: 0.2
	scheduler.tenants["tenant-1"].AllocatedMemory = 200
	scheduler.tenants["tenant-1"].AllocatedGPU = 2
	scheduler.updateDominantShare(scheduler.tenants["tenant-1"])

	scheduler.tenants["tenant-2"].AllocatedCPU = 30  // dominant share: 0.3, weighted: 0.15
	scheduler.tenants["tenant-2"].AllocatedMemory = 300
	scheduler.tenants["tenant-2"].AllocatedGPU = 3
	scheduler.updateDominantShare(scheduler.tenants["tenant-2"])

	scheduler.tenants["tenant-3"].AllocatedCPU = 10  // dominant share: 0.1, weighted: 0.1
	scheduler.tenants["tenant-3"].AllocatedMemory = 100
	scheduler.tenants["tenant-3"].AllocatedGPU = 1
	scheduler.updateDominantShare(scheduler.tenants["tenant-3"])

	order := scheduler.GetSchedulingOrder()

	// Should be sorted by weighted share (ascending)
	assert.Len(t, order, 3)
	assert.Equal(t, "tenant-3", order[0].ID) // lowest weighted share (0.1)
	assert.Equal(t, "tenant-2", order[1].ID) // middle weighted share (0.15)
	assert.Equal(t, "tenant-1", order[2].ID) // highest weighted share (0.2)
}

func TestDRFScheduler_ReleaseResources(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)

	// Allocate some resources first
	scheduler.tenants["tenant-1"].AllocatedCPU = 20
	scheduler.tenants["tenant-1"].AllocatedMemory = 200
	scheduler.tenants["tenant-1"].AllocatedGPU = 2
	scheduler.tenants["tenant-1"].Usage.CPU = 20
	scheduler.tenants["tenant-1"].Usage.Memory = 200
	scheduler.tenants["tenant-1"].Usage.GPU = 2
	scheduler.tenants["tenant-1"].Usage.Disk = 1000

	tests := []struct {
		name      string
		tenantID  string
		resources Resources
		expectedCPU    float64
		expectedMemory float64
		expectedGPU    float64
		expectedDisk   float64
	}{
		{
			name:      "Release partial resources",
			tenantID:  "tenant-1",
			resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			expectedCPU:    10,
			expectedMemory: 100,
			expectedGPU:    1,
			expectedDisk:   500,
		},
		{
			name:      "Release all resources",
			tenantID:  "tenant-1",
			resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			expectedCPU:    0,
			expectedMemory: 0,
			expectedGPU:    0,
			expectedDisk:   0,
		},
		{
			name:      "Non-existent tenant",
			tenantID:  "non-existent",
			resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			expectedCPU:    0,
			expectedMemory: 0,
			expectedGPU:    0,
			expectedDisk:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scheduler.ReleaseResources(tt.tenantID, tt.resources)
			assert.NoError(t, err)

			if tt.tenantID == "tenant-1" {
				tenant := scheduler.tenants[tt.tenantID]
				assert.Equal(t, tt.expectedCPU, tenant.AllocatedCPU)
				assert.Equal(t, tt.expectedMemory, tenant.AllocatedMemory)
				assert.Equal(t, tt.expectedGPU, tenant.AllocatedGPU)
				assert.Equal(t, tt.expectedDisk, tenant.Usage.Disk)
			}
		})
	}
}

func TestDRFScheduler_ReleaseResources_NegativeValues(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)

	// Allocate some resources
	scheduler.tenants["tenant-1"].AllocatedCPU = 10
	scheduler.tenants["tenant-1"].AllocatedMemory = 100
	scheduler.tenants["tenant-1"].AllocatedGPU = 1
	scheduler.tenants["tenant-1"].Usage.CPU = 10
	scheduler.tenants["tenant-1"].Usage.Memory = 100
	scheduler.tenants["tenant-1"].Usage.GPU = 1
	scheduler.tenants["tenant-1"].Usage.Disk = 500

	// Try to release more than allocated
	err := scheduler.ReleaseResources("tenant-1", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000})
	assert.NoError(t, err)

	tenant := scheduler.tenants["tenant-1"]
	assert.Equal(t, 0.0, tenant.AllocatedCPU)
	assert.Equal(t, 0.0, tenant.AllocatedMemory)
	assert.Equal(t, 0.0, tenant.AllocatedGPU)
	assert.Equal(t, -10.0, tenant.Usage.CPU)
	assert.Equal(t, -100.0, tenant.Usage.Memory)
	assert.Equal(t, -1.0, tenant.Usage.GPU)
	assert.Equal(t, -500.0, tenant.Usage.Disk)
}

func TestDRFScheduler_GetTenantStats(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 2.0)

	// Allocate some resources
	scheduler.tenants["tenant-1"].AllocatedCPU = 25
	scheduler.tenants["tenant-1"].AllocatedMemory = 250
	scheduler.tenants["tenant-1"].AllocatedGPU = 2.5
	scheduler.tenants["tenant-1"].Usage.CPU = 25
	scheduler.tenants["tenant-1"].Usage.Memory = 250
	scheduler.tenants["tenant-1"].Usage.GPU = 2.5
	scheduler.tenants["tenant-1"].Usage.Disk = 1250
	scheduler.updateDominantShare(scheduler.tenants["tenant-1"])

	tests := []struct {
		name     string
		tenantID string
		expected *TenantStats
	}{
		{
			name:     "Valid tenant",
			tenantID: "tenant-1",
			expected: &TenantStats{
				TenantID:          "tenant-1",
				TenantName:        "Tenant One",
				DominantShare:     0.25, // 25/100
				WeightedShare:    0.125, // 0.25/2.0
				CPUUtilization:   0.5, // 25/50
				MemoryUtilization: 0.5, // 250/500
				GPUUtilization:   0.5, // 2.5/5
				QuotaRemaining: Resources{
					CPU:    25, // 50-25
					Memory: 250, // 500-250
					GPU:    2.5, // 5-2.5
					Disk:   1250, // 2500-1250
				},
			},
		},
		{
			name:     "Non-existent tenant",
			tenantID: "non-existent",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := scheduler.GetTenantStats(tt.tenantID)

			if tt.expected == nil {
				assert.NoError(t, err)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Equal(t, tt.expected.TenantID, stats.TenantID)
				assert.Equal(t, tt.expected.TenantName, stats.TenantName)
				assert.InDelta(t, tt.expected.DominantShare, stats.DominantShare, 0.001)
				assert.InDelta(t, tt.expected.WeightedShare, stats.WeightedShare, 0.001)
				assert.InDelta(t, tt.expected.CPUUtilization, stats.CPUUtilization, 0.001)
				assert.InDelta(t, tt.expected.MemoryUtilization, stats.MemoryUtilization, 0.001)
				assert.InDelta(t, tt.expected.GPUUtilization, stats.GPUUtilization, 0.001)
				assert.Equal(t, tt.expected.QuotaRemaining, stats.QuotaRemaining)
			}
		})
	}
}

func TestDRFScheduler_GetClusterUtilization(t *testing.T) {
	clusterTotal := Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}
	scheduler := NewDRFScheduler(clusterTotal)

	// Register tenants
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)
	scheduler.RegisterTenant("tenant-2", "Tenant Two", Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500}, 1.0)

	// Allocate resources
	scheduler.tenants["tenant-1"].AllocatedCPU = 40
	scheduler.tenants["tenant-1"].AllocatedMemory = 400
	scheduler.tenants["tenant-1"].AllocatedGPU = 4

	scheduler.tenants["tenant-2"].AllocatedCPU = 20
	scheduler.tenants["tenant-2"].AllocatedMemory = 200
	scheduler.tenants["tenant-2"].AllocatedGPU = 2

	utilization := scheduler.GetClusterUtilization()

	expectedCPU := 60.0 / 100.0    // (40+20)/100
	expectedMemory := 600.0 / 1000.0 // (400+200)/1000
	expectedGPU := 6.0 / 10.0     // (4+2)/10

	assert.InDelta(t, expectedCPU, utilization.CPUUtilization, 0.001)
	assert.InDelta(t, expectedMemory, utilization.MemoryUtilization, 0.001)
	assert.InDelta(t, expectedGPU, utilization.GPUUtilization, 0.001)
	assert.Equal(t, 2, utilization.TotalTenants)
}

func TestDRFScheduler_ConcurrentAccess(t *testing.T) {
	clusterTotal := Resources{CPU: 1000, Memory: 10000, GPU: 100, Disk: 50000}
	scheduler := NewDRFScheduler(clusterTotal)

	// Register multiple tenants
	for i := 0; i < 10; i++ {
		scheduler.RegisterTenant(
			fmt.Sprintf("tenant-%d", i),
			fmt.Sprintf("Tenant %d", i),
			Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000},
			1.0,
		)
	}

	// Test concurrent scheduling
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(tenantID string) {
			defer func() { done <- true }()
			
			request := TaskRequest{
				TenantID: tenantID,
				TaskID:   fmt.Sprintf("task-%s", tenantID),
				Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			}
			
			allowed, reason := scheduler.ScheduleTask(request)
			assert.True(t, allowed, "Task should be scheduled for %s: %s", tenantID, reason)
		}(fmt.Sprintf("tenant-%d", i))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all tenants have allocated resources
	for i := 0; i < 10; i++ {
		tenantID := fmt.Sprintf("tenant-%d", i)
		tenant := scheduler.tenants[tenantID]
		assert.Equal(t, 10.0, tenant.AllocatedCPU)
		assert.Equal(t, 100.0, tenant.AllocatedMemory)
		assert.Equal(t, 1.0, tenant.AllocatedGPU)
	}
}

func TestDRFScheduler_EdgeCases(t *testing.T) {
	t.Run("Empty cluster", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{})
		
		order := scheduler.GetSchedulingOrder()
		assert.Empty(t, order)
		
		utilization := scheduler.GetClusterUtilization()
		// With zero cluster resources, utilization should be 0 or NaN
		assert.True(t, utilization.CPUUtilization == 0.0 || math.IsNaN(utilization.CPUUtilization))
		assert.True(t, utilization.MemoryUtilization == 0.0 || math.IsNaN(utilization.MemoryUtilization))
		assert.True(t, utilization.GPUUtilization == 0.0 || math.IsNaN(utilization.GPUUtilization))
		assert.Equal(t, 0, utilization.TotalTenants)
	})

	t.Run("Zero resource request", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
		scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)

		request := TaskRequest{
			TenantID: "tenant-1",
			TaskID:   "task-1",
			Resources: Resources{}, // Zero resources
		}

		allowed, reason := scheduler.ScheduleTask(request)
		assert.True(t, allowed)
		assert.Equal(t, "scheduled", reason)
	})

	t.Run("Very small resource values", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{CPU: 0.001, Memory: 0.001, GPU: 0.001, Disk: 0.001})
		scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 0.0005, Memory: 0.0005, GPU: 0.0005, Disk: 0.0005}, 1.0)

		request := TaskRequest{
			TenantID: "tenant-1",
			TaskID:   "task-1",
			Resources: Resources{CPU: 0.0001, Memory: 0.0001, GPU: 0.0001, Disk: 0.0001},
		}

		allowed, reason := scheduler.ScheduleTask(request)
		assert.True(t, allowed)
		assert.Equal(t, "scheduled", reason)
	})
}
