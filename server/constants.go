package server

const (
	HeaderAccept              = "Accept"
	HeaderContentType         = "Content-Type"
	HeaderEthConsensusVersion = "Eth-Consensus-Version"
	HeaderKeySlotUID          = "X-MEVBoost-SlotID"
	HeaderKeyVersion          = "X-MEVBoost-Version"
	HeaderStartTimeUnixMS     = "X-MEVBoost-StartTimeUnixMS"
	HeaderUserAgent           = "User-Agent"

	MediaTypeJSON        = "application/json"
	MediaTypeOctetStream = "application/octet-stream"

	EthConsensusVersionBellatrix = "bellatrix"
	EthConsensusVersionCapella   = "capella"
	EthConsensusVersionDeneb     = "deneb"
	EthConsensusVersionElectra   = "electra"
)
