package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

type FakeRequestHandler struct {
	timesCalled   int
	catalogConfig CatalogConfig
}

func (frh *FakeRequestHandler) getRequest(r io.Reader) ([]byte, error) {
	var s []byte
	frh.timesCalled++
	return s, nil
}

func (frh *FakeRequestHandler) parseRequest(b []byte) (*RequestMessage, error) {
	frh.timesCalled++
	return &RequestMessage{}, nil
}

func (frh *FakeRequestHandler) processRequest(req *RequestMessage, config CatalogConfig, wh WorkHandler) {
	frh.timesCalled++
	frh.catalogConfig = config
}

func TestMain(t *testing.T) {
	os.Args = []string{"catalog_worker",
		"--debug",
		"--token", "gobbledygook",
		"--url", "https://www.example.com"}
	b := []byte(`{"account":"12345","sender":"buzz", "message_id":"4567","payload":{"jobs": [{"method":"monitor","href_slug":"/api/v2/jobs/7008","accept_encoding":"gzip"}]}}`)
	bo := bytes.NewBuffer(b)
	frh := &FakeRequestHandler{}
	startRun(bo, frh)
	if frh.timesCalled != 3 {
		t.Errorf("Request handler not called 3 times")
	}
	if !frh.catalogConfig.Debug {
		t.Errorf("Debug is not being enabled")
	}
	if frh.catalogConfig.URL != "https://www.example.com" {
		t.Errorf("Debug is not being enabled")
	}
	if frh.catalogConfig.Token != "gobbledygook" {
		t.Errorf("Token has not been set")
	}
}
