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
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncer(t *testing.T) {
	t.Parallel()

	t.Run("it syncs configuration every given interval of time", func(t *testing.T) {
		t.Parallel()

		// arrange
		errCh := make(chan error, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithRandomRelays(t),
			rcm.SyncerWithInterval(10*time.Millisecond),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(errCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assert.NoError(t, <-errCh)
	})

	t.Run("it handles sync failures", func(t *testing.T) {
		t.Parallel()

		// arrange
		errCh := make(chan error, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithFaultyProvider(t),
			rcm.SyncerWithInterval(10*time.Millisecond),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(errCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assert.ErrorIs(t, <-errCh, rcm.ErrConfigProviderFailure)
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
		errCh := make(chan error, 1)

		sut := rcm.NewSyncer(
			createConfigManagerWithRandomRelays(t),
			rcm.SyncerWithOnSyncHandler(createTestOnSyncHandler(errCh)))

		// act
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		sut.SyncConfig(ctx)

		// assert
		assertDefaultIntervalIsUsed(t)(ctx, errCh)
	})

	t.Run("it panics if config manager is not provided", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithValue(t, "configManager is require and cannot be nil", func() {
			_ = rcm.NewSyncer(nil)
		})
	})
}

func assertDefaultIntervalIsUsed(t *testing.T) func(context.Context, chan error) {
	t.Helper()

	// We don't want to wait more than 3 minutes to check the rcm.DefaultSyncInterval value.
	// if context is timed-out earlier that then onSyncHandler was executed,
	// it means the interval is bigger than the context timeout,
	// so we may deduce that the rcm.DefaultSyncInterval is used.
	return func(ctx context.Context, errCh chan error) {
		select {
		case <-errCh:
			assert.Fail(t, "sync interval is less than rcm.DefaultSyncInterval")
		case <-ctx.Done():
			return
		}
	}
}

func createConfigManagerWithRandomRelays(t *testing.T) *rcm.Configurator {
	t.Helper()

	relays := testutil.RandomRelaySet(t, 3)
	relayProvider := rcp.NewDefault(relays).FetchConfig

	cm, err := rcm.New(rcm.NewRegistryCreator(relayProvider))
	require.NoError(t, err)

	return cm
}

func createTestOnSyncHandler(errCh chan error) func(_ time.Time, err error) {
	onSyncHandler := func(_ time.Time, err error) {
		errCh <- err
	}

	return onSyncHandler
}

func createConfigManagerWithFaultyProvider(t *testing.T) *rcm.Configurator {
	t.Helper()

	validatorPublicKey := testutil.RandomBLSPublicKey(t)
	proposerRelays := testutil.RandomRelaySet(t, 3)
	defaultRelays := testutil.RandomRelaySet(t, 2)
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
