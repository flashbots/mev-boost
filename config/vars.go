package config

import (
	"github.com/flashbots/go-utils/cli"
)

// Set during build
var (
	// Version is the version of the software, set at build time
	Version = "v1.3.3-dev"
)

// Other settings
var (
	// ServerReadTimeoutMs sets the maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout.
	ServerReadTimeoutMs = cli.GetEnvInt("MEV_BOOST_SERVER_READ_TIMEOUT_MS", 1000)

	// ServerReadHeaderTimeoutMs sets the amount of time allowed to read request headers.
	ServerReadHeaderTimeoutMs = cli.GetEnvInt("MEV_BOOST_SERVER_READ_HEADER_TIMEOUT_MS", 1000)

	// ServerWriteTimeoutMs sets the maximum duration before timing out writes of the response.
	ServerWriteTimeoutMs = cli.GetEnvInt("MEV_BOOST_SERVER_WRITE_TIMEOUT_MS", 0)

	// ServerIdleTimeoutMs sets the maximum amount of time to wait for the next request when keep-alives are enabled.
	ServerIdleTimeoutMs = cli.GetEnvInt("MEV_BOOST_SERVER_IDLE_TIMEOUT_MS", 0)

	ServerMaxHeaderBytes = cli.GetEnvInt("MAX_HEADER_BYTES", 4000) // max header byte size for requests for dos prevention
)
