package params

const (
	// Router paths
	// These paths will be deprecated in favor of the new Gloas builder-api.
	PathStatus            = "/eth/v1/builder/status"
	PathRegisterValidator = "/eth/v1/builder/validators"
	PathGetHeader         = "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}"
	PathGetPayload        = "/eth/v1/builder/blinded_blocks"
	PathGetPayloadV2      = "/eth/v2/builder/blinded_blocks"

	// gloas builder-api
	PathGetExecutionPayloadBid   = "/eth/v1/builder/execution_payload_bid/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{parent_root:0x[a-fA-F0-9]+}/{proposer_pubkey:0x[a-fA-F0-9]+}"
	PathSubmitBuilderPreferences = "/eth/v1/builder/builder_preferences/{proposer_pubkey:0x[a-fA-F0-9]+}"
	PathSubmitSignedBeaconBlock  = "/eth/v1/builder/beacon_blocks"
)
