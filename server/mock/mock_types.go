package mock

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/utils"
	"github.com/sirupsen/logrus"
)

// TestLog is used to log information in the test methods
var TestLog = logrus.NewEntry(logrus.New())

// HexToBytes converts a hexadecimal string to a byte array
func HexToBytes(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

// HexToHash converts a hexadecimal string to an Ethereum hash
func HexToHash(s string) (ret phase0.Hash32) {
	ret, err := utils.HexToHash(s)
	if err != nil {
		TestLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

// HexToAddress converts a hexadecimal string to an Ethereum address
func HexToAddress(s string) (ret bellatrix.ExecutionAddress) {
	ret, err := utils.HexToAddress(s)
	if err != nil {
		TestLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

// HexToPubkey converts a hexadecimal string to a BLS Public Key
func HexToPubkey(s string) (ret phase0.BLSPubKey) {
	ret, err := utils.HexToPubkey(s)
	if err != nil {
		TestLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

// HexToSignature converts a hexadecimal string to a BLS Signature
func HexToSignature(s string) (ret phase0.BLSSignature) {
	ret, err := utils.HexToSignature(s)
	if err != nil {
		TestLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
