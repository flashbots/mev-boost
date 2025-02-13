package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

func (m *BoostService) registerValidator(log *logrus.Entry, regBytes []byte, header http.Header) error {
	respErrCh := make(chan error, len(m.relays))

	// Forward request to each relay
	for _, relay := range m.relays {
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

			// Send the request
			resp, err := m.httpClientRegVal.Do(req)
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay")
				respErrCh <- err
				return
			}
			resp.Body.Close()

			// Check if response is successful
			if resp.StatusCode == http.StatusOK {
				respErrCh <- nil
			} else {
				respErrCh <- fmt.Errorf("%w: %d", errHTTPErrorResponse, resp.StatusCode)
			}
		}(relay)
	}

	// Return OK if any relay responds OK
	for range m.relays {
		respErr := <-respErrCh
		if respErr == nil {
			// Goroutines are independent, so if there are a lot of configured
			// relays and the first one responds OK, this will continue to send
			// validator registrations to the other relays.
			return nil
		}
	}

	// None of the relays responded OK
	return errNoSuccessfulRelayResponse
}

func (m *BoostService) sendValidatorRegistrationsToRelayMonitors(log *logrus.Entry, regBytes []byte, header http.Header) {
	// Forward request to each relay monitor
	for _, relayMonitor := range m.relayMonitors {
		go func(relayMonitor *url.URL) {
			// Get the URL for this relay monitor
			requestURL := types.GetURI(relayMonitor, params.PathRegisterValidator)
			log := log.WithField("url", requestURL)

			// Build the new request
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, bytes.NewReader(regBytes))
			if err != nil {
				log.WithError(err).Warn("error creating new request")
				return
			}

			// Extend the request header with our values
			for key, values := range header {
				req.Header[key] = values
			}

			// Send the request
			resp, err := m.httpClientRegVal.Do(req)
			if err != nil {
				log.WithError(err).Warn("error calling registerValidator on relay monitor")
				return
			}
			resp.Body.Close()
		}(relayMonitor)
	}
}
