package rcm

import (
	"sync"
	"sync/atomic"

	"github.com/flashbots/mev-boost/config/relay"
)

// ConfigProvider provider relay configuration.
type ConfigProvider func() (*relay.Config, error)

// Default is a general implementation for an RCM.
//
// It holds a thread-safe Relay Registry under the hood,
// which holds both proposer and default relays.
type Default struct {
	configProvider  ConfigProvider
	registryCreator *relay.RegistryCreator
	relayRegistry   atomic.Value
	relayRegistryMu sync.RWMutex
}

// NewDefault creates a new instance of Default.
//
// It creates a new instances and immediately synchronises the config with an RCP.
// It returns a newly created instance on success.
// It returns an error if config synchronisation fails.
func NewDefault(configProvider ConfigProvider) (*Default, error) {
	cm := &Default{
		configProvider:  configProvider,
		registryCreator: relay.NewRegistryCreator(),
	}

	if err := cm.SyncConfig(); err != nil {
		return nil, err
	}

	return cm, nil
}

// SyncConfig synchronises the Relay Registry with an RCP.
//
// It returns an error if it cannot fetch a config from an RCP.
// It returns an error if it cannot populate the new Relay Registry based on the config.
// If the config is valid and the new Relay Registry is populated,
// It atomically replaces the content of currently used Relay Registry with the new one.
func (m *Default) SyncConfig() error {
	cfg, err := m.configProvider()
	if err != nil {
		return err
	}

	m.relayRegistryMu.Lock()
	defer m.relayRegistryMu.Unlock()

	r, err := m.registryCreator.Create(cfg)
	if err != nil {
		return err
	}

	// atomically replace registry
	m.relayRegistry.Store(r)

	return nil
}

// RelaysForValidator looks up the Relay Registry to get a list of relay for the given public key.
func (m *Default) RelaysForValidator(publicKey relay.ValidatorPublicKey) relay.List {
	m.relayRegistryMu.RLock()
	defer m.relayRegistryMu.RUnlock()

	return m.loadRegistry().RelaysForValidator(publicKey).ToList()
}

// AllRelays retrieves a list of all unique relays from the Relay Registry.
func (m *Default) AllRelays() relay.List {
	m.relayRegistryMu.RLock()
	defer m.relayRegistryMu.RUnlock()

	return m.loadRegistry().AllRelays().ToList()
}

func (m *Default) loadRegistry() *relay.Registry {
	r, ok := m.relayRegistry.Load().(*relay.Registry)
	if !ok {
		panic("unexpected relay registry type") // this must never happen
	}

	return r
}
