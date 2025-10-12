package migration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMigrationManager(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		sourceCluster *ZookeeperCluster
		targetCluster *ZookeeperCluster
	}{
		{
			name: "Valid migration manager",
			id:   "migration-1",
			sourceCluster: &ZookeeperCluster{
				ID:    "cluster-a",
				Hosts: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
				Port:  2181,
			},
			targetCluster: &ZookeeperCluster{
				ID:    "cluster-b",
				Hosts: []string{"zk4:2181", "zk5:2181", "zk6:2181"},
				Port:  2181,
			},
		},
		{
			name: "Empty ID",
			id:   "",
			sourceCluster: &ZookeeperCluster{
				ID:    "cluster-a",
				Hosts: []string{"zk1:2181"},
				Port:  2181,
			},
			targetCluster: &ZookeeperCluster{
				ID:    "cluster-b",
				Hosts: []string{"zk2:2181"},
				Port:  2181,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewMigrationManager(tt.id, tt.sourceCluster, tt.targetCluster)

			assert.Equal(t, tt.id, manager.ID)
			assert.Equal(t, tt.sourceCluster, manager.SourceCluster)
			assert.Equal(t, tt.targetCluster, manager.TargetCluster)
			assert.Equal(t, "initialized", manager.Status)
			assert.Equal(t, 0, manager.CurrentPhase)
			assert.Len(t, manager.Phases, 6)
			assert.NotNil(t, manager.SyncStatus)
			assert.Equal(t, tt.sourceCluster.ID, manager.SyncStatus.SourceCluster)
			assert.Equal(t, tt.targetCluster.ID, manager.SyncStatus.TargetCluster)
			assert.Equal(t, "not_started", manager.SyncStatus.Status)
		})
	}
}

func TestCreateMigrationPhases(t *testing.T) {
	phases := createMigrationPhases()

	assert.Len(t, phases, 6)

	expectedPhases := []struct {
		number      int
		name        string
		description string
	}{
		{1, "Deploy Target Zookeeper Cluster", "Deploy and configure Zookeeper Cluster-B"},
		{2, "Start Bidirectional Synchronization", "Start real-time sync between Cluster-A and Cluster-B"},
		{3, "Deploy Mesos Master Cluster-B", "Deploy Mesos masters pointing to Cluster-B"},
		{4, "Deploy Mesos Agent Cluster-B", "Deploy Mesos agents connected to Cluster-B"},
		{5, "Drain Tasks from Cluster-A", "Gracefully drain tasks from Cluster-A agents"},
		{6, "Final Cutover", "Complete migration to Cluster-B"},
	}

	for i, expected := range expectedPhases {
		phase := phases[i]
		assert.Equal(t, expected.number, phase.Number)
		assert.Equal(t, expected.name, phase.Name)
		assert.Equal(t, expected.description, phase.Description)
		assert.Equal(t, "pending", phase.Status)
		assert.NotNil(t, phase.RollbackSteps)
		assert.Greater(t, len(phase.RollbackSteps), 0)
	}
}

func TestMigrationManager_StartMigration(t *testing.T) {
	tests := []struct {
		name        string
		initialStatus string
		expectError bool
	}{
		{
			name:        "Valid start",
			initialStatus: "initialized",
			expectError: false,
		},
		{
			name:        "Already started",
			initialStatus: "running",
			expectError: true,
		},
		{
			name:        "Already completed",
			initialStatus: "completed",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)
			manager.Status = tt.initialStatus

			err := manager.StartMigration()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "running", manager.Status)
			}
		})
	}
}

func TestMigrationManager_StartPhase(t *testing.T) {
	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
	}{
		{
			name:        "Valid phase 1",
			phaseNumber: 1,
			expectError: false,
		},
		{
			name:        "Valid phase 6",
			phaseNumber: 6,
			expectError: false,
		},
		{
			name:        "Invalid phase 0",
			phaseNumber: 0,
			expectError: true,
		},
		{
			name:        "Invalid phase 7",
			phaseNumber: 7,
			expectError: true,
		},
		{
			name:        "Invalid phase -1",
			phaseNumber: -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

			err := manager.StartPhase(tt.phaseNumber)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				phase := manager.Phases[tt.phaseNumber-1]
				assert.Equal(t, "running", phase.Status)
				assert.NotNil(t, phase.StartTime)
			}
		})
	}
}

