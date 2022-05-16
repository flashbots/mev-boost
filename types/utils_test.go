package types

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bls/blst"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	sk, err := blst.RandKey()
	require.NoError(t, err)

	var pubkey PublicKey
	pubkey.FromSlice(sk.PublicKey().Marshal())
	require.Equal(t, sk.PublicKey().Marshal(), pubkey[:])

	msg := RegisterValidatorRequestMessage{
		FeeRecipient: Address{0x42},
		GasLimit:     15_000_000,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       pubkey,
	}
	root, err := msg.HashTreeRoot()
	require.NoError(t, err)

	// Create signature
	sig := sk.Sign(root[:]).Marshal()
	var signature Signature
	signature.FromSlice(sig)
	require.Equal(t, sig[:], signature[:])

	// Verify signature
	ok, err := VerifySignature(&msg, pubkey[:], signature[:])
	require.NoError(t, err)
	require.True(t, ok)

	// Test wrong signature (TODO)
	// wrongSig := signature[:]
	// wrongSig[len(wrongSig)-1] = 0x00
	// ok, err = VerifySignature(&msg, pubkey[:], signature[:])
	// require.NoError(t, err)
	// require.False(t, ok)
}

func TestVerifySignatureManualPk(t *testing.T) {
	msg2 := RegisterValidatorRequestMessage{
		FeeRecipient: Address{0x42},
		GasLimit:     15_000_000,
		Timestamp:    1652369368,
		Pubkey:       PublicKey{0x0d},
	}
	root2, err := msg2.HashTreeRoot()
	require.NoError(t, err)

	// Verify expected signature with manual pk
	pkBytes := make([]byte, 32)
	pkBytes[0] = 0x01
	sk2, err := blst.SecretKeyFromBytes(pkBytes)
	require.NoError(t, err)
	sig2 := sk2.Sign(root2[:]).Marshal()
	var signature2 Signature
	signature2.FromSlice(sig2)
	require.Equal(t, "0x8e09a0ae7af113da2043001cc19fb1b3b24bbe022c1b8050ba2297ad1186f4217dd7095edad1d16d83d10f3297883d9e1674c81da95f10d3358c5afdb2500279e720b32879219c9a3b33415239bf46a66cd92b9d1750a6dd7cc7ec936a357128", signature2.String())
}

func TestHexToAddress(t *testing.T) {
	_, err := HexToAddress("0x01")
	require.Error(t, err)

	a, err := HexToAddress("0x0100000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, "0x0100000000000000000000000000000000000000", a.String())
}

func TestHexToPubkey(t *testing.T) {
	_, err := HexToPubkey("0x01")
	require.Error(t, err)

	a, err := HexToPubkey("0xed7f862045422bd51ba732730ce993c94d2545e5db1112102026343904fcdf6f5cf37926a3688444703772ed80fa223f")
	require.NoError(t, err)
	require.Equal(t, "0xed7f862045422bd51ba732730ce993c94d2545e5db1112102026343904fcdf6f5cf37926a3688444703772ed80fa223f", a.String())
}

func TestHexToSignature(t *testing.T) {
	_, err := HexToSignature("0x01")
	require.Error(t, err)

	a, err := HexToSignature("0xb8f03e639b91fa8e9892f66c798f07f6e7b3453234f643b2c06a35c5149cf6d85e4e1572c33549fe749292445fbff9e0739c78159324c35dc1a90e5745ca70c8caf1b63fb6678d81bd2d5cb6baeb1462df7a93877d0e22a31dd6438334536d9a")
	require.NoError(t, err)
	require.Equal(t, "0xb8f03e639b91fa8e9892f66c798f07f6e7b3453234f643b2c06a35c5149cf6d85e4e1572c33549fe749292445fbff9e0739c78159324c35dc1a90e5745ca70c8caf1b63fb6678d81bd2d5cb6baeb1462df7a93877d0e22a31dd6438334536d9a", a.String())
}
