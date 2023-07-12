package rcm

import (
	"fmt"

	"github.com/flashbots/mev-boost/config/relay"
)

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
		return nil, fmt.Errorf("%w: %w", ErrConfigProviderFailure, err)
	}

	if err := validateProposerConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidProposerConfig, err)
	}

	relayRegistry := relay.NewProposerRegistry()

	relayRegistry, err = r.populateProposers(cfg, relayRegistry)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotPopulateProposerRelays, err)
	}

	relayRegistry, err = r.populateDefault(cfg, relayRegistry)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotPopulateDefaultRelays, err)
	}

	return relayRegistry, nil
}

func (r *RegistryCreator) populateProposers(cfg *relay.Config, relayRegistry *relay.Registry) (*relay.Registry, error) {
	for publicKey, proposerCfg := range cfg.ProposerConfig {
		if err := r.populateProposer(publicKey, proposerCfg, relayRegistry); err != nil {
			return nil, fmt.Errorf("%w: proposer %s", err, publicKey)
		}
	}

	return relayRegistry, nil
}

func (r *RegistryCreator) populateProposer(publicKey relay.ValidatorPublicKey, cfg relay.Relay, relayRegistry *relay.Registry) error {
	if cfg.Builder == nil {
		return nil
	}

	if !cfg.Builder.Enabled {
		relayRegistry.AddDisabledProposer(publicKey)

		return nil
	}

	walker := func(entry relay.Entry) {
		relayRegistry.AddRelayForProposer(publicKey, entry)
	}

	return r.processRelays(cfg, walker)
}

func (r *RegistryCreator) populateDefault(cfg *relay.Config, relayRegistry *relay.Registry) (*relay.Registry, error) {
	walker := func(entry relay.Entry) {
		relayRegistry.AddDefaultRelay(entry)
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

// validateProposerConfig checks if the relay config has correct data.
//
// It returns nil on success.
// It returns an error if a proposer.builder is enabled and there are no relays
// It returns an error If the default builder is enabled and there are no default relays.
func validateProposerConfig(cfg *relay.Config) error {
	if err := ensureProposersHaveRelays(cfg); err != nil {
		return err
	}

	return ensureBuilderHasRelays(cfg.DefaultConfig.Builder)
}

func ensureProposersHaveRelays(cfg *relay.Config) error {
	for publicKey, proposerCfg := range cfg.ProposerConfig {
		if err := ensureBuilderHasRelays(proposerCfg.Builder); err != nil {
			return fmt.Errorf("%w: proposer %s", err, publicKey)
		}
	}

	return nil
}

func ensureBuilderHasRelays(builder *relay.Builder) error {
	if builder == nil {
		return nil
	}

	if builder.Enabled && len(builder.Relays) < 1 {
		return ErrEmptyBuilderRelays
	}

	return nil
}
