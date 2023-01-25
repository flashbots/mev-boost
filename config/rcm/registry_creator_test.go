package rcm_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp/rcptest"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure *relay.Registry implements rcm.RelayRegistry interface.
var _ rcm.RelayRegistry = (*relay.Registry)(nil)

func TestRegistryCreator(t *testing.T) {
	t.Parallel()

	t.Run("it creates a valid registry with proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		proposerRelays := reltest.RandomRelaySet(t, 3)
		defaultRelays := reltest.RandomRelaySet(t, 2)
		want := reltest.JoinSets(proposerRelays, defaultRelays).ToList()

		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(validatorPublicKey.String(), proposerRelays),
			rcptest.WithDefaultRelays(defaultRelays))

		sut := rcm.NewRegistryCreator(configProvider)

		// act
		got, err := sut.Create()

		// assert
		require.NoError(t, err)
		assert.ElementsMatch(t, got.AllRelays(), want)
		assert.ElementsMatch(t, got.RelaysForProposer(validatorPublicKey.String()), proposerRelays.ToList())
		assert.ElementsMatch(t, got.RelaysForProposer(
			reltest.RandomBLSPublicKey(t).String()),
			defaultRelays.ToList())
	})

	t.Run("it creates a valid registry with two different proposers and no default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		proposer1 := reltest.RandomBLSPublicKey(t)
		proposer2 := reltest.RandomBLSPublicKey(t)

		proposer1Relays := reltest.RandomRelaySet(t, 3)
		proposer2Relays := reltest.RandomRelaySet(t, 2)

		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(proposer1.String(), proposer1Relays),
			rcptest.WithProposerRelays(proposer2.String(), proposer2Relays),
		)

		sut := rcm.NewRegistryCreator(configProvider)

		// act
		got, err := sut.Create()

		// assert
		require.NoError(t, err)

		assertRegistryHasProposerRelays(t, got)(proposer1.String(), proposer1Relays)
		assertRegistryHasProposerRelays(t, got)(proposer2.String(), proposer2Relays)
		assertRegistryHasAllRelays(t, got)(proposer1Relays, proposer2Relays)
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
		assert.ErrorIs(t, err, rcm.ErrConfigProviderFailure)
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

func assertRegistryHasAllRelays(t *testing.T, got rcm.RelayRegistry) func(...relay.Set) {
	t.Helper()

	return func(relays ...relay.Set) {
		assert.ElementsMatch(t, reltest.JoinSets(relays...).ToStringSlice(), got.AllRelays().ToStringSlice())
	}
}

func assertRegistryHasProposerRelays(t *testing.T, sut rcm.RelayRegistry) func(relay.ValidatorPublicKey, relay.Set) {
	t.Helper()

	return func(pk relay.ValidatorPublicKey, want relay.Set) {
		assert.ElementsMatch(t, want.ToStringSlice(), sut.RelaysForProposer(pk).ToStringSlice())
	}
}
