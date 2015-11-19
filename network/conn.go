package network

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"time"

	"github.com/fatih/color"

	"github.com/degdb/degdb/protocol"
)

var Timeout = errors.New("request timed-out")

// NewConn creates a new Conn with the specified net.Conn.
func (s *Server) NewConn(c net.Conn) *Conn {
	return &Conn{
		Conn:             c,
		server:           s,
		expectedMessages: make(map[uint64]chan *protocol.Message),
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
	expectedMessages   map[uint64]chan *protocol.Message

	net.Conn
}

// Send a message on the specified connection. Consider Request.
func (c *Conn) Send(m *protocol.Message) error {
	msg, err := m.Marshal()
	if err != nil {
		return err
	}
	packet := make([]byte, len(msg)+4)
	binary.BigEndian.PutUint32(packet, uint32(len(msg)))
	copy(packet[4:], msg)
	_, err = c.Conn.Write(packet)
	return err
}

// Request sends a message on a connection and waits for a response.
// Returns error network.Timeout if no response in 10 seconds.
func (c *Conn) Request(m *protocol.Message) (*protocol.Message, error) {
	m.Id = uint64(rand.Int63())
	m.ResponseRequired = true
	if err := c.Send(m); err != nil {
		return nil, err
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()
	resp := make(chan *protocol.Message, 1)
	c.expectedMessages[m.Id] = resp

	var msg *protocol.Message
	var err error
	select {
	case msg = <-resp:
	case <-timeout:
		err = Timeout
	}
	delete(c.expectedMessages, m.Id)
	return msg, err
}

// RespondTo sends `resp` as a response to the request `to`.
func (c *Conn) RespondTo(to *protocol.Message, resp *protocol.Message) error {
	resp.ResponseTo = to.Id
	return c.Send(resp)
}

// Close closes the connection and sets Closed to true.
func (c *Conn) Close() error {
	c.Closed = true
	return c.Conn.Close()
}

// PrettyID returns a terminal colored format of the connection ID.
func (c *Conn) PrettyID() string {
	var remote string
	if c.Peer != nil {
		remote = c.Peer.Id
	} else {
		remote = c.RemoteAddr().String()
	}
	return color.CyanString(remote)
}
