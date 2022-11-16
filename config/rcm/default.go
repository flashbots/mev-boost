package rcm

import "errors"

var ErrNoRelays = errors.New("no relays")

type DefaultConfigManager struct {
	relays []RelayEntry
}

func NewDefault(relays []RelayEntry) (*DefaultConfigManager, error) {
	if len(relays) == 0 {
		return nil, ErrNoRelays
	}

	return &DefaultConfigManager{relays: relays}, nil
}

func (m *DefaultConfigManager) RelaysByValidatorPublicKey(_ ValidatorPublicKey) ([]RelayEntry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) RelaysByValidatorIndex(_ ValidatorIndex) ([]RelayEntry, error) {
	return m.relays, nil
}

func (m *DefaultConfigManager) AllRegisteredRelays() []RelayEntry {
	return m.relays
}
