package kafka

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gliderlabs/logspout/router"
	"gopkg.in/Shopify/sarama.v1"
)

func init() {
	router.AdapterFactories.Register(NewKafkaAdapter, "kafka")
}

type KafkaAdapter struct {
	route    *router.Route
	brokers  []string
	topic    string
	config   *sarama.Config
	client   sarama.Client
	producer sarama.AsyncProducer
	tmpl     *template.Template
}

func NewKafkaAdapter(route *router.Route) (router.LogAdapter, error) {
	brokers, topic, err := parseKafkaAddress(route.Address)
	if err != nil {
		return nil, err
	}

	var tmpl *template.Template
	if text := os.Getenv("KAFKA_TEMPLATE"); text != "" {
		tmpl, err = template.New("kafka").Parse(text)
		if err != nil {
			return nil, err
		}
	}

	config := buildConfig(route.Options)
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("couldn't create Kafka client: %v", err)
	}

	producer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("couldn't create Kafka producer: %v", err)
	}

	return &KafkaAdapter{
		route:    route,
		brokers:  brokers,
		topic:    topic,
		config:   config,
		client:   client,
		producer: producer,
		tmpl:     tmpl,
	}, nil
}

func (a *KafkaAdapter) Stream(logstream chan *router.Message) {
	for rm := range logstream {
		message, err := a.formatMessage(rm)
		if err != nil {
			log.Println("kafka:", err)
			a.route.Close()
			break
		}

		a.producer.Input() <- message
	}

	a.producer.Close()
	a.client.Close()
}

func buildConfig(options map[string]string) *sarama.Config {
	config := sarama.NewConfig()
	config.ClientID = "logspout"
	config.Producer.Flush.Frequency = 1 * time.Second
	config.Producer.RequiredAcks = sarama.WaitForLocal

	if opt := options["compression.codec"]; opt != "" {
		switch opt {
		case "gzip":
			config.Producer.Compression = sarama.CompressionGZIP
		case "snappy":
			config.Producer.Compression = sarama.CompressionSnappy
		}
	}

	return config
}

func (a *KafkaAdapter) formatMessage(message *router.Message) (*sarama.ProducerMessage, error) {
	var encoder sarama.Encoder
	if a.tmpl != nil {
		var w bytes.Buffer
		if err := a.tmpl.Execute(&w, message); err != nil {
			return nil, err
		}
		encoder = sarama.ByteEncoder(w.Bytes())
	} else {
		encoder = sarama.StringEncoder(message.Data)
	}

	return &sarama.ProducerMessage{
		Topic: a.topic,
		Value: encoder,
	}, nil
}

func parseKafkaAddress(routeAddress string) ([]string, string, error) {
	if !strings.Contains(routeAddress, "/") {
		return []string{}, "", errors.New("the route address didn't specify the Kafka topic")
	}

	slash := strings.Index(routeAddress, "/")
	topic := routeAddress[slash+1:]
	addrs := strings.Split(routeAddress[:slash], ",")
	return addrs, topic, nil
}
