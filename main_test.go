package main

import (
	"os"
	"testing"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

type FakeRequestHandler struct {
	mqttClient    mqtt.Client
	catalogConfig common.CatalogConfig
	workHandler   towerapiworker.WorkHandler
}

func (frh *FakeRequestHandler) StartHandlingRequests(mqttClient mqtt.Client, config *common.CatalogConfig, wh towerapiworker.WorkHandler) {
	frh.catalogConfig = *config
	frh.mqttClient = mqttClient
	frh.workHandler = wh
}

func TestMain(t *testing.T) {
	os.Args = []string{"catalog_worker", "--config", "./sample.conf"}

	frh := &FakeRequestHandler{}
	mqttClient := mqtt.NewClient(mqtt.NewClientOptions())

	initConfig()
	logf := configLogger()
	startRun(makeConfig(), mqttClient, frh)

	info, err := logf.Stat()
	assert.NoError(t, err)
	logf.Close()
	os.Remove(info.Name())

	assert.True(t, info.Size() > 0)
	assert.True(t, frh.catalogConfig.Debug)
	assert.Equal(t, "<<Your Tower URL>>", frh.catalogConfig.URL)
	assert.Equal(t, "<<Your Tower Token>>", frh.catalogConfig.Token)
	assert.Equal(t, &towerapiworker.DefaultAPIWorker{}, frh.workHandler)
	assert.Equal(t, mqttClient, frh.mqttClient)
}
