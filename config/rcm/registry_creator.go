package rcm

import (
	"errors"
	"fmt"

	"github.com/flashbots/mev-boost/config/relay"
)

var (
	ErrCannotPopulateProposeRelays = errors.New("cannot populate proposer relays")
	ErrCannotPopulateDefaultRelays = errors.New("cannot populate default relays")
)

type RegistryCreator struct {
	relayRegistry *relay.Registry
}

func NewRegistryCreator() *RegistryCreator {
	return &RegistryCreator{relayRegistry: relay.NewProposerRegistry()}
}

func (r *RegistryCreator) Create(cfg *relay.Config) (*relay.Registry, error) {
	// TODO(screwyprof): should we allow creating registries with no relays in proposer_config section?
	// What about relay configs which have no proposer_config?
	//  1) Allow it, don't populate the registry and let the caller fallback to default relays
	//  2) Don't allow it, in which case, we will have to implement either
	// 	   a) custom config validators for diff cases
	//     b) implement different instances of RCM.
	//     c) custom registry creator for default config provider which doesn't have proposer_config
	//
	// m.validateConfig(cfg)

	if err := r.populateProposerRelays(cfg.ProposerConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateProposeRelays, err)
	}

	if err := r.populateDefaultRelays(cfg.DefaultConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateDefaultRelays, err)
	}

	return r.relayRegistry, nil
}

func (r *RegistryCreator) populateProposerRelays(proposerCfg relay.ProposerConfig) error {
	for validatorPubKey, cfg := range proposerCfg {
		err := r.populateRelays(cfg, func(relay relay.Entry) {
			r.relayRegistry.AddRelayForValidator(validatorPubKey, relay)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RegistryCreator) populateDefaultRelays(relayCfg relay.Relay) error {
	err := r.populateRelays(relayCfg, func(relay relay.Entry) {
		r.relayRegistry.AddDefaultRelay(relay)
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *RegistryCreator) populateRelays(cfg relay.Relay, fn func(entry relay.Entry)) error {
	if !cfg.Builder.Enabled {
		return nil
	}

	for _, relayURL := range cfg.Builder.Relays {
		relayEntry, err := relay.NewRelayEntry(relayURL)
		if err != nil {
			return err
		}

		fn(relayEntry)
	}

	return nil
}
