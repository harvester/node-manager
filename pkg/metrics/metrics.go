package metrics

import (
	"net/http"
	"time"

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
	metricServer := &http.Server{
		Addr:              MetricPort,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := metricServer.ListenAndServe(); err != nil {
		logrus.Fatalf("failed to start metrics server: %s", err)
	}
}
