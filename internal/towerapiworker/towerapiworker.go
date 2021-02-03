package towerapiworker

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/rhc_catalog_worker/internal/artifacts"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/common"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/filters"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/logger"
)

// WorkChannels collects all channels for communication between the api worker and client request goroutines
type WorkChannels struct {
	Shutdown        chan struct{}
	ErrorChannel    chan string
	DispatchChannel chan common.JobParam
	FinishedChannel chan bool
	WaitChannel     chan bool
	ResponseChannel chan common.Page
}

type relatedObject struct {
	predicate    string
	relAttribute string
	jobExtra     common.JobParam
}

// WorkHandler is an interface to start a worker
type WorkHandler interface {
	StartWork(ctx context.Context, config *common.CatalogConfig, params common.JobParam, client *http.Client, wc WorkChannels) error
}

// DefaultAPIWorker is struct to start a worker
type DefaultAPIWorker struct {
}

// StartWork can be started as a go routine to start a unit of work based on a given JobParam
// The responses are sent to the Responder's channel so that it can relay it to the Receptor
func (aw *DefaultAPIWorker) StartWork(ctx context.Context, config *common.CatalogConfig, params common.JobParam, client *http.Client, wc WorkChannels) error {
	glog := logger.GetLogger(ctx)
	glog.Info("Worker starting")
	w := &workUnit{}
	w.glog = glog
	w.setConfig(config)
	w.setJobParameters(params)
	w.errorChannel = wc.ErrorChannel
	w.shutdown = wc.Shutdown
	w.dispatchChannel = wc.DispatchChannel
	w.responseChannel = wc.ResponseChannel
	err := w.setURL()
	if err != nil {
		glog.Errorf("Error %v", err)
		return err
	}
	w.setClient(client)
	w.glog.Info("Dispatch started")
	return w.dispatch()
}

// workUnit is a data struct to store a single unit of work
type workUnit struct {
	glog            logger.Logger
	config          *common.CatalogConfig
	hostURL         *url.URL
	client          *http.Client
	input           *common.JobParam
	filterValue     *filters.Value
	parsedURL       *url.URL
	parsedValues    url.Values
	errorChannel    chan string
	dispatchChannel chan common.JobParam
	responseChannel chan common.Page
	shutdown        chan struct{}
	relatedObjects  []relatedObject
}

func (w *workUnit) setConfig(p *common.CatalogConfig) {
	w.config = p
	w.parseHost(p.URL)
}

func (w *workUnit) setJobParameters(data common.JobParam) {
	if data.ApplyFilter != nil {
		fltr := filters.Value{}
		fltr.Parse(data.ApplyFilter)
		w.filterValue = &fltr
	}
	if data.Params == nil {
		data.Params = make(map[string]interface{})
	}
	if data.PagePrefix == "" {
		data.PagePrefix = "page"
	}

	w.setRelatedObjects(data)
	w.input = &data
}

func (w *workUnit) setRelatedObjects(data common.JobParam) {
	if data.FetchRelated != nil {
		for _, o := range data.FetchRelated {
			obj := o.(map[string]interface{})
			w.setRelated(obj)
		}
	}
}

func (w *workUnit) setClient(c *http.Client) error {
	w.glog.Infof("Setting client %v", c)
	if c == nil {
		var tr *http.Transport
		if w.config.SkipVerifyCertificate {
			config := &tls.Config{InsecureSkipVerify: true}
			tr = &http.Transport{TLSClientConfig: config}
		}
		w.client = &http.Client{Transport: tr}
	} else {
		w.client = c
	}
	return nil
}

func (w *workUnit) dispatch() error {
	var err error
	switch strings.ToLower(w.input.Method) {
	case "get":
		err = w.get()
	case "post", "launch":
		err = w.post()
	case "monitor":
		err = w.monitor()
	default:
		err = errors.New("Invalid method received " + w.input.Method)
		w.sendError(err.Error(), 0)
	}
	return err
}

func (w *workUnit) setURL() error {
	w.glog.Info("Setting URL")
	var err error
	w.parsedURL, err = url.Parse(w.input.HrefSlug)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	w.parsedValues, err = url.ParseQuery(w.parsedURL.RawQuery)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	w.parsedURL.Scheme = w.hostURL.Scheme
	w.parsedURL.Host = w.hostURL.Host
	return nil
}

