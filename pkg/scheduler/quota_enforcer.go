package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// QuotaEnforcer enforces resource quotas and prevents oversubscription
type QuotaEnforcer struct {
	scheduler        *DRFScheduler
	quotaViolations  map[string]int
	violationsMux    sync.RWMutex
	enforcementMode  EnforcementMode
	gracePeriod      time.Duration
	preemptionPolicy PreemptionPolicy
}

// EnforcementMode defines how strictly quotas are enforced
type EnforcementMode string

const (
	HardEnforcement EnforcementMode = "hard" // Reject tasks immediately
	SoftEnforcement EnforcementMode = "soft" // Allow temporary oversubscription
	AdaptiveMode    EnforcementMode = "adaptive" // Dynamic based on cluster load
)

// PreemptionPolicy defines task preemption behavior
type PreemptionPolicy string

const (
	PreemptNever       PreemptionPolicy = "never"
	PreemptLowPriority PreemptionPolicy = "low-priority"
	PreemptOldest      PreemptionPolicy = "oldest"
)

// QuotaEnforcementResult represents enforcement decision
type QuotaEnforcementResult struct {
	Allowed        bool
	Reason         string
	SuggestedWait  time.Duration
	PreemptedTasks []string
}

// NewQuotaEnforcer creates a new quota enforcer
func NewQuotaEnforcer(scheduler *DRFScheduler, mode EnforcementMode, policy PreemptionPolicy) *QuotaEnforcer {
	return &QuotaEnforcer{
		scheduler:        scheduler,
		quotaViolations:  make(map[string]int),
		enforcementMode:  mode,
		gracePeriod:      5 * time.Minute,
		preemptionPolicy: policy,
	}
}

// EnforceQuota checks if a task can be scheduled within quota limits
func (qe *QuotaEnforcer) EnforceQuota(ctx context.Context, request TaskRequest) QuotaEnforcementResult {
	// Check tenant quota
	stats, err := qe.scheduler.GetTenantStats(request.TenantID)
	if err != nil {
		return QuotaEnforcementResult{
			Allowed: false,
			Reason:  fmt.Sprintf("failed to get tenant stats: %v", err),
		}
	}

	if stats == nil {
		return QuotaEnforcementResult{
			Allowed: false,
			Reason:  "tenant not found",
		}
	}

	// Check if quota would be exceeded
	wouldExceedQuota := false
	var exceedReason string

	if stats.QuotaRemaining.CPU < request.Resources.CPU {
		wouldExceedQuota = true
		exceedReason = fmt.Sprintf("CPU quota exceeded (available: %.2f, requested: %.2f)",
			stats.QuotaRemaining.CPU, request.Resources.CPU)
	}
	if stats.QuotaRemaining.Memory < request.Resources.Memory {
		wouldExceedQuota = true
		exceedReason = fmt.Sprintf("Memory quota exceeded (available: %.2f GB, requested: %.2f GB)",
			stats.QuotaRemaining.Memory, request.Resources.Memory)
	}
	if stats.QuotaRemaining.GPU < request.Resources.GPU {
		wouldExceedQuota = true
		exceedReason = fmt.Sprintf("GPU quota exceeded (available: %.2f, requested: %.2f)",
			stats.QuotaRemaining.GPU, request.Resources.GPU)
	}

	// Apply enforcement policy
	switch qe.enforcementMode {
	case HardEnforcement:
		if wouldExceedQuota {
			qe.recordViolation(request.TenantID)
			return QuotaEnforcementResult{
				Allowed:       false,
				Reason:        exceedReason,
				SuggestedWait: qe.estimateWaitTime(request),
			}
		}

	case SoftEnforcement:
		if wouldExceedQuota {
			log.Printf("Soft quota violation for tenant %s: %s", request.TenantID, exceedReason)
			qe.recordViolation(request.TenantID)
			// Allow anyway in soft mode
		}

	case AdaptiveMode:
		clusterUtil := qe.scheduler.GetClusterUtilization()
		// If cluster is under-utilized, allow oversubscription
		if wouldExceedQuota {
			if clusterUtil.CPUUtilization < 0.7 && clusterUtil.MemoryUtilization < 0.7 {
				log.Printf("Allowing oversubscription for tenant %s due to low cluster utilization", request.TenantID)
			} else {
				return QuotaEnforcementResult{
					Allowed:       false,
					Reason:        exceedReason + " (adaptive mode, cluster at high utilization)",
					SuggestedWait: qe.estimateWaitTime(request),
				}
			}
		}
	}

	// Check if preemption is needed
	if wouldExceedQuota && qe.preemptionPolicy != PreemptNever {
		preemptedTasks := qe.attemptPreemption(request)
		if len(preemptedTasks) > 0 {
			return QuotaEnforcementResult{
				Allowed:        true,
				Reason:         "scheduled after preemption",
				PreemptedTasks: preemptedTasks,
			}
		}
	}

	return QuotaEnforcementResult{
		Allowed: true,
		Reason:  "within quota limits",
	}
}

