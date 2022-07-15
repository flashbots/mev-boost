package server

import "github.com/flashbots/go-boost-utils/types"

// proposerPublicKeyStorage is used to store a proposer's public key according to a specific slot
// along with the block hash it is currently proposing.
type proposerPublicKeyStorage struct {
	storage map[string]types.PublicKey
}
