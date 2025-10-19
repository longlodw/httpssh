package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type SshServer struct {
	logger      *zap.Logger
	promMetrics *PrometheusMetrics
	hosts       map[string]chan<- net.Conn
	confg       *ssh.ServerConfig
	listener    net.Listener
	closed      bool
}

func NewSshServer(logger *zap.Logger, promMetrics *PrometheusMetrics, hosts map[string]chan<- net.Conn, confg *ssh.ServerConfig, listener net.Listener) *SshServer {
	return &SshServer{
		logger:      logger,
		promMetrics: promMetrics,
		hosts:       hosts,
		confg:       confg,
		listener:    listener,
	}
}

func (sh *SshServer) Serve() {
	for !sh.closed {
		conn, err := sh.listener.Accept()
		if err != nil {
			if sh.closed {
				return
			}
			sh.logger.Error("Error acceptomg tcp conn", zap.Error(err))
		} else {
			go sh.handleConn(conn)
		}
	}
}

func (sh *SshServer) ShutDown() error {
	sh.closed = true
	err := sh.listener.Close()
	for _, v := range sh.hosts {
		close(v)
	}
	return err
}

func (sh *SshServer) handleSshNewChan(ch ssh.NewChannel, sshConn *ssh.ServerConn) {
	switch ch.ChannelType() {
	case "direct-tcpip":
		extraData := ch.ExtraData()
		targetUrl, _, err := parseExtraData(extraData)
		if err != nil {
			sh.logger.Info("Invalid connection extra data", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())), zap.String("extraData", base64.RawStdEncoding.EncodeToString(extraData)))
			ch.Reject(ssh.ConnectionFailed, "Failed to parse connection extra data")
			return
		}
		targetUrlStr := targetUrl.String()
		acceptChan, ok := sh.hosts[targetUrlStr]
		if !ok {
			sh.logger.Info("Target url not allowed", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())), zap.String("target", targetUrlStr))
			ch.Reject(ssh.Prohibited, "Target url not allowed")
			return
		}
		acceptedCh, reqs, err := ch.Accept()
		if err != nil {
			sh.logger.Error("Failed to accept channel", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())))
			return
		}
		ssh.DiscardRequests(reqs)
		acceptChan <- &sshChanConn{
			Channel: acceptedCh,
			sshConn: sshConn,
		}
	default:
		ch.Reject(ssh.UnknownChannelType, "unsupported channel type")
		sh.logger.Info("Rejecting unknown channel type", zap.String("channel_type", ch.ChannelType()))
	}
}

func (sh *SshServer) handleConn(c net.Conn) {
	defer c.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(c, sh.confg)
	if err != nil {
		sh.logger.Error("Failed to create ssh server connection", zap.Error(err))
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)
	for ch := range chans {
		go sh.handleSshNewChan(ch, sshConn)
	}
}

func parseExtraData(extraData []byte) (*url.URL, *url.URL, error) {
	targetHost, extraData, err := parseLengthPrefixString(extraData)
	if err != nil {
		return nil, nil, err
	}
	if len(extraData) < 4 {
		return nil, nil, io.ErrShortBuffer
	}
	targetPort := binary.BigEndian.Uint32(extraData)
	originIP, extraData, err := parseLengthPrefixString(extraData)
	if err != nil {
		return nil, nil, err
	}
	if len(extraData) < 4 {
		return nil, nil, io.ErrShortBuffer
	}
	originPort := binary.BigEndian.Uint32(extraData)
	targetUrl, err := url.Parse(fmt.Sprintf("%s:%d", targetHost, targetPort))
	if err != nil {
		return nil, nil, err
	}
	originUrl, err := url.Parse(fmt.Sprintf("%s:%d", originIP, originPort))
	if err != nil {
		return nil, nil, err
	}
	return targetUrl, originUrl, nil
}

func parseLengthPrefixString(b []byte) (string, []byte, error) {
	if len(b) < 4 {
		return "", b, io.ErrShortBuffer
	}
	length := binary.BigEndian.Uint32(b)
	b = b[4:]
	if uint32(len(b)) < length {
		return "", b, io.ErrShortBuffer
	}
	b = b[:length]
	return string(b), b[length:], nil
}
