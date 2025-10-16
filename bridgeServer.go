package main

import (
	"go.uber.org/zap"
	"net"
	"net/http"
)

type bridgeServerConfig struct {
	ServerConfig
	BridgeAddr string `json:"bridge_addr"`
}

type bridgeHandler struct {
	logger      *zap.Logger
	promMetrics *prometheusMetrics
	bridgeAddr  string
}

func (bs *bridgeHandler) handle(w *http.ResponseWriter, r *http.Request) {
	bs.logger.Info("Handling new request", zap.String("remote_addr", r.RemoteAddr))
	bs.promMetrics.totalRequests.Inc()
	httpCon := net.Dial
}
