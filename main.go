package main

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/longlodw/lazyiterate"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func loadSshPrivateKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}

func loadRawPrivateKey(path string) (crypto.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ed25519.PrivateKey(key), nil
}

func loadRawPublicKey(path string) (crypto.PublicKey, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(key), nil
}

func main() {
	configPathPtr := flag.String("config", "config.json", "config json")
	flag.Parse()
	serverConfig, err := LoadServerConfig(*configPathPtr)
	if err != nil {
		panic(err)
	}
	hostPrivateKey, err := loadSshPrivateKey(serverConfig.SshPrivateKeyPath)
	if err != nil {
		panic(err)
	}
	urlPath, err := url.Parse(serverConfig.AuthorizationEndPoint)
	if err != nil {
		panic(err)
	}
	sshConfig := ssh.ServerConfig{
		NoClientAuth: serverConfig.NoAuth,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			q := urlPath.Query()
			q.Add("username", conn.User())
			q.Add("password", base64.RawURLEncoding.EncodeToString(password))
			urlPath.RawQuery = q.Encode()
			res, err := http.Get(urlPath.String())
			if err != nil {
				return nil, err
			}
			if res.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("Status %v: %s", res.StatusCode, res.Status)
			}
			return nil, nil
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			q := urlPath.Query()
			q.Add("username", conn.User())
			q.Add("key", base64.RawURLEncoding.EncodeToString(key.Marshal()))
			urlPath.RawQuery = q.Encode()
			res, err := http.Get(urlPath.String())
			if err != nil {
				return nil, err
			}
			if res.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("Status %v: %s", res.StatusCode, res.Status)
			}
			return nil, nil
		},
	}
	sshConfig.AddHostKey(hostPrivateKey)
	promMetrics := NewPrometheusMetrics()
	promMetrics.Register()
	metricsServerMux := InitMetricsServer()
	logger := zap.Must(zap.NewProduction())
	defer logger.Sync()
	allowedHosts := slices.Collect(lazyiterate.Map(slices.Values(serverConfig.AllowedBackends), func(host string) *url.URL {
		urlObj, err := url.Parse(host)
		if err != nil {
			panic(err)
		}
		return urlObj
	}))
	inChans := make(map[string]chan<- net.Conn)
	outChans := make(map[string]<-chan net.Conn)
	for _, allowedHost := range allowedHosts {
		connChan := make(chan net.Conn)
		hostString := allowedHost.String()
		inChans[hostString] = connChan
		outChans[hostString] = connChan
	}
	chanListeners := MakeChanListeners(outChans)
	listener, err := net.Listen("tcp", serverConfig.SshListenAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	sshServer := NewSshServer(logger, promMetrics, inChans, &sshConfig, listener)
	jwtPrivateKey, err := loadRawPrivateKey(serverConfig.JwtPrivateKeyPath)
	if err != nil {
		panic(err)
	}
	jwtPublicKey, err := loadRawPublicKey(serverConfig.JwtPublicKeyPath)
	if err != nil {
		panic(err)
	}
	bridgeServerMux := MakeBridgeServerMux(logger, promMetrics, jwtPrivateKey, serverConfig.KeyListenAddr, allowedHosts)
	go sshServer.Serve()
	bridgeServers := []*http.Server{}
	for _, allowedHost := range allowedHosts {
		hostString := allowedHost.String()
		bridgeServer := &http.Server{
			Handler: bridgeServerMux[hostString],
		}
		bridgeServers = append(bridgeServers, bridgeServer)
		go func(bs *http.Server) {
			if err := bs.Serve(chanListeners[hostString]); err != nil {
				logger.Error("bridge server error", zap.Error(err))
			}
		}(bridgeServer)
	}
	metricsServer := &http.Server{
		Addr:    serverConfig.PrometheusListenAddr,
		Handler: metricsServerMux,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			logger.Error("Metrics server error", zap.Error(err))
		}
	}()
	keyServerMux := http.NewServeMux()
	keyServerMux.HandleFunc("/key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Println(w, base64.RawStdEncoding.EncodeToString(jwtPublicKey.([]byte)))
	})
	keyServer := &http.Server{
		Addr:    serverConfig.KeyListenAddr,
		Handler: keyServerMux,
	}
	go func() {
		if err := keyServer.ListenAndServe(); err != nil {
			logger.Error("Key server error", zap.Error(err))
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	sshServer.ShutDown()
	keyServer.Shutdown(context.Background())
	metricsServer.Shutdown(context.Background())
	for _, s := range bridgeServers {
		s.Shutdown(context.Background())
	}
}
