package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Generate SSZ encoding: make generate-ssz

type Eth1Data struct {
	DepositRoot  Root   `json:"deposit_root" ssz-size:"32"`
	DepositCount uint64 `json:"deposit_count,string"`
	BlockHash    Hash   `json:"block_hash" ssz-size:"32"`
}

type BeaconBlockHeader struct {
	Slot          uint64 `json:"slot,string"`
	ProposerIndex uint64 `json:"proposer_index,string"`
	ParentRoot    Root   `json:"parent_root" ssz-size:"32"`
	StateRoot     Root   `json:"state_root" ssz-size:"32"`
	BodyRoot      Root   `json:"body_root" ssz-size:"32"`
}

type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader `json:"message"`
	Signature Signature          `json:"signature" ssz-size:"96"`
}

type ProposerSlashing struct {
	A *SignedBeaconBlockHeader `json:"signed_header_1"`
	B *SignedBeaconBlockHeader `json:"signed_header_2"`
}

type Checkpoint struct {
	Epoch uint64 `json:"epoch,string"`
	Root  Root   `json:"root" ssz-size:"32"`
}

type AttestationData struct {
	Slot      uint64      `json:"slot,string"`
	Index     uint64      `json:"index,string"`
	BlockRoot Root        `json:"beacon_block_root" ssz-size:"32"`
	Source    *Checkpoint `json:"source"`
	Target    *Checkpoint `json:"target"`
}

type IndexedAttestation struct {
	AttestingIndices []uint64         `json:"attesting_indices" ssz-max:"2048"` // MAX_VALIDATORS_PER_COMMITTEE
	Data             *AttestationData `json:"data"`
	Signature        Signature        `json:"signature" ssz-size:"96"`
}

type AttesterSlashing struct {
	A *IndexedAttestation `json:"attestation_1"`
	B *IndexedAttestation `json:"attestation_2"`
}

type Attestation struct {
	AggregationBits hexutil.Bytes    `json:"aggregation_bits" ssz-max:"2048"` // MAX_VALIDATORS_PER_COMMITTEE
	Data            *AttestationData `json:"data"`
	Signature       Signature        `json:"signature" ssz-size:"96"`
}

type Deposit struct {
	Pubkey                PublicKey `json:"pubkey" ssz-size:"48"`
	WithdrawalCredentials Hash      `json:"withdrawal_credentials" ssz-size:"32"`
	Amount                uint64    `json:"amount,string"`
	Signature             Signature `json:"signature" ssz-size:"96"`
}

type VoluntaryExit struct {
	Epoch          uint64 `json:"epoch,string"`
	ValidatorIndex uint64 `json:"validator_index,string"`
}

type SyncAggregate struct {
	CommitteeBits      CommitteeBits `json:"sync_committee_bits" ssz-size:"64"`
	CommitteeSignature Signature     `json:"sync_committee_signature" ssz-size:"96"`
}

type ExecutionPayloadHeader struct {
	ParentHash       Hash    `json:"parent_hash" ssz-size:"32"`
	FeeRecipient     Address `json:"fee_recipient" ssz-size:"20"`
	StateRoot        Root    `json:"state_root" ssz-size:"32"`
	ReceiptsRoot     Root    `json:"receipts_root" ssz-size:"32"`
	LogsBloom        Bloom   `json:"logs_bloom" ssz-size:"256"`
	Random           Hash    `json:"prev_randao" ssz-size:"32"`
	BlockNumber      uint64  `json:"block_number,string"`
	GasLimit         uint64  `json:"gas_limit,string"`
	GasUsed          uint64  `json:"gas_used,string"`
	Timestamp        uint64  `json:"timestamp,string"`
	ExtraData        Hash    `json:"extra_data" ssz-size:"32"`
	BaseFeePerGas    U256Str `json:"base_fee_per_gas" ssz-max:"32"`
	BlockHash        Hash    `json:"block_hash" ssz-size:"32"`
	TransactionsRoot Root    `json:"transactions_root" ssz-size:"32"`
}

type ExecutionPayloadREST struct {
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
	ExtraData     Hash            `json:"extra_data" ssz-size:"32"`
	BaseFeePerGas U256Str         `json:"base_fee_per_gas" ssz-max:"32"`
	BlockHash     Hash            `json:"block_hash" ssz-size:"32"`
	Transactions  []hexutil.Bytes `json:"transactions"` // ssz-size/ssz-max:"2048"?
}

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

type BlindedBeaconBlock struct {
	Slot          uint64                  `json:"slot,string"`
	ProposerIndex uint64                  `json:"proposer_index,string"`
	ParentRoot    Root                    `json:"parent_root" ssz-size:"32"`
	StateRoot     Root                    `json:"state_root" ssz-size:"32"`
	Body          *BlindedBeaconBlockBody `json:"body"`
}

type RegisterValidatorRequestMessage struct {
	FeeRecipient Address   `json:"fee_recipient" ssz-size:"20"` // type was Address
	GasLimit     uint64    `json:"gas_limit,string"`
	Timestamp    uint64    `json:"timestamp,string"`
	Pubkey       PublicKey `json:"pubkey" ssz-size:"48"` // type was PublicKey
}

type RegisterValidatorRequest struct {
	Message   *RegisterValidatorRequestMessage `json:"message"`
	Signature Signature                        `json:"signature"`
}

type BuilderBid struct {
	Header *ExecutionPayloadHeader `json:"header"`
	Value  U256Str                 `json:"value" ssz-size:"32"`
	Pubkey PublicKey               `json:"pubkey" ssz-size:"48"`
}

type SignedBuilderBid struct {
	Message   *BuilderBid `json:"message"`
	Signature Signature   `json:"signature"`
}

type GetHeaderResponse struct {
	Version string            `json:"version"`
	Data    *SignedBuilderBid `json:"data"`
}

type SignedBlindedBeaconBlock struct {
	Message   *BlindedBeaconBlock `json:"message"`
	Signature Signature           `json:"signature"`
}

type GetPayloadResponse struct {
	Version string                `json:"version"`
	Data    *ExecutionPayloadREST `json:"data"`
}
