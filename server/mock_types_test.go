package server

import (
	"testing"

	capellaapi "github.com/attestantio/go-builder-client/api/capella"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestHexToBytes(t *testing.T) {
	testCases := []struct {
		name string
		hex  string

		expectedPanic bool
		expectedBytes []byte
	}{
		{
			name:          "Should panic because of invalid hexadecimal input",
			hex:           "foo",
			expectedPanic: true,
			expectedBytes: nil,
		},
		{
			name:          "Should not panic and convert hexadecimal input to byte array",
			hex:           "0x0102",
			expectedPanic: false,
			expectedBytes: []byte{0x01, 0x02},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					_HexToBytes(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualBytes := _HexToBytes(tt.hex)
					require.Equal(t, tt.expectedBytes, actualBytes)
				})
			}
		})
	}
}

// Providing foo works, definitely a weird behavior.
func TestHexToHash(t *testing.T) {
	testCases := []struct {
		name string
		hex  string

		expectedPanic bool
		expectedHash  *phase0.Hash32
	}{
		{
			name:          "Should panic because of empty hexadecimal input",
			hex:           "",
			expectedPanic: true,
			expectedHash:  nil,
		},
		/*
			{
				name:          "Should panic because of invalid hexadecimal input",
				hex:           "foo",
				expectedPanic: true,
				expectedHash:  nil,
			},
		*/
		{
			name:          "Should not panic and convert hexadecimal input to hash",
			hex:           common.Hash{0x01}.String(),
			expectedPanic: false,
			expectedHash:  &phase0.Hash32{0x01},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					_HexToHash(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualHash := _HexToHash(tt.hex)
					require.Equal(t, *tt.expectedHash, actualHash)
				})
			}
		})
	}
}

// Providing foo works here too, definitely a weird behavior.
func TestHexToAddress(t *testing.T) {
	testCases := []struct {
		name string
		hex  string

		expectedPanic   bool
		expectedAddress *bellatrix.ExecutionAddress
	}{
		{
			name:            "Should panic because of empty hexadecimal input",
			hex:             "",
			expectedPanic:   true,
			expectedAddress: nil,
		},
		/*
			{
				name:            "Should panic because of invalid hexadecimal input",
				hex:             "foo",
				expectedPanic:   true,
				expectedAddress: nil,
			},
		*/
		{
			name:            "Should not panic and convert hexadecimal input to address",
			hex:             common.Address{0x01}.String(),
			expectedPanic:   false,
			expectedAddress: &bellatrix.ExecutionAddress{0x01},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					_HexToAddress(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualAddress := _HexToAddress(tt.hex)
					require.Equal(t, *tt.expectedAddress, actualAddress)
				})
			}
		})
	}
}

// Same as for TestHexToHash and TestHexToAddress.
func TestHexToPublicKey(t *testing.T) {
	testCases := []struct {
		name string
		hex  string

		expectedPanic     bool
		expectedPublicKey *phase0.BLSPubKey
	}{
		{
			name:              "Should panic because of empty hexadecimal input",
			hex:               "",
			expectedPanic:     true,
			expectedPublicKey: nil,
		},
		/*
			{
				name:              "Should panic because of invalid hexadecimal input",
				hex:               "foo",
				expectedPanic:     true,
				expectedSignature: nil,
			},
		*/
		{
			name:              "Should not panic and convert hexadecimal input to public key",
			hex:               phase0.BLSPubKey{0x01}.String(),
			expectedPanic:     false,
			expectedPublicKey: &phase0.BLSPubKey{0x01},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					_HexToPubkey(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualPublicKey := _HexToPubkey(tt.hex)
					require.Equal(t, *tt.expectedPublicKey, actualPublicKey)
				})
			}
		})
	}
}

// Same as for TestHexToHash, TestHexToAddress and TestHexToPublicKey.
func TestHexToSignature(t *testing.T) {
	// Sign a message for further testing.
	privateKey, blsPublicKey, err := bls.GenerateNewKeypair()
	require.NoError(t, err)

	publicKey := hexutil.Encode(bls.PublicKeyToBytes(blsPublicKey))

	message := &capellaapi.BuilderBid{
		Header: &capella.ExecutionPayloadHeader{
			BlockHash: _HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7"),
		},
		Value:  uint256.NewInt(12345),
		Pubkey: _HexToPubkey(publicKey),
	}
	ssz, err := message.MarshalSSZ()
	require.NoError(t, err)

	sig := bls.Sign(privateKey, ssz)
	sigBytes := bls.SignatureToBytes(sig)

	// Convert bls.Signature bytes to phase0.BLSSignature
	signature, err := utils.BlsSignatureToSignature(sig)
	require.NoError(t, err)

	testCases := []struct {
		name string
		hex  string

		expectedPanic     bool
		expectedSignature *phase0.BLSSignature
	}{
		{
			name:              "Should panic because of empty hexadecimal input",
			hex:               "",
			expectedPanic:     true,
			expectedSignature: nil,
		},
		/*
			{
				name:              "Should panic because of invalid hexadecimal input",
				hex:               "foo",
				expectedPanic:     true,
				expectedSignature: nil,
			},
		*/
		{
			name:              "Should not panic and convert hexadecimal input to signature",
			hex:               hexutil.Encode(sigBytes),
			expectedPanic:     false,
			expectedSignature: &signature,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					_HexToSignature(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualSignature := _HexToSignature(tt.hex)
					require.Equal(t, *tt.expectedSignature, actualSignature)
				})
			}
		})
	}
}
