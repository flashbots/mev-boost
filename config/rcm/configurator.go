package rcm

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/flashbots/mev-boost/config/relay"
)

var ErrCannotCreateConfigurator = errors.New("cannot create configurator")

// Configurator is a general implementation for an RCM.
//
// It holds a thread-safe Relay Registry under the hood,
// which holds both proposer and default relays.
type Configurator struct {
	registryCreator *RegistryCreator
	relayRegistry   atomic.Value
}

// New creates a new instance of Configurator.
//
// It synchronises the config via Configurator.SyncConfig.
// It returns a newly created instance on success.
// It returns an error if config synchronisation fails.
func New(registryCreator *RegistryCreator) (*Configurator, error) {
	cm := &Configurator{
		registryCreator: registryCreator,
	}

	if registryCreator == nil {
		return nil, fmt.Errorf("%w: registry creator is required and cannot be nil", ErrCannotCreateConfigurator)
	}

	if err := cm.SyncConfig(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotCreateConfigurator, err)
	}

	return cm, nil
}

// SyncConfig synchronises the Relay Registry with an RCP.
//
// It returns an error if it cannot create a new relay.Registry.
// It atomically replaces the content of currently used Relay Registry with the new one on success.
func (m *Configurator) SyncConfig() error {
	r, err := m.registryCreator.Create()
	if err != nil {
		return fmt.Errorf("cannot syncronise configuration: %w", err)
	}

	// atomically replace registry
	m.relayRegistry.Store(r)

	return nil
}

// RelaysForProposer looks up the Relay Registry to get a list of relay for the given public key.
func (m *Configurator) RelaysForProposer(publicKey relay.ValidatorPublicKey) relay.List {
	return m.loadRegistry().RelaysForProposer(publicKey)
}

// AllRelays retrieves a list of all unique relays from the Relay Registry.
func (m *Configurator) AllRelays() relay.List {
	return m.loadRegistry().AllRelays()
}

func (m *Configurator) loadRegistry() RelayRegistry {
	r, ok := m.relayRegistry.Load().(RelayRegistry)
	if !ok {
		panic("unexpected relay registry type")
	}

	return r
}