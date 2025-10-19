package migration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationManager_Creation tests creation of migration manager
func TestMigrationManager_Creation(t *testing.T) {
	sourceCluster := &ZookeeperCluster{
		ID:     "cluster-a",
		Hosts:  []string{"zk1:2181", "zk2:2181"},
		Port:   2181,
		Status: "active",
	}

	targetCluster := &ZookeeperCluster{
		ID:     "cluster-b",
		Hosts:  []string{"zk3:2181", "zk4:2181"},
		Port:   2181,
		Status: "active",
	}

	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	assert.NotNil(t, manager)
	assert.Equal(t, "test-migration", manager.ID)
	assert.Equal(t, "initialized", manager.Status)
	assert.Equal(t, 0, manager.CurrentPhase)
	assert.Equal(t, 6, len(manager.Phases))
	assert.NotNil(t, manager.SyncStatus)
	assert.Equal(t, "not_started", manager.SyncStatus.Status)
}

// TestMigrationPhaseCreation tests migration phase creation
func TestMigrationPhaseCreation(t *testing.T) {
	phases := createMigrationPhases()

	assert.Equal(t, 6, len(phases))

	expectedPhases := []string{
		"Deploy Target Zookeeper Cluster",
		"Start Bidirectional Synchronization",
		"Deploy Mesos Master Cluster-B",
		"Deploy Mesos Agent Cluster-B",
		"Drain Tasks from Cluster-A",
		"Final Cutover",
	}

	for i, phase := range phases {
		assert.Equal(t, i+1, phase.Number)
		assert.Equal(t, expectedPhases[i], phase.Name)
		assert.Equal(t, "pending", phase.Status)
		assert.NotEmpty(t, phase.RollbackSteps)
		assert.Nil(t, phase.StartTime)
		assert.Nil(t, phase.EndTime)
	}
}

// TestStartMigration tests starting migration
func TestStartMigration(t *testing.T) {
	manager := createTestMigrationManager()

	// Test successful start
	err := manager.StartMigration()
	assert.NoError(t, err)
	assert.Equal(t, "running", manager.Status)

	// Test starting already started migration
	err = manager.StartMigration()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}

// TestStartPhase tests starting individual phases
func TestStartPhase(t *testing.T) {
	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
		errorMsg    string
	}{
		{"valid phase 1", 1, false, ""},
		{"valid phase 6", 6, false, ""},
		{"invalid phase 0", 0, true, "invalid phase number"},
		{"invalid phase 7", 7, true, "invalid phase number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestMigrationManager()

			err := manager.StartPhase(tt.phaseNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				phase := manager.Phases[tt.phaseNumber-1]
				assert.Equal(t, "running", phase.Status)
				assert.NotNil(t, phase.StartTime)

				// Wait for phase to complete
				time.Sleep(6 * time.Second)

				assert.Equal(t, "completed", phase.Status)
				assert.NotNil(t, phase.EndTime)
				assert.NotNil(t, phase.Validation)
				assert.True(t, phase.Validation.Passed)
			}
		})
	}
}

// TestStartPhaseNotPending tests starting non-pending phase
func TestStartPhaseNotPending(t *testing.T) {
	manager := createTestMigrationManager()

	// Start phase 1
	err := manager.StartPhase(1)
	assert.NoError(t, err)

	// Try to start phase 1 again
	err = manager.StartPhase(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not pending")
}

// TestValidatePhase tests phase validation
func TestValidatePhase(t *testing.T) {
	manager := createTestMigrationManager()

	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
	}{
		{"valid phase 1", 1, false},
		{"valid phase 6", 6, false},
		{"invalid phase 0", 0, true},
		{"invalid phase 7", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation, err := manager.ValidatePhase(tt.phaseNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, validation)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validation)
				assert.True(t, validation.Passed)
				assert.Equal(t, 2, len(validation.Checks))
				assert.Empty(t, validation.Errors)
				assert.Empty(t, validation.Warnings)

				// Verify validation checks
				assert.Equal(t, "Cluster Connectivity", validation.Checks[0].Name)
				assert.Equal(t, "passed", validation.Checks[0].Status)
				assert.True(t, validation.Checks[0].Required)

				assert.Equal(t, "Data Consistency", validation.Checks[1].Name)
				assert.Equal(t, "passed", validation.Checks[1].Status)
			}
		})
	}
}

// TestRollbackPhase tests phase rollback
func TestRollbackPhase(t *testing.T) {
	manager := createTestMigrationManager()

	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
	}{
		{"valid rollback phase 1", 1, false},
		{"valid rollback phase 6", 6, false},
		{"invalid rollback phase 0", 0, true},
		{"invalid rollback phase 7", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RollbackPhase(tt.phaseNumber)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				phase := manager.Phases[tt.phaseNumber-1]
				assert.Equal(t, "rolled_back", phase.Status)
			}
		})
	}
}

