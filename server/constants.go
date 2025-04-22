package server

const (
	HeaderAccept              = "Accept"
	HeaderContentType         = "Content-Type"
	HeaderEthConsensusVersion = "Eth-Consensus-Version"
	HeaderKeySlotUID          = "X-MEVBoost-SlotID"
	HeaderKeyVersion          = "X-MEVBoost-Version"
	HeaderUserAgent           = "User-Agent"
	// Header which communicates when a request was sent. Used to measure latency.
	HeaderDateMilliseconds = "Date-Milliseconds"

	MediaTypeJSON        = "application/json"
	MediaTypeOctetStream = "application/octet-stream"

	EthConsensusVersionBellatrix = "bellatrix"
	EthConsensusVersionCapella   = "capella"
	EthConsensusVersionDeneb     = "deneb"
	EthConsensusVersionElectra   = "electra"
)
