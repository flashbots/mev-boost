package rcm

import "github.com/flashbots/mev-boost/server"

type ValidatorPublicKey = string

type RelayConfigProvider interface {
	RelaysByValidatorPublicKey(publicKey ValidatorPublicKey) ([]server.RelayEntry, error)
}

type RelayConfigManager struct {
	configProvider RelayConfigProvider
}

func New(configProvider RelayConfigProvider) *RelayConfigManager {
	return &RelayConfigManager{configProvider: configProvider}
}

func (m *RelayConfigManager) RelaysByValidatorPublicKey(publicKey ValidatorPublicKey) ([]server.RelayEntry, error) {
	return m.configProvider.RelaysByValidatorPublicKey(publicKey)
}
