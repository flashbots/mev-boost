package cli

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server"
	"github.com/sirupsen/logrus"
)

const (
	genesisForkVersionMainnet = "0x00000000"
	genesisForkVersionKiln    = "0x70000069" // https://github.com/eth-clients/merge-testnets/blob/main/kiln/config.yaml#L10
	genesisForkVersionRopsten = "0x80000069"
	genesisForkVersionSepolia = "0x90000069"
	genesisForkVersionGoerli  = "0x00001020"
)

var (
	// defaults
	defaultLogJSON            = os.Getenv("LOG_JSON") != ""
	defaultLogLevel           = getEnv("LOG_LEVEL", "info")
	defaultListenAddr         = getEnv("BOOST_LISTEN_ADDR", "localhost:18550")
	defaultRelayCheck         = os.Getenv("RELAY_STARTUP_CHECK") != ""
	defaultGenesisForkVersion = getEnv("GENESIS_FORK_VERSION", "")

	// mev-boost relay request timeouts (see also https://github.com/flashbots/mev-boost/issues/287)
	defaultTimeoutMsGetHeader         = getEnvInt("RELAY_TIMEOUT_MS_GETHEADER", 950)   // timeout for getHeader requests
	defaultTimeoutMsGetPayload        = getEnvInt("RELAY_TIMEOUT_MS_GETPAYLOAD", 4000) // timeout for getPayload requests
	defaultTimeoutMsRegisterValidator = getEnvInt("RELAY_TIMEOUT_MS_REGVAL", 3000)     // timeout for registerValidator requests

	// cli flags
	printVersion = flag.Bool("version", false, "only print version")
	logJSON      = flag.Bool("json", defaultLogJSON, "log in JSON format instead of text")
	logLevel     = flag.String("loglevel", defaultLogLevel, "minimum loglevel: trace, debug, info, warn/warning, error, fatal, panic")
	logDebug     = flag.Bool("debug", false, "shorthand for '-loglevel debug'")

	listenAddr       = flag.String("addr", defaultListenAddr, "listen-address for mev-boost server")
	relayURLs        = flag.String("relays", "", "relay urls - single entry or comma-separated list (scheme://pubkey@host)")
	relayCheck       = flag.Bool("relay-check", defaultRelayCheck, "check relay status on startup and on the status API call")
	relayMonitorURLs = flag.String("relay-monitors", "", "relay monitor urls - single entry or comma-separated list (scheme://host)")

	relayTimeoutMsGetHeader  = flag.Int("request-timeout-getheader", defaultTimeoutMsGetHeader, "timeout for getHeader requests to the relay [ms]")
	relayTimeoutMsGetPayload = flag.Int("request-timeout-getpayload", defaultTimeoutMsGetPayload, "timeout for getPayload requests to the relay [ms]")
	relayTimeoutMsRegVal     = flag.Int("request-timeout-regval", defaultTimeoutMsRegisterValidator, "timeout for registerValidator requests [ms]")

	// helpers
	useGenesisForkVersionMainnet = flag.Bool("mainnet", false, "use Mainnet")
	useGenesisForkVersionKiln    = flag.Bool("kiln", false, "use Kiln")
	useGenesisForkVersionRopsten = flag.Bool("ropsten", false, "use Ropsten")
	useGenesisForkVersionSepolia = flag.Bool("sepolia", false, "use Sepolia")
	useGenesisForkVersionGoerli  = flag.Bool("goerli", false, "use Goerli")
	useCustomGenesisForkVersion  = flag.String("genesis-fork-version", defaultGenesisForkVersion, "use a custom genesis fork version")
)

var log = logrus.NewEntry(logrus.New())

