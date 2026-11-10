package gloas

import "github.com/attestantio/go-eth2-client/spec/gloas"

// ExecutionPayloadBidResponse is the JSON body of a 200 getExecutionPayloadBid response.
type ExecutionPayloadBidResponse struct {
	Version string                           `json:"version"`
	Data    *gloas.SignedExecutionPayloadBid `json:"data"`
}
