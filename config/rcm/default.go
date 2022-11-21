package rcm

import (
	"sync/atomic"

	"github.com/flashbots/mev-boost/config/relay"
)

type ConfigProvider func() (*relay.Config, error)

type Default struct {
	relayRegistry   atomic.Value
	registryCreator *relay.RegistryCreator
	configProvider  ConfigProvider
}

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

func (m *Default) SyncConfig() error {
	cfg, err := m.configProvider()
	if err != nil {
		return err
	}

	r, err := m.populateRegistry(cfg)
	if err != nil {
		return err
	}

	// atomically replace registry
	m.relayRegistry.Store(r)

	return nil
}

func (m *Default) populateRegistry(cfg *relay.Config) (*relay.Registry, error) {
	r, err := m.registryCreator.Create(cfg)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (m *Default) RelaysForValidator(publicKey relay.ValidatorPublicKey) relay.List {
	return m.loadRegistry().RelaysForValidator(publicKey).ToList()
}

func (m *Default) AllRelays() relay.List {
	return m.loadRegistry().AllRelays().ToList()
}

func (m *Default) loadRegistry() *relay.Registry {
	r, ok := m.relayRegistry.Load().(*relay.Registry)
	if !ok {
		panic("unexpected relay registry type") // this must never happen
	}

	return r
}
