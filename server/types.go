package server

import (
	"time"

	"github.com/flashbots/go-boost-utils/types"
)

// bidResp are entries in the bids cache
type bidResp struct {
	proposerPubkey types.PublicKey

	t         time.Time
	response  types.GetHeaderResponse
	blockHash string
	relays    []string
}

// bidRespKey is used as key for the bids cache
type bidRespKey struct {
	slot      uint64
	blockHash string
}
