package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/request"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	viper "github.com/spf13/viper"
)

var Version = "development"
var Sha1 = "unknown"

func main() {
	initConfig()

	logf := configLogger()
	if logf != nil {
		defer logf.Close()
	}

	config := makeConfig()

	mqttURL := viper.GetString("MQTT_BROKER.url")
	uri, err := url.Parse(mqttURL)
	if err != nil {
		log.Errorf("Error parsing MQTT URL %s %v", mqttURL, err)
		return
	}

	mqttClient, err := connect("tower_client_"+viper.GetString("MQTT_BROKER.uuid"), uri)
	if err != nil {
		log.Errorf("Error connecting to MQTT Server %v", err)
		return
	}
	log.Infof("Connected to MQTT Server %s", mqttURL)

	startRun(config, mqttClient, &request.DefaultRequestHandler{})
}

func initConfig() {
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "/etc/redhat_mqtt_client/catalog.conf", "location of the config file")
	flag.Parse()

	dir, file := filepath.Split(configFilePath)
	viper.SetConfigName(file)
	viper.SetConfigType("ini")
	viper.AddConfigPath(dir)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Failed to import configuration file %s", configFilePath))
	}
}

func connect(clientID string, uri *url.URL) (mqtt.Client, error) {
	opts := createClientOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}

func createClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	return opts
}

func startRun(config *common.CatalogConfig, mqttClient mqtt.Client, rh request.Handler) {
	defer log.Info("Finished Catalog Worker")
	log.Infof("Catalog MQTT Client version %s GIT SHA %s", Version, Sha1)

	rh.StartHandlingRequests(mqttClient, config, &towerapiworker.DefaultAPIWorker{})
}

func makeConfig() *common.CatalogConfig {
	config := common.CatalogConfig{}

	config.Token = viper.GetString("ANSIBLE_TOWER.token")
	config.URL = viper.GetString("ANSIBLE_TOWER.url")
	config.SkipVerifyCertificate = !viper.GetBool("ANSIBLE_TOWER.verify_ssl")
	config.Debug = viper.GetBool("logger.debug")
	config.MQTTURL = viper.GetString("MQTT_BROKER.url")
	config.GUID = viper.GetString("MQTT_BROKER.uuid")

	flag.Parse()
	if config.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	if config.Token == "" || config.URL == "" || config.GUID == "" {
		log.Fatal("Token, GUID and URL parameters are required")
	}

	return &config
}

// Configure the logger
func configLogger() *os.File {
	logFileName := viper.GetString("logger.logfile") + strconv.Itoa(os.Getpid()) + ".log"
	logf, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(logf)
	log.SetReportCaller(true)
	return logf
}
