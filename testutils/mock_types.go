package testutils

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/sirupsen/logrus"
)

// TestLog is used to log information in the test methods
var TestLog = logrus.WithField("testing", true)

// HexToBytesP converts a hexadecimal string to a byte array
func HexToBytesP(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

// HexToHashP converts a hexadecimal string to an Ethereum hash
func HexToHashP(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " HexToHashP: ", s)
		panic(err)
	}
	return ret
}

// HexToAddressP converts a hexadecimal string to an Ethereum address
func HexToAddressP(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " HexToAddressP: ", s)
		panic(err)
	}
	return ret
}

// HexToPubkeyP converts a hexadecimal string to a BLS Public Key
func HexToPubkeyP(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " HexToPubkeyP: ", s)
		panic(err)
	}
	return
}

// HexToSignatureP converts a hexadecimal string to a BLS Signature
func HexToSignatureP(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " HexToSignatureP: ", s)
		panic(err)
	}
	return
}
