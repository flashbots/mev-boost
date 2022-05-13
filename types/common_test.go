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

type commonMarshallable interface {
	MarshalText() ([]byte, error)
	UnmarshalJSON(input []byte) error
	UnmarshalText(input []byte) error
	String() string
	FromSlice(x []byte)
}

func TestCommonSZEncoding(t *testing.T) {
	u256 := IntToU256(12345)
	tests := []commonMarshallable{
		&Signature{0x01},
		&PublicKey{0x02},
		&Address{0x03},
		&Hash{0x04},
		&Root{0x05},
		&CommitteeBits{0x06},
		&Bloom{0x07},
		&u256,
	}
	for _, test := range tests {
		// fmt.Printf("%T \n", test)
		buf1, err := test.MarshalText()
		require.NoError(t, err)
		require.LessOrEqual(t, 0, len(buf1))

		err = test.UnmarshalText(buf1)
		require.NoError(t, err)

		err = test.UnmarshalJSON([]byte(fmt.Sprintf(`"%s"`, test.String())))
		require.NoError(t, err)

		s := test.String()
		require.LessOrEqual(t, 0, len(s))

		test.FromSlice([]byte{0x01})
	}
}
