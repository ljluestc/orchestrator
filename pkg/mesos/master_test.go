package mesos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMaster(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		hostname     string
		port         int
		zookeeperURL string
	}{
		{
			name:         "Valid Master",
			id:           "master-1",
			hostname:     "localhost",
			port:         5050,
			zookeeperURL: "zk://localhost:2181/mesos",
		},
		{
			name:         "Empty ID",
			id:           "",
			hostname:     "localhost",
			port:         5050,
			zookeeperURL: "zk://localhost:2181/mesos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master := NewMaster(tt.id, tt.hostname, tt.port, tt.zookeeperURL)

			assert.Equal(t, tt.id, master.ID)
			assert.Equal(t, tt.hostname, master.Hostname)
			assert.Equal(t, tt.port, master.Port)
			assert.Equal(t, tt.zookeeperURL, master.ZookeeperURL)
			assert.False(t, master.IsLeader)
			assert.NotNil(t, master.Agents)
			assert.NotNil(t, master.Frameworks)
			assert.NotNil(t, master.Resources)
			assert.NotNil(t, master.Offers)
			assert.NotNil(t, master.State)
			assert.Equal(t, "1.0.0", master.State.Version)
			assert.NotNil(t, master.State.Agents)
			assert.NotNil(t, master.State.Frameworks)
			assert.NotNil(t, master.State.Tasks)
			assert.NotNil(t, master.State.Offers)
		})
	}
}

func TestMaster_RegisterAgent(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
			Ports: []PortRange{
				{Begin: 31000, End: 32000},
			},
		},
		Tasks:    make(map[string]*Task),
		Status:   "inactive",
		LastSeen: time.Now().Add(-1 * time.Hour),
	}

	err := master.RegisterAgent(agent)

	assert.NoError(t, err)
	assert.Equal(t, "active", agent.Status)
	assert.True(t, time.Since(agent.LastSeen) < time.Minute)

	// Verify agent is stored
	storedAgent, exists := master.Agents[agent.ID]
	assert.True(t, exists)
	assert.Equal(t, agent, storedAgent)

	// Verify agent is in state
	stateAgent, exists := master.State.Agents[agent.ID]
	assert.True(t, exists)
	assert.Equal(t, agent, stateAgent)

	// Verify resources are updated
	assert.Equal(t, 4.0, master.Resources.TotalCPUs)
	assert.Equal(t, 8192.0, master.Resources.TotalMemory)
	assert.Equal(t, 100000.0, master.Resources.TotalDisk)
	assert.Equal(t, 4.0, master.Resources.AvailableCPUs)
	assert.Equal(t, 8192.0, master.Resources.AvailableMemory)
	assert.Equal(t, 100000.0, master.Resources.AvailableDisk)
}

func TestMaster_RegisterFramework(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
		Status:    "inactive",
	}

	err := master.RegisterFramework(framework)

	assert.NoError(t, err)
	assert.Equal(t, "active", framework.Status)
	assert.True(t, time.Since(framework.RegisteredAt) < time.Minute)
	assert.NotNil(t, framework.Tasks)
	assert.NotNil(t, framework.Offers)

	// Verify framework is stored
	storedFramework, exists := master.Frameworks[framework.ID]
	assert.True(t, exists)
	assert.Equal(t, framework, storedFramework)

	// Verify framework is in state
	stateFramework, exists := master.State.Frameworks[framework.ID]
	assert.True(t, exists)
	assert.Equal(t, framework, stateFramework)
}

func TestMaster_LaunchTask(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "pending",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
		Command: &Command{
			Value: "echo 'Hello World'",
			Shell: true,
			User:  "root",
		},
	}

	err := master.LaunchTask(task)

	assert.NoError(t, err)
	assert.Equal(t, "starting", task.State)
	assert.True(t, time.Since(task.CreatedAt) < time.Minute)

	// Verify task is in agent
	agentTask, exists := agent.Tasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, task, agentTask)

	// Verify task is in framework
	frameworkTask, exists := framework.Tasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, task, frameworkTask)

	// Verify task is in global state
	stateTask, exists := master.State.Tasks[task.ID]
	assert.True(t, exists)
	assert.Equal(t, task, stateTask)
}

