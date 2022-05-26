package testing

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/sirupsen/logrus"
)

var testLog = logrus.WithField("testing", true)

func _HexToBytes(hex string) []byte {
	res, err := hexutil.Decode(hex)
	if err != nil {
		panic(err)
	}
	return res
}

func _HexToHash(s string) (ret types.Hash) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToHash: ", s)
		panic(err)
	}
	return ret
}

func _HexToAddress(s string) (ret types.Address) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToAddress: ", s)
		panic(err)
	}
	return ret
}

func _HexToPubkey(s string) (ret types.PublicKey) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToPubkey: ", s)
		panic(err)
	}
	return
}

func _HexToSignature(s string) (ret types.Signature) {
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		testLog.Error(err, " _HexToSignature: ", s)
		panic(err)
	}
	return
}
