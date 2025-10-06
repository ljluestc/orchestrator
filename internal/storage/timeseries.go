package storage

import (
	"sync"
	"time"

	"github.com/ljluestc/orchestrator/pkg/probe"
)

// TimeSeriesPoint represents a single data point with timestamp
type TimeSeriesPoint struct {
	Timestamp time.Time
	Report    *probe.ReportData
}

// TimeSeriesData stores time-series data for a specific agent
type TimeSeriesData struct {
	AgentID    string
	Points     []TimeSeriesPoint
	mu         sync.RWMutex
	maxAge     time.Duration
	resolution time.Duration
}

// TimeSeriesStore manages time-series data for all agents
type TimeSeriesStore struct {
	data       map[string]*TimeSeriesData
	mu         sync.RWMutex
	maxAge     time.Duration
	resolution time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewTimeSeriesStore creates a new time-series storage with 15-second resolution
func NewTimeSeriesStore(maxAge time.Duration) *TimeSeriesStore {
	if maxAge == 0 {
		maxAge = 1 * time.Hour // Default to 1 hour retention
	}

	store := &TimeSeriesStore{
		data:       make(map[string]*TimeSeriesData),
		maxAge:     maxAge,
		resolution: 15 * time.Second, // 15-second resolution as specified
		stopCh:     make(chan struct{}),
	}

	// Start cleanup goroutine
	store.wg.Add(1)
	go store.cleanupLoop()

	return store
}

// AddReport adds a new report to the time-series store
func (ts *TimeSeriesStore) AddReport(report *probe.ReportData) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	agentID := report.AgentID
	if _, exists := ts.data[agentID]; !exists {
		ts.data[agentID] = &TimeSeriesData{
			AgentID:    agentID,
			Points:     make([]TimeSeriesPoint, 0, 240), // Pre-allocate for 1 hour at 15s resolution
			maxAge:     ts.maxAge,
			resolution: ts.resolution,
		}
	}

	ts.data[agentID].AddPoint(report)
}

// AddPoint adds a data point to the time-series data
func (tsd *TimeSeriesData) AddPoint(report *probe.ReportData) {
	tsd.mu.Lock()
	defer tsd.mu.Unlock()

	point := TimeSeriesPoint{
		Timestamp: report.Timestamp,
		Report:    report,
	}

	// Add point
	tsd.Points = append(tsd.Points, point)

	// Remove old points that exceed maxAge
	cutoff := time.Now().Add(-tsd.maxAge)
	newStart := 0
	for i, p := range tsd.Points {
		if p.Timestamp.After(cutoff) {
			newStart = i
			break
		}
	}
	if newStart > 0 {
		tsd.Points = tsd.Points[newStart:]
	}
}

// GetRecentPoints returns all points within the specified duration
func (ts *TimeSeriesStore) GetRecentPoints(agentID string, duration time.Duration) []TimeSeriesPoint {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	data, exists := ts.data[agentID]
	if !exists {
		return nil
	}

	return data.GetRecentPoints(duration)
}

// GetRecentPoints returns all points within the specified duration
func (tsd *TimeSeriesData) GetRecentPoints(duration time.Duration) []TimeSeriesPoint {
	tsd.mu.RLock()
	defer tsd.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]TimeSeriesPoint, 0)

	for _, point := range tsd.Points {
		if point.Timestamp.After(cutoff) {
			result = append(result, point)
		}
	}

	return result
}

// GetLatestReport returns the most recent report for an agent
func (ts *TimeSeriesStore) GetLatestReport(agentID string) *probe.ReportData {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	data, exists := ts.data[agentID]
	if !exists {
		return nil
	}

	return data.GetLatestReport()
}

// GetLatestReport returns the most recent report
func (tsd *TimeSeriesData) GetLatestReport() *probe.ReportData {
	tsd.mu.RLock()
	defer tsd.mu.RUnlock()

	if len(tsd.Points) == 0 {
		return nil
	}

	return tsd.Points[len(tsd.Points)-1].Report
}

// GetAllAgents returns a list of all agent IDs
func (ts *TimeSeriesStore) GetAllAgents() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	agents := make([]string, 0, len(ts.data))
	for agentID := range ts.data {
		agents = append(agents, agentID)
	}

	return agents
}

// GetTimeSeriesData returns the time-series data for a specific agent
func (ts *TimeSeriesStore) GetTimeSeriesData(agentID string) *TimeSeriesData {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.data[agentID]
}

// DeleteAgent removes all data for a specific agent
func (ts *TimeSeriesStore) DeleteAgent(agentID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	delete(ts.data, agentID)
}

// cleanupLoop periodically removes old data
func (ts *TimeSeriesStore) cleanupLoop() {
	defer ts.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.cleanup()
		case <-ts.stopCh:
			return
		}
	}
}

// cleanup removes expired data points and empty agents
func (ts *TimeSeriesStore) cleanup() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	cutoff := time.Now().Add(-ts.maxAge)
	emptyAgents := make([]string, 0)

	for agentID, data := range ts.data {
		data.mu.Lock()

		// Find first non-expired point
		newStart := 0
		for i, p := range data.Points {
			if p.Timestamp.After(cutoff) {
				newStart = i
				break
			}
		}

		// Remove old points
		if newStart > 0 {
			data.Points = data.Points[newStart:]
		}

		// Mark agent for deletion if no points remain
		if len(data.Points) == 0 {
			emptyAgents = append(emptyAgents, agentID)
		}

		data.mu.Unlock()
	}

	// Remove empty agents
	for _, agentID := range emptyAgents {
		delete(ts.data, agentID)
	}
}

// Stop stops the cleanup goroutine
func (ts *TimeSeriesStore) Stop() {
	close(ts.stopCh)
	ts.wg.Wait()
}

// GetStats returns statistics about the store
func (ts *TimeSeriesStore) GetStats() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	totalPoints := 0
	for _, data := range ts.data {
		data.mu.RLock()
		totalPoints += len(data.Points)
		data.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_agents": len(ts.data),
		"total_points": totalPoints,
		"max_age":      ts.maxAge.String(),
		"resolution":   ts.resolution.String(),
	}
}
