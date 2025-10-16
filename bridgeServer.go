package main

import (
	"net"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	"github.com/golang-jwt/jwt/v5"
)

type bridgeServerConfig struct {
	ServerConfig
	BridgeAddr string `json:"bridge_addr"`
}

type bridgeHandler struct {
	logger      *zap.Logger
	promMetrics *prometheusMetrics
	sub string
	secretKey []byte
	aud  *url.URL
	iss string
}

func (bs *bridgeHandler) direct(r *http.Request) {
	bs.logger.Info("Handling new request", zap.String("remote_addr", r.RemoteAddr))
	bs.promMetrics.totalRequests.Inc()
	r.URL.Scheme = bs.aud.Scheme
	r.URL.Host = bs.aud.Host
	r.Header.Set("X-Forwarded-Host", r.Host)
	claims := jwt.MapClaim{
		"sub": bs.sub,
		"aud": bs.aud.String()
	}
	r.Header.Set("X-Public-Key", bs.sshPK)
}

func initBridgeServer(cfg *bridgeServerConfig, logger *zap.Logger, promMetrics *prometheusMetrics) *http.Server {
	
	muxBridge := http.NewServeMux()
	muxBridge.HandleFunc()
}
