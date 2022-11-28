package rcm_test

import (
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp/rcptest"
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

		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(validatorPublicKey.String(), proposerRelays.ToStringSlice()),
			rcptest.WithDefaultRelays(defaultRelays.ToStringSlice()))

		sut := rcm.NewRegistryCreator(configProvider)

		// act
		got, err := sut.Create()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, got.AllRelays(), want)
		assert.ElementsMatch(t, got.RelaysForValidator(validatorPublicKey.String()), proposerRelays.ToList())
		assert.ElementsMatch(t, got.RelaysForValidator(
			testutil.RandomBLSPublicKey(t).String()),
			defaultRelays.ToList())
	})

	t.Run("it skips disabled builders", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithSomeDisabledProposerBuilders(t))
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
		sut := rcm.NewRegistryCreator(rcptest.MockRelayConfigProvider(rcptest.WithErr()))

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrCannotFetchRelayConfig)
	})

	t.Run("it returns an error if a proposer builder is enabled but has not relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithProposerEnabledBuilderAndNoRelays(t))
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrEmptyBuilderRelays)
	})

	t.Run("it returns an error if a default builder is enabled but has not relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithDefaultEnabledBuilderAndNoRelays())
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrEmptyBuilderRelays)
	})

	t.Run("it returns an error if it cannot populate proposer relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithInvalidProposerRelays(t))
		sut := rcm.NewRegistryCreator(configProvider)

		// act
		_, err := sut.Create()

		// assert
		assert.ErrorIs(t, err, rcm.ErrCannotPopulateProposerRelays)
	})

	t.Run("it returns an error if it cannot populate default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithInvalidDefaultRelays())
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
		rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(
				pubKey.String(),
				proposerRelays.ToStringSlice(),
			),
			rcptest.WithDefaultRelays(defaultRelays.ToStringSlice()),
		), // first call is successful
		rcptest.MockRelayConfigProvider(rcptest.WithErr()), // second call is an error
	}

	return rcptest.MockRelayConfigProvider(rcptest.WithCalls(calls))
}
