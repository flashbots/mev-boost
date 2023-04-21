package server

import (
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	retBytes, err := hexutil.Decode(s)
	if err != nil {
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	if len(retBytes) != len(ret) {
		testLog.Error(" _HexToHash: invalid length", len(retBytes))
		panic(err)
	}
	copy(ret[:], retBytes)
	return ret
}

// _HexToAddress converts a hexadecimal string to an Ethereum address
func _HexToAddress(s string) (ret bellatrix.ExecutionAddress) {
	retBytes, err := hexutil.Decode(s)
	if err != nil {
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	if len(retBytes) != len(ret) {
		testLog.Error(" _HexToAddress: invalid length", len(retBytes))
		panic(err)
	}
	copy(ret[:], retBytes)
	return ret
}

// _HexToPubkey converts a hexadecimal string to a BLS Public Key
func _HexToPubkey(s string) (ret phase0.BLSPubKey) {
	retBytes, err := hexutil.Decode(s)
	if err != nil {
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	if len(retBytes) != len(ret) {
		testLog.Error(" _HexToPubkey: invalid length", len(retBytes))
		panic(err)
	}
	copy(ret[:], retBytes)
	return
}

// _HexToSignature converts a hexadecimal string to a BLS Signature
func _HexToSignature(s string) (ret phase0.BLSSignature) {
	retBytes, err := hexutil.Decode(s)
	if err != nil {
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	if len(retBytes) != len(ret) {
		testLog.Error(" _HexToSignature: invalid length", len(retBytes))
		panic(err)
	}
	copy(ret[:], retBytes)
	return
}
