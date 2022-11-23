package server

import (
	"encoding/hex"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// testLog is used to log information in the test methods
var testLog = logrus.NewEntry(logrus.New())

// _HexToBytes converts a hexadecimal string to a byte array
func _HexToBytes(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

// _HexToHash converts a hexadecimal string to an Ethereum hash
func _HexToHash(s string) (ret phase0.Hash32) {
	hash, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		err = errors.Wrap(err, "invalid value for hash")
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	if len(hash) != phase0.Hash32Length {
		err = errors.New("incorrect length for hash")
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	copy(ret[:], hash)
	return
}

// _HexToAddress converts a hexadecimal string to an Ethereum address
func _HexToAddress(s string) (ret bellatrix.ExecutionAddress) {
	address, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		err = errors.Wrap(err, "invalid value for fee recipient")
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	if len(address) == 0 {
		err = errors.New("incorrect length for public key")
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	copy(ret[:], address)
	return
}

// _HexToPubkey converts a hexadecimal string to a BLS Public Key
func _HexToPubkey(s string) (ret phase0.BLSPubKey) {
	pubKey, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		err = errors.Wrap(err, "invalid value for public key")
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	if len(pubKey) != phase0.PublicKeyLength {
		err = errors.New("incorrect length for public key")
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	copy(ret[:], pubKey)
	return
}

// _HexToSignature converts a hexadecimal string to a BLS Signature
func _HexToSignature(s string) (ret phase0.BLSSignature) {
	signature, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		err = errors.Wrap(err, "invalid value for signature")
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	if len(signature) != phase0.SignatureLength {
		err = errors.Errorf("incorrect length %d for signature", len(signature))
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	copy(ret[:], signature)
	return
}
