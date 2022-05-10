package types

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestJSONSerialization(t *testing.T) {
	a := Signature{0x01}
	b, err := json.Marshal(a)
	require.NoError(t, err)

	expectedHex := `0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000`
	expectedJSON := fmt.Sprintf(`"%s"`, expectedHex)
	require.JSONEq(t, expectedJSON, string(b))

	ax := new(hexutil.Bytes)
	err = ax.UnmarshalJSON([]byte(expectedJSON))
	require.NoError(t, err)
	require.Equal(t, expectedHex, ax.String())

	a2 := Signature{}
	err = a2.UnmarshalJSON([]byte(expectedJSON))
	require.NoError(t, err)
	require.Equal(t, a, a2)
}

func TestU256Str(t *testing.T) {
	a := U256Str{}
	a[31] = 0x01
	require.Equal(t, "1", a.String())

	b, err := json.Marshal(a)
	require.NoError(t, err)

	expectedStr := `1`
	expectedJSON := fmt.Sprintf(`"%s"`, expectedStr)
	require.JSONEq(t, expectedJSON, string(b))

	// UnmarshalText
	a2 := U256Str{}
	err = a2.UnmarshalText([]byte(expectedStr))
	require.NoError(t, err)
	require.Equal(t, a, a2)

	// UnmarshalJSON
	a3 := U256Str{}
	err = a3.UnmarshalJSON([]byte(expectedJSON))
	require.NoError(t, err)
	require.Equal(t, a, a3)

	// IntToU256
	u := IntToU256(123)
	require.Equal(t, "123", u.String())
}

func TestHexToAddress(t *testing.T) {
	a := HexToAddress("0x01")
	require.Equal(t, "0x010000000000000000000000000000000000000000000000000000000000000000", a.String())
}
