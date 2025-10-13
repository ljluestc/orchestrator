package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewQuotaEnforcer(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	
	tests := []struct {
		name   string
		mode   EnforcementMode
		policy PreemptionPolicy
	}{
		{
			name:   "Hard enforcement with no preemption",
			mode:   HardEnforcement,
			policy: PreemptNever,
		},
		{
			name:   "Soft enforcement with low priority preemption",
			mode:   SoftEnforcement,
			policy: PreemptLowPriority,
		},
		{
			name:   "Adaptive enforcement with oldest preemption",
			mode:   AdaptiveMode,
			policy: PreemptOldest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enforcer := NewQuotaEnforcer(scheduler, tt.mode, tt.policy)

			assert.NotNil(t, enforcer)
			assert.Equal(t, scheduler, enforcer.scheduler)
			assert.Equal(t, tt.mode, enforcer.enforcementMode)
			assert.Equal(t, tt.policy, enforcer.preemptionPolicy)
			assert.Equal(t, 5*time.Minute, enforcer.gracePeriod)
			assert.NotNil(t, enforcer.quotaViolations)
			assert.Empty(t, enforcer.quotaViolations)
		})
	}
}

func TestQuotaEnforcer_EnforceQuota_HardEnforcement(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Register tenant with limited quota
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)

	tests := []struct {
		name     string
		request  TaskRequest
		expected bool
		reason   string
	}{
		{
			name: "Within quota limits",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-1",
				Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			},
			expected: true,
			reason:   "within quota limits",
		},
		{
			name: "At quota limit",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-2",
				Resources: Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000},
			},
			expected: true,
			reason:   "within quota limits",
		},
		{
			name: "Exceeds CPU quota",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-3",
				Resources: Resources{CPU: 25, Memory: 100, GPU: 1, Disk: 500},
			},
			expected: false,
			reason:   "CPU quota exceeded",
		},
		{
			name: "Exceeds memory quota",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-4",
				Resources: Resources{CPU: 10, Memory: 250, GPU: 1, Disk: 500},
			},
			expected: false,
			reason:   "Memory quota exceeded",
		},
		{
			name: "Non-existent tenant",
			request: TaskRequest{
				TenantID: "non-existent",
				TaskID:   "task-5",
				Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			},
			expected: false,
			reason:   "tenant not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := enforcer.EnforceQuota(ctx, tt.request)

			assert.Equal(t, tt.expected, result.Allowed)
			assert.Contains(t, result.Reason, tt.reason)

			if !tt.expected && tt.request.TenantID == "tenant-1" {
				assert.Greater(t, result.SuggestedWait, time.Duration(0))
			}
		})
	}
}

func TestQuotaEnforcer_EnforceQuota_SoftEnforcement(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, SoftEnforcement, PreemptNever)

	// Register tenant with limited quota
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)

	tests := []struct {
		name     string
		request  TaskRequest
		expected bool
	}{
		{
			name: "Within quota limits",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-1",
				Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			},
			expected: true,
		},
		{
			name: "Exceeds quota but allowed in soft mode",
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-2",
				Resources: Resources{CPU: 25, Memory: 250, GPU: 3, Disk: 1500},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := enforcer.EnforceQuota(ctx, tt.request)

			assert.Equal(t, tt.expected, result.Allowed)
			assert.Equal(t, "within quota limits", result.Reason)
		})
	}
}

