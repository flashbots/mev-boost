package server

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/utils"
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
	ret, err := utils.HexToHash(s)
	if err != nil {
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

// _HexToAddress converts a hexadecimal string to an Ethereum address
func _HexToAddress(s string) (ret bellatrix.ExecutionAddress) {
	ret, err := utils.HexToAddress(s)
	if err != nil {
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

// _HexToPubkey converts a hexadecimal string to a BLS Public Key
func _HexToPubkey(s string) (ret phase0.BLSPubKey) {
	ret, err := utils.HexToPubkey(s)
	if err != nil {
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

// _HexToSignature converts a hexadecimal string to a BLS Signature
func _HexToSignature(s string) (ret phase0.BLSSignature) {
	ret, err := utils.HexToSignature(s)
	if err != nil {
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
