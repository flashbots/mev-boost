package rcp

import "github.com/flashbots/mev-boost/server"

type ValidatorPublicKey = string

type DefaultConfigProvider struct {
	relays []server.RelayEntry
}

func NewDefaultConfigProvider(relays []server.RelayEntry) *DefaultConfigProvider {
	return &DefaultConfigProvider{relays: relays}
}

func (p *DefaultConfigProvider) RelaysByValidatorPublicKey(_ ValidatorPublicKey) ([]server.RelayEntry, error) {
	return p.relays, nil
}
