package scheduler

import (
	"math"
	"sort"
	"sync"
)

// DRFScheduler implements Dominant Resource Fairness algorithm
// Ensures fair resource allocation across multiple tenants in multi-resource environments
type DRFScheduler struct {
	tenants      map[string]*Tenant
	clusterTotal Resources
	mu           sync.RWMutex
}

// Tenant represents a single tenant with resource allocation
type Tenant struct {
	ID               string
	Name             string
	Weight           float64 // Weight for Weighted DRF (default: 1.0)
	AllocatedCPU     float64
	AllocatedMemory  float64
	AllocatedGPU     float64
	DominantShare    float64
	WeightedShare    float64
	Quota            Resources
	Usage            Resources
}

// Resources represents multi-dimensional resources
type Resources struct {
	CPU    float64 // cores
	Memory float64 // GB
	GPU    float64 // count
	Disk   float64 // GB
}

// TaskRequest represents a resource request for scheduling
type TaskRequest struct {
	TenantID  string
	TaskID    string
	Resources Resources
}

// NewDRFScheduler creates a new DRF scheduler
func NewDRFScheduler(clusterTotal Resources) *DRFScheduler {
	return &DRFScheduler{
		tenants:      make(map[string]*Tenant),
		clusterTotal: clusterTotal,
	}
}

// RegisterTenant registers a new tenant with quota and weight
func (d *DRFScheduler) RegisterTenant(id, name string, quota Resources, weight float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if weight <= 0 {
		weight = 1.0
	}

	d.tenants[id] = &Tenant{
		ID:               id,
		Name:             name,
		Weight:           weight,
		AllocatedCPU:     0,
		AllocatedMemory:  0,
		AllocatedGPU:     0,
		DominantShare:    0,
		WeightedShare:    0,
		Quota:            quota,
		Usage:            Resources{},
	}
}

// ScheduleTask schedules a task using Weighted DRF algorithm
func (d *DRFScheduler) ScheduleTask(request TaskRequest) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tenant, exists := d.tenants[request.TenantID]
	if !exists {
		return false, "tenant not found"
	}

	// Check if tenant has quota available
	if !d.checkQuota(tenant, request.Resources) {
		return false, "quota exceeded"
	}

	// Check if cluster has resources available
	if !d.checkClusterCapacity(request.Resources) {
		return false, "insufficient cluster resources"
	}

	// Allocate resources
	tenant.AllocatedCPU += request.Resources.CPU
	tenant.AllocatedMemory += request.Resources.Memory
	tenant.AllocatedGPU += request.Resources.GPU
	tenant.Usage.CPU += request.Resources.CPU
	tenant.Usage.Memory += request.Resources.Memory
	tenant.Usage.GPU += request.Resources.GPU
	tenant.Usage.Disk += request.Resources.Disk

	// Update dominant share
	d.updateDominantShare(tenant)

	return true, "scheduled"
}

// checkQuota validates tenant quota limits
func (d *DRFScheduler) checkQuota(tenant *Tenant, resources Resources) bool {
	// Check hard quota limits
	if tenant.Usage.CPU+resources.CPU > tenant.Quota.CPU {
		return false
	}
	if tenant.Usage.Memory+resources.Memory > tenant.Quota.Memory {
		return false
	}
	if tenant.Usage.GPU+resources.GPU > tenant.Quota.GPU {
		return false
	}
	if tenant.Usage.Disk+resources.Disk > tenant.Quota.Disk {
		return false
	}
	return true
}

// checkClusterCapacity validates cluster-wide resource availability
func (d *DRFScheduler) checkClusterCapacity(resources Resources) bool {
	totalAllocatedCPU := 0.0
	totalAllocatedMemory := 0.0
	totalAllocatedGPU := 0.0

	for _, tenant := range d.tenants {
		totalAllocatedCPU += tenant.AllocatedCPU
		totalAllocatedMemory += tenant.AllocatedMemory
		totalAllocatedGPU += tenant.AllocatedGPU
	}

	// Check if cluster has capacity
	if totalAllocatedCPU+resources.CPU > d.clusterTotal.CPU {
		return false
	}
	if totalAllocatedMemory+resources.Memory > d.clusterTotal.Memory {
		return false
	}
	if totalAllocatedGPU+resources.GPU > d.clusterTotal.GPU {
		return false
	}

	return true
}

