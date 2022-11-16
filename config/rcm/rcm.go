package rcm

import (
	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/server"
)

type RelayConfigProvider interface {
	RelaysByValidatorPublicKey(publicKey rcp.ValidatorPublicKey) ([]server.RelayEntry, error)
}

type RelayConfigManager struct {
	configProvider RelayConfigProvider
}

func New(configProvider RelayConfigProvider) *RelayConfigManager {
	return &RelayConfigManager{configProvider: configProvider}
}

func (m *RelayConfigManager) RelaysByValidatorPublicKey(publicKey rcp.ValidatorPublicKey) ([]server.RelayEntry, error) {
	return m.configProvider.RelaysByValidatorPublicKey(publicKey)
}
