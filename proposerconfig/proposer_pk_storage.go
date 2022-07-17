package proposerconfig

import "github.com/flashbots/go-boost-utils/types"

// StorageKey is used as the key in the proposer's PublicKeyStorage.
type StorageKey struct {
	Slot       uint64
	ParentHash types.Hash
}

// PublicKeyStorage is used to store a proposer's public key according to a specific slot
// along with the block hash it is currently proposing.
type PublicKeyStorage struct {
	Storage map[StorageKey]types.PublicKey
}
