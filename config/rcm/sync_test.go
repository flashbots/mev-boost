package rcm_test

import (
	"context"
	"testing"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/config/rcp/rcptest"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestSyncer(t *testing.T) {
	t.Parallel()

	t.Run("it syncs configuration every given interval of time", func(t *testing.T) {
		t.Parallel()

		// arrange
		syncCh := make(chan relaySync, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithRandomRelays(t),
			rcm.SyncerWithInterval(10*time.Millisecond),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(syncCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assert.NoError(t, (<-syncCh).err)
	})

	t.Run("it handles sync failures", func(t *testing.T) {
		t.Parallel()

		// arrange
		syncCh := make(chan relaySync, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithFaultyProvider(t),
			rcm.SyncerWithInterval(10*time.Millisecond),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(syncCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assert.ErrorIs(t, (<-syncCh).err, rcm.ErrConfigProviderFailure)
	})

	t.Run("it handles empty proposer and default relays properly on sync", func(t *testing.T) {
		t.Parallel()

		// arrange
		syncCh := make(chan relaySync, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithFilledRelaysAtFirstAndEmptyProposersConsequentially(t),
			rcm.SyncerWithInterval(10*time.Millisecond),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(syncCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		got := <-syncCh
		assert.NoError(t, got.err)
		assert.Empty(t, got.relays)
	})

	t.Run("it uses a nop onSyncHandler if none specified", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcm.NewSyncer(
			createConfigManagerWithRandomRelays(t),
			rcm.SyncerWithInterval(10*time.Millisecond))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		assert.NotPanics(t, func() {
			sut.SyncConfig(ctx)
		})
	})

	t.Run("it uses a default time interval if none specified", func(t *testing.T) {
		t.Parallel()

		// arrange
		syncCh := make(chan relaySync, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithRandomRelays(t),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(syncCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assertDefaultIntervalIsUsed(t)(ctx, syncCh)
	})

	t.Run("it panics if config manager is not provided", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithValue(t, "configurator is required and cannot be nil", func() {
			_ = rcm.NewSyncer(nil)
		})
	})
}

func assertDefaultIntervalIsUsed(t *testing.T) func(context.Context, chan relaySync) {
	t.Helper()

	// We don't want to wait more than 3 minutes to check the rcm.DefaultSyncInterval value.
	// if context is timed-out earlier that then onSyncHandler was executed,
	// it means the interval is bigger than the context timeout,
	// so we may deduce that the rcm.DefaultSyncInterval is used.
	return func(ctx context.Context, syncCh chan relaySync) {
		select {
		case <-syncCh:
			assert.Fail(t, "sync interval is less than rcm.DefaultSyncInterval")
		case <-ctx.Done():
			return
		}
	}
}

func createConfigManagerWithRandomRelays(t *testing.T) *rcm.Configurator {
	t.Helper()

	relays := reltest.RandomRelaySet(t, 3)
	relayProvider := rcp.NewDefault(relays).FetchConfig

	cm, err := rcm.New(rcm.NewRegistryCreator(relayProvider))
	require.NoError(t, err)

	return cm
}

type relaySync struct {
	err    error
	relays relay.List
}

func createTestOnSyncHandler(res chan relaySync) func(_ time.Time, err error, _ relay.List) {
	onSyncHandler := func(_ time.Time, err error, relays relay.List) {
		res <- relaySync{err: err, relays: relays}
	}

	return onSyncHandler
}

func createConfigManagerWithFaultyProvider(t *testing.T) *rcm.Configurator {
	t.Helper()

	validatorPublicKey := reltest.RandomBLSPublicKey(t)
	proposerRelays := reltest.RandomRelaySet(t, 3)
	defaultRelays := reltest.RandomRelaySet(t, 2)
	relayProvider := onceOnlySuccessfulProvider(validatorPublicKey, proposerRelays, defaultRelays)

	cm, err := rcm.New(rcm.NewRegistryCreator(relayProvider))
	require.NoError(t, err)

	return cm
}

func onceOnlySuccessfulProvider(
	pubKey types.PublicKey, proposerRelays, defaultRelays relay.Set,
) rcm.ConfigProvider {
	calls := []rcm.ConfigProvider{
		rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(
				pubKey.String(),
				proposerRelays,
			),
			rcptest.WithDefaultRelays(defaultRelays),
		), // first call is successful
		rcptest.MockRelayConfigProvider(rcptest.WithErr()), // second call is an error
	}

	return rcptest.MockRelayConfigProvider(rcptest.WithCalls(calls))
}

func createConfigManagerWithFilledRelaysAtFirstAndEmptyProposersConsequentially(t *testing.T) *rcm.Configurator {
	t.Helper()

	validatorPublicKey := reltest.RandomBLSPublicKey(t)
	proposerRelays := reltest.RandomRelaySet(t, 3)
	defaultRelays := reltest.RandomRelaySet(t, 2)
	relayProvider := filledFirstEmptyConsequentially(validatorPublicKey, proposerRelays, defaultRelays)

	cm, err := rcm.New(rcm.NewRegistryCreator(relayProvider))
	require.NoError(t, err)

	return cm
}

func filledFirstEmptyConsequentially(
	pubKey types.PublicKey, proposerRelays, defaultRelays relay.Set,
) rcm.ConfigProvider {
	calls := []rcm.ConfigProvider{
		rcptest.MockRelayConfigProvider(
			rcptest.WithProposerRelays(
				pubKey.String(),
				proposerRelays,
			),
			rcptest.WithDefaultRelays(defaultRelays),
		), // first call is successful
		rcptest.MockRelayConfigProvider(), // second call returns no data
	}

	return rcptest.MockRelayConfigProvider(rcptest.WithCalls(calls))
}
