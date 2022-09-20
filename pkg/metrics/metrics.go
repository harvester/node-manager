package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	MetricPort = ":9526"
	MetricPath = "/metrics"
)

var (
	KsmdUtilizationGV = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ksmd_utilization",
		Help: "ksmd utilization of cpu in second",
	}, []string{"nodename"})
)

func Run() {
	logrus.Info("starting metrics server")
	prometheus.MustRegister(KsmdUtilizationGV)

	http.Handle(MetricPath, promhttp.Handler())
	if err := http.ListenAndServe(MetricPort, nil); err != nil {
		logrus.Fatalf("failed to start metrics server: %s", err)
	}
}
