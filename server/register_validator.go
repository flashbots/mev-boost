package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"time"

	builderApiV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/flashbots/mev-boost/server/params"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
)

var ErrUnsupportedContentType = errors.New("unsupported content type")

func (m *BoostService) registerValidator(log *logrus.Entry, regBytes []byte, header http.Header) error {
	m.relayConfigsLock.RLock()
	relayConfigs := m.relayConfigs
	m.relayConfigsLock.RUnlock()

	respErrCh := make(chan error, len(relayConfigs))

	log.WithFields(logrus.Fields{
		"timeout":   m.httpClientRegVal.Timeout,
		"numRelays": len(relayConfigs),
	if m.muxConfig == nil {
		return m.sendRegistrationsToRelays(log, regBytes, header, m.relays)
	}
	pubkeys, err := m.extractValidatorPubkeys(regBytes, header)
	if err != nil {
		return m.sendRegistrationsToRelays(log, regBytes, header, m.relays)
	}
	relaysToUse := m.getRelaysForValidators(pubkeys)

	log.WithFields(logrus.Fields{
		"numValidators": len(pubkeys),
		"numRelays":     len(relaysToUse),
	}).Debug("sending validator registrations to relevant relays")

	return m.sendRegistrationsToRelays(log, regBytes, header, relaysToUse)
}

func (m *BoostService) sendRegistrationsToRelays(log *logrus.Entry, regBytes []byte, header http.Header, relays []types.RelayEntry) error {
	respErrCh := make(chan error, len(relays))

	log.WithFields(logrus.Fields{
		"timeout":   m.httpClientRegVal.Timeout,
		"numRelays": len(relays),
		"regBytes":  len(regBytes),
	}).Info("calling registerValidator on relays")

	// Forward request to each relay
	for _, relayConfig := range relayConfigs {
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
				"request": req,
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
		}(relayConfig.RelayEntry)
	}

	// Return OK if any relay responds OK
	for range relayConfigs {
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

func (m *BoostService) extractValidatorPubkeys(regBytes []byte, header http.Header) ([]string, error) {
	contentType := header.Get("Content-Type")
	parsedContentType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		parsedContentType = "application/json"
	}

	var registrations []*builderApiV1.SignedValidatorRegistration

	switch parsedContentType {
	case "application/json":
		if err := json.Unmarshal(regBytes, &registrations); err != nil {
			return nil, err
		}
	case "application/octet-stream":
		var sszRegistrations builderApiV1.SignedValidatorRegistrations
		if err := sszRegistrations.UnmarshalSSZ(regBytes); err != nil {
			return nil, err
		}
		registrations = sszRegistrations.Registrations
	default:
		return nil, ErrUnsupportedContentType
	}
	pubkeys := make([]string, len(registrations))
	for i, reg := range registrations {
		pubkeys[i] = reg.Message.Pubkey.String()
	}

	return pubkeys, nil
}

func (m *BoostService) getRelaysForValidators(pubkeys []string) []types.RelayEntry {
	relayMap := make(map[string]types.RelayEntry)

	for _, pubkey := range pubkeys {
		validatorRelays := m.getRelaysForValidator(pubkey)
		for _, relay := range validatorRelays {
			relayMap[relay.URL.String()] = relay
		}
	}
	relays := make([]types.RelayEntry, 0, len(relayMap))
	for _, relay := range relayMap {
		relays = append(relays, relay)
	}

	return relays
}
