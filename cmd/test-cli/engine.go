package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type engineBlockData struct {
	ParentHash common.Hash    `json:"parentHash"`
	Hash       common.Hash    `json:"hash"`
	Timestamp  hexutil.Uint64 `json:"timestamp"`
}

func getLatestEngineBlock(engineEndpoint string) (engineBlockData, error) {
	dst := engineBlockData{}
	payload := map[string]any{"id": 67, "jsonrpc": "2.0", "method": "eth_getBlockByNumber", "params": []any{"latest", false}}
	err := sendJSONRequest(engineEndpoint, payload, &dst)
	if err != nil {
		log.WithError(err).Info("could not get latest block")
		return engineBlockData{}, err
	}

	return dst, nil
}

type forkchoiceState struct {
	HeadBlockHash      common.Hash `json:"headBlockHash"`
	SafeBlockHash      common.Hash `json:"safeBlockHash"`
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}

type payloadAttributes struct {
	Timestamp             hexutil.Uint64 `json:"timestamp"`
	PrevRandao            common.Hash    `json:"prevRandao"`
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"`
}

func sendForkchoiceUpdate(engineEndpoint string, block engineBlockData) error {
	log.WithField("hash", block.Hash).Info("sending FCU")
	params := []any{
		forkchoiceState{block.Hash, block.Hash, block.Hash},
		payloadAttributes{block.Timestamp + 1, common.Hash{0x01}, common.Address{0x02}},
	}
	payload := map[string]any{"id": 67, "jsonrpc": "2.0", "method": "engine_forkchoiceUpdatedV1", "params": params}
	return sendJSONRequest(engineEndpoint, payload, nil)
}
