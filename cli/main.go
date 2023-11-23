package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/common"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server"
	"github.com/sirupsen/logrus"
)

const (
	genesisForkVersionMainnet = "0x00000000"
	genesisForkVersionSepolia = "0x90000069"
	genesisForkVersionGoerli  = "0x00001020"
	genesisForkVersionHolesky = "0x01017000"

	genesisTimeMainnet = 1606824023
	genesisTimeSepolia = 1655733600
	genesisTimeGoerli  = 1614588812
	genesisTimeHolesky = 1695902400
)

var (
	// errors
	errInvalidLoglevel = errors.New("invalid loglevel")

	// defaults
	defaultLogJSON           = os.Getenv("LOG_JSON") != ""
	defaultLogLevel          = common.GetEnv("LOG_LEVEL", "info")
	defaultListenAddr        = common.GetEnv("BOOST_LISTEN_ADDR", "localhost:18550")
	defaultRelayCheck        = os.Getenv("RELAY_STARTUP_CHECK") != ""
	defaultRelayMinBidEth    = common.GetEnvFloat64("MIN_BID_ETH", 0)
	defaultDisableLogVersion = os.Getenv("DISABLE_LOG_VERSION") == "1" // disables adding the version to every log entry
	defaultDebug             = os.Getenv("DEBUG") != ""
	defaultLogServiceTag     = os.Getenv("LOG_SERVICE_TAG")
	defaultRelays            = os.Getenv("RELAYS")
	defaultRelayMonitors     = os.Getenv("RELAY_MONITORS")
	defaultMaxRetries        = common.GetEnvInt("REQUEST_MAX_RETRIES", 5)

	defaultGenesisForkVersion = common.GetEnv("GENESIS_FORK_VERSION", "")
	defaultGenesisTime        = common.GetEnvInt("GENESIS_TIMESTAMP", -1)
	defaultUseSepolia         = os.Getenv("SEPOLIA") != ""
	defaultUseGoerli          = os.Getenv("GOERLI") != ""
	defaultUseHolesky         = os.Getenv("HOLESKY") != ""

	// mev-boost relay request timeouts (see also https://github.com/flashbots/mev-boost/issues/287)
	defaultTimeoutMsGetHeader         = common.GetEnvInt("RELAY_TIMEOUT_MS_GETHEADER", 950)   // timeout for getHeader requests
	defaultTimeoutMsGetPayload        = common.GetEnvInt("RELAY_TIMEOUT_MS_GETPAYLOAD", 4000) // timeout for getPayload requests
	defaultTimeoutMsRegisterValidator = common.GetEnvInt("RELAY_TIMEOUT_MS_REGVAL", 3000)     // timeout for registerValidator requests

	relays        relayList
	relayMonitors relayMonitorList

	// cli flags
	printVersion = flag.Bool("version", false, "only print version")
	logJSON      = flag.Bool("json", defaultLogJSON, "log in JSON format instead of text")
	logLevel     = flag.String("loglevel", defaultLogLevel, "minimum loglevel: trace, debug, info, warn/warning, error, fatal, panic")
	logDebug     = flag.Bool("debug", defaultDebug, "shorthand for '-loglevel debug'")
	logService   = flag.String("log-service", defaultLogServiceTag, "add a 'service=...' tag to all log messages")
	logNoVersion = flag.Bool("log-no-version", defaultDisableLogVersion, "disables adding the version to every log entry")

	listenAddr       = flag.String("addr", defaultListenAddr, "listen-address for mev-boost server")
	relayURLs        = flag.String("relays", defaultRelays, "relay urls - single entry or comma-separated list (scheme://pubkey@host)")
	relayCheck       = flag.Bool("relay-check", defaultRelayCheck, "check relay status on startup and on the status API call")
	relayMinBidEth   = flag.Float64("min-bid", defaultRelayMinBidEth, "minimum bid to accept from a relay [eth]")
	relayMonitorURLs = flag.String("relay-monitors", defaultRelayMonitors, "relay monitor urls - single entry or comma-separated list (scheme://host)")

	relayTimeoutMsGetHeader  = flag.Int("request-timeout-getheader", defaultTimeoutMsGetHeader, "timeout for getHeader requests to the relay [ms]")
	relayTimeoutMsGetPayload = flag.Int("request-timeout-getpayload", defaultTimeoutMsGetPayload, "timeout for getPayload requests to the relay [ms]")
	relayTimeoutMsRegVal     = flag.Int("request-timeout-regval", defaultTimeoutMsRegisterValidator, "timeout for registerValidator requests [ms]")

	relayRequestMaxRetries = flag.Int("request-max-retries", defaultMaxRetries, "maximum number of retries for a relay get payload request")

	// helpers
	mainnet = flag.Bool("mainnet", true, "use Mainnet")
	sepolia = flag.Bool("sepolia", defaultUseSepolia, "use Sepolia")
	goerli  = flag.Bool("goerli", defaultUseGoerli, "use Goerli")
	holesky = flag.Bool("holesky", defaultUseHolesky, "use Holesky")

	useCustomGenesisForkVersion = flag.String("genesis-fork-version", defaultGenesisForkVersion, "use a custom genesis fork version")
	useCustomGenesisTime        = flag.Int("genesis-timestamp", defaultGenesisTime, "use a custom genesis timestamp (unix seconds)")
)

var log = logrus.NewEntry(logrus.New())