func (w *workUnit) setRelated(data map[string]interface{}) error {
	r := relatedObject{}
	for key, element := range data {
		switch v := element.(type) {
		case string:
			if key == "href_slug" {
				r.relAttribute = v
			} else if key == "predicate" {
				r.predicate = v
			} else if key == "apply_filter" {
				r.jobExtra.ApplyFilter = v
			}
		}
	}
	// If there is no href_slug ignore this relation
	if r.relAttribute != "" {
		w.relatedObjects = append(w.relatedObjects, r)
	}
	return nil
}

func (w *workUnit) overrideQueryParams(override map[string]interface{}) error {
	for key, element := range override {
		switch v := element.(type) {
		case int64:
			w.parsedValues.Set(key, strconv.FormatInt(element.(int64), 10))
		case string:
			w.parsedValues.Set(key, element.(string))
		case float64:
			w.parsedValues.Set(key, strconv.FormatFloat(element.(float64), 'E', -1, 64))
		case bool:
			w.parsedValues.Set(key, strconv.FormatBool(element.(bool)))
		case json.Number:
			w.parsedValues.Set(key, element.(json.Number).String())
		default:
			w.glog.Infof("I don't know about type %T!\n", v)
		}
	}
	for key, element := range w.parsedValues {
		w.glog.Infof("Key:%s => Element:%s", key, element[0])
	}
	w.parsedURL.RawQuery = w.parsedValues.Encode()
	return nil
}

func (w *workUnit) parseHost(host string) error {
	u, err := url.Parse(host)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	w.hostURL = u
	return nil
}

func (w *workUnit) getPage() ([]byte, int, error) {
	err := w.overrideQueryParams(w.input.Params)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, 0, err
	}

	req, err := http.NewRequest("GET", w.parsedURL.String(), nil)
	req.Header.Add("Authorization", "Bearer "+w.config.Token)
	resp, err := w.client.Do(req)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, 0, err
	}

	w.glog.Info("GET " + w.parsedURL.String() + " Status " + resp.Status)

	err = w.validateHTTPResponse(resp, body)
	if err != nil {
		return nil, 0, err
	}
	return []byte(body), resp.StatusCode, nil
}

func (w *workUnit) validateHTTPResponse(resp *http.Response, body []byte) error {
	if !successHTTPCode(resp.StatusCode) {
		err := errors.New("HTTP GET call failed with " + resp.Status)
		w.sendError(string(body), resp.StatusCode)
		w.glog.Errorf("%v", err)
		return err
	}
	return nil
}

func (w *workUnit) post() error {
	b, err := json.Marshal(w.input.Params)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}

	req, err := http.NewRequest("POST", w.parsedURL.String(), bytes.NewBuffer(b))
	req.Header.Add("Authorization", "Bearer "+w.config.Token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	w.glog.Info("POST " + w.parsedURL.String() + " Status " + resp.Status)
	err = w.validateHTTPResponse(resp, body)
	if err != nil {
		return err
	}

	job, err := w.writeResponse(body, filepath.Join(w.parsedURL.Path, "response.json"))
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}

	if strings.ToLower(w.input.Method) == "launch" {
		u := job["url"].(string)
		w.dispatchChannel <- common.JobParam{Method: "monitor", HrefSlug: u, ApplyFilter: w.input.ApplyFilter}
	}
	return nil
}

func (w *workUnit) writeResponse(body []byte, fileName string) (map[string]interface{}, error) {
	jsonBody, err := w.createJSON(body)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, err
	}
	err = w.writePage(jsonBody, fileName)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, err
	}
	return jsonBody, nil
}

func (w *workUnit) get() error {
	body, _, err := w.getPage()
	if err != nil {
		w.glog.Errorf("Get failed Error %v", err)
		return err
	}
	filename := fmt.Sprintf("%s%d.json", w.input.PagePrefix, 1)

	jsonBody, err := w.writeResponse(body, filepath.Join(w.parsedURL.Path, filename))
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}

	err = w.requestAllRelations(jsonBody)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}

	if w.input.FetchAllPages {
		nextPage := jsonBody["next"]
		for page := 2; reflect.TypeOf(nextPage) == reflect.TypeOf("string"); page++ {
			w.input.Params["page"] = strconv.Itoa(page)
			body, _, err := w.getPage()
			if err != nil {
				w.glog.Errorf("Get failed %v", err)
				return err
			}
			filename = fmt.Sprintf("%s%d.json", w.input.PagePrefix, page)
			jsonBody, err := w.writeResponse(body, filepath.Join(w.parsedURL.Path, filename))
			if err != nil {
				w.glog.Errorf("Error %v", err)
				return err
			}
			err = w.requestAllRelations(jsonBody)
			if err != nil {
				w.glog.Errorf("Error %v", err)
				return err
			}
			nextPage = jsonBody["next"]
		}
	}
	return nil
}

