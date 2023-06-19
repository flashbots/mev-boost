package config

import (
	"os"

	"github.com/flashbots/mev-boost/common"
)

// Set during build
const (
	// Version is the version of the software, set at build time
	Version = "v1.6-alpha1"

	// ForkVersion is the latest supported fork version at build time
	ForkVersion = "capella"
)

// Other settings
var (
	// ServerReadTimeoutMs sets the maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout.
	ServerReadTimeoutMs = common.GetEnvInt("MEV_BOOST_SERVER_READ_TIMEOUT_MS", 1000)

	// ServerReadHeaderTimeoutMs sets the amount of time allowed to read request headers.
	ServerReadHeaderTimeoutMs = common.GetEnvInt("MEV_BOOST_SERVER_READ_HEADER_TIMEOUT_MS", 1000)

	// ServerWriteTimeoutMs sets the maximum duration before timing out writes of the response.
	ServerWriteTimeoutMs = common.GetEnvInt("MEV_BOOST_SERVER_WRITE_TIMEOUT_MS", 0)

	// ServerIdleTimeoutMs sets the maximum amount of time to wait for the next request when keep-alives are enabled.
	ServerIdleTimeoutMs = common.GetEnvInt("MEV_BOOST_SERVER_IDLE_TIMEOUT_MS", 0)

	// ServerMaxHeaderBytes defines the max header byte size for requests (for dos prevention)
	ServerMaxHeaderBytes = common.GetEnvInt("MAX_HEADER_BYTES", 4000)

	// SkipRelaySignatureCheck can be used to disable relay signature check
	SkipRelaySignatureCheck = os.Getenv("SKIP_RELAY_SIGNATURE_CHECK") == "1"

	GenesisTime = int64(common.GetEnvInt("GENESIS_TIMESTAMP", common.GenesisTimeMainnet))
	SlotTimeSec = int64(common.GetEnvInt("SLOT_SEC", common.SlotTimeSecMainnet))
)
