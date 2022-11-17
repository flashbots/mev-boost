package rcm

import (
	"errors"

	"github.com/flashbots/mev-boost/config/rcp"
)

var ErrNoRelays = errors.New("no relays")

type DefaultConfigManager struct {
	relays []rcp.RelayEntry
}

func NewDefault(relays []rcp.RelayEntry) (*DefaultConfigManager, error) {
	if len(relays) == 0 {
		return nil, ErrNoRelays
	}

	return &DefaultConfigManager{relays: relays}, nil
}

func (m *DefaultConfigManager) RelaysByValidatorPublicKey(_ rcp.ValidatorPublicKey) ([]rcp.RelayEntry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) RelaysByValidatorIndex(_ rcp.ValidatorIndex) ([]rcp.RelayEntry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) AllRegisteredRelays() []rcp.RelayEntry {
	return m.relays
}
