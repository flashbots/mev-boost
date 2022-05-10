package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func sendRESTRequest(url string, method string, payload any, dst any) error {
	fetchLog := log.WithField("url", url)
	var req *http.Request
	if payload == nil {
		var err error
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			fetchLog.WithField("err", err).Error("invalid request")
			return err
		}
	} else {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err = http.NewRequest(method, url, bytes.NewReader(payloadBytes))
		if err != nil {
			fetchLog.WithField("err", err).Error("invalid request")
			return err
		}
	}

	req.Header.Set("accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fetchLog.WithField("err", err).Error("client refused")
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fetchLog.WithField("err", err).Error("could not read response body")
		return err
	}

	fetchLog = fetchLog.WithField("body", string(bodyBytes))

	if resp.StatusCode >= 300 {
		ec := &struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal(bodyBytes, ec); err != nil {
			fetchLog.WithField("err", err).Error("Couldn't unmarshal error from beacon node")
			return errors.New("could not unmarshal error response from beacon node")
		}
		return errors.New(ec.Message)
	}

	if dst != nil {
		err = json.Unmarshal(bodyBytes, dst)
		if err != nil {
			fetchLog.WithField("err", err).Error("could not unmarshal response")
			return err
		}

		fetchLog.WithField("res", dst).Info("fetched")
	} else {
		fetchLog.Info("fetched")
	}
	return nil
}

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

func sendJSONRequest(endpoint string, payload any, dst any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body := bytes.NewReader(payloadBytes)
	fetchLog := log.WithField("endpoint", endpoint).WithField("method", "POST").WithField("payload", string(payloadBytes))
	fetchLog.Info("sending request")
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		fetchLog.WithField("err", err).Error("could not prepare request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fetchLog.WithField("err", err).Error("could not send request")
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	fetchLog.WithField("bodyBytes", string(bodyBytes)).Info("got response")
	if err != nil {
		return err
	}

	var jsonResp jsonrpcMessage
	if err = json.Unmarshal(bodyBytes, &jsonResp); err != nil {
		fetchLog.WithField("response", string(bodyBytes)).WithField("err", err).Error("could not unmarshal response")
		return err
	}

	if jsonResp.Error != nil {
		fetchLog.WithField("code", jsonResp.Error.Code).WithField("err", jsonResp.Error.Message).Error("error response")
		return errors.New(jsonResp.Error.Message)
	}

	if dst != nil {
		if err = json.Unmarshal(jsonResp.Result, dst); err != nil {
			fetchLog.WithField("result", string(jsonResp.Result)).WithField("err", err).Error("could not unmarshal result")
			return err
		}
	}
	return nil
}