func TestMaster_LaunchTaskAgentNotFound(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "nonexistent-agent",
		State:       "pending",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}

	err := master.LaunchTask(task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent nonexistent-agent not found")
}

func TestMaster_KillTask(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	// Kill task
	err := master.KillTask("task-1")

	assert.NoError(t, err)
	assert.Equal(t, "killed", task.State)

	// Verify task is removed from agent
	_, exists := agent.Tasks[task.ID]
	assert.False(t, exists)

	// Verify task is removed from framework
	_, exists = framework.Tasks[task.ID]
	assert.False(t, exists)

	// Verify task is removed from global state
	_, exists = master.State.Tasks[task.ID]
	assert.False(t, exists)
}

func TestMaster_KillTaskNotFound(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	err := master.KillTask("nonexistent-task")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task nonexistent-task not found")
}

func TestMaster_GenerateResourceOffers(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks:  make(map[string]*Task),
		Status: "active",
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
		Status:    "active",
	}
	master.RegisterFramework(framework)

	// Generate offers
	master.generateResourceOffers()

	// Verify offers are created
	assert.Len(t, master.Offers, 1)
	assert.Len(t, master.State.Offers, 1)

	offer := master.Offers[0]
	assert.NotEmpty(t, offer.ID)
	assert.Equal(t, "agent-1", offer.AgentID)
	assert.Equal(t, agent.Resources, offer.Resources)
	assert.True(t, time.Since(offer.CreatedAt) < time.Minute)
	assert.True(t, time.Until(offer.ExpiresAt) < 6*time.Minute)
}

func TestMaster_CheckAgentHealth(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent with old last seen time
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks:    make(map[string]*Task),
		Status:   "active",
		LastSeen: time.Now().Add(-3 * time.Minute), // 3 minutes ago
	}
	master.RegisterAgent(agent)
	
	// Override the LastSeen time after registration (since RegisterAgent sets it to now)
	agent.LastSeen = time.Now().Add(-3 * time.Minute)

	// Check health
	master.checkAgentHealth()

	// Verify agent is marked as inactive
	assert.Equal(t, "inactive", agent.Status)
}

