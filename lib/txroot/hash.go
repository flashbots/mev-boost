package txroot

import (
	"errors"
	"hash"
	"sync"

	"github.com/minio/sha256-simd"
	"golang.org/x/crypto/sha3"
)

// ErrNilProto can occur when attempting to hash a protobuf message that is nil
// or has nil objects within lists.
var ErrNilProto = errors.New("cannot hash a nil protobuf message")

var sha256Pool = sync.Pool{New: func() interface{} {
	return sha256.New()
}}

// Hash defines a function that returns the sha256 checksum of the data passed in.
// https://github.com/ethereum/consensus-specs/blob/v0.9.3/specs/core/0_beacon-chain.md#hash
func Hash(data []byte) [32]byte {
	h, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		h = sha256.New()
	}
	defer sha256Pool.Put(h)
	h.Reset()

	var b [32]byte

	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash

	// #nosec G104
	h.Write(data)
	h.Sum(b[:0])

	return b
}

// CustomSHA256Hasher returns a hash function that uses
// an enclosed hasher. This is not safe for concurrent
// use as the same hasher is being called throughout.
//
// Note: that this method is only more performant over
// hashutil.Hash if the callback is used more than 5 times.
func CustomSHA256Hasher() func([]byte) [32]byte {
	hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		hasher = sha256.New()
	} else {
		hasher.Reset()
	}
	var h [32]byte

	return func(data []byte) [32]byte {
		// The hash interface never returns an error, for that reason
		// we are not handling the error below. For reference, it is
		// stated here https://golang.org/pkg/hash/#Hash

		// #nosec G104
		hasher.Write(data)
		hasher.Sum(h[:0])
		hasher.Reset()

		return h
	}
}

var keccak256Pool = sync.Pool{New: func() interface{} {
	return sha3.NewLegacyKeccak256()
}}

// HashKeccak256 defines a function which returns the Keccak-256/SHA3
// hash of the data passed in.
func HashKeccak256(data []byte) [32]byte {
	var b [32]byte

	h, ok := keccak256Pool.Get().(hash.Hash)
	if !ok {
		h = sha3.NewLegacyKeccak256()
	}
	defer keccak256Pool.Put(h)
	h.Reset()

	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash

	// #nosec G104
	h.Write(data)
	h.Sum(b[:0])

	return b
}
