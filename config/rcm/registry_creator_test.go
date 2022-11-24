package rcm_test

import (
	"io"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryCreator(t *testing.T) {
	t.Parallel()

	t.Run("it creates a valid registry with proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		proposerRelays := testutil.RandomRelaySet(t, 3)
		defaultRelays := testutil.RandomRelaySet(t, 2)
		want := testutil.JoinSets(proposerRelays, defaultRelays).ToList()

		configProvider := createMockRelayConfigProvider(
			withProposerRelays(validatorPublicKey.String(), proposerRelays.ToStringSlice()),
			withDefaultRelays(defaultRelays.ToStringSlice()))

		sut := rcm.NewRegistryCreator(configProvider)

		// act
		got, err := sut.Create()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, got.AllRelays().ToList(), want)
		assert.ElementsMatch(t, got.RelaysForValidator(validatorPublicKey.String()).ToList(), proposerRelays.ToList())
		assert.ElementsMatch(t, got.RelaysForValidator(
			testutil.RandomBLSPublicKey(t).String()).ToList(),
			defaultRelays.ToList())
	})

	t.Run("it skips disabled builders", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := createMockRelayConfigProvider(withSomeDisabledProposerBuilders(t))
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		got, err := sut.Create()

		// assert
		require.NoError(t, err)
		assert.Len(t, got.AllRelays(), 0)
	})

	t.Run("it returns an error if it cannot fetch relay config", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcm.NewRegistryCreator(createMockRelayConfigProvider(withErr()))

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrCannotFetchRelayConfig)
	})

	t.Run("it returns an error if a proposer builder is enabled but has not relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := createMockRelayConfigProvider(withProposerEnabledBuilderAndNoRelays(t))
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrEmptyBuilderRelays)
	})

	t.Run("it returns an error if a default builder is enabled but has not relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := createMockRelayConfigProvider(withDefaultEnabledBuilderAndNoRelays())
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrEmptyBuilderRelays)
	})

	t.Run("it returns an error if it cannot populate proposer relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := createMockRelayConfigProvider(withInvalidProposerRelays(t))
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrCannotPopulateProposerRelays)
	})

	t.Run("it returns an error if it cannot populate default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := createMockRelayConfigProvider(withInvalidDefaultRelays())
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrCannotPopulateDefaultRelays)
	})

	t.Run("it panics if config provider is not supplied", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			rcm.NewRegistryCreator(nil)
		})
	})
}

func onceOnlySuccessfulProvider(
	pubKey types.PublicKey, proposerRelays, defaultRelays relay.Set,
) rcm.ConfigProvider {
	calls := []rcm.ConfigProvider{
		createMockRelayConfigProvider(
			withProposerRelays(
				pubKey.String(),
				proposerRelays.ToStringSlice(),
			),
			withDefaultRelays(defaultRelays.ToStringSlice()),
		), // first call is successful
		createMockRelayConfigProvider(withErr()), // second call is an error
	}

	return createMockRelayConfigProvider(withCalls(calls))
}

func withErr() option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.err = io.ErrUnexpectedEOF
	}
}

func withProposerRelays(pubKey relay.ValidatorPublicKey, relays []string) option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.relayCfg.ProposerConfig = map[string]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
					Relays:  relays,
				},
			},
		}
	}
}

func withInvalidProposerRelays(t *testing.T) option {
	t.Helper()

	return func(cfg *mockRelayConfigProviderOptions) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
					Relays:  []string{"invalid-relay-url"},
				},
			},
		}
	}
}

func withDefaultRelays(relays []string) option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  relays,
		}
	}
}

func withInvalidDefaultRelays() option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  []string{"htt://fef/"},
		}
	}
}

func withProposerEnabledBuilderAndNoRelays(t *testing.T) option {
	t.Helper()

	return func(cfg *mockRelayConfigProviderOptions) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
				},
			},
		}
	}
}

func withSomeDisabledProposerBuilders(t *testing.T) option {
	t.Helper()

	return func(cfg *mockRelayConfigProviderOptions) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: false,
					Relays:  testutil.RandomRelaySet(t, 3).ToStringSlice(),
				},
			},
		}
	}
}

func withDefaultEnabledBuilderAndNoRelays() option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
		}
	}
}

func withCalls(calls []rcm.ConfigProvider) option {
	return func(cfg *mockRelayConfigProviderOptions) {
		cfg.calls = calls
	}
}

type option func(cfg *mockRelayConfigProviderOptions)

type mockRelayConfigProviderOptions struct {
	relayCfg *relay.Config
	err      error

	calls []rcm.ConfigProvider
}

func stubRelayConfigProvider(cfg *relay.Config, err error) (*relay.Config, error) {
	return cfg, err
}

func createMockRelayConfigProvider(opt ...option) func() (*relay.Config, error) {
	cfg := &mockRelayConfigProviderOptions{relayCfg: &relay.Config{}}
	for _, o := range opt {
		o(cfg)
	}

	curCall := 0

	return func() (*relay.Config, error) {
		if len(cfg.calls) > curCall {
			nextCall := cfg.calls[curCall]
			curCall++

			return nextCall()
		}

		return stubRelayConfigProvider(cfg.relayCfg, cfg.err)
	}
}
