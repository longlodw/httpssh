package httpssh

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshChanConnAddr struct {
	username string
	net.Addr
}

func (shca *sshChanConnAddr) String() string {
	return fmt.Sprintf("%s@%s", shca.username, shca.Addr.String())
}

type sshChanConn struct {
	ssh.Channel
	sshConn *ssh.ServerConn
}

func (shc *sshChanConn) LocalAddr() net.Addr {
	return shc.sshConn.LocalAddr()
}

func (shc *sshChanConn) RemoteAddr() net.Addr {
	return &sshChanConnAddr{
		username: shc.sshConn.User(),
		Addr:     shc.sshConn.RemoteAddr(),
	}
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