func TestMigrationManager_StartPhaseAlreadyRunning(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	// Start phase 1
	err := manager.StartPhase(1)
	assert.NoError(t, err)

	// Try to start phase 1 again
	err = manager.StartPhase(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not pending")
}

func TestMigrationManager_ValidatePhase(t *testing.T) {
	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
	}{
		{
			name:        "Valid phase 1",
			phaseNumber: 1,
			expectError: false,
		},
		{
			name:        "Valid phase 6",
			phaseNumber: 6,
			expectError: false,
		},
		{
			name:        "Invalid phase 0",
			phaseNumber: 0,
			expectError: true,
		},
		{
			name:        "Invalid phase 7",
			phaseNumber: 7,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

			validation, err := manager.ValidatePhase(tt.phaseNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, validation)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validation)
				assert.True(t, validation.Passed)
				assert.Len(t, validation.Checks, 2)
				assert.Len(t, validation.Errors, 0)
				assert.Len(t, validation.Warnings, 0)
				assert.NotNil(t, validation.Timestamp)

				// Check validation checks
				checkNames := make(map[string]bool)
				for _, check := range validation.Checks {
					checkNames[check.Name] = true
				}
				assert.True(t, checkNames["Cluster Connectivity"])
				assert.True(t, checkNames["Data Consistency"])
			}
		})
	}
}

func TestMigrationManager_RollbackPhase(t *testing.T) {
	tests := []struct {
		name        string
		phaseNumber int
		expectError bool
	}{
		{
			name:        "Valid phase 1",
			phaseNumber: 1,
			expectError: false,
		},
		{
			name:        "Valid phase 6",
			phaseNumber: 6,
			expectError: false,
		},
		{
			name:        "Invalid phase 0",
			phaseNumber: 0,
			expectError: true,
		},
		{
			name:        "Invalid phase 7",
			phaseNumber: 7,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

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

func TestMigrationManager_StartSync(t *testing.T) {
	tests := []struct {
		name        string
		initialStatus string
		expectError bool
	}{
		{
			name:        "Valid start",
			initialStatus: "not_started",
			expectError: false,
		},
		{
			name:        "Already running",
			initialStatus: "running",
			expectError: true,
		},
		{
			name:        "Already stopped",
			initialStatus: "stopped",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)
			manager.SyncStatus.Status = tt.initialStatus

			err := manager.StartSync()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "running", manager.SyncStatus.Status)
				assert.False(t, manager.SyncStatus.LastSync.IsZero())
			}
		})
	}
}

func TestMigrationManager_StopSync(t *testing.T) {
	tests := []struct {
		name        string
		initialStatus string
		expectError bool
	}{
		{
			name:        "Valid stop",
			initialStatus: "running",
			expectError: false,
		},
		{
			name:        "Not running",
			initialStatus: "not_started",
			expectError: true,
		},
		{
			name:        "Already stopped",
			initialStatus: "stopped",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
			targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
			manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)
			manager.SyncStatus.Status = tt.initialStatus

			err := manager.StopSync()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "stopped", manager.SyncStatus.Status)
			}
		})
	}
}

func TestMigrationManager_UpdateSyncStatus(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	// Start sync
	manager.SyncStatus.Status = "running"
	manager.SyncStatus.LastSync = time.Now().Add(-10 * time.Second)

	// Update sync status
	manager.updateSyncStatus()

	// Verify sync lag is calculated
	assert.True(t, manager.SyncStatus.SyncLag > 0)
	assert.True(t, manager.SyncStatus.LastSync.After(time.Now().Add(-1*time.Second)))

	// Verify sync progress
	if manager.SyncStatus.TotalPaths == 0 {
		assert.Equal(t, 1000, manager.SyncStatus.TotalPaths)
	}
	assert.GreaterOrEqual(t, manager.SyncStatus.SyncedPaths, 0)
}

