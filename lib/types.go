package lib

import "github.com/ethereum/go-ethereum/common"

// ExecutePayloadResponse TODO
type ExecutePayloadResponse struct {
	// enum - "VALID" | "INVALID" | "SYNCING"
	Status          string      `json:"status"`
	Message         string      `json:"message"`
	LatestValidHash common.Hash `json:"latestValidHash"`
}
