package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusMetrics struct {
	activeConnections prometheus.Gauge
	totalConnections  prometheus.Counter
	totalRequests     prometheus.Counter
}

func newPrometheusMetrics() *prometheusMetrics {
	return &prometheusMetrics{
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

func (pm *prometheusMetrics) register() {
	prometheus.MustRegister(pm.activeConnections, pm.totalConnections, pm.totalRequests)
}

func initMetricsServer(cfg *ServerConfig) *http.Server {
	muxMetrics := http.NewServeMux()
	muxMetrics.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      muxMetrics,
		ReadTimeout:  time.Duration(cfg.ReadTimeout),
		WriteTimeout: time.Duration(cfg.WriteTimeout),
		IdleTimeout:  time.Duration(cfg.IdleTimeout),
	}
	return metricsSrv
}
