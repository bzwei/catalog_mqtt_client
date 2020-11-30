package main

import (
	"bytes"
	"net/http"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestGetRequest(t *testing.T) {
	b := []byte(`{"account":"12345","sender":"buzz", "message_id":"4567","payload":{"jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008","accept_encoding":"gzip"}]}}`)
	log.SetOutput(os.Stdout)
	drh := &DefaultRequestHandler{}
	bo := bytes.NewBuffer(b)
	b, err := drh.getRequest(bo)
	if err != nil {
		t.Fatalf("Error getting request data %v", err)
	}
}
func TestGetRequestNewLine(t *testing.T) {
	b := []byte(`{"account":"12345","sender":"buzz", "message_id":"4567","payload":{"jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008","accept_encoding":"gzip"}]}}\n abab`)
	log.SetOutput(os.Stdout)
	bo := bytes.NewBuffer(b)
	drh := &DefaultRequestHandler{}
	b, err := drh.getRequest(bo)
	if err != nil {
		t.Fatalf("Error getting request data %v", err)
	}
}

func TestParseRequest(t *testing.T) {
	b := []byte(`{"account":"12345","sender":"buzz", "message_id":"4567","payload":{"jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008","accept_encoding":"gzip"}]}}`)
	log.SetOutput(os.Stdout)
	drh := &DefaultRequestHandler{}
	_, err := drh.parseRequest(b)
	if err != nil {
		t.Fatalf("Error parsing request data %v", err)
	}
}

type FakeHandler struct {
	timesCalled int
}

func (fh *FakeHandler) StartWork(config *CatalogConfig, params JobParam, client *http.Client, channel chan ResponsePayload) error {
	fh.timesCalled++
	return nil
}

func TestProcessRequest(t *testing.T) {
	b := []byte(`{"account":"12345","sender":"buzz", "message_id":"4567","payload":{"jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008","accept_encoding":"gzip"},{"method":"get","href_slug":"/api/v2/inventories/899"}]}}`)
	log.SetOutput(os.Stdout)
	drh := &DefaultRequestHandler{}
	req, err := drh.parseRequest(b)
	if err != nil {
		t.Fatalf("Error parsing request data %v", err)
	}
	fh := FakeHandler{}
	drh.processRequest(req, CatalogConfig{}, &fh)
	if fh.timesCalled != 2 {
		t.Fatalf("2 workers should have been started only %d were started", fh.timesCalled)
	}
}
