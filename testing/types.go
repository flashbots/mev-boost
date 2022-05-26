package testing

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/sirupsen/logrus"
)

var TestLog = logrus.WithField("testing", true)

func HexToBytes(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

func HexToHash(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

func HexToAddress(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

func HexToPubkey(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

func HexToSignature(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		TestLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
