package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

// Pump message send counter
var pumpMsgSend *prometheus.CounterVec

func init() {
	pumpMsgSend = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pump_msg_send",
			Help: "Number of message sent",
		},
		[]string{"container_id"},
	)
	prometheus.MustRegister(pumpMsgSend)
	HttpHandlers.Register(func() http.Handler { return prometheus.Handler() }, "metrics")
}
