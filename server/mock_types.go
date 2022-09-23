package server

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/types"
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
func _HexToHash(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

// _HexToAddress converts a hexadecimal string to an Ethereum address
func _HexToAddress(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

// _HexToPubkey converts a hexadecimal string to a BLS Public Key
func _HexToPubkey(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

// _HexToSignature converts a hexadecimal string to a BLS Signature
func _HexToSignature(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
