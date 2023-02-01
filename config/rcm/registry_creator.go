package rcm

import (
	"fmt"

	"github.com/flashbots/mev-boost/config/relay"
)

// proposerWalkerFn a helper used for traversing proposal config.
type proposerWalkerFn func(publicKey relay.ValidatorPublicKey, cfg relay.Relay) error

// RegistryCreator creates a new instance of Relay Registry.
//
// It invokes ConfigProvider to fetch proposer configuration, then validates the retrieved config,
// and finally creates a new instance of relay.Registry.
type RegistryCreator struct {
	configProvider ConfigProvider
}

// NewRegistryCreator creates a new instance of RegistryCreator.
func NewRegistryCreator(configProvider ConfigProvider) *RegistryCreator {
	if configProvider == nil {
		panic("configProvider is required, but nil")
	}

	return &RegistryCreator{
		configProvider: configProvider,
	}
}

// Create builds a relay.Registry from the proposer config retrieved from an RCP.
//
// It returns a valid instance of *relay.Registry on success.
// It returns an error if config is invalid.
func (r *RegistryCreator) Create() (*relay.Registry, error) {
	cfg, err := r.configProvider()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigProviderFailure, err)
	}

	if err := r.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%v: %w", ErrInvalidProposerConfig, err)
	}

	relayRegistry := relay.NewProposerRegistry()

	relayRegistry, err = r.populateProposerRelays(cfg, relayRegistry)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateProposerRelays, err)
	}

	relayRegistry, err = r.populateDefaultRelays(cfg, relayRegistry)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateDefaultRelays, err)
	}

	return relayRegistry, nil
}

// validateConfig check if the relay config is correct.
//
// It returns nil on success.
// It returns an error if the builder is true and there are no relays
// It returns an error If the builder is true and there are no default relays.
func (r *RegistryCreator) validateConfig(cfg *relay.Config) error {
	walker := func(publicKey relay.ValidatorPublicKey, cfg relay.Relay) error {
		if err := r.checkIfBuilderHasRelays(cfg.Builder); err != nil {
			return fmt.Errorf("%w: proposer %s", err, publicKey)
		}

		return nil
	}

	if err := r.walkProposerCfg(cfg.ProposerConfig, walker); err != nil {
		return err
	}

	if err := r.checkIfBuilderHasRelays(cfg.DefaultConfig.Builder); err != nil {
		return err
	}

	return nil
}

func (r *RegistryCreator) checkIfBuilderHasRelays(builder *relay.Builder) error {
	if builder == nil {
		return nil
	}

	if builder.Enabled && len(builder.Relays) < 1 {
		return ErrEmptyBuilderRelays
	}

	return nil
}

func (r *RegistryCreator) walkProposerCfg(proposerCfg relay.ProposerConfig, fn proposerWalkerFn) error {
	for validatorPubKey, cfg := range proposerCfg {
		if err := fn(validatorPubKey, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (r *RegistryCreator) populateProposerRelays(cfg *relay.Config, relayRegistry *relay.Registry) (*relay.Registry, error) {
	proposerCfgWalker := func(publicKey relay.ValidatorPublicKey, cfg relay.Relay) error {
		if cfg.Builder == nil {
			return nil
		}

		if !cfg.Builder.Enabled {
			relayRegistry.AddEmptyProposer(publicKey)

			return nil
		}

		walker := func(relay relay.Entry) {
			relayRegistry.AddRelayForProposer(publicKey, relay)
		}

		return r.processRelays(cfg, walker)
	}

	if err := r.walkProposerCfg(cfg.ProposerConfig, proposerCfgWalker); err != nil {
		return nil, err
	}

	return relayRegistry, nil
}

func (r *RegistryCreator) populateDefaultRelays(cfg *relay.Config, relayRegistry *relay.Registry) (*relay.Registry, error) {
	walker := func(relay relay.Entry) {
		relayRegistry.AddDefaultRelay(relay)
	}

	if err := r.processRelays(cfg.DefaultConfig, walker); err != nil {
		return nil, err
	}

	return relayRegistry, nil
}

func (r *RegistryCreator) processRelays(cfg relay.Relay, fn func(entry relay.Entry)) error {
	if cfg.Builder == nil {
		return nil
	}

	for _, relayURL := range cfg.Builder.Relays {
		relayEntry, err := relay.NewRelayEntry(relayURL)
		if err != nil {
			return fmt.Errorf("cannot process relays: %w", err)
		}

		fn(relayEntry)
	}

	return nil
}