// TestStartSync tests starting synchronization
func TestStartSync(t *testing.T) {
	manager := createTestMigrationManager()

	// Test successful start
	err := manager.StartSync()
	assert.NoError(t, err)
	assert.Equal(t, "running", manager.SyncStatus.Status)

	// Test starting already running sync
	err = manager.StartSync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

// TestStopSync tests stopping synchronization
func TestStopSync(t *testing.T) {
	manager := createTestMigrationManager()

	// Test stopping when not running
	err := manager.StopSync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start sync
	err = manager.StartSync()
	assert.NoError(t, err)

	// Test successful stop
	err = manager.StopSync()
	assert.NoError(t, err)
	assert.Equal(t, "stopped", manager.SyncStatus.Status)
}

// TestUpdateSyncStatus tests sync status updates
func TestUpdateSyncStatus(t *testing.T) {
	manager := createTestMigrationManager()

	// Start sync
	manager.SyncStatus.Status = "running"
	manager.SyncStatus.TotalPaths = 100
	manager.SyncStatus.SyncedPaths = 0

	// Update multiple times
	for i := 0; i < 15; i++ {
		manager.updateSyncStatus()
		time.Sleep(10 * time.Millisecond)
	}

	assert.Greater(t, manager.SyncStatus.SyncedPaths, 0)
	assert.LessOrEqual(t, manager.SyncStatus.SyncedPaths, manager.SyncStatus.TotalPaths)

	// Continue until complete
	for manager.SyncStatus.Status == "running" {
		manager.updateSyncStatus()
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, "completed", manager.SyncStatus.Status)
	assert.Equal(t, manager.SyncStatus.TotalPaths, manager.SyncStatus.SyncedPaths)
}

// HTTP Handler Tests

// TestHandleMigrationStatus tests migration status endpoint
func TestHandleMigrationStatus(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("GET", "/api/v1/migration/status", nil)
	w := httptest.NewRecorder()

	manager.handleMigrationStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var status map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&status)
	assert.NoError(t, err)

	assert.Equal(t, manager.ID, status["id"])
	assert.Equal(t, manager.Status, status["status"])
	assert.Equal(t, float64(manager.CurrentPhase), status["current_phase"])
	assert.Equal(t, float64(len(manager.Phases)), status["total_phases"])
}

// TestHandleListPhases tests list phases endpoint
func TestHandleListPhases(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("GET", "/api/v1/migration/phases", nil)
	w := httptest.NewRecorder()

	manager.handleListPhases(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var phases []*MigrationPhase
	err := json.NewDecoder(w.Body).Decode(&phases)
	assert.NoError(t, err)
	assert.Equal(t, 6, len(phases))
}

// TestHandleStartPhaseHTTP tests start phase HTTP endpoint
func TestHandleStartPhaseHTTP(t *testing.T) {
	tests := []struct {
		name       string
		phaseID    string
		expectCode int
	}{
		{"valid phase", "1", http.StatusOK},
		{"invalid phase ID", "invalid", http.StatusBadRequest},
		{"out of range phase", "10", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestMigrationManager()
			router := manager.setupRoutes()

			req := httptest.NewRequest("POST", "/api/v1/migration/phases/"+tt.phaseID+"/start", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			if tt.expectCode == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, "success", response["status"])
			}
		})
	}
}

// TestHandleValidatePhaseHTTP tests validate phase HTTP endpoint
func TestHandleValidatePhaseHTTP(t *testing.T) {
	manager := createTestMigrationManager()
	router := manager.setupRoutes()

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/1/validate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var validation ValidationResult
	err := json.NewDecoder(w.Body).Decode(&validation)
	assert.NoError(t, err)
	assert.True(t, validation.Passed)
}

// TestHandleRollbackPhaseHTTP tests rollback phase HTTP endpoint
func TestHandleRollbackPhaseHTTP(t *testing.T) {
	manager := createTestMigrationManager()
	router := manager.setupRoutes()

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/1/rollback", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
}

// TestHandleSyncStatus tests sync status endpoint
func TestHandleSyncStatus(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("GET", "/api/v1/sync/status", nil)
	w := httptest.NewRecorder()

	manager.handleSyncStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var syncStatus SyncStatus
	err := json.NewDecoder(w.Body).Decode(&syncStatus)
	assert.NoError(t, err)
	assert.Equal(t, manager.SourceCluster.ID, syncStatus.SourceCluster)
	assert.Equal(t, manager.TargetCluster.ID, syncStatus.TargetCluster)
}

