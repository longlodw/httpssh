package httpssh

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshChanConn struct {
	ssh.Channel
	sshConn *ssh.ServerConn
}

func (shc *sshChanConn) LocalAddr() net.Addr {
	return shc.LocalAddr()
}

func (shc *sshChanConn) RemoteAddr() net.Addr {
	return shc.RemoteAddr()
}

func (shc *sshChanConn) SetDeadline(_ time.Time) error {
	return nil
}

func (shc *sshChanConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (shc *sshChanConn) SetWriteDeadline(_ time.Time) error {
	return nil
}
