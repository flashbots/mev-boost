package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Generate SSZ encoding: make generate-ssz

// NilHash is an empty hash.
var NilHash = Hash{}

// Eth1Data https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#eth1data
type Eth1Data struct {
	DepositRoot  Root   `json:"deposit_root" ssz-size:"32"`
	DepositCount uint64 `json:"deposit_count,string"`
	BlockHash    Hash   `json:"block_hash" ssz-size:"32"`
}

// BeaconBlockHeader https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
type BeaconBlockHeader struct {
	Slot          uint64 `json:"slot,string"`
	ProposerIndex uint64 `json:"proposer_index,string"`
	ParentRoot    Root   `json:"parent_root" ssz-size:"32"`
	StateRoot     Root   `json:"state_root" ssz-size:"32"`
	BodyRoot      Root   `json:"body_root" ssz-size:"32"`
}

// SignedBeaconBlockHeader https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#signedbeaconblockheader
type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader `json:"message"`
	Signature Signature          `json:"signature" ssz-size:"96"`
}

// ProposerSlashing https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#proposerslashing
type ProposerSlashing struct {
	A *SignedBeaconBlockHeader `json:"signed_header_1"`
	B *SignedBeaconBlockHeader `json:"signed_header_2"`
}

// Checkpoint ...
type Checkpoint struct {
	Epoch uint64 `json:"epoch,string"`
	Root  Root   `json:"root" ssz-size:"32"`
}

// AttestationData https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#attestationdata
type AttestationData struct {
	Slot      uint64      `json:"slot,string"`
	Index     uint64      `json:"index,string"`
	BlockRoot Root        `json:"beacon_block_root" ssz-size:"32"`
	Source    *Checkpoint `json:"source"`
	Target    *Checkpoint `json:"target"`
}

// IndexedAttestation https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#indexedattestation
type IndexedAttestation struct {
	AttestingIndices []uint64         `json:"attesting_indices" ssz-max:"2048"` // MAX_VALIDATORS_PER_COMMITTEE
	Data             *AttestationData `json:"data"`
	Signature        Signature        `json:"signature" ssz-size:"96"`
}

// AttesterSlashing https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#attesterslashing
type AttesterSlashing struct {
	A *IndexedAttestation `json:"attestation_1"`
	B *IndexedAttestation `json:"attestation_2"`
}

// Attestation https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#attestation
type Attestation struct {
	AggregationBits hexutil.Bytes    `json:"aggregation_bits" ssz-max:"2048"` // MAX_VALIDATORS_PER_COMMITTEE
	Data            *AttestationData `json:"data"`
	Signature       Signature        `json:"signature" ssz-size:"96"`
}

// Deposit https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#deposit
type Deposit struct {
	Pubkey                PublicKey `json:"pubkey" ssz-size:"48"`
	WithdrawalCredentials Hash      `json:"withdrawal_credentials" ssz-size:"32"`
	Amount                uint64    `json:"amount,string"`
	Signature             Signature `json:"signature" ssz-size:"96"`
}

// VoluntaryExit https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#voluntaryexit
type VoluntaryExit struct {
	Epoch          uint64 `json:"epoch,string"`
	ValidatorIndex uint64 `json:"validator_index,string"`
}

// SyncAggregate ...
type SyncAggregate struct {
	CommitteeBits      CommitteeBits `json:"sync_committee_bits" ssz-size:"64"`
	CommitteeSignature Signature     `json:"sync_committee_signature" ssz-size:"96"`
}

// ExecutionPayloadHeader https://github.com/ethereum/consensus-specs/blob/dev/specs/bellatrix/beacon-chain.md#executionpayloadheader
type ExecutionPayloadHeader struct {
	ParentHash       Hash          `json:"parent_hash" ssz-size:"32"`
	FeeRecipient     Address       `json:"fee_recipient" ssz-size:"20"`
	StateRoot        Root          `json:"state_root" ssz-size:"32"`
	ReceiptsRoot     Root          `json:"receipts_root" ssz-size:"32"`
	LogsBloom        Bloom         `json:"logs_bloom" ssz-size:"256"`
	Random           Hash          `json:"prev_randao" ssz-size:"32"`
	BlockNumber      uint64        `json:"block_number,string"`
	GasLimit         uint64        `json:"gas_limit,string"`
	GasUsed          uint64        `json:"gas_used,string"`
	Timestamp        uint64        `json:"timestamp,string"`
	ExtraData        hexutil.Bytes `json:"extra_data" ssz-size:"32"`
	BaseFeePerGas    U256Str       `json:"base_fee_per_gas" ssz-max:"32"`
	BlockHash        Hash          `json:"block_hash" ssz-size:"32"`
	TransactionsRoot Root          `json:"transactions_root" ssz-size:"32"`
}

