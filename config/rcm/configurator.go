package rcm

import (
	"sync"
	"sync/atomic"

	"github.com/flashbots/mev-boost/config/relay"
)

// Configurator is a general implementation for an RCM.
//
// It holds a thread-safe Relay Registry under the hood,
// which holds both proposer and default relays.
type Configurator struct {
	registryCreator *RegistryCreator
	relayRegistry   atomic.Value
	relayRegistryMu sync.RWMutex
}

// NewDefault creates a new instance of Configurator.
//
// It synchronises the config via Configurator.SyncConfig.
// It returns a newly created instance on success.
// It returns an error if config synchronisation fails.
func NewDefault(registryCreator *RegistryCreator) (*Configurator, error) {
	cm := &Configurator{
		registryCreator: registryCreator,
	}

	if registryCreator == nil {
		panic("registryCreator is required")
	}

	if err := cm.SyncConfig(); err != nil {
		return nil, err
	}

	return cm, nil
}

// SyncConfig synchronises the Relay Registry with an RCP.
//
// It returns an error if it cannot create the new Relay Registry.
// It atomically replaces the content of currently used Relay Registry with the new one on success.
func (m *Configurator) SyncConfig() error {
	m.relayRegistryMu.Lock()
	defer m.relayRegistryMu.Unlock()

	r, err := m.registryCreator.Create()
	if err != nil {
		return err
	}

	// atomically replace registry
	m.relayRegistry.Store(r)

	return nil
}

// RelaysForValidator looks up the Relay Registry to get a list of relay for the given public key.
func (m *Configurator) RelaysForValidator(publicKey relay.ValidatorPublicKey) relay.List {
	m.relayRegistryMu.RLock()
	defer m.relayRegistryMu.RUnlock()

	return m.loadRegistry().RelaysForValidator(publicKey)
}

// AllRelays retrieves a list of all unique relays from the Relay Registry.
func (m *Configurator) AllRelays() relay.List {
	m.relayRegistryMu.RLock()
	defer m.relayRegistryMu.RUnlock()

	return m.loadRegistry().AllRelays()
}

func (m *Configurator) loadRegistry() RelayRegistry {
	r, ok := m.relayRegistry.Load().(RelayRegistry)
	if !ok {
		panic("unexpected relay registry type") // this must never happen
	}

	return r
}