// recordViolation records a quota violation for a tenant
func (qe *QuotaEnforcer) recordViolation(tenantID string) {
	qe.violationsMux.Lock()
	defer qe.violationsMux.Unlock()
	qe.quotaViolations[tenantID]++
}

// GetViolationCount returns the number of quota violations for a tenant
func (qe *QuotaEnforcer) GetViolationCount(tenantID string) int {
	qe.violationsMux.RLock()
	defer qe.violationsMux.RUnlock()
	return qe.quotaViolations[tenantID]
}

// estimateWaitTime estimates how long until resources become available
func (qe *QuotaEnforcer) estimateWaitTime(request TaskRequest) time.Duration {
	// Simple estimation: assume average task duration is 10 minutes
	// In production, this should use historical task duration data
	return 10 * time.Minute
}

// attemptPreemption tries to preempt lower-priority tasks to make room
func (qe *QuotaEnforcer) attemptPreemption(request TaskRequest) []string {
	// TODO: Implement actual preemption logic
	// This would involve:
	// 1. Finding running tasks from the same tenant
	// 2. Selecting tasks based on preemption policy
	// 3. Gracefully terminating those tasks
	// 4. Returning list of preempted task IDs
	return []string{}
}

// MonitorQuotas monitors and logs quota usage periodically
func (qe *QuotaEnforcer) MonitorQuotas(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			qe.logQuotaStatus()
		}
	}
}

// logQuotaStatus logs current quota status for all tenants
func (qe *QuotaEnforcer) logQuotaStatus() {
	order := qe.scheduler.GetSchedulingOrder()

	log.Println("=== Quota Status Report ===")
	for _, tenant := range order {
		stats, err := qe.scheduler.GetTenantStats(tenant.ID)
		if err != nil {
			continue
		}

		log.Printf("Tenant: %s | CPU: %.1f%% | Memory: %.1f%% | GPU: %.1f%% | Dominant Share: %.3f | Violations: %d",
			stats.TenantName,
			stats.CPUUtilization*100,
			stats.MemoryUtilization*100,
			stats.GPUUtilization*100,
			stats.DominantShare,
			qe.GetViolationCount(tenant.ID),
		)
	}

	clusterUtil := qe.scheduler.GetClusterUtilization()
	log.Printf("Cluster Utilization: CPU: %.1f%%, Memory: %.1f%%, GPU: %.1f%%",
		clusterUtil.CPUUtilization*100,
		clusterUtil.MemoryUtilization*100,
		clusterUtil.GPUUtilization*100,
	)
}

// SetEnforcementMode changes the enforcement mode dynamically
func (qe *QuotaEnforcer) SetEnforcementMode(mode EnforcementMode) {
	qe.enforcementMode = mode
	log.Printf("Quota enforcement mode changed to: %s", mode)
}

// ResetViolations clears violation history for a tenant
func (qe *QuotaEnforcer) ResetViolations(tenantID string) {
	qe.violationsMux.Lock()
	defer qe.violationsMux.Unlock()
	delete(qe.quotaViolations, tenantID)
}
