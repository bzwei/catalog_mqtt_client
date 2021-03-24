package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/RedHatInsights/rhc-worker-catalog/build"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/request"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/towerapiworker"
	log "github.com/sirupsen/logrus"
	viper "github.com/spf13/viper"
)

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
	var err error
	flag.StringVar(&configFilePath, "config", "", "location of the config file")
	flag.Parse()

	if configFilePath == "" {
		if configFilePath, err = getConfigFile(); err != nil {
			panic(err)
		}
	}
	dir, file := filepath.Split(configFilePath)
	viper.SetConfigName(file)
	viper.SetConfigType("toml")
	viper.AddConfigPath(dir)
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Failed to import configuration file %s, reason %v", configFilePath, err))
	}
}

func startRun(config *common.CatalogConfig, rh request.Handler) {
	defer log.Info("Finished Catalog Worker")
	log.Infof("Catalog Worker Version %s GIT SHA %s Build %s", build.Version, build.Sha1, build.Build)

	rh.StartHandlingRequests(config, &towerapiworker.DefaultAPIWorker{})
}

func getConfigFile() (string, error) {
	for _, filename := range candidateConfigFiles() {
		if fileExists(filename) {
			return filename, nil
		}
	}

	return "", fmt.Errorf("Cannot find catalog.toml at default locations")
}

func candidateConfigFiles() []string {
	var s []string
	s = append(s, "./rhc/workers/catalog.toml")
	usr, err := user.Current()
	if err == nil {
		s = append(s, fmt.Sprintf("%s/.config/rhc/workers/catalog.toml", usr.HomeDir))
	}
	s = append(s, "/etc/rhc/workers/catalog.toml")
	return s
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func makeConfig() *common.CatalogConfig {
	config := common.CatalogConfig{}

	config.Token = viper.GetString("ANSIBLE_TOWER.token")
	config.URL = viper.GetString("ANSIBLE_TOWER.url")
	config.SkipVerifyCertificate = !viper.GetBool("ANSIBLE_TOWER.verify_ssl")
	config.Level = viper.GetString("logger.level")
	config.MQTTURL = viper.GetString("MQTT_BROKER.url")
	config.GUID = viper.GetString("MQTT_BROKER.uuid")

	flag.Parse()
	level, err := log.ParseLevel(config.Level)
	if err == nil {
		log.SetLevel(level)
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
