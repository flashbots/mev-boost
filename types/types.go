// Package types provides various types used by the boost service.
package types

import "github.com/ethereum/go-ethereum/common/hexutil"

// RelayEntry represents a relay that mev-boost connects to
type RelayEntry struct {
	Address string
	Pubkey  hexutil.Bytes
}
