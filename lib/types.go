package lib

// SignedBeaconBlockHeader see https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#signedbeaconblockheader
type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader `json:"message"`
	Signature string             `json:"signature"`
}

// BeaconBlockHeader see https://github.com/ethereum/consensus-specs/blob/v1.1.6/specs/phase0/beacon-chain.md#beaconblockheader
type BeaconBlockHeader struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}