func (w *workUnit) requestAllRelations(jsonBody map[string]interface{}) error {
	for _, rel := range w.relatedObjects {
		err := w.requestRelated(jsonBody, rel)
		if err != nil {
			w.glog.Errorf("Error fetching related objects %v", err)
			return err
		}
	}
	return nil
}

func (w *workUnit) requestRelated(jsonBody map[string]interface{}, related relatedObject) error {
	if val, ok := jsonBody["results"]; ok {
		for _, o := range val.([]interface{}) {
			obj := o.(map[string]interface{})
			if enabled, found := obj[related.predicate]; found {
				if !enabled.(bool) {
					continue
				}
			}
			if rel, found := obj[related.relAttribute]; found {
				url := rel.(string)
				w.dispatchChannel <- common.JobParam{Method: "GET", HrefSlug: url, ApplyFilter: related.jobExtra.ApplyFilter}
			}

		}
	}
	return nil
}

func (w *workUnit) monitor() error {
	var completedStatus = []string{"successful", "failed", "error", "canceled"}
	var allKnownStatus = []string{"new", "pending", "waiting", "running", "successful", "failed", "error", "canceled"}
	var body []byte
	var err error
	if w.input.RefreshIntervalSeconds == 0 {
		w.input.RefreshIntervalSeconds = 10
	}
	for {
		body, _, err = w.getPage()
		if err != nil {
			w.glog.Errorf("Get failed %v", err)
			return err
		}

		jsonBody, err := w.createJSON(body)
		if err != nil {
			w.glog.Errorf("create JSON failed %v", err)
			return err
		}

		v, ok := jsonBody["status"]
		if !ok {
			err = errors.New("Object does not contain a status attribute")
			w.sendError(err.Error(), 0)
			w.glog.Errorf("Error %v", err)
			return err
		}

		status := v.(string)
		if !includes(status, allKnownStatus) {
			err = errors.New("Status " + status + " is not one of the known status")
			w.sendError(err.Error(), 0)
			w.glog.Errorf("Error %v", err)
			return err
		}

		if includes(status, completedStatus) {
			break
		} else {
			time.Sleep(time.Duration(w.input.RefreshIntervalSeconds) * time.Second)
		}
	}

	_, err = w.writeResponse(body, filepath.Join(w.parsedURL.Path, "response.json"))
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}

	return nil
}

func includes(s string, values []string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

func (w *workUnit) createJSON(body []byte) (map[string]interface{}, error) {
	var jsonBody map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	err := decoder.Decode(&jsonBody)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return nil, err
	}

	if w.filterValue != nil {
		jsonBody, err = w.filterValue.Apply(jsonBody)
		if err != nil {
			w.glog.Errorf("Error %v", err)
			return nil, err
		}
	}

	v, ok := jsonBody["artifacts"]
	if ok && v != nil {
		s, err := artifacts.Sanctify(v.(map[string]interface{}))
		if err != nil {
			w.glog.Errorf("Error %v", err)
			return nil, err
		}
		jsonBody["artifacts"] = s
	}
	return jsonBody, nil
}

func (w *workUnit) writePage(jsonBody map[string]interface{}, fileName string) error {
	b, err := json.Marshal(jsonBody)
	if err != nil {
		w.glog.Errorf("Error %v", err)
		return err
	}
	w.responseChannel <- common.Page{Name: fileName, Data: b}
	return nil
}

func (w *workUnit) sendError(message string, httpStatus int) error {
	s := fmt.Sprintf("URL: %s Status: %d Message: %s", w.input.HrefSlug, httpStatus, message)
	w.errorChannel <- s
	return nil
}

func successHTTPCode(code int) bool {
	var validCodes = [...]int{200, 201, 202}
	for _, v := range validCodes {
		if v == code {
			return true
		}
	}
	return false
}
