package rcm_test

import (
	"testing"

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

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(WithProposerRelays(validatorPublicKey.String(), want.ToStringSlice())),
		}

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.RelaysForValidator(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it returns default relays for an unknown validator", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelaySet(t, 3)

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(WithDefaultRelays(want.ToStringSlice())),
		}

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.RelaysForValidator(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it returns an error if config provider fails to retrieve configuration", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := MockConfigProvider{
			FetchConfigFn: func() (*relay.Config, error) {
				return nil, relay.ErrMissingRelayPubKey
			},
		}

		// act
		_, err := rcm.NewDefault(configProvider.FetchConfig)

		// assert
		assert.ErrorIs(t, err, relay.ErrMissingRelayPubKey)
	})

	t.Run("it returns an error it cannot populate proposer registry", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		malformedRelays := []string{"https://relay-with-no-pub-key.com"}

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(WithProposerRelays(validatorPublicKey.String(), malformedRelays)),
		}

		// act
		_, err := rcm.NewDefault(configProvider.FetchConfig)

		// assert
		assert.ErrorIs(t, err, relay.ErrCannotPopulateProposeRelays)
	})

	t.Run("it returns an error if it cannot populate default registry", func(t *testing.T) {
		t.Parallel()

		// arrange
		malformedRelays := []string{"https://relay-with-no-pub-key.com"}

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(WithDefaultRelays(malformedRelays)),
		}

		// act
		_, err := rcm.NewDefault(configProvider.FetchConfig)

		// assert
		assert.ErrorIs(t, err, relay.ErrCannotPopulateDefaultRelays)
	})

	t.Run("it returns only default relays if no proposer relays populated", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := testutil.RandomRelaySet(t, 3)

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(WithDefaultRelays(want.ToStringSlice())),
		}

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it returns proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		proposerRelays := testutil.RandomRelaySet(t, 3)
		defaultRelays := testutil.RandomRelaySet(t, 2)

		want := make(relay.List, 0, 5)
		want = append(want, proposerRelays.ToList()...)
		want = append(want, defaultRelays.ToList()...)

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(
				WithProposerRelays(validatorPublicKey.String(), proposerRelays.ToStringSlice()),
				WithDefaultRelays(defaultRelays.ToStringSlice())),
		}

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, want, got)
	})

	t.Run("it returns only unique relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelaySet(t, 3)

		configProvider := MockConfigProvider{
			FetchConfigFn: CreateTestConfig(
				WithProposerRelays(validatorPublicKey.String(), want.ToStringSlice()),
				WithDefaultRelays(want.ToStringSlice())),
		}

		sut, err := rcm.NewDefault(configProvider.FetchConfig)
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, want.ToList(), got)
	})

	t.Run("it panics if relay provider is not supplied", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			_, _ = rcm.NewDefault(nil)
		})
	})
}

type Option func(cfg *relay.Config)

func WithProposerRelays(pubKey relay.ValidatorPublicKey, relays []string) Option {
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

func WithDefaultRelays(relays []string) Option {
	return func(cfg *relay.Config) {
		cfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  relays,
		}
	}
}

func CreateTestConfig(opt ...Option) func() (*relay.Config, error) {
	cfg := &relay.Config{}
	for _, o := range opt {
		o(cfg)
	}

	return func() (*relay.Config, error) {
		return cfg, nil
	}
}

type MockConfigProvider struct {
	FetchConfigFn func() (*relay.Config, error)
}

func (m MockConfigProvider) FetchConfig() (*relay.Config, error) {
	return m.FetchConfigFn()
}