func TestQuotaEnforcer_EnforceQuota_AdaptiveMode(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, AdaptiveMode, PreemptNever)

	// Register tenant with limited quota
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)

	tests := []struct {
		name           string
		clusterUtil    ClusterUtilization
		request        TaskRequest
		expected       bool
		expectedReason string
	}{
		{
			name: "Low cluster utilization - allow oversubscription",
			clusterUtil: ClusterUtilization{
				CPUUtilization:    0.5,
				MemoryUtilization: 0.6,
				GPUUtilization:    0.4,
				TotalTenants:      1,
			},
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-1",
				Resources: Resources{CPU: 25, Memory: 250, GPU: 3, Disk: 1500},
			},
			expected:       true,
			expectedReason: "within quota limits",
		},
		{
			name: "High cluster utilization - reject oversubscription",
			clusterUtil: ClusterUtilization{
				CPUUtilization:    0.8,
				MemoryUtilization: 0.9,
				GPUUtilization:    0.7,
				TotalTenants:      1,
			},
			request: TaskRequest{
				TenantID: "tenant-1",
				TaskID:   "task-2",
				Resources: Resources{CPU: 25, Memory: 250, GPU: 3, Disk: 1500},
			},
			expected:       false,
			expectedReason: "adaptive mode, cluster at high utilization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock cluster utilization by allocating resources
			if tt.clusterUtil.CPUUtilization > 0.7 {
				scheduler.RegisterTenant("high-util-tenant", "High Util Tenant", Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000}, 1.0)
				scheduler.tenants["high-util-tenant"].AllocatedCPU = tt.clusterUtil.CPUUtilization * 100
				scheduler.tenants["high-util-tenant"].AllocatedMemory = tt.clusterUtil.MemoryUtilization * 1000
				scheduler.tenants["high-util-tenant"].AllocatedGPU = tt.clusterUtil.GPUUtilization * 10
			}

			ctx := context.Background()
			result := enforcer.EnforceQuota(ctx, tt.request)

			assert.Equal(t, tt.expected, result.Allowed)
			assert.Contains(t, result.Reason, tt.expectedReason)
		})
	}
}

func TestQuotaEnforcer_recordViolation(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Record violations for different tenants
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-2")
	enforcer.recordViolation("tenant-1")

	assert.Equal(t, 3, enforcer.GetViolationCount("tenant-1"))
	assert.Equal(t, 1, enforcer.GetViolationCount("tenant-2"))
	assert.Equal(t, 0, enforcer.GetViolationCount("tenant-3"))
}

func TestQuotaEnforcer_GetViolationCount(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Test non-existent tenant
	assert.Equal(t, 0, enforcer.GetViolationCount("non-existent"))

	// Record some violations
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-2")

	assert.Equal(t, 2, enforcer.GetViolationCount("tenant-1"))
	assert.Equal(t, 1, enforcer.GetViolationCount("tenant-2"))
}

func TestQuotaEnforcer_estimateWaitTime(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	request := TaskRequest{
		TenantID: "tenant-1",
		TaskID:   "task-1",
		Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
	}

	waitTime := enforcer.estimateWaitTime(request)
	assert.Equal(t, 10*time.Minute, waitTime)
}

func TestQuotaEnforcer_attemptPreemption(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptLowPriority)

	request := TaskRequest{
		TenantID: "tenant-1",
		TaskID:   "task-1",
		Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
	}

	preemptedTasks := enforcer.attemptPreemption(request)
	assert.Empty(t, preemptedTasks) // TODO implementation returns empty slice
}

func TestQuotaEnforcer_MonitorQuotas(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Register some tenants
	scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 20, Memory: 200, GPU: 2, Disk: 1000}, 1.0)
	scheduler.RegisterTenant("tenant-2", "Tenant Two", Resources{CPU: 30, Memory: 300, GPU: 3, Disk: 1500}, 2.0)

	// Allocate some resources
	scheduler.tenants["tenant-1"].AllocatedCPU = 10
	scheduler.tenants["tenant-1"].AllocatedMemory = 100
	scheduler.tenants["tenant-1"].AllocatedGPU = 1
	scheduler.updateDominantShare(scheduler.tenants["tenant-1"])

	scheduler.tenants["tenant-2"].AllocatedCPU = 15
	scheduler.tenants["tenant-2"].AllocatedMemory = 150
	scheduler.tenants["tenant-2"].AllocatedGPU = 1.5
	scheduler.updateDominantShare(scheduler.tenants["tenant-2"])

	// Record some violations
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-1")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start monitoring with short interval
	go enforcer.MonitorQuotas(ctx, 50*time.Millisecond)

	// Wait for context to be done
	<-ctx.Done()

	// The function should have logged quota status at least once
	// We can't easily test the log output, but we can verify the function doesn't panic
}

