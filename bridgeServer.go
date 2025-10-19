package main

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type tokenCacheEntry struct {
	exp         time.Time
	signedToken string
}

type bridgeHandler struct {
	logger      *zap.Logger
	promMetrics *PrometheusMetrics
	secretKey   crypto.Signer
	aud         *url.URL
	tokensCache sync.Map
	iss         string
}

func newBridgeHandler(logger *zap.Logger, promMetrics *PrometheusMetrics, secretKey crypto.Signer, aud *url.URL, iss string) *bridgeHandler {
	return &bridgeHandler{
		logger:      logger,
		promMetrics: promMetrics,
		secretKey:   secretKey,
		aud:         aud,
		iss:         iss,
	}
}

func (bs *bridgeHandler) makeToken(sub string) string {
	iat := time.Now()
	if cacheEntry, ok := bs.tokensCache.Load(sub); !ok || iat.After(cacheEntry.(tokenCacheEntry).exp) {
		jtiBytes := make([]byte, 32)
		rand.Read(jtiBytes)
		exp := iat.Add(time.Hour)
		claims := jwt.MapClaims{
			"sub": sub,
			"aud": bs.aud.String(),
			"iss": bs.iss,
			"iat": iat.Unix(),
			"exp": exp.Unix(),
			"nbf": iat.Unix(),
			"jti": base64.RawStdEncoding.EncodeToString(jtiBytes),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		signedToken, err := token.SignedString(bs.secretKey)
		if err != nil {
			bs.logger.Fatal("Failed to sign token", zap.Error(err))
		}
		bs.tokensCache.Store(sub, tokenCacheEntry{
			exp:         exp,
			signedToken: signedToken,
		})
		return signedToken
	} else {
		return cacheEntry.(tokenCacheEntry).signedToken
	}
}

func (bs *bridgeHandler) direct(r *http.Request) {
	bs.logger.Info("Handling new request", zap.String("remote_addr", r.RemoteAddr))
	bs.promMetrics.totalRequests.Inc()
	r.URL.Scheme = bs.aud.Scheme
	r.URL.Host = bs.aud.Host
	r.Header.Set("X-Identity", bs.makeToken(r.RemoteAddr))
}

func initBridgeServer(bs *bridgeHandler) *http.ServeMux {
	muxBridge := http.NewServeMux()
	proxy := httputil.NewSingleHostReverseProxy(bs.aud)
	proxy.Director = bs.direct
	muxBridge.Handle("/", proxy)
	return muxBridge
}

func MakeBridgeServerMux(logger *zap.Logger, promMetrics *PrometheusMetrics, secretKey crypto.Signer, iss string, urls []*url.URL) map[string]*http.ServeMux {
	results := make(map[string]*http.ServeMux)
	for _, e := range urls {
		results[e.String()] = initBridgeServer(newBridgeHandler(logger, promMetrics, secretKey, e, iss))
	}
	return results
}
