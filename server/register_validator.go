package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

func (m *BoostService) registerValidator(log *logrus.Entry, regBytes []byte, header http.Header) error {
	m.relayConfigsLock.RLock()
	allConfigs := m.AllRelayConfigs()
	m.relayConfigsLock.RUnlock()

	relays := relayEntries(allConfigs)
	respErrCh := make(chan error, len(relays))

	log.WithFields(logrus.Fields{
		"timeout":   m.httpClientRegVal.Timeout,
		"numRelays": len(relays),
		"regBytes":  len(regBytes),
	}).Info("calling registerValidator on relays")

	// Forward request to each relay
	for _, relay := range relays {
		go func(relay types.RelayEntry) {
			// Get the URL for this relay
			requestURL := relay.GetURI(params.PathRegisterValidator)
			log := log.WithField("url", requestURL)

			// Build the new request
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, bytes.NewReader(regBytes))
			if err != nil {
				log.WithError(err).Warn("error creating new request")
				respErrCh <- err
				return
			}

			// Extend the request header with our values
			for key, values := range header {
				req.Header[key] = values
			}

			log.WithFields(logrus.Fields{
				"httpMethod":    req.Method,
				"contentLength": req.ContentLength,
			}).Debug("sending the registerValidator request")

			// Send the request
			start := time.Now()
			resp, err := m.httpClientRegVal.Do(req)
			RecordRelayLatency(params.PathRegisterValidator, relay.URL.Hostname(), float64(time.Since(start).Milliseconds()))
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay")
				respErrCh <- err
				return
			}
			resp.Body.Close()

			RecordRelayStatusCode(strconv.Itoa(resp.StatusCode), params.PathRegisterValidator, relay.URL.Hostname())
			// Check if response is successful
			if resp.StatusCode == http.StatusOK {
				log.Debug("relay accepted registrations")
				respErrCh <- nil
			} else {
				log.WithFields(logrus.Fields{
					"statusCode": resp.StatusCode,
				}).Debug("received an error response from relay")
				respErrCh <- fmt.Errorf("%w: %d", errHTTPErrorResponse, resp.StatusCode)
			}
		}(relay)
	}

	// Return OK if any relay responds OK
	for range relays {
		respErr := <-respErrCh
		if respErr == nil {
			// Goroutines are independent, so if there are a lot of configured
			// relays and the first one responds OK, this will continue to send
			// validator registrations to the other relays.
			log.Debug("one or more relays accepted the registrations")
			return nil
		}
	}

	// None of the relays responded OK
	log.Debug("no relays accepted the registrations")
	return errNoSuccessfulRelayResponse
}
