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

func OldUpload(url string, name string, contentType string) ([]byte, error) {
	rp, wp := io.Pipe()
	m := multipart.NewWriter(wp)
	go func() {
		defer m.Close()
		defer wp.Close()

		mh := make(textproto.MIMEHeader)
		mh.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="file"; filename="%s"`, "inventory.tgz"))
		mh.Set("Content-Type", "application/vnd.redhat.topological-inventory.filename+tgz")
		/*
			mh.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="file"; filename="%s"`, "inventory.tgz"))
			mh.Set("Content-Type", contentType)
		*/
		part, err := m.CreatePart(mh)
		file, err := os.Open(name)
		if err != nil {
			log.Errorf("Error opening file %s %v", name, err)
			return
		}
		defer file.Close()
		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, rp)
	if err != nil {
		log.Errorf("Error creating a new request %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.SetBasicAuth("insights-qa", "redhatqa")
	/*
		xrh := "eyJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IjE0NjAyOTAiLCJ0eXBlIjoiVXNlciIsInVzZXIiOnsidXNlcm5hbWUiOiJnbWNjdWxsb0ByZWRoYXQuY29tIiwiZW1haWwiOiJnbWNjdWxsb0ByZWRoYXQuY29tIiwiZmlyc3RfbmFtZSI6Ik1hZGh1IiwibGFzdF9uYW1lIjoiS2Fub29yIiwiaXNfYWN0aXZlIjp0cnVlLCJpc19vcmdfYWRtaW4iOmZhbHNlLCJpc19pbnRlcm5hbCI6ZmFsc2UsImxvY2FsZSI6ImVuX1VTIn0sImludGVybmFsIjp7Im9yZ19pZCI6IjMzNDA4NTEiLCJhdXRoX3R5cGUiOiJiYXNpYy1hdXRoIiwiYXV0aF90aW1lIjo2MzAwfX19"
		req.Header.Set("x-rh-identity", xrh)
	*/
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error processing request %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading body %v", err)
		return nil, err
	}

	log.Info("Response from upload " + url + " Status " + resp.Status)
	log.Infof("Reponse from Post %s", string(body))
	return body, nil
}