func TestQuotaEnforcer_SetEnforcementMode(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Test mode changes
	enforcer.SetEnforcementMode(SoftEnforcement)
	assert.Equal(t, SoftEnforcement, enforcer.enforcementMode)

	enforcer.SetEnforcementMode(AdaptiveMode)
	assert.Equal(t, AdaptiveMode, enforcer.enforcementMode)

	enforcer.SetEnforcementMode(HardEnforcement)
	assert.Equal(t, HardEnforcement, enforcer.enforcementMode)
}

func TestQuotaEnforcer_ResetViolations(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Record some violations
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-1")
	enforcer.recordViolation("tenant-2")

	assert.Equal(t, 2, enforcer.GetViolationCount("tenant-1"))
	assert.Equal(t, 1, enforcer.GetViolationCount("tenant-2"))

	// Reset violations for tenant-1
	enforcer.ResetViolations("tenant-1")

	assert.Equal(t, 0, enforcer.GetViolationCount("tenant-1"))
	assert.Equal(t, 1, enforcer.GetViolationCount("tenant-2"))

	// Reset violations for non-existent tenant
	enforcer.ResetViolations("non-existent")
	assert.Equal(t, 0, enforcer.GetViolationCount("non-existent"))
}

func TestQuotaEnforcer_ConcurrentAccess(t *testing.T) {
	scheduler := NewDRFScheduler(Resources{CPU: 1000, Memory: 10000, GPU: 100, Disk: 50000})
	enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

	// Register multiple tenants
	for i := 0; i < 10; i++ {
		scheduler.RegisterTenant(
			fmt.Sprintf("tenant-%d", i),
			fmt.Sprintf("Tenant %d", i),
			Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000},
			1.0,
		)
	}

	// Test concurrent quota enforcement
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(tenantID string) {
			defer func() { done <- true }()
			
			request := TaskRequest{
				TenantID: tenantID,
				TaskID:   fmt.Sprintf("task-%s", tenantID),
				Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
			}
			
			ctx := context.Background()
			result := enforcer.EnforceQuota(ctx, request)
			assert.True(t, result.Allowed, "Quota enforcement should allow task for %s: %s", tenantID, result.Reason)
		}(fmt.Sprintf("tenant-%d", i))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestQuotaEnforcer_EdgeCases(t *testing.T) {
	t.Run("Empty scheduler", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{})
		enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)

		request := TaskRequest{
			TenantID: "non-existent",
			TaskID:   "task-1",
			Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
		}

		ctx := context.Background()
		result := enforcer.EnforceQuota(ctx, request)
		assert.False(t, result.Allowed)
		assert.Equal(t, "tenant not found", result.Reason)
	})

	t.Run("Zero resource request", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
		enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)
		scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)

		request := TaskRequest{
			TenantID: "tenant-1",
			TaskID:   "task-1",
			Resources: Resources{}, // Zero resources
		}

		ctx := context.Background()
		result := enforcer.EnforceQuota(ctx, request)
		assert.True(t, result.Allowed)
		assert.Equal(t, "within quota limits", result.Reason)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		scheduler := NewDRFScheduler(Resources{CPU: 100, Memory: 1000, GPU: 10, Disk: 5000})
		enforcer := NewQuotaEnforcer(scheduler, HardEnforcement, PreemptNever)
		scheduler.RegisterTenant("tenant-1", "Tenant One", Resources{CPU: 50, Memory: 500, GPU: 5, Disk: 2500}, 1.0)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		request := TaskRequest{
			TenantID: "tenant-1",
			TaskID:   "task-1",
			Resources: Resources{CPU: 10, Memory: 100, GPU: 1, Disk: 500},
		}

		result := enforcer.EnforceQuota(ctx, request)
		assert.True(t, result.Allowed) // Context cancellation doesn't affect quota enforcement
	})
}

func TestEnforcementMode_Constants(t *testing.T) {
	assert.Equal(t, EnforcementMode("hard"), HardEnforcement)
	assert.Equal(t, EnforcementMode("soft"), SoftEnforcement)
	assert.Equal(t, EnforcementMode("adaptive"), AdaptiveMode)
}

func TestPreemptionPolicy_Constants(t *testing.T) {
	assert.Equal(t, PreemptionPolicy("never"), PreemptNever)
	assert.Equal(t, PreemptionPolicy("low-priority"), PreemptLowPriority)
	assert.Equal(t, PreemptionPolicy("oldest"), PreemptOldest)
}
