package cli

import "github.com/urfave/cli/v3"

const (
	LoggingCategory = "LOGGING AND DEBUGGING"
	GenesisCategory = "GENESIS"
	RelayCategory   = "RELAYS"
	GeneralCategory = "GENERAL"
)

var flags = []cli.Flag{
	// general
	addrFlag,
	versionFlag,
	// logging
	jsonFlag,
	debugFlag,
	logLevelFlag,
	logServiceFlag,
	logNoVersionFlag,
	// genesis
	customGenesisForkFlag,
	customGenesisTimeFlag,
	mainnetFlag,
	sepoliaFlag,
	holeskyFlag,
	// relay
	relaysFlag,
	relayMonitorFlag,
	minBidFlag,
	relayCheckFlag,
	timeoutGetHeaderFlag,
	timeoutGetPayloadFlag,
	timeoutRegValFlag,
	maxRetriesFlag,
	privilegedBuildersFlag,
}

var (
	// General
	addrFlag = &cli.StringFlag{
		Name:     "addr",
		Sources:  cli.EnvVars("BOOST_LISTEN_ADDR"),
		Value:    "localhost:18550",
		Usage:    "listen-address for mev-boost server",
		Category: GeneralCategory,
	}
	versionFlag = &cli.BoolFlag{
		Name:     "version",
		Usage:    "print version",
		Category: GeneralCategory,
	}
	// Logging and debugging
	jsonFlag = &cli.BoolFlag{
		Name:     "json",
		Sources:  cli.EnvVars("LOG_JSON"),
		Usage:    "log in JSON format instead of text",
		Category: LoggingCategory,
	}
	debugFlag = &cli.BoolFlag{
		Name:     "debug",
		Sources:  cli.EnvVars("DEBUG"),
		Usage:    "shorthand for '--loglevel debug'",
		Category: LoggingCategory,
	}
	logLevelFlag = &cli.StringFlag{
		Name:     "loglevel",
		Sources:  cli.EnvVars("LOG_LEVEL"),
		Value:    "info",
		Usage:    "minimum loglevel: trace, debug, info, warn/warning, error, fatal, panic",
		Category: LoggingCategory,
	}
	logServiceFlag = &cli.StringFlag{
		Name:     "log-service",
		Sources:  cli.EnvVars("LOG_SERVICE_TAG"),
		Value:    "",
		Usage:    "add a 'service=...' tag to all log messages",
		Category: LoggingCategory,
	}
	logNoVersionFlag = &cli.BoolFlag{
		Name:     "log-no-version",
		Sources:  cli.EnvVars("DISABLE_LOG_VERSION"),
		Usage:    "disables adding the version to every log entry",
		Category: LoggingCategory,
	}
	// Genesis Flags
	customGenesisForkFlag = &cli.StringFlag{
		Name:     "genesis-fork-version",
		Sources:  cli.EnvVars("GENESIS_FORK_VERSION"),
		Usage:    "use a custom genesis fork version",
		Category: GenesisCategory,
	}
	customGenesisTimeFlag = &cli.UintFlag{
		Name:     "genesis-timestamp",
		Sources:  cli.EnvVars("GENESIS_TIMESTAMP"),
		Usage:    "use a custom genesis timestamp (unix seconds)",
		Category: GenesisCategory,
	}
	mainnetFlag = &cli.BoolFlag{
		Name:     "mainnet",
		Sources:  cli.EnvVars("MAINNET"),
		Usage:    "use Mainnet",
		Value:    true,
		Category: GenesisCategory,
	}
	sepoliaFlag = &cli.BoolFlag{
		Name:     "sepolia",
		Sources:  cli.EnvVars("SEPOLIA"),
		Usage:    "use Sepolia",
		Category: GenesisCategory,
	}
	holeskyFlag = &cli.BoolFlag{
		Name:     "holesky",
		Sources:  cli.EnvVars("HOLESKY"),
		Usage:    "use Holesky",
		Category: GenesisCategory,
	}
	// Relay
	relaysFlag = &cli.StringSliceFlag{
		Name:     "relay",
		Aliases:  []string{"relays"},
		Sources:  cli.EnvVars("RELAYS"),
		Usage:    "relay urls - single entry or comma-separated list (scheme://pubkey@host)",
		Category: RelayCategory,
	}
	relayMonitorFlag = &cli.StringSliceFlag{
		Name:     "relay-monitors",
		Aliases:  []string{"relay-monitor"},
		Sources:  cli.EnvVars("RELAY_MONITORS"),
		Usage:    "relay monitor urls - single entry or comma-separated list (scheme://host)",
		Category: RelayCategory,
	}
	minBidFlag = &cli.FloatFlag{
		Name:     "min-bid",
		Sources:  cli.EnvVars("MIN_BID_ETH"),
		Usage:    "minimum bid to accept from a relay [eth]",
		Category: RelayCategory,
	}
	relayCheckFlag = &cli.BoolFlag{
		Name:     "relay-check",
		Sources:  cli.EnvVars("RELAY_STARTUP_CHECK"),
		Usage:    "check relay status on startup and on the status API call",
		Category: RelayCategory,
	}
	// mev-boost relay request timeouts (see also https://github.com/flashbots/mev-boost/issues/287)
	timeoutGetHeaderFlag = &cli.IntFlag{
		Name:     "request-timeout-getheader",
		Sources:  cli.EnvVars("RELAY_TIMEOUT_MS_GETHEADER"),
		Usage:    "timeout for getHeader requests to the relay [ms]",
		Value:    950,
		Category: RelayCategory,
	}
	timeoutGetPayloadFlag = &cli.IntFlag{
		Name:     "request-timeout-getpayload",
		Sources:  cli.EnvVars("RELAY_TIMEOUT_MS_GETPAYLOAD"),
		Usage:    "timeout for getPayload requests to the relay [ms]",
		Value:    4000,
		Category: RelayCategory,
	}
	timeoutRegValFlag = &cli.IntFlag{
		Name:     "request-timeout-regval",
		Sources:  cli.EnvVars("RELAY_TIMEOUT_MS_REGVAL"),
		Usage:    "timeout for registerValidator requests [ms]",
		Value:    3000,
		Category: RelayCategory,
	}
	maxRetriesFlag = &cli.IntFlag{
		Name:     "request-max-retries",
		Sources:  cli.EnvVars("REQUEST_MAX_RETRIES"),
		Usage:    "maximum number of retries for a relay get payload request",
		Value:    5,
		Category: RelayCategory,
	}
	privilegedBuildersFlag = &cli.StringSliceFlag{
		Name:     "privileged-builder",
		Sources:  cli.EnvVars("PRIVILEGED_BUILDER"),
		Usage:    "relay username/pubkey  - single entry or comma-separated list",
		Category: RelayCategory,
	}
)
