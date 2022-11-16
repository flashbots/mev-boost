package rcp_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns all known relays ignoring validator public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelayEntries(t, 3)
		sut := rcp.NewDefaultConfigProvider(want)

		// act
		got, err := sut.RelaysByValidatorPublicKey(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
