package rcm_test

import (
	"math/rand"
	"runtime"
	"sync"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/config/rcp/rcptest"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure *rcm.Configurator implements rcm.RelayRegistry interface.
var _ rcm.RelayRegistry = (*rcm.Configurator)(nil)

func TestConfigurator(t *testing.T) {
	t.Parallel()

	t.Run("proposer is not found & default with no relays is disabled -> empty", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithDisabledEmptyDefaultRelays())

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, emptyRelayList(), got)
	})

	t.Run("proposer is not found & default with relays is enabled -> default", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		want := reltest.RandomRelaySet(t, 3)
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithDefaultRelays(want))

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, want.ToList(), got)
	})

	t.Run("proposer with no builder & default with no relays is disabled -> empty", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerWithNoBuilder(validatorPublicKey.String()),
			rcptest.WithDisabledEmptyDefaultRelays())

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, emptyRelayList(), got)
	})

	t.Run("proposer with no relays is disabled & default with no relays is disabled -> empty", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithDisabledEmptyProposer(validatorPublicKey.String()),
			rcptest.WithDisabledEmptyDefaultRelays())

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, emptyRelayList(), got)
	})

	t.Run("proposer with no builder & default with relays is enabled -> default", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		want := reltest.RandomRelaySet(t, 3)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerWithNoBuilder(validatorPublicKey.String()),
			rcptest.WithDefaultRelays(want))

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, want.ToList(), got)
	})

	t.Run("proposer with no relays is disabled & default with relays is enabled -> empty", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithDisabledEmptyProposer(validatorPublicKey.String()),
			rcptest.WithDisabledDefaultRelays(relay.NewRelaySet()))

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, emptyRelayList(), got)
	})

	t.Run("proposer with relays is enabled & default with relays is enabled -> proposer", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelaySet(t, 3)

		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(validatorPublicKey.String(), want),
			rcptest.WithDefaultRelays(reltest.RandomRelaySet(t, 3)))

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, want.ToList(), got)
	})

	t.Run("proposer with relays is enabled & default with no relays is disabled -> proposer", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelaySet(t, 3)

		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(validatorPublicKey.String(), want),
			rcptest.WithDisabledEmptyDefaultRelays())

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.RelaysForProposer(validatorPublicKey.String())

		// assert
		assertRelayListsMatch(t, want.ToList(), got)
	})

	t.Run("it returns an error if it cannot create the registry", func(t *testing.T) {
		t.Parallel()

		// arrange
		configProvider := rcptest.MockRelayConfigProvider(rcptest.WithErr())

		// act
		_, err := rcm.New(rcm.NewRegistryCreator(configProvider))

		// assert
		assert.ErrorIs(t, err, rcm.ErrConfigProviderFailure)
	})

	t.Run("it returns only unique relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		proposerRelays := reltest.RelaySetWithRelaysHavingTheSameURL(t, 3)
		defaultRelays := reltest.RelaySetWithRelaysHavingTheSameURL(t, 2)

		want := reltest.JoinSets(proposerRelays, defaultRelays).ToList()
		configProvider := rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(validatorPublicKey.String(), proposerRelays),
			rcptest.WithDefaultRelays(defaultRelays))

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		got := sut.AllRelays()

		// assert
		assertRelayListsMatch(t, want, got)
		assert.Len(t, got, 2)
	})

	t.Run("it uses the previously stored relays if synchronisation error occurs", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := reltest.RandomBLSPublicKey(t)
		proposerRelays := reltest.RandomRelaySet(t, 3)
		defaultRelays := reltest.RandomRelaySet(t, 2)

		configProvider := onceOnlySuccessfulProvider(validatorPublicKey, proposerRelays, defaultRelays)

		sut, err := rcm.New(rcm.NewRegistryCreator(configProvider))
		require.NoError(t, err)

		// act
		err = sut.SyncConfig()

		// assert
		require.Error(t, err)
		assertRelaysHaveNotChanged(t, sut)(validatorPublicKey, proposerRelays)
		assertRelaysHaveNotChanged(t, sut)(reltest.RandomBLSPublicKey(t), defaultRelays)
	})

	t.Run("it panics if relay provider is not supplied", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			_, _ = rcm.New(nil)
		})
	})

	t.Run("it is thread-safe", func(t *testing.T) {
		t.Parallel()

		relays := reltest.RandomRelaySet(t, 5)

		sut, err := rcm.New(rcm.NewRegistryCreator(rcp.NewDefault(relays).FetchConfig))
		require.NoError(t, err)

		assertWasRunConCurrently(t, sut)
	})
}

func emptyRelayList() relay.List {
	return relay.NewRelaySet().ToList()
}

func assertRelayListsMatch(t *testing.T, want, got relay.List) {
	t.Helper()

	assert.ElementsMatch(t, want.ToStringSlice(), got.ToStringSlice())
}

func assertRelaysHaveNotChanged(t *testing.T, sut *rcm.Configurator) func(types.PublicKey, relay.Set) {
	t.Helper()

	return func(pk types.PublicKey, want relay.Set) {
		assert.ElementsMatch(t, want.ToStringSlice(), sut.RelaysForProposer(pk.String()).ToStringSlice())
	}
}

func assertWasRunConCurrently(t *testing.T, sut *rcm.Configurator) {
	t.Helper()

	const iterations = 10000
	numOfWorkers := runtime.NumCPU()

	runConcurrently(numOfWorkers, iterations, func(r *rand.Rand, num int64) {
		randomlyCallRCMMethods(t, sut)(r, num)
	})
}

func runConcurrently(numOfWorkers int, num int64, fn func(*rand.Rand, int64)) {
	var wg sync.WaitGroup
	wg.Add(numOfWorkers)

	for g := numOfWorkers; g > 0; g-- {
		r := rand.New(rand.NewSource(int64(g)))

		go func(r *rand.Rand) {
			defer wg.Done()

			for n := int64(1); n <= num; n++ {
				fn(r, n)
			}
		}(r)
	}

	wg.Wait()
}

func randomlyCallRCMMethods(t *testing.T, sut *rcm.Configurator) func(*rand.Rand, int64) {
	t.Helper()

	return func(r *rand.Rand, num int64) {
		switch {
		case r.Int63n(num)%2 == 0:
			require.NoError(t, sut.SyncConfig())
		case r.Int63n(num)%3 == 0:
			sut.RelaysForProposer(reltest.RandomBLSPublicKey(t).String())
		default:
			sut.AllRelays()
		}
	}
}
