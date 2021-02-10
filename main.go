package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/request"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/towerapiworker"
	log "github.com/sirupsen/logrus"
	viper "github.com/spf13/viper"
)

// Version of the release
var Version = "development"

// Sha1 is the sha of source commit for the release
var Sha1 = "unknown"

func main() {
	initConfig()

	logf := configLogger()
	if logf != nil {
		defer logf.Close()
	}

	config := makeConfig()

	startRun(config, &request.DefaultRequestHandler{})
}

func initConfig() {
	var configFilePath string
	flag.StringVar(&configFilePath, "config", "/etc/rhc/workers/catalog.toml", "location of the config file")
	flag.Parse()

	dir, file := filepath.Split(configFilePath)
	viper.SetConfigName(file)
	viper.SetConfigType("toml")
	viper.AddConfigPath(dir)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Failed to import configuration file %s, reason %v", configFilePath, err))
	}
}

func startRun(config *common.CatalogConfig, rh request.Handler) {
	defer log.Info("Finished Catalog Worker")
	log.Infof("Catalog MQTT Client version %s GIT SHA %s", Version, Sha1)

	rh.StartHandlingRequests(config, &towerapiworker.DefaultAPIWorker{})
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
	/*
		if config.Token == "" || config.URL == "" || config.GUID == "" {
			log.Fatal("Token, GUID and URL parameters are required")
		}
	*/

	return &config
}

// Configure the logger
func configLogger() *os.File {
	logf := os.Stdout
	logFileName := viper.GetString("logger.logfile")
	if logFileName != "" {
		var err error
		logFileName = logFileName + strconv.Itoa(os.Getpid()) + ".log"
		logf, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(logf)
	return logf
}
