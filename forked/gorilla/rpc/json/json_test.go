// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/rpc"
)

var ErrResponseError = errors.New("response error")

type Service1Request struct {
	A int
	B int
}

type Service1BadRequest struct {
	M string `json:"method"`
}

type Service1Response struct {
	Result int
}

type Service1 struct {
	beforeAfterContext map[string]string
}

func (t *Service1) Multiply(r *http.Request, req *Service1Request, res *Service1Response) error {
	res.Result = req.A * req.B
	return nil
}

func (t *Service1) ResponseError(r *http.Request, req *Service1Request, res *Service1Response) error {
	return ErrResponseError
}

func (t *Service1) BeforeAfter(r *http.Request, req *Service1Request, res *Service1Response) error {
	if _, ok := t.beforeAfterContext["before"]; !ok {
		return fmt.Errorf("before value not found in context")
	}
	res.Result = 1
	return nil
}

func execute(t *testing.T, s *rpc.Server, method string, req, res interface{}) error {
	if !s.HasMethod(method) {
		t.Fatal("Expected to be registered:", method)
	}

	buf, _ := EncodeClientRequest(method, req)
	body := bytes.NewBuffer(buf)
	r, _ := http.NewRequest("POST", "http://localhost:8080/", body)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	return DecodeClientResponse(w.Body, res)
}

func executeRaw(t *testing.T, s *rpc.Server, req interface{}, res interface{}) int {
	j, _ := json.Marshal(req)
	r, _ := http.NewRequest("POST", "http://localhost:8080/", bytes.NewBuffer(j))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	return w.Code
}

func TestService(t *testing.T) {
	s := rpc.NewServer()
	s.RegisterCodec(NewCodec(), "application/json")
	s.RegisterService(new(Service1), "")

	var res Service1Response
	if err := execute(t, s, "Service1.Multiply", &Service1Request{4, 2}, &res); err != nil {
		t.Error("Expected err to be nil, but got:", err)
	}
	if res.Result != 8 {
		t.Errorf("Wrong response: %v.", res.Result)
	}

	if err := execute(t, s, "Service1.ResponseError", &Service1Request{4, 2}, &res); err == nil {
		t.Errorf("Expected to get %q, but got nil", ErrResponseError)
	} else if err.Error() != ErrResponseError.Error() {
		t.Errorf("Expected to get %q, but got %q", ErrResponseError, err)
	}

	if code := executeRaw(t, s, &Service1BadRequest{"Service1.Multiply"}, &res); code != 400 {
		t.Errorf("Expected http response code 400, but got %v", code)
	}
}

func TestServiceBeforeAfter(t *testing.T) {
	s := rpc.NewServer()
	s.RegisterCodec(NewCodec(), "application/json")
	service := &Service1{}
	service.beforeAfterContext = map[string]string{}
	s.RegisterService(service, "")

	s.RegisterBeforeFunc(func(i *rpc.RequestInfo) {
		service.beforeAfterContext["before"] = "Before is true"
	})
	s.RegisterAfterFunc(func(i *rpc.RequestInfo) {
		service.beforeAfterContext["after"] = "After is true"
	})

	var res Service1Response
	if err := execute(t, s, "Service1.BeforeAfter", &Service1Request{}, &res); err != nil {
		t.Error("Expected err to be nil, but got:", err)
	}

	if res.Result != 1 {
		t.Errorf("Expected Result = 1, got %d", res.Result)
	}

	if afterValue, ok := service.beforeAfterContext["after"]; !ok || afterValue != "After is true" {
		t.Errorf("Expected after in context to be 'After is true', got %s", afterValue)
	}
}