// TestHandleStartSyncHTTP tests start sync HTTP endpoint
func TestHandleStartSyncHTTP(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("POST", "/api/v1/sync/start", nil)
	w := httptest.NewRecorder()

	manager.handleStartSync(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "started", response["sync"])
}

// TestHandleStopSyncHTTP tests stop sync HTTP endpoint
func TestHandleStopSyncHTTP(t *testing.T) {
	manager := createTestMigrationManager()

	// Start sync first
	manager.StartSync()

	req := httptest.NewRequest("POST", "/api/v1/sync/stop", nil)
	w := httptest.NewRecorder()

	manager.handleStopSync(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "stopped", response["sync"])
}

// TestHandleListConflicts tests list conflicts endpoint
func TestHandleListConflicts(t *testing.T) {
	manager := createTestMigrationManager()

	// Add some conflicts
	manager.SyncStatus.Conflicts = []*SyncConflict{
		{
			Path:        "/mesos/test",
			SourceValue: "value1",
			TargetValue: "value2",
			Timestamp:   time.Now(),
			Resolution:  "manual",
		},
	}

	req := httptest.NewRequest("GET", "/api/v1/sync/conflicts", nil)
	w := httptest.NewRecorder()

	manager.handleListConflicts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var conflicts []*SyncConflict
	err := json.NewDecoder(w.Body).Decode(&conflicts)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(conflicts))
}

// TestHandleListClusters tests list clusters endpoint
func TestHandleListClusters(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("GET", "/api/v1/clusters", nil)
	w := httptest.NewRecorder()

	manager.handleListClusters(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var clusters []*ZookeeperCluster
	err := json.NewDecoder(w.Body).Decode(&clusters)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(clusters))
}

// TestHandleClusterStatus tests cluster status endpoint
func TestHandleClusterStatus(t *testing.T) {
	manager := createTestMigrationManager()
	router := manager.setupRoutes()

	tests := []struct {
		name       string
		clusterID  string
		expectCode int
	}{
		{"source cluster", manager.SourceCluster.ID, http.StatusOK},
		{"target cluster", manager.TargetCluster.ID, http.StatusOK},
		{"unknown cluster", "unknown", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/clusters/"+tt.clusterID+"/status", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			if tt.expectCode == http.StatusOK {
				var cluster ZookeeperCluster
				err := json.NewDecoder(w.Body).Decode(&cluster)
				assert.NoError(t, err)
				assert.Equal(t, tt.clusterID, cluster.ID)
			}
		})
	}
}

// TestHandleHealth tests health check endpoint
func TestHandleHealth(t *testing.T) {
	manager := createTestMigrationManager()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	manager.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var health map[string]string
	err := json.NewDecoder(w.Body).Decode(&health)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])
}

// TestStartStopServer tests starting and stopping the HTTP server
func TestStartStopServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	manager := createTestMigrationManager()

	// Start server in background
	serverStarted := make(chan bool)
	serverError := make(chan error)

	go func() {
		serverStarted <- true
		err := manager.Start()
		if err != http.ErrServerClosed {
			serverError <- err
		}
	}()

	// Wait for server to start
	<-serverStarted
	time.Sleep(100 * time.Millisecond)

	// Test that server is running
	resp, err := http.Get("http://localhost:8080/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Stop server
	err = manager.Stop()
	assert.NoError(t, err)

	// Verify no server errors
	select {
	case err := <-serverError:
		t.Fatalf("Server error: %v", err)
	case <-time.After(100 * time.Millisecond):
		// No error, good
	}
}

// TestConcurrentAccess tests concurrent access to migration manager
func TestConcurrentAccess(t *testing.T) {
	manager := createTestMigrationManager()

	// Start sync
	manager.StartSync()

	done := make(chan bool, 3)

	// Concurrent sync status updates
	go func() {
		for i := 0; i < 100; i++ {
			manager.updateSyncStatus()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.SyncStatus.Status
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Concurrent phase validation
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = manager.ValidatePhase(1)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

// Helper functions

func createTestMigrationManager() *MigrationManager {
	sourceCluster := &ZookeeperCluster{
		ID:     "cluster-a",
		Hosts:  []string{"zk1:2181", "zk2:2181"},
		Port:   2181,
		Status: "active",
	}

	targetCluster := &ZookeeperCluster{
		ID:     "cluster-b",
		Hosts:  []string{"zk3:2181", "zk4:2181"},
		Port:   2181,
		Status: "active",
	}

	return NewMigrationManager("test-migration", sourceCluster, targetCluster)
}

// Benchmarks

func BenchmarkUpdateSyncStatus(b *testing.B) {
	manager := createTestMigrationManager()
	manager.SyncStatus.Status = "running"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.updateSyncStatus()
	}
}

func BenchmarkValidatePhase(b *testing.B) {
	manager := createTestMigrationManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidatePhase(1)
	}
}
