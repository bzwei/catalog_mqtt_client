package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/request"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

var Version = "development"
var Sha1 = "unknown"

func main() {
	startRun(os.Stdin, &request.DefaultRequestHandler{})
}

func connect(clientId string, uri *url.URL) (mqtt.Client, error) {
	opts := createClientOptions(clientId, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}

func createClientOptions(clientId string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientId)
	return opts
}

func startRun(reader io.Reader, rh request.Handler) {
	config := common.CatalogConfig{}
	logFileName := "/tmp/catalog_mqtt_client" + strconv.Itoa(os.Getpid()) + ".log"
	logf, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logf.Close()
	defer log.Info("Finished Catalog Worker")

	setConfig(&config)

	configLogger(&config, logf)
	log.Infof("Catalog MQTT Client version %s GIT SHA %s", Version, Sha1)
	log.Infof("Config Debug: %v", config.Debug)
	log.Infof("Config URL: %v", config.URL)
	log.Infof("Config Token: %v", config.Token)
	log.Infof("Config SkipVerifyCertificate: %v", config.SkipVerifyCertificate)
	log.Infof("Config MQTTURL: %v", config.MQTTURL)
	log.Infof("Config GUID: %v", config.GUID)

	log.Debug("Processing request")
	uri, err := url.Parse(config.MQTTURL)
	if err != nil {
		log.Errorf("Error parsing MQTT URL %s %v", config.MQTTURL, err)
		return
	}

	mqttClient, err := connect("tower_client_"+config.GUID, uri)
	if err != nil {
		log.Errorf("Error connecting to MQTT Server %v", err)
		return
	}

	log.Infof("Connected to MQTT Server %s", config.MQTTURL)
	rh.StartHandlingRequests(mqttClient, &config, &towerapiworker.DefaultAPIWorker{})
}

func setConfig(config *common.CatalogConfig) {
	flag.StringVar(&config.Token, "token", "", "Ansible Tower token")
	flag.StringVar(&config.URL, "url", "", "Ansible Tower URL")
	flag.BoolVar(&config.Debug, "debug", false, "log debug messages")
	flag.BoolVar(&config.SkipVerifyCertificate, "skip_verify_ssl", false, "skip tower certificate verification")
	flag.StringVar(&config.MQTTURL, "mqtturl", "", "MQTTURL")
	flag.StringVar(&config.GUID, "guid", "", "Client GUID")

	flag.Parse()

	if config.Token == "" || config.URL == "" || config.GUID == "" {
		log.Fatal("Token, GUID and URL parameters are required")
	}

}

// Configure the logger
func configLogger(config *common.CatalogConfig, f *os.File) {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(f)
	if config.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	log.SetReportCaller(true)
}
