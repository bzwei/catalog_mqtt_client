package upload

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	log "github.com/sirupsen/logrus"
)

// Upload uploads a file with metadata to the url
func Upload(url string, filename string, contentType string, metadata map[string]string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Error opening file %s %v", filename, err)
	}
	defer file.Close()
	r, w := io.Pipe()
	m := multipart.NewWriter(w)
	go func() {
		defer w.Close()
		defer m.Close()

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="file"; filename="%s"`, "inventory.tgz"))
		// TODO : Set the metadata when the ingress service supports it
		// For now override the ContentType to include the task_id

		h.Set("Content-Type", overrideContentType(metadata))
		part, err := m.CreatePart(h)
		if err != nil {
			return
		}
		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()

	req, err := http.NewRequest("POST", url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", m.FormDataContentType())

	client, err := common.MakeHTTPClient(req)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Error reading body %v", err)
		return nil, err
	}
	if res.StatusCode != http.StatusAccepted {
		err = fmt.Errorf("Upload failed %d %s", res.StatusCode, string(body))
		return nil, err
	}
	log.Info("Response from upload " + url + " Status " + res.Status)
	log.Infof("Response from Post %s", string(body))
	return body, nil
}

// overrideContentType inserts the task_id as part of the content tye
// till we can get a more permanent solution to send metadata along
// with multipart contents
func overrideContentType(metadata map[string]string) string {
	ct := "application/vnd.redhat.topological-inventory.filename+tgz"
	if val, ok := metadata["task_url"]; ok {
		parts := strings.Split(val, "/")
		taskID := parts[len(parts)-1]
		ct = fmt.Sprintf("application/vnd.redhat.topological-inventory.%s+tgz", taskID)
	}
	return ct
}
