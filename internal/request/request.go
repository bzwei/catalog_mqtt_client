package request

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/catalogtask"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/jsonwriter"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/tarwriter"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/towerapiworker"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Handler interface allows for easy mocking during testing
type Handler interface {
	StartHandlingRequests(config *common.CatalogConfig, wh towerapiworker.WorkHandler)
	//parseRequest(b []byte) (*RequestMessage, error)
}

// DefaultRequestHandler implements the 3 RequestHandler methods
type DefaultRequestHandler struct {
}

type listener interface {
	stop()
}

// StartHandlingRequests starts a request listener. It will not stop until receives a system signal.
func (drh *DefaultRequestHandler) StartHandlingRequests(config *common.CatalogConfig, wh towerapiworker.WorkHandler) {
	sigs := make(chan os.Signal, 1)
	shutdown := make(chan struct{})
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	if _, ok := os.LookupEnv("YGG_SOCKET_ADDR"); ok {
		grpcListener, err := startGRPCListener(config, wh, shutdown)
		if err == nil {
			defer grpcListener.stop()
		}
	}
	if config.MQTTURL != "" {
		mqttListener, err := startMQTTListener(config, wh, shutdown)
		if err == nil {
			defer mqttListener.stop()
		}
	}
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
	log.Info("Request listener stopped")
}

func startDispatcher(ctx context.Context, config *common.CatalogConfig, wc towerapiworker.WorkChannels, pw common.PageWriter, wh towerapiworker.WorkHandler) {
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
	makePageWriter(ctx context.Context, input common.RequestInput, task catalogtask.CatalogTask, metadata map[string]string) (common.PageWriter, error)
}

type defaultPageWriterFactory struct{}

func (factory *defaultPageWriterFactory) makePageWriter(ctx context.Context, input common.RequestInput, task catalogtask.CatalogTask, metadata map[string]string) (common.PageWriter, error) {
	var pw common.PageWriter
	var err error
	switch strings.ToLower(input.ResponseFormat) {
	case "tar":
		pw, err = tarwriter.MakeTarWriter(ctx, task, input, metadata)
	case "json":
		pw = jsonwriter.MakeJSONWriter(ctx, task)
	default:
		err = fmt.Errorf("Invalid response format %s", input.ResponseFormat)
	}
	return pw, err
}

// Process the incoming Work Request
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
	metadata := map[string]string{"task_url": url}

	pw, err := pwFactory.makePageWriter(ctx, req.Input, task, metadata)
	if err != nil {
		glog.Errorf("Error creating a page writer for type %s, reason %v", req.Input.ResponseFormat, err)
		return
	}

	err = task.Update(map[string]interface{}{"state": "running", "message": "Catalog Worker Started at " + time.Now().Format(time.RFC3339)})
	if err != nil {
		glog.Errorf("Error updating the task with the starting message, reason %v", err)
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
	defer close(wc.WaitChannel)

	wc.Shutdown = shutdown
	go startDispatcher(ctx, config, wc, pw, wh)

	for _, j := range req.Input.Jobs {
		wc.DispatchChannel <- j
	}

	timeout := viper.GetInt64("worker.timeout_minutes")
	if timeout == 0 {
		timeout = 10
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
		case <-time.After(time.Duration(timeout) * time.Minute):
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
	glog.Info(stats())
	defer glog.Info("Worker finished")
	wh.StartWork(ctx, config, job, nil, wc)
	wc.FinishedChannel <- true
}

// stats
func stats() string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return fmt.Sprintf("Current Stats Alloc = %v MiB TotalAlloc = %v MiB Sys = %v MiB NumGC = %v NumGoroutine = %v",
		bToMb(ms.Alloc), bToMb(ms.TotalAlloc), bToMb(ms.Sys), ms.NumGC, runtime.NumGoroutine())

}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
