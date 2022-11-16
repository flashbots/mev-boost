package rcm

import (
	"github.com/flashbots/mev-boost/config/rcp"
)

type RelayConfigManager struct {
	configProvider rcp.RelayConfigProvider
}

func New(configProvider rcp.RelayConfigProvider) *RelayConfigManager {
	return &RelayConfigManager{configProvider: configProvider}
}

func (m *RelayConfigManager) RelaysByValidatorPublicKey(publicKey rcp.ValidatorPublicKey) ([]rcp.RelayEntry, error) {
	return m.configProvider.RelaysByValidatorPublicKey(publicKey)
}
