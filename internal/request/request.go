package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/jsonwriter"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/tarwriter"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type PageWriter interface {
	Write(name string, b []byte) error
	Flush() error
	FlushErrors(msg []string) error
}

// RequestHandler interface allows for easy mocking during testing
type RequestHandler interface {
	StartHandlingRequests(mqttClient mqtt.Client, config *common.CatalogConfig, wh towerapiworker.WorkHandler)
	//parseRequest(b []byte) (*RequestMessage, error)
}

// DefaultRequestHandler implements the 3 RequestHandler methods
type DefaultRequestHandler struct {
}

// getRequest get data from the Receptor via Stdin
func (drh *DefaultRequestHandler) StartHandlingRequests(mqttClient mqtt.Client, config *common.CatalogConfig, wh towerapiworker.WorkHandler) {
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

func startMQTTListener(mqttClient mqtt.Client, config *common.CatalogConfig, wh towerapiworker.WorkHandler, shutdown chan struct{}) {
	ctx := context.Background()
	topic := "out/" + config.GUID
	log.Infof("Subscribing to topic %s", topic)
	counter := 0
	fn := func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("Received a MQTT request %s", string(msg.Payload()))
		m := common.MQTTMessage{}
		mqttDecoder := json.NewDecoder(bytes.NewReader(msg.Payload()))
		mqttDecoder.UseNumber()
		err := mqttDecoder.Decode(&m)
		if err != nil {
			log.Errorf("Error decoding mqtt json %v", err)
			return
		}
		log.Infof("Process Request %s", m.URL)
		counter++
		nextCtx := logger.CtxWithLoggerID(ctx, counter)
		go processRequest(nextCtx, m.URL, config, wh, catalogtask.MakeCatalogTask(nextCtx, m.URL), &defaultPageWriterFactory{}, shutdown)
	}

	if token := mqttClient.Subscribe(topic, 0, fn); token.Wait() && token.Error() != nil {
		log.Errorf("Encountered Token Error %v", token.Error())
	}
}

func startDispatcher(ctx context.Context, config *common.CatalogConfig, wc towerapiworker.WorkChannels, pw PageWriter, wh towerapiworker.WorkHandler) {
	glog := logger.GetLogger(ctx)
	done := false
	totalCount := 0
	finishedCount := 0
	for !done {
		select {
		case job := <-wc.DispatchChannel:
			glog.Infof("Job Input Data %v", job)
			totalCount++
			go startWorker(ctx, config, job, wh, wc)
		case <-wc.Shutdown:
			done = true
		case page := <-wc.ResponseChannel:
			glog.Infof("Data received on response channel %s", page.Name)
			pw.Write(page.Name, page.Data)
		case <-wc.FinishedChannel:
			finishedCount++
		default:
			if totalCount > 0 && totalCount == finishedCount {
				done = true
			}
		}
	}
	wc.WaitChannel <- true
}

type pageWriterFactory interface {
	makePageWriter(ctx context.Context, format string, uploadURL string, task catalogtask.CatalogTask, taskURL string) (PageWriter, error)
}

type defaultPageWriterFactory struct{}

func (factory *defaultPageWriterFactory) makePageWriter(ctx context.Context, format string, uploadURL string, task catalogtask.CatalogTask, taskURL string) (PageWriter, error) {
	var pw PageWriter
	var err error
	switch strings.ToLower(format) {
	case "tar":
		metadata := map[string]string{
			"task_url": taskURL,
		}
		pw, err = tarwriter.MakeTarWriter(ctx, task, uploadURL, metadata)
	case "json":
		pw = jsonwriter.MakeJSONWriter(ctx, task)
	default:
		err = fmt.Errorf("Invalid response format %s", format)
	}
	return pw, err
}

// Process the incoming MQTT Work Request
// Fetch the Actual WorkPayload and start the work
func processRequest(ctx context.Context,
	url string, config *common.CatalogConfig,
	wh towerapiworker.WorkHandler,
	task catalogtask.CatalogTask,
	pwFactory pageWriterFactory,
	shutdown chan struct{}) {

	glog := logger.GetLogger(ctx)
	defer glog.Info("Request finished")

	req, err := task.Get()
	if err != nil {
		glog.Errorf("Error parsing payload in %s, reason %v", url, err)
		return
	}

	pw, err := pwFactory.makePageWriter(ctx, req.Input.ResponseFormat, req.Input.UploadURL, task, url)
	if err != nil {
		glog.Errorf("Error creating a page writer for type %s, reason %v", req.Input.ResponseFormat, err)
		return
	}

	wc := towerapiworker.WorkChannels{}
	wc.ErrorChannel = make(chan string)
	wc.DispatchChannel = make(chan common.JobParam)
	wc.ResponseChannel = make(chan common.Page)
	wc.FinishedChannel = make(chan bool)
	wc.WaitChannel = make(chan bool)
	defer close(wc.ErrorChannel)
	defer close(wc.DispatchChannel)
	defer close(wc.FinishedChannel)
	defer close(wc.ResponseChannel)

	wc.Shutdown = shutdown
	go startDispatcher(ctx, config, wc, pw, wh)

	for _, j := range req.Input.Jobs {
		wc.DispatchChannel <- j
	}
	var allErrors []string
	allDone := false
	for !allDone {
		select {
		case <-wc.WaitChannel:
			glog.Info("Workers finished")
			allDone = true
		case data := <-wc.ErrorChannel:
			glog.Infof("Error received %s", data)
			allErrors = append(allErrors, data)
		case <-time.After(10 * time.Minute):
			glog.Infof("Waitgroup timed out")
			allDone = true
		case <-wc.Shutdown:
			glog.Infof("Shutdown received")
			allDone = true
		}
	}

	if len(allErrors) > 0 {
		pw.FlushErrors(allErrors)
	} else {
		pw.Flush()
	}
}

// Start a work
func startWorker(ctx context.Context, config *common.CatalogConfig, job common.JobParam, wh towerapiworker.WorkHandler, wc towerapiworker.WorkChannels) {
	glog := logger.GetLogger(ctx)
	glog.Info("Worker starting")
	defer glog.Info("Worker finished")
	wh.StartWork(ctx, config, job, nil, wc)
	wc.FinishedChannel <- true
}
