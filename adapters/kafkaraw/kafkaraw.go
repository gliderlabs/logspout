package kafkaraw

import (
	"bytes"
	"errors"
	"log"
	"os"
	"text/template"

	kafka "github.com/Shopify/sarama"
	"github.com/gliderlabs/logspout/router"
	"github.com/gliderlabs/logspout/utils"
	"strings"
	"time"
)

func init() {
	router.AdapterFactories.Register(NewKafkaRawAdapter, "kafkaraw")
}

var topic string

func NewKafkaRawAdapter(route *router.Route) (router.LogAdapter, error) {
	topic = os.Getenv("TOPIC")
	if topic == "" {
		err := errors.New("not found kafka topic")
		return nil, err
	}
	transport, found := router.AdapterTransports.Lookup(route.AdapterTransport("udp"))
	if !found {
		return nil, errors.New("bad transport: " + route.Adapter)
	}
	_ = transport
	config := kafka.NewConfig()
	config.Producer.Compression = kafka.CompressionGZIP
	producer, err := kafka.NewSyncProducer([]string{route.Address}, config)
	if err != nil {
		return nil, err
	}
	tmplStr := "{{.Data}}\n"
	if os.Getenv("RAW_FORMAT") != "" {
		tmplStr = os.Getenv("RAW_FORMAT")
	}
	tmpl, err := template.New("raw").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	return &RawAdapter{
		route:    route,
		producer: producer,
		tmpl:     tmpl,
	}, nil
}

type RawAdapter struct {
	producer kafka.SyncProducer
	route    *router.Route
	tmpl     *template.Template
}

func (a *RawAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("raw:", err)
			return
		}
		if cn := utils.M1[message.Container.Name]; cn != "" {
			t := time.Unix(time.Now().Unix(), 0)
			timestr := t.Format("2006-01-02T15:04:05")
			logmsg := strings.Replace(string(timestr), "\"", "", -1) + " " +
				utils.UserId + " " +
				utils.ClusterId + " " +
				utils.UUID + " " +
				utils.IP + " " +
				utils.Hostname + " " +
				cn + " " +
				buf.String()
			msg := &kafka.ProducerMessage{Topic: "test", Value: kafka.StringEncoder(logmsg)}
			partition, offset, err := a.producer.SendMessage(msg)
			_, _, _ = partition, offset, err
		}
	}

}
