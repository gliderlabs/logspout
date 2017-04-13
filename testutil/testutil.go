package testutil

import (
	"net"
	"sync"
)

// Test constants
const (
	CloseOnMsgIdx = 5
	MaxMsgCount   = 10
)

// LocalTCPServer tcp server for testing specific network errors
type LocalTCPServer struct {
	lnmu     sync.RWMutex
	MockConn *Conn
	net.Listener
}

// Accept returns server side of the LocalTCPServer MockConn
func (ls *LocalTCPServer) Accept() (*End, error) {
	return ls.MockConn.Server, nil
}

// Teardown locks and tears down a LocalTCPServer
func (ls *LocalTCPServer) Teardown() error {
	ls.lnmu.Lock()
	if ls.Listener != nil {
		ls.Listener.Close()
		ls.Listener = nil
	}
	ls.lnmu.Unlock()
	return nil
}

// NewLocalTCPServer return a new LocalTCPServer
func NewLocalTCPServer() (*LocalTCPServer, error) {
	ln, err := newLocalListener()
	if err != nil {
		return nil, err
	}
	return &LocalTCPServer{Listener: ln, MockConn: NewConn()}, nil
}

func newLocalListener() (net.Listener, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return ln, nil
}

// Dial returns the client side of our MockConn
func Dial(ls *LocalTCPServer) net.Conn {
	return ls.MockConn.Client
}

// AcceptAndCloseConn opens the listener side of a server, reads data and closes the connection after CloseOnMsgIdx
func AcceptAndCloseConn(ls *LocalTCPServer, datac chan []byte, errc chan error) {
	defer func() {
		close(datac)
		close(errc)
	}()
	c, err := ls.Accept()
	if err != nil {
		errc <- err
		return
	}

	count := 0
	for {
		switch count {
		case MaxMsgCount - CloseOnMsgIdx:
			c.CloseTest()
			readConn(count, c, datac)
			count++
		case MaxMsgCount:
			return
		default:
			readConn(count, c, datac)
			count++
		}
	}
}

func readConn(count int, c net.Conn, ch chan []byte) error {
	b := make([]byte, 256)
	_, err := c.Read(b)
	if err != nil {
		return err
	}
	// Simulate real-life network drop
	if count != MaxMsgCount-CloseOnMsgIdx {
		ch <- b
	}
	return nil
}
