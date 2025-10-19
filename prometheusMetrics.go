package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusMetrics struct {
	activeConnections prometheus.Gauge
	totalConnections  prometheus.Counter
	totalRequests     prometheus.Counter
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		activeConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wstcp_active_connections",
			Help: "Number of currently active WebSocket connections",
		}),
		totalConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wstcp_connections_total",
			Help: "Total number of WebSocket connections handled",
		}),
		totalRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wstcp_requests_total",
			Help: "Total number of requests processed",
		}),
	}
}

func (pm *PrometheusMetrics) Register() {
	prometheus.MustRegister(pm.activeConnections, pm.totalConnections, pm.totalRequests)
}

func InitMetricsServer() *http.ServeMux {
	muxMetrics := http.NewServeMux()
	muxMetrics.Handle("/metrics", promhttp.Handler())
	return muxMetrics
}
