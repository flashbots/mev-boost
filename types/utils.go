package types

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

type HashTreeRoot interface {
	HashTreeRoot() ([32]byte, error)
}

type SSZable interface {
	MarshalSSZ() ([]byte, error)
	MarshalSSZTo(buf []byte) (dst []byte, err error)
	UnmarshalSSZ(buf []byte) error
	SizeSSZ() (size int)
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh *ssz.Hasher) (err error)
}

func VerifySignature(obj HashTreeRoot, pk, s []byte) (bool, error) {
	msg, err := obj.HashTreeRoot()
	if err != nil {
		return false, err
	}
	sig, err := bls.SignatureFromBytes(s)
	if err != nil {
		return false, err
	}
	pubkey, err := bls.PublicKeyFromBytes(pk)
	if err != nil {
		return false, err
	}
	return sig.Verify(pubkey, msg[:]), nil
}
