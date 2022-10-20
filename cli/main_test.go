package cli

import (
	"math/big"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
)

func TestFloatEthTo256Wei(t *testing.T) {
	// test with small input
	i := 0.000000000000012345
	weiU256, err := floatEthTo256Wei(i)
	require.NoError(t, err)
	require.Equal(t, types.IntToU256(12345), *weiU256)

	// test with zero
	i = 0
	weiU256, err = floatEthTo256Wei(i)
	require.NoError(t, err)
	require.Equal(t, types.IntToU256(0), *weiU256)

	// test with large input
	i = 987654.3
	weiU256, err = floatEthTo256Wei(i)
	require.NoError(t, err)

	r := big.NewInt(9876543)
	r.Mul(r, big.NewInt(1e17))
	referenceWeiU256 := new(types.U256Str)
	err = referenceWeiU256.FromBig(r)
	require.NoError(t, err)

	require.Equal(t, *referenceWeiU256, *weiU256)
}
