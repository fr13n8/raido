package agent

import (
	"sync"
)

var manager *Manager

type Manager struct {
	agents map[string]*Agent
	mu     sync.Mutex
}

var once sync.Once

func NewAgentManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			agents: make(map[string]*Agent),
		}
	})

	return manager
}

func (cm *Manager) GetAgent(id string) *Agent {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.agents[id]
}

func (cm *Manager) RemoveAgent(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.agents, id)
}

func (cm *Manager) AddAgent(id string, conn *Agent) *Agent {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.agents[id] = conn

	return cm.agents[id]
}

func (cm *Manager) GetAgents() map[string]*Agent {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.agents
}
