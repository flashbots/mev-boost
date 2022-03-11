package lib

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var nilHash = common.Hash{}

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

// BlindedBeaconBlockBodyPartial a partial block body only containing a payload, in both snake_case and camelCase
type BlindedBeaconBlockBodyPartial struct {
	ExecutionPayload      ExecutionPayloadHeaderOnlyBlockHash `json:"execution_payload_header"`
	ExecutionPayloadCamel ExecutionPayloadHeaderOnlyBlockHash `json:"executionPayloadHeader"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadWithTxRootV1 -field-override executionPayloadHeaderMarshaling -out gen_ed.go

// ExecutionPayloadWithTxRootV1 is the same as ExecutionPayloadV1 with a transactionsRoot in addition to transactions
type ExecutionPayloadWithTxRootV1 struct {
	ParentHash       common.Hash    `json:"parentHash" gencodec:"required"`
	FeeRecipient     common.Address `json:"feeRecipient" gencodec:"required"`
	StateRoot        common.Hash    `json:"stateRoot" gencodec:"required"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot" gencodec:"required"`
	LogsBloom        []byte         `json:"logsBloom" gencodec:"required"`
	PrevRandao       common.Hash    `json:"prevRandao" gencodec:"required"`
	Number           uint64         `json:"blockNumber" gencodec:"required"`
	GasLimit         uint64         `json:"gasLimit" gencodec:"required"`
	GasUsed          uint64         `json:"gasUsed" gencodec:"required"`
	Timestamp        uint64         `json:"timestamp" gencodec:"required"`
	ExtraData        []byte         `json:"extraData" gencodec:"required"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash        common.Hash    `json:"blockHash" gencodec:"required"`
	Transactions     *[]string      `json:"transactions,omitempty"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	FeeRecipientDiff *big.Int       `json:"feeRecipientDiff" gencodec:"required"`
}

// ExecutionPayloadHeaderOnlyBlockHash an execution payload with only a block hash, used for BlindedBeaconBlockBodyPartial
type ExecutionPayloadHeaderOnlyBlockHash struct {
	BlockHash      string `json:"block_hash"`
	BlockHashCamel string `json:"blockHash"`
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

// ForkchoiceStatus as defined in the engine spec: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_forkchoiceupdatedv1
type ForkchoiceStatus string

var (
	// ForkchoiceStatusValid indicates the fork choice is valid
	ForkchoiceStatusValid ForkchoiceStatus = "VALID"

	// ForkchoiceStatusInvalid indicates the fork choice is invalid
	ForkchoiceStatusInvalid ForkchoiceStatus = "INVALID"

	// ForkchoiceStatusAccepted indicates the fork choice is accepted
	ForkchoiceStatusAccepted ForkchoiceStatus = "ACCEPTED"

	// ForkchoiceStatusSyncing indicates the node is still syncing
	ForkchoiceStatusSyncing ForkchoiceStatus = "SYNCING"

	// ForkchoiceStatusInvalidBlockHash indicates supplied blockhash is unknown
	ForkchoiceStatusInvalidBlockHash ForkchoiceStatus = "INVALID_BLOCK_HASH"

	// ForkchoiceStatusInvalidTerminalBlock indicates the latest valid ancestor is not a descendant of configured TTD block
	ForkchoiceStatusInvalidTerminalBlock ForkchoiceStatus = "INVALID_TERMINAL_BLOCK"
)

// ForkChoiceResponse is a workaround for mergemock allowing these fields to be null
type ForkChoiceResponse struct {
	PayloadStatus PayloadStatus  `json:"payloadStatus,omitempty"`
	PayloadID     *hexutil.Bytes `json:"payloadId,omitempty"`
}

// PayloadStatus is used in ForkChoiceResponse
type PayloadStatus struct {
	Status          ForkchoiceStatus `json:"status,omitempty"`
	LatestValidHash string           `json:"latestValidHash,omitempty"`
	ValidationError string           `json:"validationError,omitempty"`
}
