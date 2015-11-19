package network

import (
	"encoding/binary"
	"net"

	"github.com/fatih/color"

	"github.com/degdb/degdb/protocol"
)

// NewConn creates a new Conn with the specified net.Conn.
func (s *Server) NewConn(c net.Conn) *Conn {
	return &Conn{
		Conn:   c,
		server: s,
	}
}

// Conn is a net.Conn with extensions.
type Conn struct {
	Peer   *protocol.Peer
	Closed bool

	// Notify channel for heartbeats
	peerRequest        chan bool
	peerRequestRetries int
	server             *Server

	net.Conn
}

// Send a message to the specified connection.
func (c *Conn) Send(m *protocol.Message) error {
	msg, err := m.Marshal()
	if err != nil {
		return err
	}
	packet := make([]byte, len(msg)+4)
	binary.BigEndian.PutUint32(packet, uint32(len(msg)))
	copy(packet[4:], msg)

	if _, err := c.Conn.Write(packet); err != nil {
		return err
	}
	return nil
}

func (c *Conn) PrettyID() string {
	var remote string
	if c.Peer != nil {
		remote = c.Peer.Id
	} else {
		remote = c.RemoteAddr().String()
	}
	return color.CyanString(remote)
}

func (c *Conn) Close() error {
	c.Closed = true
	return c.Conn.Close()
}
