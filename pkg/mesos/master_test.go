package mesos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMaster(t *testing.T) {
	// Test creating a new Mesos master
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	assert.NotNil(t, master)
	assert.Equal(t, "master-1", master.ID)
	assert.Equal(t, "localhost", master.Hostname)
	assert.Equal(t, 5050, master.Port)
	assert.Equal(t, "localhost:2181", master.ZookeeperURL)
	assert.False(t, master.IsLeader)
}

func TestMaster_Start(t *testing.T) {
	// Test starting Mesos master
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	err := master.Start()
	assert.NoError(t, err)
	
	// Test stopping master
	err = master.Stop()
	assert.NoError(t, err)
}

func TestMaster_Stop(t *testing.T) {
	// Test stopping Mesos master
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	err := master.Stop()
	assert.NoError(t, err)
}

func TestMaster_RegisterAgent(t *testing.T) {
	// Test registering agent
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Verify agent was registered
	agents, err := master.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "agent-1", agents[0].ID)
}

func TestMaster_DeregisterAgent(t *testing.T) {
	// Test deregistering agent
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent first
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Deregister agent
	err = master.DeregisterAgent("agent-1")
	assert.NoError(t, err)
	
	// Verify agent was deregistered
	agents, err := master.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 0)
}

func TestMaster_ListAgents(t *testing.T) {
	// Test listing agents
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register some agents
	agent1 := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	agent2 := &AgentInfo{
		ID:       "agent-2",
		Hostname: "localhost",
		Port:     5052,
		Resources: Resource{
			CPUs:  2.0,
			Memory: 4096,
			Disk:   50000,
		},
	}
	
	err := master.RegisterAgent(agent1)
	assert.NoError(t, err)
	
	err = master.RegisterAgent(agent2)
	assert.NoError(t, err)
	
	// List agents
	agents, err := master.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 2)
	
	// Verify agent IDs
	agentIDs := make([]string, len(agents))
	for i, agent := range agents {
		agentIDs[i] = agent.ID
	}
	assert.Contains(t, agentIDs, "agent-1")
	assert.Contains(t, agentIDs, "agent-2")
}

func TestMaster_GetAgent(t *testing.T) {
	// Test getting agent
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Get agent
	agent, err := master.GetAgent("agent-1")
	assert.NoError(t, err)
	assert.Equal(t, "agent-1", agent.ID)
	assert.Equal(t, "localhost", agent.Hostname)
	assert.Equal(t, 5051, agent.Port)
	assert.Equal(t, 4.0, agent.Resources.CPUs)
	assert.Equal(t, 8192, agent.Resources.Memory)
	assert.Equal(t, 100000, agent.Resources.Disk)
	
	// Test getting non-existent agent
	_, err = master.GetAgent("nonexistent")
	assert.Error(t, err)
}

func TestMaster_AllocateResources(t *testing.T) {
	// Test resource allocation
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Allocate resources
	offer, err := master.AllocateResources(Resource{
		CPUs:  2.0,
		Memory: 4096,
		Disk:   50000,
	})
	assert.NoError(t, err)
	assert.NotNil(t, offer)
	assert.Equal(t, "agent-1", offer.AgentID)
	assert.Equal(t, 2.0, offer.Resources.CPUs)
	assert.Equal(t, 4096, offer.Resources.Memory)
	assert.Equal(t, 50000, offer.Resources.Disk)
}

func TestMaster_LaunchTask(t *testing.T) {
	// Test launching task
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Launch task
	task := Task{
		ID:        "task-1",
		Name:      "Test Task",
		Image:     "nginx:latest",
		CPUs:      1.0,
		Memory:    1024,
		Disk:      10000,
	}
	
	err = master.LaunchTask(task)
	assert.NoError(t, err)
	
	// Verify task was launched
	state, err := master.GetState()
	assert.NoError(t, err)
	assert.Len(t, state.Tasks, 1)
	assert.Equal(t, "task-1", state.Tasks[0].ID)
}

func TestMaster_KillTask(t *testing.T) {
	// Test killing task
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Launch task
	task := Task{
		ID:        "task-1",
		Name:      "Test Task",
		Image:     "nginx:latest",
		CPUs:      1.0,
		Memory:    1024,
		Disk:      10000,
	}
	
	err = master.LaunchTask(task)
	assert.NoError(t, err)
	
	// Kill task
	err = master.KillTask("task-1")
	assert.NoError(t, err)
	
	// Verify task was killed
	state, err := master.GetState()
	assert.NoError(t, err)
	assert.Len(t, state.Tasks, 0)
}

func TestMaster_GetState(t *testing.T) {
	// Test getting cluster state
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Register agent
	agentInfo := &AgentInfo{
		ID:       "agent-1",
		Hostname: "localhost",
		Port:     5051,
		Resources: Resource{
			CPUs:  4.0,
			Memory: 8192,
			Disk:   100000,
		},
	}
	
	err := master.RegisterAgent(agentInfo)
	assert.NoError(t, err)
	
	// Get state
	state, err := master.GetState()
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "master-1", state.MasterID)
	assert.Len(t, state.Agents, 1)
	assert.Len(t, state.Tasks, 0)
}

func TestMaster_ErrorHandling(t *testing.T) {
	// Test error handling
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Test invalid operations
	err := master.DeregisterAgent("nonexistent")
	assert.Error(t, err)
	
	_, err = master.GetAgent("nonexistent")
	assert.Error(t, err)
	
	err = master.KillTask("nonexistent")
	assert.Error(t, err)
}

func TestMaster_ConcurrentAccess(t *testing.T) {
	// Test concurrent access
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Test concurrent agent registrations
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(i int) {
			defer func() { done <- true }()
			agentInfo := &AgentInfo{
				ID:       "agent" + string(rune(i)),
				Hostname: "localhost",
				Port:     5051 + i,
				Resources: Resource{
					CPUs:  2.0,
					Memory: 4096,
					Disk:   50000,
				},
			}
			err := master.RegisterAgent(agentInfo)
			assert.NoError(t, err)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
	
	// Verify all agents were registered
	agents, err := master.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 5)
}

func TestMaster_Performance(t *testing.T) {
	// Test performance
	master := NewMaster("master-1", "localhost", 5050, "localhost:2181")
	
	// Test registering many agents
	for i := 0; i < 100; i++ {
		agentInfo := &AgentInfo{
			ID:       "agent" + string(rune(i)),
			Hostname: "localhost",
			Port:     5051 + i,
			Resources: Resource{
				CPUs:  2.0,
				Memory: 4096,
				Disk:   50000,
			},
		}
		err := master.RegisterAgent(agentInfo)
		assert.NoError(t, err)
	}
	
	// Verify all agents were registered
	agents, err := master.ListAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, 100)
}
