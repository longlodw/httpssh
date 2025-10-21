package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"

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
		targetAddr, originAddr, err := parseExtraData(extraData)
		if err != nil {
			sh.logger.Info("Invalid connection extra data", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())), zap.String("extraData", base64.RawStdEncoding.EncodeToString(extraData)), zap.Error(err))
			ch.Reject(ssh.ConnectionFailed, "Failed to parse connection extra data")
			return
		}
		sh.logger.Info("connection attempt", zap.String("from", originAddr), zap.String("to", targetAddr))
		targetUrlHost := targetAddr
		acceptChan, ok := sh.hosts[targetUrlHost]
		if !ok {
			sh.logger.Info("Target url not allowed", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())), zap.String("target", targetUrlHost))
			ch.Reject(ssh.Prohibited, "Target url not allowed")
			return
		}
		acceptedCh, reqs, err := ch.Accept()
		if err != nil {
			sh.logger.Error("Failed to accept channel", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())))
			return
		}
		sh.logger.Info("Accepted channel", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())))
		go ssh.DiscardRequests(reqs)
		acceptChan <- &sshChanConn{
			Channel: acceptedCh,
			sshConn: sshConn,
		}
		sh.logger.Info("Handling channel", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())))
	default:
		ch.Reject(ssh.UnknownChannelType, "unsupported channel type")
		sh.logger.Info("Rejecting unknown channel type", zap.String("channel_type", ch.ChannelType()))
	}
}

func (sh *SshServer) handleConn(c net.Conn) {
	sh.logger.Info("Tcp connection", zap.String("address", c.RemoteAddr().String()))
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
	sh.logger.Info("No more channels", zap.String("sessionId", base64.RawStdEncoding.EncodeToString(sshConn.SessionID())))
}

func parseExtraData(extraData []byte) (string, string, error) {
	targetHost, extraData, err := parseLengthPrefixString(extraData)
	if err != nil {
		return "", "", err
	}
	targetPort, extraData, err := parseUint32(extraData)
	if err != nil {
		return "", "", err
	}
	originIP, extraData, err := parseLengthPrefixString(extraData)
	if err != nil {
		return "", "", err
	}
	originPort, extraData, err := parseUint32(extraData)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s:%d", targetHost, targetPort), fmt.Sprintf("%s:%d", originIP, originPort), nil
}

func parseUint32(b []byte) (uint32, []byte, error) {
	if len(b) < 4 {
		return 0, b, io.ErrUnexpectedEOF
	}
	return binary.BigEndian.Uint32(b), b[4:], nil
}

func parseLengthPrefixString(b []byte) (string, []byte, error) {
	if len(b) < 4 {
		return "", b, io.ErrUnexpectedEOF
	}
	length := binary.BigEndian.Uint32(b)
	b = b[4:]
	if uint32(len(b)) < length {
		return "", b, io.ErrUnexpectedEOF
	}
	return string(b[:length]), b[length:], nil
}
