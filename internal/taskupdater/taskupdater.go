package taskupdater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mkanoor/catalog_mqtt_client/internal/logger"
)

type TaskUpdater struct {
	Url  string
	ctx  context.Context
	glog logger.Logger
}

func MakeTaskUpdater(ctx context.Context, url string) *TaskUpdater {
	glog := logger.GetLogger(ctx)

	return &TaskUpdater{Url: url, glog: glog, ctx: ctx}
}

// Write a Page given the name and the number of bytes to write
func (tu *TaskUpdater) Do(state string, status string, result *map[string]interface{}) ([]byte, error) {
	var payload []byte
	var err error

	if result == nil {
		payload, err = json.Marshal(map[string]interface{}{
			"state":  state,
			"status": status})
	} else {
		payload, err = json.Marshal(map[string]interface{}{
			"state":  state,
			"status": status,
			"result": result})
	}

	if err != nil {
		tu.glog.Errorf("Error Marshaling Payload %v", err)
		return nil, err
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPatch, tu.Url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	xrh := os.Getenv("X_RH_IDENTITY")
	if xrh == "" {
		err = fmt.Errorf("Environmental variable X_RH_IDENTITY is not set")
		tu.glog.Errorf("%v", err)
		return nil, err
	}
	req.Header.Set("x-rh-identity", xrh)
	if err != nil {
		tu.glog.Errorf("Error creating a new request %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		tu.glog.Errorf("Error processing request %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tu.glog.Errorf("Error reading body %v", err)
		return nil, err
	}
	if resp.StatusCode != 204 {
		err = fmt.Errorf("Invalid HTTP Status code from post %d", resp.StatusCode)
		tu.glog.Errorf("Error %v", err)
		return nil, err
	}
	tu.glog.Infof("Task Update Statue Code %d", resp.StatusCode)

	tu.glog.Infof("Reponse from Patch %s", string(body))
	return body, nil
}
