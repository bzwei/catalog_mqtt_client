package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mkanoor/catalog_mqtt_client/internal/jsonwriter"
	"github.com/mkanoor/catalog_mqtt_client/internal/logger"
	"github.com/mkanoor/catalog_mqtt_client/internal/tarwriter"
	log "github.com/sirupsen/logrus"
)

type PageWriter interface {
	Write(name string, b []byte) error
	Flush() error
	FlushErrors(msg []string) error
}

// JobParam stores the single parameter set for a job
type JobParam struct {
	Method                 string                 `json:"method"`
	HrefSlug               string                 `json:"href_slug"`
	FetchAllPages          bool                   `json:"fetch_all_pages"`
	Params                 map[string]interface{} `json:"params"`
	AcceptEncoding         string                 `json:"accept_encoding"`
	ApplyFilter            interface{}            `json:"apply_filter"`
	RefreshIntervalSeconds int64                  `json:"refresh_interval_seconds"`
	FetchRelated           []interface{}          `json:"fetch_related"`
	PagePrefix             string                 `json:"page_prefix"`
}

type Page struct {
	Data []byte
	Name string
}

type RequestMessage struct {
	Context struct {
		ResponseFormat string     `json:"response_format"`
		UploadURL      string     `json:"upload_url"`
		Jobs           []JobParam `json:"jobs"`
	} `json:"context"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MQTTMessage struct {
	URL  string `json:"url"`
	Kind string `json:"kind"`
	Sent string `json:"string"`
}

// RequestHandler interface allows for easy mocking during testing
type RequestHandler interface {
	startHandlingRequests(mqttClient mqtt.Client, config *CatalogConfig, wh WorkHandler)
	//parseRequest(b []byte) (*RequestMessage, error)
}

// DefaultRequestHandler implements the 3 RequestHandler methods
type DefaultRequestHandler struct {
}

// getRequest get data from the Receptor via Stdin
func (drh *DefaultRequestHandler) startHandlingRequests(mqttClient mqtt.Client, config *CatalogConfig, wh WorkHandler) {
	defer mqttClient.Disconnect(10)
	sigs := make(chan os.Signal, 1)
	shutdown := make(chan struct{})
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	startMQTTListener(mqttClient, config, wh, shutdown)
	done := false
	for !done {
		select {
		case sig := <-sigs:
			log.Info("Signal Received")
			fmt.Println(sig)
			done = true
			close(shutdown)
		}
	}
	log.Info("MQTT Client Ending")
}

func startMQTTListener(mqttClient mqtt.Client, config *CatalogConfig, wh WorkHandler, shutdown chan struct{}) {
	ctx := context.Background()
	topic := "out/" + config.GUID
	log.Infof("Subscribing to topic %s", topic)
	counter := 0
	fn := func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("Received a MQTT request %s", string(msg.Payload()))
		m := MQTTMessage{}
		mqttDecoder := json.NewDecoder(bytes.NewReader(msg.Payload()))
		mqttDecoder.UseNumber()
		err := mqttDecoder.Decode(&m)
		if err != nil {
			log.Errorf("Error decoding mqtt json %v", err)
			return
		}
		log.Infof("Process Request %s", m.URL)
		counter++
		go processRequest(logger.CtxWithLoggerID(ctx, counter), m.URL, config, wh, shutdown)
	}

	if token := mqttClient.Subscribe(topic, 0, fn); token.Wait() && token.Error() != nil {
		log.Errorf("Encountered Token Error %v", token.Error())
	}
}

// Parse the request into RequestMessage
func parseRequest(b []byte) (*RequestMessage, error) {
	req := RequestMessage{}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	err := decoder.Decode(&req)
	if err != nil {
		log.Errorf("Error decoding json %v", err)
		return nil, err
	}
	return &req, nil
}

func startDispatcher(ctx context.Context, config *CatalogConfig, wc WorkChannels, pw PageWriter, wh WorkHandler) {
	glog := logger.GetLogger(ctx)
	done := false
	totalCount := 0
	finishedCount := 0
	for !done {
		select {
		case j := <-wc.dispatchChannel:
			glog.Infof("Job Input Data %v", j)
			totalCount++
			go startWorker(ctx, config, j, wh, wc)
		case <-wc.shutdown:
			done = true
		case page := <-wc.responseChannel:
			glog.Infof("Data received on response channel %s", page.Name)
			pw.Write(page.Name, page.Data)
		case <-wc.finishedChannel:
			finishedCount++
		default:
			if totalCount > 0 && totalCount == finishedCount {
				done = true
			}
		}
	}
	wc.waitChannel <- true
}

// Process the incoming MQTT Work Request
// Fetch the Actual WorkPayload and start the work
func processRequest(ctx context.Context, url string, config *CatalogConfig, wh WorkHandler, shutdown chan struct{}) {
	glog := logger.GetLogger(ctx)
	defer glog.Info("Request finished")
	var pw PageWriter
	body, err := getWorkPayload(ctx, url)
	if err != nil {
		glog.Errorf("Error reading payload in %s %v", url, err)
		return
	}

	req, err := parseRequest(body)
	if err != nil {
		glog.Errorf("Error parsing payload in %s %v", url, err)
		return
	}
	switch strings.ToLower(req.Context.ResponseFormat) {
	case "tar":
		pw, err = tarwriter.MakeTarWriter(ctx, url, req.Context.UploadURL)
		if err != nil {
			glog.Errorf("Error creating Tar Writer")
			return
		}
	case "json":
		pw = jsonwriter.MakeJSONWriter(ctx, url)
	default:
		glog.Errorf("Invalid response format %s for url %s", req.Context.ResponseFormat, url)
		return
	}

	wc := WorkChannels{}
	wc.errorChannel = make(chan string)
	wc.dispatchChannel = make(chan JobParam)
	wc.responseChannel = make(chan Page)
	wc.finishedChannel = make(chan bool)
	wc.waitChannel = make(chan bool)
	defer close(wc.errorChannel)
	defer close(wc.dispatchChannel)
	defer close(wc.finishedChannel)
	defer close(wc.responseChannel)

	wc.shutdown = shutdown
	go startDispatcher(ctx, config, wc, pw, wh)

	for _, j := range req.Context.Jobs {
		wc.dispatchChannel <- j
	}
	var allErrors []string
	allDone := false
	for !allDone {
		select {
		case <-wc.waitChannel:
			glog.Info("Workers finished")
			allDone = true
		case data := <-wc.errorChannel:
			glog.Infof("Error received %s", data)
			allErrors = append(allErrors, data)
		case <-time.After(10 * time.Minute):
			glog.Infof("Waitgroup timedout")
			allDone = true
		case <-wc.shutdown:
			glog.Infof("SHutdown received")
			allDone = true
		}
	}

	if len(allErrors) > 0 {
		pw.FlushErrors(allErrors)
	} else {
		pw.Flush()
	}

}

func getWorkPayload(ctx context.Context, url string) ([]byte, error) {
	glog := logger.GetLogger(ctx)
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

// Start a work
func startWorker(ctx context.Context, config *CatalogConfig, job JobParam, wh WorkHandler, wc WorkChannels) {
	glog := logger.GetLogger(ctx)
	glog.Info("Worker starting")
	defer glog.Info("Worker finished")
	wh.StartWork(ctx, config, job, nil, wc)
	wc.finishedChannel <- true
}
