package config

import (
	"os"

	"github.com/flashbots/go-utils/cli"
)

// Set during build
var (
	// Version is the version of the software, set at build time
	Version = "v1.3.2-dev"
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

	// ServerMaxHeaderBytes sets the maximum header byte size for requests for dos prevention
	ServerMaxHeaderBytes = cli.GetEnvInt("MAX_HEADER_BYTES", 4000)

	// NewReleaseCheckIntervalHours is the interval between checking for new mev-boost releases in hours
	NewReleaseCheckIntervalHours = cli.GetEnvInt("NEW_RELEASE_CHECK_INTERVAL_H", 2)

	// DisableNewReleaseCheck disables the check for new mev-boost releases
	DisableNewReleaseCheck = os.Getenv("DISABLE_NEW_RELEASE_CHECK") == "1"
)
