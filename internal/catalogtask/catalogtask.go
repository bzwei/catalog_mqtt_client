package catalogtask

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
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
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPatch, ct.url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	xrh := os.Getenv("X_RH_IDENTITY")
	if xrh == "" {
		err = fmt.Errorf("Environmental variable X_RH_IDENTITY is not set")
		ct.glog.Errorf("%v", err)
		return err
	}
	req.Header.Set("x-rh-identity", xrh)
	if err != nil {
		ct.glog.Errorf("Error creating a new request %v", err)
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
		err = fmt.Errorf("Invalid HTTP Status code from post %d", resp.StatusCode)
		ct.glog.Errorf("Error %v", err)
		return err
	}
	ct.glog.Infof("Task Update Statue Code %d", resp.StatusCode)

	ct.glog.Infof("Response from Patch %s", string(body))
	return nil
}

func getWorkPayload(glog logger.Logger, url string) ([]byte, error) {
	client := &http.Client{}
	xrh := "eyJlbnRpdGxlbWVudHMiOnsiaW5zaWdodHMiOnsiaXNfZW50aXRsZWQiOnRydWUsImlzX3RyaWFsIjpmYWxzZX0sImNvc3RfbWFuYWdlbWVudCI6eyJpc19lbnRpdGxlZCI6dHJ1ZSwiaXNfdHJpYWwiOmZhbHNlfSwibWlncmF0aW9ucyI6eyJpc19lbnRpdGxlZCI6dHJ1ZSwiaXNfdHJpYWwiOmZhbHNlfSwiYW5zaWJsZSI6eyJpc19lbnRpdGxlZCI6dHJ1ZSwiaXNfdHJpYWwiOmZhbHNlfSwidXNlcl9wcmVmZXJlbmNlcyI6eyJpc19lbnRpdGxlZCI6dHJ1ZSwiaXNfdHJpYWwiOmZhbHNlfSwib3BlbnNoaWZ0Ijp7ImlzX2VudGl0bGVkIjp0cnVlLCJpc190cmlhbCI6ZmFsc2V9LCJzbWFydF9tYW5hZ2VtZW50Ijp7ImlzX2VudGl0bGVkIjp0cnVlLCJpc190cmlhbCI6ZmFsc2V9LCJzdWJzY3JpcHRpb25zIjp7ImlzX2VudGl0bGVkIjp0cnVlLCJpc190cmlhbCI6ZmFsc2V9LCJzZXR0aW5ncyI6eyJpc19lbnRpdGxlZCI6dHJ1ZSwiaXNfdHJpYWwiOmZhbHNlfX0sImlkZW50aXR5Ijp7ImludGVybmFsIjp7ImF1dGhfdGltZSI6Nzk5LCJvcmdfaWQiOiIxMTc4OTc3MiJ9LCJhY2NvdW50X251bWJlciI6IjYwODk3MTkiLCJhdXRoX3R5cGUiOiJiYXNpYy1hdXRoIiwidXNlciI6eyJpc19hY3RpdmUiOnRydWUsImxvY2FsZSI6ImVuX1VTIiwiaXNfb3JnX2FkbWluIjp0cnVlLCJ1c2VybmFtZSI6Imluc2lnaHRzLXFhIiwiZW1haWwiOiJkYWpvaG5zb0ByZWRoYXQuY29tIiwiZmlyc3RfbmFtZSI6Ikluc2lnaHRzIiwidXNlcl9pZCI6IjUxODM0Nzc2IiwibGFzdF9uYW1lIjoiUUEiLCJpc19pbnRlcm5hbCI6dHJ1ZX0sInR5cGUiOiJVc2VyIn19"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		glog.Errorf("Error creating request %s %v", url, err)
		return nil, err
	}
	req.Header.Add("x-rh-identity", xrh)
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Error fetching request %s %v", url, err)
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
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
