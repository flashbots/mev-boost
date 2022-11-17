package cli

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestFloatEthTo256Wei(t *testing.T) {
	// test with small input
	i := 0.000000000000012345
	weiU256, overflow := floatEthTo256Wei(i)
	require.False(t, overflow)
	require.Equal(t, uint256.NewInt(12345), weiU256)

	// test with zero
	i = 0
	weiU256, overflow = floatEthTo256Wei(i)
	require.False(t, overflow)
	require.Equal(t, uint256.NewInt(0), weiU256)

	// test with large input
	i = 987654.3
	weiU256, overflow = floatEthTo256Wei(i)
	require.False(t, overflow)

	r := big.NewInt(9876543)
	r.Mul(r, big.NewInt(1e17))
	referenceWeiU256 := new(uint256.Int)
	overflow = referenceWeiU256.SetFromBig(r)
	require.False(t, overflow)

	require.Equal(t, referenceWeiU256, weiU256)
}
