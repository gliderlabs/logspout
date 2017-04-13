package testutil

import "net"

// MockTransport allows us to dial our test listener
type MockTransport struct {
	Listener *LocalTCPServer
}

// Dial always returns the client from our test listener
func (mt MockTransport) Dial(addr string, opt map[string]string) (net.Conn, error) {
	return mt.Listener.MockConn.Client, nil
}
