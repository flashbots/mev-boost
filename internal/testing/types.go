package testing

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/sirupsen/logrus"
)

// TestLog is used to log information in the test methods
var TestLog = logrus.WithField("testing", true)

// HexToBytes converts a hexadecimal string to a byte array
func HexToBytes(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

// HexToHash converts a hexadecimal string to an Ethereum hash
func HexToHash(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

// HexToAddress converts a hexadecimal string to an Ethereum address
func HexToAddress(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

// HexToPubkey converts a hexadecimal string to a BLS Public Key
func HexToPubkey(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

// HexToSignature converts a hexadecimal string to a BLS Signature
func HexToSignature(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
