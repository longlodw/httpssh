package httpssh

import (
	"io"
	"net"
	"sync/atomic"
)

type chanListener struct {
	channel  <-chan net.Conn
	closing  chan struct{}
	isClosed atomic.Bool
}

func (cl *chanListener) Accept() (net.Conn, error) {
	if cl.isClosed.Load() {
		return nil, io.ErrClosedPipe
	}
	select {
	case conn := <-cl.channel:
		return conn, nil
	case _ = <-cl.closing:
		return nil, io.ErrClosedPipe
	}
}

func (cl *chanListener) Addr() net.Addr {
	return nil
}

func (cl *chanListener) Close() error {
	cl.isClosed.Store(true)
	cl.closing <- struct{}{}
	close(cl.closing)
	return nil
}

func NewChanListener(channel <-chan net.Conn) net.Listener {
	return &chanListener{
		channel: channel,
		closing: make(chan struct{}, 1),
	}
}

func MakeChanListeners(hosts map[string]<-chan net.Conn) map[string]net.Listener {
	result := make(map[string]net.Listener)
	for k, v := range hosts {
		result[k] = NewChanListener(v)
	}
	return result
}
