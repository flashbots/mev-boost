package proposerconfig

import "github.com/flashbots/go-boost-utils/types"

// StorageKey is used as the key in the proposer's PublicKeyStorage.
type StorageKey struct {
	Slot       uint64
	ParentHash types.Hash
}

// PublicKeyStorage is used to store public keys for a specific slot.
type PublicKeyStorage struct {
	Storage map[StorageKey][]types.PublicKey
}

// Store stores a public key in the array slot corresponding to a given combination of slot
// parent hash.
func (s *PublicKeyStorage) Store(pk types.PublicKey, key StorageKey) {
	s.Storage[key] = append(s.Storage[key], pk)
}

// Get retrieves all public keys associated to a given StorageKey.
func (s *PublicKeyStorage) Get(key StorageKey) []types.PublicKey {
	return s.Storage[key]
}

// Prune deletes all public key entries for a given StorageKey.
func (s *PublicKeyStorage) Prune(key StorageKey) {
	delete(s.Storage, key)
}