func TestMaster_HTTPHandlers(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register test framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch test task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	testCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"Master Info", "GET", "/api/v1/master/info", http.StatusOK},
		{"Master State", "GET", "/api/v1/master/state", http.StatusOK},
		{"List Agents", "GET", "/api/v1/agents", http.StatusOK},
		{"Get Agent", "GET", "/api/v1/agents/agent-1", http.StatusOK},
		{"Get Agent Tasks", "GET", "/api/v1/agents/agent-1/tasks", http.StatusOK},
		{"List Frameworks", "GET", "/api/v1/frameworks", http.StatusOK},
		{"Get Framework", "GET", "/api/v1/frameworks/framework-1", http.StatusOK},
		{"Get Framework Tasks", "GET", "/api/v1/frameworks/framework-1/tasks", http.StatusOK},
		{"List Tasks", "GET", "/api/v1/tasks", http.StatusOK},
		{"Get Task", "GET", "/api/v1/tasks/task-1", http.StatusOK},
		{"List Offers", "GET", "/api/v1/offers", http.StatusOK},
		{"Health", "GET", "/health", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			router := master.setupRoutes()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.status, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

func TestMaster_HandleMasterInfo(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	req := httptest.NewRequest("GET", "/api/v1/master/info", nil)
	rr := httptest.NewRecorder()

	master.handleMasterInfo(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var info map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &info)
	assert.NoError(t, err)

	assert.Equal(t, "test-master", info["id"])
	assert.Equal(t, "localhost", info["hostname"])
	assert.Equal(t, float64(5050), info["port"])
	assert.Equal(t, false, info["leader"])
	assert.Equal(t, "1.0.0", info["version"])
}

func TestMaster_HandleMasterState(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	req := httptest.NewRequest("GET", "/api/v1/master/state", nil)
	rr := httptest.NewRecorder()

	master.handleMasterState(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var state ClusterState
	err := json.Unmarshal(rr.Body.Bytes(), &state)
	assert.NoError(t, err)

	assert.Equal(t, "1.0.0", state.Version)
	assert.NotNil(t, state.Agents)
	assert.NotNil(t, state.Frameworks)
	assert.NotNil(t, state.Tasks)
	assert.NotNil(t, state.Offers)
}

func TestMaster_HandleListAgents(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test agents
	agent1 := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	agent2 := &AgentInfo{
		ID:       "agent-2",
		Hostname: "localhost",
		Port:     5052,
		Resources: &Resources{
			CPUs:   8.0,
			Memory: 16384.0,
			Disk:   200000.0,
		},
		Tasks: make(map[string]*Task),
	}

	master.RegisterAgent(agent1)
	master.RegisterAgent(agent2)

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rr := httptest.NewRecorder()

	master.handleListAgents(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var agents []AgentInfo
	err := json.Unmarshal(rr.Body.Bytes(), &agents)
	assert.NoError(t, err)
	assert.Len(t, agents, 2)
}

func TestMaster_HandleGetAgent(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	req := httptest.NewRequest("GET", "/api/v1/agents/agent-1", nil)
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := master.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var returnedAgent AgentInfo
	err := json.Unmarshal(rr.Body.Bytes(), &returnedAgent)
	assert.NoError(t, err)
	assert.Equal(t, "agent-1", returnedAgent.ID)
	assert.Equal(t, "localhost", returnedAgent.Hostname)
	assert.Equal(t, 5051, returnedAgent.Port)
}

func TestMaster_HandleGetAgentNotFound(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	req := httptest.NewRequest("GET", "/api/v1/agents/nonexistent", nil)
	rr := httptest.NewRecorder()

	master.handleGetAgent(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMaster_HandleGetAgentTasks(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	req := httptest.NewRequest("GET", "/api/v1/agents/agent-1/tasks", nil)
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := master.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var tasks []Task
	err := json.Unmarshal(rr.Body.Bytes(), &tasks)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "task-1", tasks[0].ID)
}

func TestMaster_HandleListFrameworks(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test frameworks
	framework1 := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	framework2 := &Framework{
		ID:        "framework-2",
		Name:      "chronos",
		Principal: "chronos-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8081,
	}

	master.RegisterFramework(framework1)
	master.RegisterFramework(framework2)

	req := httptest.NewRequest("GET", "/api/v1/frameworks", nil)
	rr := httptest.NewRecorder()

	master.handleListFrameworks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var frameworks []Framework
	err := json.Unmarshal(rr.Body.Bytes(), &frameworks)
	assert.NoError(t, err)
	assert.Len(t, frameworks, 2)
}

func TestMaster_HandleRegisterFramework(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}

	frameworkJSON, _ := json.Marshal(framework)
	var buf bytes.Buffer
	buf.Write(frameworkJSON)
	req := httptest.NewRequest("POST", "/api/v1/frameworks", &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	master.handleRegisterFramework(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify framework was registered
	_, exists := master.Frameworks["framework-1"]
	assert.True(t, exists)
}

func TestMaster_HandleGetFramework(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	req := httptest.NewRequest("GET", "/api/v1/frameworks/framework-1", nil)
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := master.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var returnedFramework Framework
	err := json.Unmarshal(rr.Body.Bytes(), &returnedFramework)
	assert.NoError(t, err)
	assert.Equal(t, "framework-1", returnedFramework.ID)
	assert.Equal(t, "marathon", returnedFramework.Name)
}

func TestMaster_HandleGetFrameworkNotFound(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	req := httptest.NewRequest("GET", "/api/v1/frameworks/nonexistent", nil)
	rr := httptest.NewRecorder()

	master.handleGetFramework(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMaster_HandleGetFrameworkTasks(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	req := httptest.NewRequest("GET", "/api/v1/frameworks/framework-1/tasks", nil)
	rr := httptest.NewRecorder()

	// Use the router to handle the request
	router := master.setupRoutes()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var tasks []Task
	err := json.Unmarshal(rr.Body.Bytes(), &tasks)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "task-1", tasks[0].ID)
}

func TestMaster_HandleListTasks(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch tasks
	task1 := &Task{
		ID:          "task-1",
		Name:        "test-task-1",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	task2 := &Task{
		ID:          "task-2",
		Name:        "test-task-2",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}

	master.LaunchTask(task1)
	master.LaunchTask(task2)

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	rr := httptest.NewRecorder()

	master.handleListTasks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var tasks []Task
	err := json.Unmarshal(rr.Body.Bytes(), &tasks)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestMaster_HandleGetTask(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	router := master.setupRoutes()
	req := httptest.NewRequest("GET", "/api/v1/tasks/task-1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var returnedTask Task
	err := json.Unmarshal(rr.Body.Bytes(), &returnedTask)
	assert.NoError(t, err)
	assert.Equal(t, "task-1", returnedTask.ID)
	assert.Equal(t, "test-task", returnedTask.Name)
}

func TestMaster_HandleGetTaskNotFound(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	router := master.setupRoutes()
	req := httptest.NewRequest("GET", "/api/v1/tasks/nonexistent", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMaster_HandleKillTask(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	// Launch task
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	router := master.setupRoutes()
	req := httptest.NewRequest("POST", "/api/v1/tasks/task-1/kill", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify task was killed
	_, exists := master.State.Tasks["task-1"]
	assert.False(t, exists)
}

func TestMaster_HandleListOffers(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks:  make(map[string]*Task),
		Status: "active",
	}
	master.RegisterAgent(agent)

	// Generate offers
	master.generateResourceOffers()

	req := httptest.NewRequest("GET", "/api/v1/offers", nil)
	rr := httptest.NewRecorder()

	master.handleListOffers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var offers []ResourceOffer
	err := json.Unmarshal(rr.Body.Bytes(), &offers)
	assert.NoError(t, err)
	assert.Len(t, offers, 1)
}

func TestMaster_HandleAcceptOffer(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks:  make(map[string]*Task),
		Status: "active",
	}
	master.RegisterAgent(agent)

	// Generate offers
	master.generateResourceOffers()

	offerID := master.Offers[0].ID
	req := httptest.NewRequest("POST", "/api/v1/offers/"+offerID+"/accept", nil)
	rr := httptest.NewRecorder()

	master.handleAcceptOffer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMaster_HandleDeclineOffer(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks:  make(map[string]*Task),
		Status: "active",
	}
	master.RegisterAgent(agent)

	// Generate offers
	master.generateResourceOffers()

	offerID := master.Offers[0].ID
	req := httptest.NewRequest("POST", "/api/v1/offers/"+offerID+"/decline", nil)
	rr := httptest.NewRecorder()

	master.handleDeclineOffer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMaster_HandleHealth(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	master.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var health map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &health)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])
}

func TestMaster_SetupRoutes(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Add test data for routes with path parameters
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
	}
	master.LaunchTask(task)

	offer := &ResourceOffer{
		ID:          "offer-1",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		Resources: &Resources{
			CPUs:   2.0,
			Memory: 2048.0,
			Disk:   5000.0,
		},
	}
	master.State.Offers = append(master.State.Offers, offer)

	router := master.setupRoutes()

	assert.NotNil(t, router)

	// Test that routes are properly configured
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/master/info"},
		{"GET", "/api/v1/master/state"},
		{"GET", "/api/v1/agents"},
		{"GET", "/api/v1/agents/agent-1"},
		{"GET", "/api/v1/agents/agent-1/tasks"},
		{"GET", "/api/v1/frameworks"},
		{"POST", "/api/v1/frameworks"},
		{"GET", "/api/v1/frameworks/framework-1"},
		{"GET", "/api/v1/frameworks/framework-1/tasks"},
		{"GET", "/api/v1/tasks"},
		{"GET", "/api/v1/tasks/task-1"},
		{"POST", "/api/v1/tasks/task-1/kill"},
		{"GET", "/api/v1/offers"},
		{"POST", "/api/v1/offers/offer-1/accept"},
		{"POST", "/api/v1/offers/offer-1/decline"},
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

func TestMaster_StartStop(t *testing.T) {
	master := NewMaster("test-master", "localhost", 0, "zk://localhost:2181/mesos")

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- master.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := master.server.Shutdown(ctx)
	if err != nil {
		t.Logf("Server shutdown error: %v", err)
	}

	// Check for start error
	select {
	case err := <-errChan:
		// Server closed error is expected when Shutdown() is called
		if err != nil && err != http.ErrServerClosed {
			assert.NoError(t, err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Server start timeout")
	}
}

func TestMaster_ConcurrentAccess(t *testing.T) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				agent := &AgentInfo{
					ID:       fmt.Sprintf("agent-%d", j),
					Hostname: "localhost",
					Port:     5051 + j,
					Resources: &Resources{
						CPUs:   4.0,
						Memory: 8192.0,
						Disk:   100000.0,
					},
					Tasks: make(map[string]*Task),
				}
				master.RegisterAgent(agent)

				framework := &Framework{
					ID:        fmt.Sprintf("framework-%d", j),
					Name:      "marathon",
					Principal: "marathon-principal",
					Role:      "*",
					Hostname:  "localhost",
					Port:      8080 + j,
				}
				master.RegisterFramework(framework)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestMaster_ResourceStructures(t *testing.T) {
	// Test Resources structure
	resources := &Resources{
		CPUs:   4.0,
		Memory: 8192.0,
		Disk:   100000.0,
		Ports: []PortRange{
			{Begin: 31000, End: 32000},
			{Begin: 33000, End: 34000},
		},
	}

	assert.Equal(t, 4.0, resources.CPUs)
	assert.Equal(t, 8192.0, resources.Memory)
	assert.Equal(t, 100000.0, resources.Disk)
	assert.Len(t, resources.Ports, 2)
	assert.Equal(t, uint32(31000), resources.Ports[0].Begin)
	assert.Equal(t, uint32(32000), resources.Ports[0].End)
	assert.Equal(t, uint32(33000), resources.Ports[1].Begin)
	assert.Equal(t, uint32(34000), resources.Ports[1].End)
}

func TestMaster_ResourcePoolStructures(t *testing.T) {
	// Test ResourcePool structure
	pool := &ResourcePool{
		TotalCPUs:       16.0,
		TotalMemory:     32768.0,
		TotalDisk:       500000.0,
		AvailableCPUs:   12.0,
		AvailableMemory: 24576.0,
		AvailableDisk:   400000.0,
		ReservedCPUs:    4.0,
		ReservedMemory:  8192.0,
		ReservedDisk:    100000.0,
	}

	assert.Equal(t, 16.0, pool.TotalCPUs)
	assert.Equal(t, 32768.0, pool.TotalMemory)
	assert.Equal(t, 500000.0, pool.TotalDisk)
	assert.Equal(t, 12.0, pool.AvailableCPUs)
	assert.Equal(t, 24576.0, pool.AvailableMemory)
	assert.Equal(t, 400000.0, pool.AvailableDisk)
	assert.Equal(t, 4.0, pool.ReservedCPUs)
	assert.Equal(t, 8192.0, pool.ReservedMemory)
	assert.Equal(t, 100000.0, pool.ReservedDisk)
}

func TestMaster_TaskStructures(t *testing.T) {
	now := time.Now()
	task := &Task{
		ID:          "task-1",
		Name:        "test-task",
		FrameworkID: "framework-1",
		AgentID:     "agent-1",
		State:       "running",
		Resources: &Resources{
			CPUs:   1.0,
			Memory: 1024.0,
			Disk:   1000.0,
		},
		Command: &Command{
			Value: "echo 'Hello World'",
			Shell: true,
			User:  "root",
		},
		Container: &Container{
			Type: "DOCKER",
			Docker: &DockerContainer{
				Image:   "nginx:latest",
				Network: "BRIDGE",
				PortMappings: []PortMapping{
					{ContainerPort: 80, HostPort: 0, Protocol: "tcp"},
				},
				Volumes: []Volume{
					{ContainerPath: "/var/log", HostPath: "/host/log", Mode: "RW"},
				},
			},
		},
		CreatedAt: now,
		StartedAt: now,
	}

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "test-task", task.Name)
	assert.Equal(t, "framework-1", task.FrameworkID)
	assert.Equal(t, "agent-1", task.AgentID)
	assert.Equal(t, "running", task.State)
	assert.Equal(t, 1.0, task.Resources.CPUs)
	assert.Equal(t, 1024.0, task.Resources.Memory)
	assert.Equal(t, 1000.0, task.Resources.Disk)
	assert.Equal(t, "echo 'Hello World'", task.Command.Value)
	assert.True(t, task.Command.Shell)
	assert.Equal(t, "root", task.Command.User)
	assert.Equal(t, "DOCKER", task.Container.Type)
	assert.Equal(t, "nginx:latest", task.Container.Docker.Image)
	assert.Equal(t, "BRIDGE", task.Container.Docker.Network)
	assert.Len(t, task.Container.Docker.PortMappings, 1)
	assert.Equal(t, 80, task.Container.Docker.PortMappings[0].ContainerPort)
	assert.Equal(t, 0, task.Container.Docker.PortMappings[0].HostPort)
	assert.Equal(t, "tcp", task.Container.Docker.PortMappings[0].Protocol)
	assert.Len(t, task.Container.Docker.Volumes, 1)
	assert.Equal(t, "/var/log", task.Container.Docker.Volumes[0].ContainerPath)
	assert.Equal(t, "/host/log", task.Container.Docker.Volumes[0].HostPath)
	assert.Equal(t, "RW", task.Container.Docker.Volumes[0].Mode)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, now, task.StartedAt)
}

func TestMaster_ResourceOfferStructures(t *testing.T) {
	now := time.Now()
	offer := &ResourceOffer{
		ID:        "offer-1",
		AgentID:   "agent-1",
		Resources: &Resources{CPUs: 4.0, Memory: 8192.0, Disk: 100000.0},
		FrameworkID: "framework-1",
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
	}

	assert.Equal(t, "offer-1", offer.ID)
	assert.Equal(t, "agent-1", offer.AgentID)
	assert.Equal(t, 4.0, offer.Resources.CPUs)
	assert.Equal(t, 8192.0, offer.Resources.Memory)
	assert.Equal(t, 100000.0, offer.Resources.Disk)
	assert.Equal(t, "framework-1", offer.FrameworkID)
	assert.Equal(t, now, offer.CreatedAt)
	assert.Equal(t, now.Add(5*time.Minute), offer.ExpiresAt)
}

func BenchmarkMaster_RegisterAgent(b *testing.B) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	for i := 0; i < b.N; i++ {
		agent := &AgentInfo{
			ID:       fmt.Sprintf("agent-%d", i),
			Hostname: "localhost",
			Port:     5051 + i,
			Resources: &Resources{
				CPUs:   4.0,
				Memory: 8192.0,
				Disk:   100000.0,
			},
			Tasks: make(map[string]*Task),
		}
		master.RegisterAgent(agent)
	}
}

func BenchmarkMaster_RegisterFramework(b *testing.B) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	for i := 0; i < b.N; i++ {
		framework := &Framework{
			ID:        fmt.Sprintf("framework-%d", i),
			Name:      "marathon",
			Principal: "marathon-principal",
			Role:      "*",
			Hostname:  "localhost",
			Port:      8080 + i,
		}
		master.RegisterFramework(framework)
	}
}

func BenchmarkMaster_LaunchTask(b *testing.B) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register agent
	agent := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: &Resources{
			CPUs:   4.0,
			Memory: 8192.0,
			Disk:   100000.0,
		},
		Tasks: make(map[string]*Task),
	}
	master.RegisterAgent(agent)

	// Register framework
	framework := &Framework{
		ID:        "framework-1",
		Name:      "marathon",
		Principal: "marathon-principal",
		Role:      "*",
		Hostname:  "localhost",
		Port:      8080,
	}
	master.RegisterFramework(framework)

	for i := 0; i < b.N; i++ {
		task := &Task{
			ID:          fmt.Sprintf("task-%d", i),
			Name:        "test-task",
			FrameworkID: "framework-1",
			AgentID:     "agent-1",
			State:       "pending",
			Resources: &Resources{
				CPUs:   1.0,
				Memory: 1024.0,
				Disk:   1000.0,
			},
		}
		master.LaunchTask(task)
	}
}

func BenchmarkMaster_HandleListAgents(b *testing.B) {
	master := NewMaster("test-master", "localhost", 5050, "zk://localhost:2181/mesos")

	// Register test agents
	for i := 0; i < 100; i++ {
		agent := &AgentInfo{
			ID:       fmt.Sprintf("agent-%d", i),
			Hostname: "localhost",
			Port:     5051 + i,
			Resources: &Resources{
				CPUs:   4.0,
				Memory: 8192.0,
				Disk:   100000.0,
			},
			Tasks: make(map[string]*Task),
		}
		master.RegisterAgent(agent)
	}

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		rr := httptest.NewRecorder()
		master.handleListAgents(rr, req)
	}
}