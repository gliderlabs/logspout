package healthcheck

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.HTTPHandlers.Register(HealthCheck, "health")
}

// HealthCheck returns a http.Handler for the health check
func HealthCheck() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Healthy!\n"))
	})
	return r
}