// ExecutionPayload https://github.com/ethereum/consensus-specs/blob/dev/specs/bellatrix/beacon-chain.md#executionpayload
type ExecutionPayload struct {
	ParentHash    Hash            `json:"parent_hash" ssz-size:"32"`
	FeeRecipient  Address         `json:"fee_recipient" ssz-size:"20"`
	StateRoot     Root            `json:"state_root" ssz-size:"32"`
	ReceiptsRoot  Root            `json:"receipts_root" ssz-size:"32"`
	LogsBloom     Bloom           `json:"logs_bloom" ssz-size:"256"`
	Random        Hash            `json:"prev_randao" ssz-size:"32"`
	BlockNumber   uint64          `json:"block_number,string"`
	GasLimit      uint64          `json:"gas_limit,string"`
	GasUsed       uint64          `json:"gas_used,string"`
	Timestamp     uint64          `json:"timestamp,string"`
	ExtraData     hexutil.Bytes   `json:"extra_data" ssz-size:"32"`
	BaseFeePerGas U256Str         `json:"base_fee_per_gas" ssz-max:"32"`
	BlockHash     Hash            `json:"block_hash" ssz-size:"32"`
	Transactions  []hexutil.Bytes `json:"transactions"` // ssz-size/ssz-max:"2048"?
}

// BlindedBeaconBlockBody https://github.com/ethereum/beacon-APIs/blob/master/types/bellatrix/block.yaml#L65
type BlindedBeaconBlockBody struct {
	RandaoReveal           Signature               `json:"randao_reveal" ssz-size:"96"`
	Eth1Data               *Eth1Data               `json:"eth1_data"`
	Graffiti               Hash                    `json:"graffiti" ssz-size:"32"`
	ProposerSlashings      []*ProposerSlashing     `json:"proposer_slashings" ssz-max:"16"`
	AttesterSlashings      []*AttesterSlashing     `json:"attester_slashings" ssz-max:"2"`
	Attestations           []*Attestation          `json:"attestations" ssz-max:"128"`
	Deposits               []*Deposit              `json:"deposits" ssz-max:"4"`
	VoluntaryExits         []*VoluntaryExit        `json:"voluntary_exits" ssz-max:"16"`
	SyncAggregate          *SyncAggregate          `json:"sync_aggregate"`
	ExecutionPayloadHeader *ExecutionPayloadHeader `json:"execution_payload_header"`
}

// BlindedBeaconBlock https://github.com/ethereum/beacon-APIs/blob/master/types/bellatrix/block.yaml#L74
type BlindedBeaconBlock struct {
	Slot          uint64                  `json:"slot,string"`
	ProposerIndex uint64                  `json:"proposer_index,string"`
	ParentRoot    Root                    `json:"parent_root" ssz-size:"32"`
	StateRoot     Root                    `json:"state_root" ssz-size:"32"`
	Body          *BlindedBeaconBlockBody `json:"body"`
}

// RegisterValidatorRequestMessage https://github.com/ethereum/beacon-APIs/blob/master/types/registration.yaml
type RegisterValidatorRequestMessage struct {
	FeeRecipient Address   `json:"fee_recipient" ssz-size:"20"` // type was Address
	GasLimit     uint64    `json:"gas_limit,string"`
	Timestamp    uint64    `json:"timestamp,string"`
	Pubkey       PublicKey `json:"pubkey" ssz-size:"48"` // type was PublicKey
}

// SignedValidatorRegistration https://github.com/ethereum/beacon-APIs/blob/master/types/registration.yaml#L18
type SignedValidatorRegistration struct {
	Message   *RegisterValidatorRequestMessage `json:"message"`
	Signature Signature                        `json:"signature"`
}

// BuilderBid https://github.com/ethereum/builder-specs/pull/2/files#diff-b37cbf48e8754483e30e7caaadc5defc8c3c6e1aaf3273ee188d787b7c75d993
type BuilderBid struct {
	Header *ExecutionPayloadHeader `json:"header"`
	Value  U256Str                 `json:"value" ssz-size:"32"`
	Pubkey PublicKey               `json:"pubkey" ssz-size:"48"`
}

// SignedBuilderBid https://github.com/ethereum/builder-specs/pull/2/files#diff-b37cbf48e8754483e30e7caaadc5defc8c3c6e1aaf3273ee188d787b7c75d993
type SignedBuilderBid struct {
	Message   *BuilderBid `json:"message"`
	Signature Signature   `json:"signature"`
}

// GetHeaderResponse is the response payload from the getHeader request: https://github.com/ethereum/builder-specs/pull/2/files#diff-c80f52e38c99b1049252a99215450a29fd248d709ffd834a9480c98a233bf32c
type GetHeaderResponse struct {
	Version string            `json:"version"`
	Data    *SignedBuilderBid `json:"data"`
}

// SignedBlindedBeaconBlock https://github.com/ethereum/beacon-APIs/blob/master/types/bellatrix/block.yaml#L83
type SignedBlindedBeaconBlock struct {
	Message   *BlindedBeaconBlock `json:"message"`
	Signature Signature           `json:"signature"`
}

// GetPayloadResponse is the response payload from the getPayload request: https://github.com/ethereum/builder-specs/pull/2/files#diff-8446716b376f3ffe88737f9773ce2ff21adc2bc0f2c9a140dcc2e9d632091ba4
type GetPayloadResponse struct {
	Version string            `json:"version"`
	Data    *ExecutionPayload `json:"data"`
}
