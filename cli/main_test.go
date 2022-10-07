package cli

import (
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
)

func TestFloatEthTo256Wei(t *testing.T) {
	// test with valid input
	i := 0.000000000000012345
	u256, err := floatEthTo256Wei(i)
	require.Equal(t, types.IntToU256(12345), *u256)
	require.NoError(t, err)
}
