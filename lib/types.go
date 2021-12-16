package lib

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SignedBlindedBeaconBlock forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#signedbeaconblockheader
type SignedBlindedBeaconBlock struct {
	Message   *BlindedBeaconBlock `json:"message"`
	Signature string              `json:"signature"`
}

// BlindedBeaconBlock forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblock
type BlindedBeaconBlock struct {
	Slot          string          `json:"slot"`
	ProposerIndex string          `json:"proposer_index"`
	ParentRoot    string          `json:"parent_root"`
	StateRoot     string          `json:"state_root"`
	Body          json.RawMessage `json:"body"`
}

// BlindedBeaconBlockBodyPartial a partial block body only containing a payload
type BlindedBeaconBlockBodyPartial struct {
	ExecutionPayload ExecutionPayloadWithTxRootV1 `json:"execution_payload_header"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadWithTxRootV1 -field-override executionPayloadHeaderMarshaling -out gen_ed.go

// ExecutionPayloadWithTxRootV1 is the same as ExecutionPayloadV1 with a transactionsRoot in addition to transactions
type ExecutionPayloadWithTxRootV1 struct {
	ParentHash       common.Hash    `json:"parentHash" gencodec:"required"`
	FeeRecipient     common.Address `json:"feeRecipient" gencodec:"required"`
	StateRoot        common.Hash    `json:"stateRoot" gencodec:"required"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot" gencodec:"required"`
	LogsBloom        []byte         `json:"logsBloom" gencodec:"required"`
	Random           common.Hash    `json:"random" gencodec:"required"`
	Number           uint64         `json:"blockNumber" gencodec:"required"`
	GasLimit         uint64         `json:"gasLimit" gencodec:"required"`
	GasUsed          uint64         `json:"gasUsed" gencodec:"required"`
	Timestamp        uint64         `json:"timestamp" gencodec:"required"`
	ExtraData        []byte         `json:"extraData" gencodec:"required"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash        common.Hash    `json:"blockHash" gencodec:"required"`
	Transactions     *[]string      `json:"transactions,omitempty"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
}

// JSON type overrides for executableData.
type executionPayloadHeaderMarshaling struct {
	Number        hexutil.Uint64
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Timestamp     hexutil.Uint64
	BaseFeePerGas *hexutil.Big
	ExtraData     hexutil.Bytes
	LogsBloom     hexutil.Bytes
}
