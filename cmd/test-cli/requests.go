package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}

func sendJSONRequest(endpoint string, payload, dst any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body := bytes.NewReader(payloadBytes)
	fetchLog := log.WithField("endpoint", endpoint).WithField("method", "POST").WithField("payload", string(payloadBytes))
	fetchLog.Info("sending request")
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		fetchLog.WithError(err).Error("could not prepare request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fetchLog.WithError(err).Error("could not send request")
		return err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			fetchLog.WithError(err).Error("could not close body")
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	fetchLog.WithField("bodyBytes", string(bodyBytes)).Info("got response")
	if err != nil {
		return err
	}

	var jsonResp jsonrpcMessage
	if err = json.Unmarshal(bodyBytes, &jsonResp); err != nil {
		fetchLog.WithField("response", string(bodyBytes)).WithError(err).Error("could not unmarshal response")
		return err
	}

	if jsonResp.Error != nil {
		fetchLog.WithField("code", jsonResp.Error.Code).WithField("err", jsonResp.Error.Message).Error("error response")
		return errors.New(jsonResp.Error.Message)
	}

	if dst != nil {
		if err = json.Unmarshal(jsonResp.Result, dst); err != nil {
			fetchLog.WithField("result", string(jsonResp.Result)).WithError(err).Error("could not unmarshal result")
			return err
		}
	}
	return nil
}
