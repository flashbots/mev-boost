package types

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

// HashTreeRoot is an interface for SSZ-encodable objects
type HashTreeRoot interface {
	HashTreeRoot() ([32]byte, error)
}

// VerifySignature verifies a signature against a message and public key.
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

// IntToU256 takes a uint64 and returns a U256 type
func IntToU256(i uint64) (ret U256Str) {
	s := fmt.Sprint(i)
	ret.UnmarshalText([]byte(s))
	return
}

// HexToAddress takes a hex string and returns an Address
func HexToAddress(s string) (ret Address, err error) {
	err = ret.UnmarshalText([]byte(s))
	return ret, err
}

// HexToPubkey takes a hex string and returns a PublicKey
func HexToPubkey(s string) (ret PublicKey, err error) {
	err = ret.UnmarshalText([]byte(s))
	return ret, err
}

// HexToSignature takes a hex string and returns a Signature
func HexToSignature(s string) (ret Signature, err error) {
	err = ret.UnmarshalText([]byte(s))
	return ret, err
}
