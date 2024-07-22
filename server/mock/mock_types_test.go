package mock

import (
	"testing"

	builderApiDeneb "github.com/attestantio/go-builder-client/api/deneb"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
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
					HexToBytes(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualBytes := HexToBytes(tt.hex)
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
		{
			name:          "Should panic because of invalid hexadecimal input",
			hex:           "foo",
			expectedPanic: true,
			expectedHash:  nil,
		},
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
					HexToHash(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualHash := HexToHash(tt.hex)
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
		{
			name:            "Should panic because of invalid hexadecimal input",
			hex:             "foo",
			expectedPanic:   true,
			expectedAddress: nil,
		},
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
					HexToAddress(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualAddress := HexToAddress(tt.hex)
					require.Equal(t, *tt.expectedAddress, actualAddress)
				})
			}
		})
	}
}

// Same as for TestHexToHash and TestHexToAddress.
func TestHexToPublicKey(t *testing.T) {
	publicKey := phase0.BLSPubKey{
		0x82, 0xf6, 0xe7, 0xcc, 0x57, 0xa2, 0xce, 0x68,
		0xec, 0x41, 0x32, 0x1b, 0xeb, 0xc5, 0x5b, 0xcb,
		0x31, 0x94, 0x5f, 0xe6, 0x6a, 0x8e, 0x67, 0xeb,
		0x82, 0x51, 0x42, 0x5f, 0xab, 0x4c, 0x6a, 0x38,
		0xc1, 0x0c, 0x53, 0x21, 0x0a, 0xea, 0x97, 0x96,
		0xdd, 0x0b, 0xa0, 0x44, 0x1b, 0x46, 0x76, 0x2a,
	}

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
		{
			name:              "Should panic because of invalid hexadecimal input",
			hex:               "foo",
			expectedPanic:     true,
			expectedPublicKey: nil,
		},
		{
			name:              "Should not panic and convert hexadecimal input to public key",
			hex:               publicKey.String(),
			expectedPanic:     false,
			expectedPublicKey: &publicKey,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				require.Panics(t, func() {
					HexToPubkey(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualPublicKey := HexToPubkey(tt.hex)
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

	message := &builderApiDeneb.BuilderBid{
		Header: &deneb.ExecutionPayloadHeader{
			BlockHash:     HexToHash("0xe28385e7bd68df656cd0042b74b69c3104b5356ed1f20eb69f1f925df47a3ab7"),
			BaseFeePerGas: uint256.NewInt(0),
		},
		Value:  uint256.NewInt(12345),
		Pubkey: HexToPubkey(publicKey),
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
		{
			name:              "Should panic because of invalid hexadecimal input",
			hex:               "foo",
			expectedPanic:     true,
			expectedSignature: nil,
		},
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
					HexToSignature(tt.hex)
				})
			} else {
				require.NotPanics(t, func() {
					actualSignature := HexToSignature(tt.hex)
					require.Equal(t, *tt.expectedSignature, actualSignature)
				})
			}
		})
	}
}
