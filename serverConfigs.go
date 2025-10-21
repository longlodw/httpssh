package main

import (
	"encoding/json"
	"os"
)

type ServerConfig struct {
	SshListenAddr         string   `json:"ssh_listen_addr"`
	PrometheusListenAddr  string   `json:"prometheus_listen_addr"`
	KeyListenAddr         string   `json:"key_listen_addr"`
	JwtKeyPath            string   `json:"jwt_key_path"`
	SshKeyPath            string   `json:"ssh_key_path"`
	AuthorizationEndPoint string   `json:"authorization_end_point"`
	AllowedBackends       []string `json:"allowed_backends"`
	NoAuth                bool     `json:"no_auth"`
}

func LoadServerConfig(configPath string) (*ServerConfig, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var s ServerConfig
	json.Unmarshal(configData, &s)
	return &s, nil
}