// Main starts the mev-boost cli
func Main() {
	// process repeatable flags
	flag.Var(&relays, "relay", "a single relay, can be specified multiple times")
	flag.Var(&relayMonitors, "relay-monitor", "a single relay monitor, can be specified multiple times")

	// parse flags and get started
	flag.Parse()

	// perhaps only print the version
	if *printVersion {
		fmt.Printf("mev-boost %s\n", config.Version) //nolint
		return
	}

	err := setupLogging()
	if err != nil {
		flag.Usage()
		log.WithError(err).Fatal("failed setting up logging")
	}

	genesisForkVersionHex := ""
	var genesisTime uint64

	switch {
	case *useCustomGenesisForkVersion != "":
		genesisForkVersionHex = *useCustomGenesisForkVersion
	case *sepolia:
		genesisForkVersionHex = genesisForkVersionSepolia
		genesisTime = genesisTimeSepolia
	case *goerli:
		genesisForkVersionHex = genesisForkVersionGoerli
		genesisTime = genesisTimeGoerli
	case *holesky:
		genesisForkVersionHex = genesisForkVersionHolesky
		genesisTime = genesisTimeHolesky
	case *mainnet:
		genesisForkVersionHex = genesisForkVersionMainnet
		genesisTime = genesisTimeMainnet
	default:
		flag.Usage()
		log.Fatal("please specify a genesis fork version (eg. -mainnet / -sepolia / -goerli / -holesky / -genesis-fork-version flags)")
	}
	log.Infof("using genesis fork version: %s", genesisForkVersionHex)

	if *useCustomGenesisTime > -1 {
		genesisTime = uint64(*useCustomGenesisTime)
	}

	// For backwards compatibility with the -relays flag.
	if *relayURLs != "" {
		for _, relayURL := range strings.Split(*relayURLs, ",") {
			err := relays.Set(strings.TrimSpace(relayURL))
			if err != nil {
				log.WithError(err).WithField("relay", relayURL).Fatal("Invalid relay URL")
			}
		}
	}

	if len(relays) == 0 {
		flag.Usage()
		log.Fatal("no relays specified")
	}
	log.Infof("using %d relays", len(relays))
	for index, relay := range relays {
		log.Infof("relay #%d: %s", index+1, relay.String())
	}

	// For backwards compatibility with the -relay-monitors flag.
	if *relayMonitorURLs != "" {
		for _, relayMonitorURL := range strings.Split(*relayMonitorURLs, ",") {
			err := relayMonitors.Set(strings.TrimSpace(relayMonitorURL))
			if err != nil {
				log.WithError(err).WithField("relayMonitor", relayMonitorURL).Fatal("Invalid relay monitor URL")
			}
		}
	}

	if len(relayMonitors) > 0 {
		log.Infof("using %d relay monitors", len(relayMonitors))
		for index, relayMonitor := range relayMonitors {
			log.Infof("relay-monitor #%d: %s", index+1, relayMonitor.String())
		}
	}

	if *relayMinBidEth < 0.0 {
		log.Fatal("Please specify a non-negative minimum bid")
	}

	if *relayMinBidEth > 1000000.0 {
		log.Fatal("Minimum bid is too large, please ensure min-bid is denominated in Ethers")
	}

	if *relayMinBidEth > 0.0 {
		log.Infof("minimum bid: %v eth", *relayMinBidEth)
	}

	relayMinBidWei, err := common.FloatEthTo256Wei(*relayMinBidEth)
	if err != nil {
		log.WithError(err).Fatal("failed converting min bid")
	}

	opts := server.BoostServiceOpts{
		Log:                      log,
		ListenAddr:               *listenAddr,
		Relays:                   relays,
		RelayMonitors:            relayMonitors,
		GenesisForkVersionHex:    genesisForkVersionHex,
		GenesisTime:              genesisTime,
		RelayCheck:               *relayCheck,
		RelayMinBid:              *relayMinBidWei,
		RequestTimeoutGetHeader:  time.Duration(*relayTimeoutMsGetHeader) * time.Millisecond,
		RequestTimeoutGetPayload: time.Duration(*relayTimeoutMsGetPayload) * time.Millisecond,
		RequestTimeoutRegVal:     time.Duration(*relayTimeoutMsRegVal) * time.Millisecond,
		RequestMaxRetries:        *relayRequestMaxRetries,
	}
	service, err := server.NewBoostService(opts)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	if *relayCheck && service.CheckRelays() == 0 {
		log.Error("no relay passed the health-check!")
	}

	log.Println("listening on", *listenAddr)
	log.Fatal(service.StartHTTPServer())
}

func setupLogging() error {
	// setup logging
	log.Logger.SetOutput(os.Stdout)
	if *logJSON {
		log.Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.RFC3339Milli,
		})
	} else {
		log.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: config.RFC3339Milli,
		})
	}
	if *logDebug {
		*logLevel = "debug"
	}
	if *logLevel != "" {
		lvl, err := logrus.ParseLevel(*logLevel)
		if err != nil {
			return fmt.Errorf("%w: %s", errInvalidLoglevel, *logLevel)
		}
		log.Logger.SetLevel(lvl)
	}
	if *logService != "" {
		log = log.WithField("service", *logService)
	}

	// Add version to logs and say hello
	addVersionToLogs := !*logNoVersion
	if addVersionToLogs {
		log = log.WithField("version", config.Version)
		log.Infof("starting mev-boost")
	} else {
		log.Infof("starting mev-boost %s", config.Version)
	}
	log.Debug("debug logging enabled")
	return nil
}
