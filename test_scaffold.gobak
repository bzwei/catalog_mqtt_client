package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
)

type fakeTransport struct {
	body          []string
	status        int
	requestNumber int
	T             *testing.T
}

type testScaffold struct {
	t             *testing.T
	output        bytes.Buffer
	req           *ResponseHeader
	outputChannel chan ResponsePayload
	responseBody  []string
	responses     []map[string]interface{}
	errorMessage  string
	work          WorkUnit
	config        *CatalogConfig
	client        *http.Client
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

func channelSetup(f io.Writer, rh *ResponseHeader) chan ResponsePayload {
	var responderGroup sync.WaitGroup

	outputChannel := make(chan ResponsePayload)
	rs := &Responder{
		Output: f,
		header: ResponseHeader{
			Account:      rh.Account,
			Sender:       rh.Sender,
			InResponseTo: rh.InResponseTo,
		},
	}
	responderGroup.Add(1)
	go startResponder(&responderGroup, rs, outputChannel)
	return outputChannel
}

func (ts *testScaffold) base(t *testing.T, jp JobParam, responseCode int, responseBody []string) {
	log.SetOutput(os.Stdout)
	ts.t = t
	ts.req = &ResponseHeader{Account: "Buzz", Sender: "Star Command", InResponseTo: "345"}
	ts.outputChannel = channelSetup(&ts.output, ts.req)
	ts.responseBody = responseBody

	ts.work = WorkUnit{outputChannel: ts.outputChannel}
	ts.config = &CatalogConfig{Debug: false, URL: "https://192.1.1.1", Token: "123", SkipVerifyCertificate: true}
	ts.client = fakeClient(t, responseBody, responseCode)
}

func (ts *testScaffold) runSuccess(t *testing.T, jp JobParam, responseCode int, responseBody []string, responses []map[string]interface{}) {
	ts.base(t, jp, responseCode, responseBody)
	ts.responses = responses
	apiw := &DefaultAPIWorker{}
	err := apiw.StartWork(ts.config, jp, ts.client, ts.outputChannel)
	if err != nil {
		t.Fatalf("StartWork failed %v", err)
	}
	ts.checkWorkResponse()
}

func (ts *testScaffold) runFail(t *testing.T, jp JobParam, responseCode int, responseBody []string, errorMessage string) {
	ts.base(t, jp, responseCode, responseBody)
	ts.errorMessage = errorMessage

	apiw := &DefaultAPIWorker{}
	err := apiw.StartWork(ts.config, jp, ts.client, ts.outputChannel)
	if err == nil {
		t.Fatalf("Test should have failed but it succedded")
	}
	ts.checkWorkFailure()
}

func (ts *testScaffold) validateResponse(m *ResponseMessage) {
	if ts.req.InResponseTo != m.InResponseTo || ts.req.Account != m.Account || ts.req.Sender != m.Sender {
		ts.t.Fatalf("request values dont match respones")
	}
}

func (ts *testScaffold) checkWorkFailure() {
	scanner := bufio.NewScanner(bufio.NewReader(&ts.output))
	var resp ResponseMessage

	for scanner.Scan() {
		err := json.Unmarshal([]byte(scanner.Text()), &resp)
		if err != nil {
			ts.t.Fatalf("Error in json unmarshal : %v", err)
		}
		ts.validateResponse(&resp)
		if resp.MessageType == "data" && resp.Code == 1 {
			if !strings.Contains(resp.Payload.Body, ts.errorMessage) {
				ts.t.Fatalf("Could not find error string %s", ts.errorMessage)
			}
		} else {
			ts.t.Fatalf("Did not receive error payload")
		}
	}

	if err := scanner.Err(); err != nil {
		ts.t.Fatalf("Error in scanner: %v", err)
	}
}

func (ts *testScaffold) checkWorkResponse() {
	scanner := bufio.NewScanner(bufio.NewReader(&ts.output))
	var resp ResponseMessage

	count := 0
	for scanner.Scan() {
		err := json.Unmarshal([]byte(scanner.Text()), &resp)
		if err != nil {
			ts.t.Fatalf("Error in json unmarshal : %v", err)
		}
		ts.validateResponse(&resp)
		if resp.MessageType == "data" {
			result := ts.parsePayload(&resp)
			ts.checkBody(result, ts.responses[count])
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		ts.t.Fatalf("Error in scanner: %v", err)
	}
}

func (ts *testScaffold) checkBody(actual map[string]interface{}, required map[string]interface{}) {
	for k, v := range required {
		value, ok := actual[k]
		if !ok {
			ts.t.Fatalf("Key Missing %s from Actual data", k)
			continue
		}
		if v == nil || value == nil {
			continue
		}
		if reflect.TypeOf(v).String() == "int" && reflect.TypeOf(value).String() == "json.Number" {
			continue
		} else if reflect.TypeOf(v) != reflect.TypeOf(value) {
			ts.t.Fatalf("Type Mismatch required %v actual %v", reflect.TypeOf(v), reflect.TypeOf(value))
		}
	}
}

func (ts *testScaffold) parsePayload(r *ResponseMessage) map[string]interface{} {
	var data []byte
	var result map[string]interface{}
	if r.Payload.Encoding == "gzip" {
		data = ts.decompress(r.Payload.Body)
	} else {
		data = []byte(r.Payload.Body)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&result)
	if err != nil {
		ts.t.Fatalf("Error parsing payload %v", err)
	}
	return result
}

func (ts *testScaffold) decompress(s string) []byte {
	var b bytes.Buffer
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		ts.t.Fatalf("Error decoding json string %v", err)
	}
	zr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		ts.t.Fatalf("Error in gzip.NewReader %v", err)
	}

	if _, err := io.Copy(bufio.NewWriter(&b), zr); err != nil {
		ts.t.Fatalf("Error in copy %v", err)
	}

	if err := zr.Close(); err != nil {
		ts.t.Fatalf("Error in Close %v", err)
	}
	return b.Bytes()
}
