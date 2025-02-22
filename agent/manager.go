package agent

import (
	"errors"
	"fmt"
	"sync"
)

type Manager struct {
	agents  map[string]*Agent
	rwMutex sync.RWMutex
}

var once sync.Once
var instance *Manager

func NewAgentManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			agents: make(map[string]*Agent),
		}
	})

	return instance
}

func (m *Manager) GetAgent(id string) *Agent {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()

	return m.agents[id]
}

func (m *Manager) RemoveAgent(id string) error {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	a, ok := m.agents[id]
	if !ok {
		return nil
	}

	if err := a.Close(); err != nil {
		return fmt.Errorf("failed to close agent: %w", err)
	}

	delete(m.agents, id)

	return nil
}

func (m *Manager) AddAgent(a *Agent) *Agent {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	m.agents[a.ID] = a

	return m.agents[a.ID]
}

func (m *Manager) Cleanup() error {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	var errs []error

	wg := sync.WaitGroup{}
	for id, a := range m.agents {
		wg.Add(1)
		go func(id string, a *Agent) {
			defer wg.Done()
			if err := a.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close agent %s: %s", id, err))
			}
		}(id, a)

		delete(m.agents, id)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %w", errors.Join(errs...))
	}

	return nil
}

func (cm *Manager) GetAllAgents() map[string]*Agent {
	cm.rwMutex.Lock()
	defer cm.rwMutex.Unlock()

	return cm.agents
}
