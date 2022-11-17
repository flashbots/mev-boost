package rcm

import (
	"errors"

	"github.com/flashbots/mev-boost/config/relay"
)

type (
	ValidatorPublicKey = string
	ValidatorIndex     = uint64
)

var ErrNoRelays = errors.New("no relays")

type DefaultConfigManager struct {
	relays []relay.Entry
}

func NewDefault(relays []relay.Entry) (*DefaultConfigManager, error) {
	if len(relays) == 0 {
		return nil, ErrNoRelays
	}

	return &DefaultConfigManager{relays: relays}, nil
}

func (m *DefaultConfigManager) RelaysByValidatorPublicKey(publicKey ValidatorPublicKey) ([]relay.Entry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) RelaysByValidatorIndex(validatorIndex ValidatorIndex) ([]relay.Entry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) AllRegisteredRelays() []relay.Entry {
	return m.relays
}
