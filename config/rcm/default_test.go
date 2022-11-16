package rcm_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/server"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ server.RelayConfigManager = (*rcm.DefaultConfigManager)(nil)

func TestDefaultConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns all known relays ignoring validator public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRCPRelayEntries(t, 3)

		sut, err := rcm.NewDefault(want)
		require.NoError(t, err)

		// act
		got, err := sut.RelaysByValidatorPublicKey(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("it returns an error if relays are empty", func(t *testing.T) {
		t.Parallel()

		_, err := rcm.NewDefault(nil)
		assert.ErrorIs(t, err, rcm.ErrNoRelays)
	})
}
