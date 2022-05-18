package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/internal/types"
	"github.com/flashbots/mev-boost/server"
	"github.com/sirupsen/logrus"
)

var (
	version = "dev" // is set during build process

	// defaults
	defaultHost           = getEnv("BOOST_HOST", "localhost")
	defaultPort           = getEnvInt("BOOST_PORT", 18550)
	defaultRelayURLs      = getEnv("RELAY_URLS", "127.0.0.1:28545") // can be IP@PORT or PUBKEY@IP:PORT or https://IP
	defaultRelayTimeoutMs = getEnvInt("RELAY_TIMEOUT_MS", 2000)     // timeout

	// cli flags
	host           = flag.String("host", defaultHost, fmt.Sprintf("host for mev-boost to listen on. default: %s", defaultHost))
	port           = flag.Int("port", defaultPort, fmt.Sprintf("port for mev-boost to listen on. default: %d", defaultPort))
	relayURLs      = flag.String("relays", defaultRelayURLs, fmt.Sprintf("relay urls - single entry or comma-separated list (pubkey@ip:port). default: %s", defaultRelayURLs))
	relayTimeoutMs = flag.Int("request-timeout", defaultRelayTimeoutMs, fmt.Sprintf("timeout for requests to a relay [ms]. default: %d", defaultRelayTimeoutMs))
)

var log = logrus.WithField("module", "cmd/mev-boost")

func main() {
	flag.Parse()
	log.Printf("mev-boost %s\n", version)

	relays := parseRelayURLs(*relayURLs)
	log.WithField("relays", relays).Infof("using %d relays", len(relays))
	// TODO: relay connection checks?

	listenAddress := fmt.Sprintf("%s:%d", *host, *port)
	relayTimeout := time.Duration(*relayTimeoutMs) * time.Millisecond
	server, err := server.NewBoostService(listenAddress, relays, log, relayTimeout)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	log.Println("listening on", listenAddress)
	log.Fatal(server.StartHTTPServer())
}

func getEnv(key string, defaultValue string) string {
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

func parseRelayURLs(relayURLs string) []types.RelayEntry {
	ret := []types.RelayEntry{}
	for _, entry := range strings.Split(relayURLs, ",") {
		relay, err := parseRelayURL(entry)
		if err != nil {
			log.WithError(err).WithField("relayURL", entry).Fatal("Invalid relay URL")
		}
		ret = append(ret, relay)
	}
	return ret
}

func parseRelayURL(relayURL string) (entry types.RelayEntry, err error) {
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	u, err := url.Parse(relayURL)
	if err != nil {
		return entry, err
	}

	entry = types.RelayEntry{Address: u.Scheme + "://" + u.Host}
	err = entry.Pubkey.UnmarshalText([]byte(u.User.Username()))
	if err != nil {
		return entry, err
	}
	return entry, err
}
