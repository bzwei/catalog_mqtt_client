package upload

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"

	log "github.com/sirupsen/logrus"
)

func Upload(url string, name string, contentType string) ([]byte, error) {
	r, w := io.Pipe()
	m := multipart.NewWriter(w)
	go func() {
		defer w.Close()
		defer m.Close()

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="file"; filename="%s"`, "inventory.tgz"))
		h.Set("Content-Type", "application/vnd.redhat.topological-inventory.filename+tgz")
		part, err := m.CreatePart(h)
		file, err := os.Open(name)
		if err != nil {
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()

	req, err := http.NewRequest("POST", url, r)
	if err != nil {
		return nil, err
	}
	fmt.Println(m.FormDataContentType())
	req.Header.Set("Content-Type", m.FormDataContentType())
	user := os.Getenv("USER")
	if user == "" {
		err = fmt.Errorf("Environmental variable USER is not set")
		return nil, err
	}
	password := os.Getenv("PASSWORD")
	if user == "" {
		err = fmt.Errorf("Environmental variable PASSWORD is not set")
		return nil, err
	}
	req.SetBasicAuth(user, password)
	client := &http.Client{}
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
	if res.StatusCode != 202 {
		err = fmt.Errorf("Upload failed %d %s", res.StatusCode, string(body))
		return nil, err
	}
	log.Info("Response from upload " + url + " Status " + res.Status)
	log.Infof("Reponse from Post %s", string(body))
	return body, nil
}
