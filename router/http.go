package router

import (
	"fmt"
	"net/http"
	"strings"
)

func init() {
	port := getopt("PORT", getopt("HTTP_PORT", "80"))
	Jobs.Register(&httpService{port}, "http")
}

type httpService struct {
	port string
}

func (s *httpService) Name() string {
	return fmt.Sprintf("http[%s]:%s",
		strings.Join(HttpHandlers.Names(), ","), s.port)
}

func (s *httpService) Setup() error {
	for name, handler := range HttpHandlers.All() {
		h := handler()
		http.Handle("/"+name, h)
		http.Handle("/"+name+"/", h)
	}
	return nil
}

func (s *httpService) Run() error {
	return http.ListenAndServe(":"+s.port, nil)
}
