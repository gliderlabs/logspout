package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gliderlabs/logspout/cfg"
)

func init() {
	bindAddress := cfg.GetEnvDefault("HTTP_BIND_ADDRESS", "0.0.0.0")
	port := cfg.GetEnvDefault("PORT", cfg.GetEnvDefault("HTTP_PORT", "80"))
	Jobs.Register(&httpService{bindAddress, port}, "http")
}

type httpService struct {
	bindAddress string
	port        string
}

func (s *httpService) Name() string {
	return fmt.Sprintf("http[%s]:%s",
		strings.Join(HTTPHandlers.Names(), ","), s.port)
}

func (s *httpService) Setup() error {
	for name, handler := range HTTPHandlers.All() {
		h := handler()
		http.Handle("/"+name, h)
		http.Handle("/"+name+"/", h)
	}
	return nil
}

func (s *httpService) Run() error {
	return http.ListenAndServe(s.bindAddress+":"+s.port, nil)
}
