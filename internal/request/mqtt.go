package request

import (
	"bytes"
	context "context"
	"crypto/tls"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/rhc_catalog_worker/internal/catalogtask"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/common"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/logger"
	"github.com/RedHatInsights/rhc_catalog_worker/internal/towerapiworker"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

func connect(clientID string, mqttURL string) (mqtt.Client, error) {
	opts := createClientOptions(clientID, mqttURL)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}

func createClientOptions(clientID string, mqttURL string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttURL)
	opts.SetClientID(clientID)
	// TODO: This is for testing remove it once we dock with RHC
	if strings.HasPrefix(mqttURL, "ssl://") {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}
	return opts
}

func makeMQTTClient(config *common.CatalogConfig) mqtt.Client {
	mqttClient, err := connect("tower_client_"+config.GUID, config.MQTTURL)
	if err != nil {
		log.Errorf("Error connecting to MQTT Server %v", err)
		return nil
	}
	log.Infof("Connected to MQTT Server %s", config.MQTTURL)
	return mqttClient
}

type mqttListener struct {
	mqttClient mqtt.Client
}

func (lis mqttListener) stop() {
	lis.mqttClient.Disconnect(10)
	log.Info("MQTT client stopped")
}

func startMQTTListener(config *common.CatalogConfig, wh towerapiworker.WorkHandler, shutdown chan struct{}) (listener, error) {
	mqttClient := makeMQTTClient(config)
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
		nextCtx := logger.CtxWithLoggerID(ctx, strconv.Itoa(counter))
		go processRequest(nextCtx, m.URL, config, wh, catalogtask.MakeCatalogTask(nextCtx, m.URL), &defaultPageWriterFactory{}, shutdown)
	}

	if token := mqttClient.Subscribe(topic, 0, fn); token.Wait() && token.Error() != nil {
		log.Errorf("Encountered Token Error %v", token.Error())
	}
	return &mqttListener{mqttClient: mqttClient}, nil
}
