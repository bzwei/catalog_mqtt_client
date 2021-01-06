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
	os.Args = []string{"catalog_worker",
		"--debug",
		"--token", "gobbledygook",
		"--guid", "guid-uuid",
		"--url", "https://www.example.com"}
	frh := &FakeRequestHandler{}
	mqttClient := mqtt.NewClient(mqtt.NewClientOptions())
	startRun(mqttClient, frh)

	assert.True(t, frh.catalogConfig.Debug)
	assert.Equal(t, "https://www.example.com", frh.catalogConfig.URL)
	assert.Equal(t, "gobbledygook", frh.catalogConfig.Token)
	assert.Equal(t, &towerapiworker.DefaultAPIWorker{}, frh.workHandler)
	assert.Equal(t, mqttClient, frh.mqttClient)
}
