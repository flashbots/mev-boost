package rcm_test

import (
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/server"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ server.RelayConfigManager = (*rcm.Default)(nil)

func TestDefaultConfigManager(t *testing.T) {
	t.Parallel()

	t.Run("it returns all relays for a known validator", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelaySet(t, 3)
		configProvider := configProviderWithProposerRelays(validatorPublicKey, want)

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.RelaysForValidator(validatorPublicKey.String())

		// assert
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it returns default relays for an unknown validator", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelaySet(t, 3)
		configProvider := configProviderWithDefaultRelays(want)

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.RelaysForValidator(validatorPublicKey.String())

		// assert
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it returns an error if config provider fails to retrieve configuration", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := faultyConfigProvider(relay.ErrMissingRelayPubKey)

		// act
		_, err := rcm.NewDefault(configProvider.FetchConfig)

		// assert
		assert.ErrorIs(t, err, relay.ErrMissingRelayPubKey)
	})

	t.Run("it returns proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		proposerRelays := testutil.RandomRelaySet(t, 3)
		defaultRelays := testutil.RandomRelaySet(t, 2)

		want := testutil.JoinSets(proposerRelays, defaultRelays).ToList()
		configProvider := integralConfigProvider(validatorPublicKey, proposerRelays, defaultRelays)

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		assert.ElementsMatch(t, want, got)
	})

	t.Run("it returns only unique relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		proposerRelays := testutil.RelaySetWithRelayHavingTheSameURL(t, 3)
		defaultRelays := testutil.RelaySetWithRelayHavingTheSameURL(t, 2)

		want := testutil.JoinSets(proposerRelays, defaultRelays).ToList()
		configProvider := integralConfigProvider(validatorPublicKey, proposerRelays, defaultRelays)

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		assert.ElementsMatch(t, want, got)
		assert.Len(t, got, 2)
	})

	t.Run("it uses the previously stored relays if synchronisation error occurs", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		proposerRelays := testutil.RandomRelaySet(t, 3)
		defaultRelays := testutil.RandomRelaySet(t, 2)

		configProvider := onceOnlySuccessfulProvider(validatorPublicKey, proposerRelays, defaultRelays)

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		err = sut.SyncConfig()

		// assert
		require.Error(t, err)
		assertRelaysHaveNotChanged(t, sut)(validatorPublicKey, proposerRelays)
		assertRelaysHaveNotChanged(t, sut)(testutil.RandomBLSPublicKey(t), defaultRelays)
	})

	t.Run("it panics if relay provider is not supplied", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			_, _ = rcm.NewDefault(nil)
		})
	})
}

func assertRelaysHaveNotChanged(t *testing.T, sut *rcm.Default) func(types.PublicKey, relay.Set) {
	t.Helper()

	return func(pk types.PublicKey, defaultRelays relay.Set) {
		got := sut.RelaysForValidator(pk.String())
		assert.ElementsMatch(t, defaultRelays.ToList(), got)
	}
}

func onceOnlySuccessfulProvider(
	pubKey types.PublicKey, proposerRelays, defaultRelays relay.Set,
) *mockConfigProvider {
	configProvider := &mockConfigProvider{}
	configProvider.FetchConfigFn = func() (*relay.Config, error) {
		if configProvider.WasCalledTimes > 1 {
			return nil, relay.ErrMissingRelayPubKey
		}

		return createTestConfig(
			withProposerRelays(pubKey.String(), proposerRelays.ToStringSlice()),
			withDefaultRelays(defaultRelays.ToStringSlice()))()
	}

	return configProvider
}

func integralConfigProvider(pk types.PublicKey, proposerRelays, defaultRelays relay.Set) mockConfigProvider {
	return mockConfigProvider{
		FetchConfigFn: createTestConfig(
			withProposerRelays(pk.String(), proposerRelays.ToStringSlice()),
			withDefaultRelays(defaultRelays.ToStringSlice())),
	}
}

func faultyConfigProvider(err error) mockConfigProvider {
	return mockConfigProvider{
		FetchConfigFn: func() (*relay.Config, error) {
			return nil, err
		},
	}
}

func configProviderWithDefaultRelays(defaultRelays relay.Set) mockConfigProvider {
	return mockConfigProvider{
		FetchConfigFn: createTestConfig(withDefaultRelays(defaultRelays.ToStringSlice())),
	}
}

func configProviderWithProposerRelays(validatorPublicKey types.PublicKey, proposerRelays relay.Set) mockConfigProvider {
	return mockConfigProvider{
		FetchConfigFn: createTestConfig(withProposerRelays(validatorPublicKey.String(), proposerRelays.ToStringSlice())),
	}
}

type option func(cfg *relay.Config)

func withProposerRelays(pubKey relay.ValidatorPublicKey, relays []string) option {
	return func(cfg *relay.Config) {
		cfg.ProposerConfig = map[string]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
					Relays:  relays,
				},
			},
		}
	}
}

func withDefaultRelays(relays []string) option {
	return func(cfg *relay.Config) {
		cfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  relays,
		}
	}
}

func createTestConfig(opt ...option) func() (*relay.Config, error) {
	cfg := &relay.Config{}
	for _, o := range opt {
		o(cfg)
	}

	return func() (*relay.Config, error) {
		return cfg, nil
	}
}

type mockConfigProvider struct {
	WasCalledTimes uint64
	FetchConfigFn  func() (*relay.Config, error)
}

func (m *mockConfigProvider) FetchConfig() (*relay.Config, error) {
	m.WasCalledTimes++

	return m.FetchConfigFn()
}
