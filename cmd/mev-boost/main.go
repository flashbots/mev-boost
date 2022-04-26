package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/server"
	"github.com/sirupsen/logrus"
)

var (
	version = "dev" // is set during build process

	// defaults
	defaultHost               = "localhost"
	defaultPort               = 18550
	defaultRelayURLs          = getEnv("RELAY_URLS", "http://127.0.0.1:28545")
	defaultGetHeaderTimeOutMs = 2000

	// cli flags
	host               = flag.String("host", defaultHost, "host for mev-boost to listen on")
	port               = flag.Int("port", defaultPort, "port for mev-boost to listen on")
	relayURLs          = flag.String("relayUrl", defaultRelayURLs, "relay urls - single entry or comma-separated list")
	getHeaderTimeoutMs = flag.Int("getHeaderTimeoutMs", defaultGetHeaderTimeOutMs, "max request timeout for getHeader in milliseconds (default: 2000 ms)")
)

func main() {
	rand.Seed(time.Now().UnixNano()) // warning: not a cryptographically secure seed

	flag.Parse()
	log := logrus.WithField("prefix", "cmd/mev-boost")
	log.Printf("mev-boost %s\n", version)

	_relayURLs := []string{}
	for _, entry := range strings.Split(*relayURLs, ",") {
		_relayURLs = append(_relayURLs, strings.Trim(entry, " "))
	}

	listenAddress := fmt.Sprintf("%s:%d", *host, *port)
	server, err := server.NewBoostRPCServer(server.BoostRPCServerOptions{
		ListenAddr:       listenAddress,
		RelayURLs:        _relayURLs,
		Cors:             []string{"*"},
		Log:              log,
		GetHeaderTimeout: time.Duration(*getHeaderTimeoutMs) * time.Millisecond,
	})
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Fatal("failed creating the server")
	}

	log.Println("listening on ", listenAddress)
	log.Fatal(server.ListenAndServe())
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
