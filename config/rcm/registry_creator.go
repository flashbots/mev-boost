package rcm

import (
	"errors"
	"fmt"

	"github.com/flashbots/mev-boost/config/relay"
)

var (
	ErrConfigProviderFailure        = errors.New("config provider failure")
	ErrInvalidProposerConfig        = errors.New("invalid proposer config")
	ErrEmptyBuilderRelays           = errors.New("builder is enabled but has no relays")
	ErrCannotPopulateProposerRelays = errors.New("cannot populate proposer relays")
	ErrCannotPopulateDefaultRelays  = errors.New("cannot populate default relays")
)

// ConfigProvider provider relay configuration.
type ConfigProvider func() (*relay.Config, error)
type proposerWalkerFn func(publicKey relay.ValidatorPublicKey, cfg relay.Relay) error

// RegistryCreator creates Relay Registries.
//
// The default and/or proposer relays may be empty.
// It ignores the disabled builders and their relays.
// If the config provider fetches a proposer config with a builder enabled
// and with no relays, the registry won't be created.
type RegistryCreator struct {
	configProvider ConfigProvider
	relayRegistry  *relay.Registry
}

// NewRegistryCreator creates a new instance of RegistryCreator.
func NewRegistryCreator(configProvider ConfigProvider) *RegistryCreator {
	if configProvider == nil {
		panic("configProvider is required, but nil")
	}

	return &RegistryCreator{
		configProvider: configProvider,
		relayRegistry:  relay.NewProposerRegistry(),
	}
}

// Create builds a Relay Registry from the proposer config retrieved from an RCP.
//
// It returns a valid instance of *relay.Registry on success.
// It returns an error if config is invalid.
// It ignores the disabled builders and their relays.
func (r *RegistryCreator) Create() (*relay.Registry, error) {
	cfg, err := r.configProvider()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigProviderFailure, err)
	}

	if err := r.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%v: %w", ErrInvalidProposerConfig, err)
	}

	if err := r.populateProposerRelays(cfg.ProposerConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateProposerRelays, err)
	}

	if err := r.populateDefaultRelays(cfg.DefaultConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCannotPopulateDefaultRelays, err)
	}

	return r.relayRegistry, nil
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

func (r *RegistryCreator) checkIfBuilderHasRelays(builder relay.Builder) error {
	if builder.Enabled && len(builder.Relays) < 1 {
		return ErrEmptyBuilderRelays
	}

	return nil
}

func (r *RegistryCreator) walkProposerCfg(proposerCfg relay.ProposerConfig, walker proposerWalkerFn) error {
	for validatorPubKey, cfg := range proposerCfg {
		return walker(validatorPubKey, cfg)
	}

	return nil
}

func (r *RegistryCreator) populateProposerRelays(proposerCfg relay.ProposerConfig) error {
	return r.walkProposerCfg(proposerCfg, func(publicKey relay.ValidatorPublicKey, cfg relay.Relay) error {
		return r.populateRelays(cfg, func(relay relay.Entry) {
			r.relayRegistry.AddRelayForValidator(publicKey, relay)
		})
	})
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