// Main starts the mev-boost cli
func Main() {
	flag.Parse()
	logrus.SetOutput(os.Stdout)

	if *printVersion {
		fmt.Printf("mev-boost %s\n", config.Version)
		return
	}

	if *logJSON {
		log.Logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	if *logDebug {
		*logLevel = "debug"
	}
	if *logLevel != "" {
		lvl, err := logrus.ParseLevel(*logLevel)
		if err != nil {
			flag.Usage()
			log.Fatalf("invalid loglevel: %s", *logLevel)
		}
		logrus.SetLevel(lvl)
	}

	log.Infof("mev-boost %s", config.Version)
	log.Debug("debug logging enabled")

	genesisForkVersionHex := ""
	if *useCustomGenesisForkVersion != "" {
		genesisForkVersionHex = *useCustomGenesisForkVersion
	} else if *useGenesisForkVersionMainnet {
		genesisForkVersionHex = genesisForkVersionMainnet
	} else if *useGenesisForkVersionKiln {
		genesisForkVersionHex = genesisForkVersionKiln
	} else if *useGenesisForkVersionRopsten {
		genesisForkVersionHex = genesisForkVersionRopsten
	} else if *useGenesisForkVersionSepolia {
		genesisForkVersionHex = genesisForkVersionSepolia
	} else if *useGenesisForkVersionGoerli {
		genesisForkVersionHex = genesisForkVersionGoerli
	} else {
		flag.Usage()
		log.Fatal("please specify a genesis fork version (eg. -mainnet / -sepolia / -goerli / -genesis-fork-version flags)")
	}
	log.Infof("using genesis fork version: %s", genesisForkVersionHex)

	relays := parseRelayURLs(*relayURLs)
	if len(relays) == 0 {
		flag.Usage()
		log.Fatal("no relays specified")
	}
	log.Infof("using %d relays", len(relays))
	for index, relay := range relays {
		log.Infof("relay #%d: %s", index+1, relay.String())
	}

	relayMonitors := parseRelayMonitorURLs(*relayMonitorURLs)
	if len(relayMonitors) > 0 {
		log.Infof("using %d relay monitors", len(relayMonitors))
		for index, relayMonitor := range relayMonitors {
			log.Infof("relay-monitor #%d: %s", index+1, relayMonitor.String())
		}
	}

	opts := server.BoostServiceOpts{
		Log:                      log,
		ListenAddr:               *listenAddr,
		Relays:                   relays,
		RelayMonitors:            relayMonitors,
		GenesisForkVersionHex:    genesisForkVersionHex,
		RelayCheck:               *relayCheck,
		RequestTimeoutGetHeader:  time.Duration(*relayTimeoutMsGetHeader) * time.Millisecond,
		RequestTimeoutGetPayload: time.Duration(*relayTimeoutMsGetPayload) * time.Millisecond,
		RequestTimeoutRegVal:     time.Duration(*relayTimeoutMsRegVal) * time.Millisecond,
	}
	server, err := server.NewBoostService(opts)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	if *relayCheck && server.CheckRelays() == 0 {
		log.Error("no relay passed the health-check!")
	}

	log.Println("listening on", *listenAddr)
	log.Fatal(server.StartHTTPServer())
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		val, err := strconv.Atoi(value)
		if err == nil {
			return val
		}
	}
	return defaultValue
}

func parseRelayURLs(relayURLs string) []server.RelayEntry {
	ret := []server.RelayEntry{}
	for _, entry := range strings.Split(relayURLs, ",") {
		relay, err := server.NewRelayEntry(entry)
		if err != nil {
			log.WithError(err).WithField("relayURL", entry).Fatal("Invalid relay URL")
		}
		ret = append(ret, relay)
	}
	return ret
}

func parseRelayMonitorURLs(relayMonitorURLs string) (ret []*url.URL) {
	for _, entry := range strings.Split(relayMonitorURLs, ",") {
		if strings.TrimSpace(entry) == "" {
			continue
		}

		relayMonitor, err := url.Parse(entry)
		if err != nil {
			log.WithError(err).WithField("relayMonitorURL", entry).Fatal("Invalid relay monitor URL")
		}
		ret = append(ret, relayMonitor)
	}
	return ret
}