// updateDominantShare calculates the dominant share for a tenant
func (d *DRFScheduler) updateDominantShare(tenant *Tenant) {
	cpuShare := tenant.AllocatedCPU / d.clusterTotal.CPU
	memoryShare := tenant.AllocatedMemory / d.clusterTotal.Memory
	gpuShare := 0.0
	if d.clusterTotal.GPU > 0 {
		gpuShare = tenant.AllocatedGPU / d.clusterTotal.GPU
	}

	// Dominant share is the maximum of all resource shares
	tenant.DominantShare = math.Max(cpuShare, math.Max(memoryShare, gpuShare))

	// Weighted share for Weighted DRF
	tenant.WeightedShare = tenant.DominantShare / tenant.Weight
}

// GetSchedulingOrder returns tenants sorted by weighted dominant share (ascending)
// Lower weighted share = higher priority for next allocation
func (d *DRFScheduler) GetSchedulingOrder() []*Tenant {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tenantList := make([]*Tenant, 0, len(d.tenants))
	for _, tenant := range d.tenants {
		tenantList = append(tenantList, tenant)
	}

	// Sort by weighted share (ascending) - fairness guarantee
	sort.Slice(tenantList, func(i, j int) bool {
		return tenantList[i].WeightedShare < tenantList[j].WeightedShare
	})

	return tenantList
}

// ReleaseResources releases resources when a task completes
func (d *DRFScheduler) ReleaseResources(tenantID string, resources Resources) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tenant, exists := d.tenants[tenantID]
	if !exists {
		return nil
	}

	tenant.AllocatedCPU -= resources.CPU
	tenant.AllocatedMemory -= resources.Memory
	tenant.AllocatedGPU -= resources.GPU
	tenant.Usage.CPU -= resources.CPU
	tenant.Usage.Memory -= resources.Memory
	tenant.Usage.GPU -= resources.GPU
	tenant.Usage.Disk -= resources.Disk

	// Ensure non-negative
	if tenant.AllocatedCPU < 0 {
		tenant.AllocatedCPU = 0
	}
	if tenant.AllocatedMemory < 0 {
		tenant.AllocatedMemory = 0
	}
	if tenant.AllocatedGPU < 0 {
		tenant.AllocatedGPU = 0
	}

	// Update dominant share
	d.updateDominantShare(tenant)

	return nil
}

// GetTenantStats returns current allocation stats for a tenant
func (d *DRFScheduler) GetTenantStats(tenantID string) (*TenantStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tenant, exists := d.tenants[tenantID]
	if !exists {
		return nil, nil
	}

	return &TenantStats{
		TenantID:         tenant.ID,
		TenantName:       tenant.Name,
		DominantShare:    tenant.DominantShare,
		WeightedShare:    tenant.WeightedShare,
		CPUUtilization:   tenant.Usage.CPU / tenant.Quota.CPU,
		MemoryUtilization: tenant.Usage.Memory / tenant.Quota.Memory,
		GPUUtilization:   tenant.Usage.GPU / tenant.Quota.GPU,
		QuotaRemaining: Resources{
			CPU:    tenant.Quota.CPU - tenant.Usage.CPU,
			Memory: tenant.Quota.Memory - tenant.Usage.Memory,
			GPU:    tenant.Quota.GPU - tenant.Usage.GPU,
			Disk:   tenant.Quota.Disk - tenant.Usage.Disk,
		},
	}, nil
}

// TenantStats represents tenant resource usage statistics
type TenantStats struct {
	TenantID          string
	TenantName        string
	DominantShare     float64
	WeightedShare     float64
	CPUUtilization    float64
	MemoryUtilization float64
	GPUUtilization    float64
	QuotaRemaining    Resources
}

// GetClusterUtilization returns overall cluster resource utilization
func (d *DRFScheduler) GetClusterUtilization() ClusterUtilization {
	d.mu.RLock()
	defer d.mu.RUnlock()

	totalAllocatedCPU := 0.0
	totalAllocatedMemory := 0.0
	totalAllocatedGPU := 0.0

	for _, tenant := range d.tenants {
		totalAllocatedCPU += tenant.AllocatedCPU
		totalAllocatedMemory += tenant.AllocatedMemory
		totalAllocatedGPU += tenant.AllocatedGPU
	}

	return ClusterUtilization{
		CPUUtilization:    totalAllocatedCPU / d.clusterTotal.CPU,
		MemoryUtilization: totalAllocatedMemory / d.clusterTotal.Memory,
		GPUUtilization:    totalAllocatedGPU / d.clusterTotal.GPU,
		TotalTenants:      len(d.tenants),
	}
}

// ClusterUtilization represents cluster-wide resource utilization
type ClusterUtilization struct {
	CPUUtilization    float64
	MemoryUtilization float64
	GPUUtilization    float64
	TotalTenants      int
}
