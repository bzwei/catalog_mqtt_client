package towerapiworker

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakeTransport struct {
	body          []string
	status        int
	requestNumber int
	T             *testing.T
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: f.status,
		Status:     http.StatusText(f.status),
		Body:       ioutil.NopCloser(bytes.NewBufferString(f.body[f.requestNumber])),
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
	}
	f.requestNumber++
	return resp, nil
}

func fakeClient(t *testing.T, body []string, status int) *http.Client {
	return &http.Client{
		Transport: &fakeTransport{body: body, status: status, T: t},
	}
}

type testScaffold struct {
	t                    *testing.T
	receivedResponses    [][]byte
	receivedErrors       []string
	responseBody         []string
	expectedResponses    []map[string]interface{}
	expectedErrors       []string
	config               *common.CatalogConfig
	client               *http.Client
	context              context.Context
	channels             WorkChannels
	terminateMain        chan bool
	numResponses         int
	terminateResponder   chan bool
	numErrors            int
	terminateErrListener chan bool
}

func (ts *testScaffold) startResponder() {
	done := false
	for !done {
		select {
		case <-ts.terminateResponder:
			done = true
			break
		case page := <-ts.channels.ResponseChannel:
			if page.Data != nil {
				ts.receivedResponses = append(ts.receivedResponses, page.Data)
				ts.numResponses++
				if ts.numResponses == len(ts.expectedResponses) {
					ts.terminateMain <- true
				}
			}
		}
	}
}

func (ts *testScaffold) startErrorListener() {
	done := false
	for !done {
		select {
		case <-ts.terminateErrListener:
			done = true
			break
		case errMsg := <-ts.channels.ErrorChannel:
			ts.receivedErrors = append(ts.receivedErrors, errMsg)
			ts.numErrors++
			if ts.numErrors == len(ts.expectedErrors) {
				ts.terminateMain <- true
			}
		}
	}
}
func (ts *testScaffold) base(t *testing.T, jp common.JobParam, responseCode int, responseBody []string) {
	log.SetOutput(os.Stdout)
	ts.t = t
	ts.channels = WorkChannels{}
	ts.responseBody = responseBody
	ts.terminateMain = make(chan bool)
	ts.terminateResponder = make(chan bool)
	ts.terminateErrListener = make(chan bool)

	ts.config = &common.CatalogConfig{Level: "error", URL: "https://192.1.1.1", Token: "123", SkipVerifyCertificate: true}
	ts.client = fakeClient(t, responseBody, responseCode)
	ts.context = logger.CtxWithLoggerID(context.Background(), "123")
}

func (ts *testScaffold) runSuccess(t *testing.T, jp common.JobParam, responseCode int, responseBody []string, responses []map[string]interface{}) {
	ts.base(t, jp, responseCode, responseBody)
	ts.channels.ResponseChannel = make(chan common.Page)
	defer close(ts.channels.ResponseChannel)
	ts.expectedResponses = responses

	go ts.startResponder()

	apiw := &DefaultAPIWorker{}
	err := apiw.StartWork(ts.context, ts.config, jp, ts.client, ts.channels)
	if err != nil {
		t.Fatalf("StartWork failed %v", err)
	}

	select {
	case <-ts.terminateMain:
		ts.checkWorkResponse()
	case <-time.After(2 * time.Second):
		t.Error("Did not receive all responses within acceptable time period")
	}
	ts.terminateResponder <- true
}

func (ts *testScaffold) runFail(t *testing.T, jp common.JobParam, responseCode int, responseBody []string, errorMessages []string) {
	ts.base(t, jp, responseCode, responseBody)
	ts.channels.ErrorChannel = make(chan string)
	defer close(ts.channels.ErrorChannel)
	ts.expectedErrors = errorMessages

	go ts.startErrorListener()

	apiw := &DefaultAPIWorker{}
	err := apiw.StartWork(ts.context, ts.config, jp, ts.client, ts.channels)
	if err == nil {
		t.Fatalf("Test should have failed but it succeeded")
	}
	select {
	case <-ts.terminateMain:
		assert.Equal(t, ts.expectedErrors, ts.receivedErrors)
	case <-time.After(2 * time.Second):
		t.Error("Did not receive all errors within acceptable time period")
	}
	ts.terminateErrListener <- true
}

func (ts *testScaffold) checkWorkResponse() {
	var resp map[string]interface{}

	for i, expectedResponse := range ts.expectedResponses {
		err := json.Unmarshal(ts.receivedResponses[i], &resp)
		if err != nil {
			ts.t.Fatalf("Error in json unmarshal : %v", err)
		}

		assert.Equal(ts.t, expectedResponse, resp)
	}
}
