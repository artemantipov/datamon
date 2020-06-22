package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

//Start run metrics http endpoint
func Start() {
	prometheus.MustRegister(NewDatamonCollector())
	fmt.Println("Starting metrics server")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}