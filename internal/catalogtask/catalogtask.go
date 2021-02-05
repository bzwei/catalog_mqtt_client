package catalogtask

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
)

// CatalogTask is an interface that gets or updates a catalog task
type CatalogTask interface {
	Get() (*common.RequestMessage, error)
	Update(data map[string]interface{}) error
}

type defaultCatalogTask struct {
	url  string
	ctx  context.Context
	glog logger.Logger
}

// MakeCatalogTask returns a struct that implements interface CatalogTask
func MakeCatalogTask(ctx context.Context, url string) CatalogTask {
	glog := logger.GetLogger(ctx)

	return &defaultCatalogTask{ctx: ctx, url: url, glog: glog}
}

func (ct *defaultCatalogTask) Get() (*common.RequestMessage, error) {
	body, err := getWorkPayload(ct.glog, ct.url)
	if err != nil {
		ct.glog.Errorf("Error reading payload in %s %v", ct.url, err)
		return nil, err
	}

	req, err := parseRequest(ct.glog, body)
	if err != nil {
		ct.glog.Errorf("Error parsing payload in %s %v", ct.url, err)
	}
	return req, err
}

func (ct *defaultCatalogTask) Update(data map[string]interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		ct.glog.Errorf("Error Marshaling Payload %v", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, ct.url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		ct.glog.Errorf("Error creating a new request %v", err)
		return err
	}

	client, err := common.MakeHTTPClient(req)
	if err != nil {
		ct.glog.Errorf("Error creating a http client %v", err)
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		ct.glog.Errorf("Error processing request %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ct.glog.Errorf("Error reading body %v", err)
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		err = fmt.Errorf("Invalid HTTP Status code from patch %d", resp.StatusCode)
		ct.glog.Errorf("Error %v", err)
		return err
	}
	ct.glog.Infof("Task Update Status Code %d", resp.StatusCode)
	ct.glog.Infof("Response from Patch %s", string(body))
	return nil
}

func getWorkPayload(glog logger.Logger, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		glog.Errorf("Error creating request %s %v", url, err)
		return nil, err
	}

	client, err := common.MakeHTTPClient(req)
	if err != nil {
		glog.Errorf("Error creating http client %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Error fetching request %s %v", url, err)
		return nil, err
	}
	if !successGetCode(resp.StatusCode) {
		err = fmt.Errorf("Invalid HTTP Status code from get %s, status: %d", url, resp.StatusCode)
		glog.Errorf("Error %v", err)
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func successGetCode(code int) bool {
	var validCodes = [...]int{200, 201, 202}
	for _, v := range validCodes {
		if v == code {
			return true
		}
	}
	return false
}

// Parse the request into RequestMessage
func parseRequest(glog logger.Logger, b []byte) (*common.RequestMessage, error) {
	req := common.RequestMessage{}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	err := decoder.Decode(&req)
	if err != nil {
		glog.Errorf("Error decoding json %v", err)
		return nil, err
	}
	return &req, nil
}
