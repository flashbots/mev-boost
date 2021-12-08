package lib

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SignedBeaconBlockHeader forked from https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#signedbeaconblockheader
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
	BodyRoot      json.RawMessage `json:"body_root"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadHeaderV1 -field-override executionPayloadHeaderMarshaling -out gen_ed.go

// ExecutionPayloadHeaderV1 is the same as ExecutionPayloadV1 with a transactionsRoot in place of transactions
type ExecutionPayloadHeaderV1 struct {
	ParentHash       common.Hash    `json:"parentHash"    gencodec:"required"`
	Coinbase         common.Address `json:"coinbase"      gencodec:"required"`
	StateRoot        common.Hash    `json:"stateRoot"     gencodec:"required"`
	ReceiptRoot      common.Hash    `json:"receiptRoot"   gencodec:"required"`
	LogsBloom        []byte         `json:"logsBloom"     gencodec:"required"`
	Random           common.Hash    `json:"random"        gencodec:"required"`
	Number           uint64         `json:"blockNumber"   gencodec:"required"`
	GasLimit         uint64         `json:"gasLimit"      gencodec:"required"`
	GasUsed          uint64         `json:"gasUsed"       gencodec:"required"`
	Timestamp        uint64         `json:"timestamp"     gencodec:"required"`
	ExtraData        []byte         `json:"extraData"     gencodec:"required"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash        common.Hash    `json:"blockHash"     gencodec:"required"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"  gencodec:"required"`
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