func TestMigrationManager_ExecutePhase(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	// Start phase 1
	err := manager.StartPhase(1)
	assert.NoError(t, err)

	// Wait for phase execution to complete
	time.Sleep(6 * time.Second)

	// Verify phase is completed
	phase := manager.Phases[0]
	assert.Equal(t, "completed", phase.Status)
	assert.NotNil(t, phase.EndTime)
	assert.NotNil(t, phase.Validation)
	assert.True(t, phase.Validation.Passed)
}

func TestMigrationManager_HTTPHandlers(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	testCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"Migration Status", "GET", "/api/v1/migration/status", http.StatusOK},
		{"List Phases", "GET", "/api/v1/migration/phases", http.StatusOK},
		{"Start Phase", "POST", "/api/v1/migration/phases/1/start", http.StatusOK},
		{"Validate Phase", "POST", "/api/v1/migration/phases/1/validate", http.StatusOK},
		{"Rollback Phase", "POST", "/api/v1/migration/phases/1/rollback", http.StatusOK},
		{"Sync Status", "GET", "/api/v1/sync/status", http.StatusOK},
		{"Start Sync", "POST", "/api/v1/sync/start", http.StatusOK},
		{"Stop Sync", "POST", "/api/v1/sync/stop", http.StatusOK},
		{"List Conflicts", "GET", "/api/v1/sync/conflicts", http.StatusOK},
		{"List Clusters", "GET", "/api/v1/clusters", http.StatusOK},
		{"Cluster Status", "GET", "/api/v1/clusters/cluster-a/status", http.StatusOK},
		{"Health Check", "GET", "/health", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router := manager.setupRoutes()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.status, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

func TestMigrationManager_HandleMigrationStatus(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/migration/status", nil)
	rr := httptest.NewRecorder()

	manager.handleMigrationStatus(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var status map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &status)
	assert.NoError(t, err)

	assert.Equal(t, "test-migration", status["id"])
	assert.Equal(t, "initialized", status["status"])
	assert.Equal(t, float64(0), status["current_phase"])
	assert.Equal(t, float64(6), status["total_phases"])
}

func TestMigrationManager_HandleListPhases(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/migration/phases", nil)
	rr := httptest.NewRecorder()

	manager.handleListPhases(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var phases []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &phases)
	assert.NoError(t, err)

	assert.Len(t, phases, 6)
	assert.Equal(t, float64(1), phases[0]["number"])
	assert.Equal(t, "Deploy Target Zookeeper Cluster", phases[0]["name"])
	assert.Equal(t, "pending", phases[0]["status"])
}

func TestMigrationManager_HandleStartPhase(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/1/start", nil)
	rr := httptest.NewRecorder()

	router := manager.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMigrationManager_HandleStartPhaseInvalidID(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/invalid/start", nil)
	rr := httptest.NewRecorder()

	manager.handleStartPhase(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid phase ID")
}

func TestMigrationManager_HandleValidatePhase(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/1/validate", nil)
	rr := httptest.NewRecorder()

	router := manager.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var validation map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &validation)
	assert.NoError(t, err)

	assert.True(t, validation["passed"].(bool))
	assert.NotNil(t, validation["checks"])
	assert.NotNil(t, validation["timestamp"])
}

func TestMigrationManager_HandleRollbackPhase(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("POST", "/api/v1/migration/phases/1/rollback", nil)
	rr := httptest.NewRecorder()

	manager.handleRollbackPhase(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMigrationManager_HandleSyncStatus(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/sync/status", nil)
	rr := httptest.NewRecorder()

	manager.handleSyncStatus(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var syncStatus map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &syncStatus)
	assert.NoError(t, err)

	assert.Equal(t, "cluster-a", syncStatus["source_cluster"])
	assert.Equal(t, "cluster-b", syncStatus["target_cluster"])
	assert.Equal(t, "not_started", syncStatus["status"])
}

func TestMigrationManager_HandleStartSync(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("POST", "/api/v1/sync/start", nil)
	rr := httptest.NewRecorder()

	manager.handleStartSync(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "running", manager.SyncStatus.Status)
}

func TestMigrationManager_HandleStopSync(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	// Start sync first
	manager.SyncStatus.Status = "running"

	req := httptest.NewRequest("POST", "/api/v1/sync/stop", nil)
	rr := httptest.NewRecorder()

	manager.handleStopSync(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "stopped", manager.SyncStatus.Status)
}

func TestMigrationManager_HandleListConflicts(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/sync/conflicts", nil)
	rr := httptest.NewRecorder()

	manager.handleListConflicts(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var conflicts []interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &conflicts)
	assert.NoError(t, err)

	assert.Len(t, conflicts, 0) // No conflicts initially
}

func TestMigrationManager_HandleListClusters(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/clusters", nil)
	rr := httptest.NewRecorder()

	manager.handleListClusters(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var clusters []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &clusters)
	assert.NoError(t, err)

	assert.Len(t, clusters, 2)
	assert.Equal(t, "cluster-a", clusters[0]["id"])
	assert.Equal(t, "cluster-b", clusters[1]["id"])
}

func TestMigrationManager_HandleClusterStatus(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/clusters/cluster-a/status", nil)
	rr := httptest.NewRecorder()

	manager.handleClusterStatus(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var cluster map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &cluster)
	assert.NoError(t, err)

	assert.Equal(t, "cluster-a", cluster["id"])
}

func TestMigrationManager_HandleClusterStatusNotFound(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/api/v1/clusters/nonexistent/status", nil)
	rr := httptest.NewRecorder()

	manager.handleClusterStatus(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMigrationManager_HandleHealth(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	manager.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var health map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &health)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
}

func TestMigrationManager_ConcurrentAccess(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				manager.StartPhase(1)
				manager.ValidatePhase(1)
				manager.StartSync()
				manager.StopSync()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestMigrationManager_StartStop(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.server.Shutdown(ctx)
	if err != nil {
		t.Logf("Server shutdown error: %v", err)
	}

	// Check for start error
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Server start timeout")
	}
}

func TestMigrationManager_SetupRoutes(t *testing.T) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	router := manager.setupRoutes()

	assert.NotNil(t, router)

	// Test that routes are properly configured
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/migration/status"},
		{"GET", "/api/v1/migration/phases"},
		{"POST", "/api/v1/migration/phases/1/start"},
		{"POST", "/api/v1/migration/phases/1/validate"},
		{"POST", "/api/v1/migration/phases/1/rollback"},
		{"GET", "/api/v1/sync/status"},
		{"POST", "/api/v1/sync/start"},
		{"POST", "/api/v1/sync/stop"},
		{"GET", "/api/v1/sync/conflicts"},
		{"GET", "/api/v1/clusters"},
		{"GET", "/api/v1/clusters/cluster-a/status"},
		{"GET", "/health"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			// Should not return 404 (route not found)
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "Route %s %s should be found", tc.method, tc.path)
		})
	}
}

func BenchmarkMigrationManager_StartPhase(b *testing.B) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	for i := 0; i < b.N; i++ {
		manager.StartPhase(1)
	}
}

func BenchmarkMigrationManager_ValidatePhase(b *testing.B) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	for i := 0; i < b.N; i++ {
		manager.ValidatePhase(1)
	}
}

func BenchmarkMigrationManager_StartSync(b *testing.B) {
	sourceCluster := &ZookeeperCluster{ID: "cluster-a", Hosts: []string{"zk1:2181"}}
	targetCluster := &ZookeeperCluster{ID: "cluster-b", Hosts: []string{"zk2:2181"}}
	manager := NewMigrationManager("test-migration", sourceCluster, targetCluster)

	for i := 0; i < b.N; i++ {
		manager.StartSync()
	}
}