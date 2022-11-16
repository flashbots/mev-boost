package rcp

type DefaultConfigProvider struct {
	relays []RelayEntry
}

func NewDefaultConfigProvider(relays []RelayEntry) *DefaultConfigProvider {
	return &DefaultConfigProvider{relays: relays}
}

func (p *DefaultConfigProvider) RelaysByValidatorPublicKey(_ ValidatorPublicKey) ([]RelayEntry, error) {
	return p.relays, nil
}
