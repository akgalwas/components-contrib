// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dapr/components-contrib/bindings"
	"github.com/dapr/dapr/pkg/logger"
)

// HTTPSource is a binding for an http url endpoint invocation
// nolint:golint
type HTTPSource struct {
	metadata httpMetadata

	logger logger.Logger
}

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type httpMetadata struct {
	URL         string       `json:"url"`
	Method      string       `json:"method"`
	Credentials *credentials `json:"credentials"`
}

// NewHTTP returns a new HTTPSource
func NewHTTP(logger logger.Logger) *HTTPSource {
	return &HTTPSource{logger: logger}
}

// Init performs metadata parsing
func (h *HTTPSource) Init(metadata bindings.Metadata) error {
	b, err := json.Marshal(metadata.Properties)
	if err != nil {
		return err
	}

	var m httpMetadata
	err = json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	h.metadata = m
	return nil
}

func (h *HTTPSource) get(url string) ([]byte, error) {
	client := http.Client{Timeout: time.Second * 60}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	addCredentials(req, h.metadata.Credentials)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return b, nil
}

func (h *HTTPSource) Read(handler func(*bindings.ReadResponse) error) error {
	b, err := h.get(h.metadata.URL)
	if err != nil {
		return err
	}

	handler(&bindings.ReadResponse{
		Data: b,
	})
	return nil
}

func (h *HTTPSource) Operations() []bindings.OperationKind {
	return []bindings.OperationKind{bindings.CreateOperation}
}

func (h *HTTPSource) Invoke(req *bindings.InvokeRequest) (*bindings.InvokeResponse, error) {

	client := http.Client{Timeout: time.Second * 5}

	r, err := http.NewRequest("POST", h.metadata.URL, bytes.NewBuffer(req.Data))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	addCredentials(r, h.metadata.Credentials)

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return nil, nil
}

func addCredentials(req *http.Request, credentials *credentials) {
	if credentials != nil && credentials.User != "" && credentials.Password != "" {
		addBasicAuthHeader(req, credentials.User, credentials.Password)
	}
}

func addBasicAuthHeader(request *http.Request, user, password string) {
	auth := user + ":" + password
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
}
