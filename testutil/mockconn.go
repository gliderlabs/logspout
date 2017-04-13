package testutil

import (
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	connIsClosed = &mutexBool{
		mutex: &sync.Mutex{},
		state: false,
	}
	connIsReset = &mutexBool{
		mutex: &sync.Mutex{},
		state: false,
	}
)

type mutexBool struct {
	mutex *sync.Mutex
	state bool
}

// Conn facilitates testing by providing two connected ReadWriteClosers
// each of which can be used in place of a net.Conn
type Conn struct {
	Server *End
	Client *End
}

// Close closes server/client pipes
func (c *Conn) Close() error {
	if err := c.Server.Close(); err != nil {
		return err
	}
	if err := c.Client.Close(); err != nil {
		return err
	}
	return nil
}

// NewConn returns a new testutil.Conn
func NewConn() *Conn {
	// A connection consists of two pipes:
	// Client      |      Server
	//   writes   ===>  reads
	//    reads  <===   writes

	serverRead, clientWrite := net.Pipe()
	clientRead, serverWrite := net.Pipe()

	return &Conn{
		Server: &End{
			Reader: serverRead,
			Writer: serverWrite,
		},
		Client: &End{
			Reader: clientRead,
			Writer: clientWrite,
		},
	}
}

// End is one 'end' of a simulated connection.
type End struct {
	Reader net.Conn
	Writer net.Conn
}

// type End struct {
// 	Reader *io.PipeReader
// 	Writer *io.PipeWriter
// }

// CloseTest simulates closing a net.Conn
func (e End) CloseTest() {
	connIsClosed.mutex.Lock()
	connIsClosed.state = true
	connIsClosed.mutex.Unlock()
	return
}

// Close closes the Reader/Writer pipes
func (e End) Close() error {
	if err := e.Writer.Close(); err != nil {
		return err
	}
	if err := e.Reader.Close(); err != nil {
		return err
	}
	return nil
}

func (e End) Read(data []byte) (n int, err error) {
	e.Reader.SetReadDeadline(time.Now().Add(5 * time.Second))
	return e.Reader.Read(data)
}
func (e End) Write(data []byte) (n int, err error) {
	connIsClosed.mutex.Lock()
	defer connIsClosed.mutex.Unlock()
	if connIsClosed.state {
		connIsClosed.state = false
		return 0, &net.OpError{
			Op:     "write",
			Net:    e.RemoteAddr().Network(),
			Source: e.LocalAddr(),
			Addr:   e.RemoteAddr(),
			Err:    fmt.Errorf("write: broken pipe"),
		}
	}
	return e.Writer.Write(data)
}

// LocalAddr satisfies the net.Conn interface
func (e End) LocalAddr() net.Addr {
	return Addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

// RemoteAddr satisfies the net.Conn interface
func (e End) RemoteAddr() net.Addr {
	return Addr{
		NetworkString: "tcp",
		AddrString:    "127.0.0.1",
	}
}

// SetDeadline satisfies the net.Conn interface
func (e End) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline satisfies the net.Conn interface
func (e End) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline satisfies the net.Conn interface
func (e End) SetWriteDeadline(t time.Time) error { return nil }

// Addr is a fake network interface which implements the net.Addr interface
type Addr struct {
	NetworkString string
	AddrString    string
}

// Network satisfies the net.Addr interface
func (a Addr) Network() string {
	return a.NetworkString
}

// Network satisfies the net.Addr interface
func (a Addr) String() string {
	return a.AddrString
}
