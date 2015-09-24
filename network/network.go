// Package network takes care of networking for degdb.
package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"

	"github.com/ccding/go-stun/stun"
	"github.com/degdb/degdb/protocol"
)

// Server handles all network traffic.
type Server struct {
	IP   string
	Port int
	NAT  stun.NATType

	handlers map[string]func(*protocol.Message)
}

// Conn is a net.Conn with extensions.
type Conn struct {
	net.Conn
}

// Send a message to the specified connection.
func (c *Conn) Send(m *protocol.Message) error {
	msg, err := m.Marshal()
	if err != nil {
		return err
	}
	msg = append(msg, '\n')
	_, err = c.Write(msg)
	if err != nil {
		return err
	}
	return nil
}

// NewServer creates a new server with routing information.
func NewServer() (*Server, error) {
	s := &Server{}
	nat, host, err := stun.NewClient().Discover()
	if err != nil {
		return nil, err
	}
	s.IP = host.IP()
	s.NAT = nat
	return s, nil
}

// Connect to another server. `addr` should be in the format "google.com:80".
func (s *Server) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	return s.handleConnection(&Conn{conn})
}

// Listen for incoming connections on the specified port.
func (s *Server) Listen(port int) error {
	s.Port = port
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	log.Printf("Listening: 0.0.0.0:%d, ip: %s", s.Port, s.IP)
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(&Conn{conn})
	}
}

// Handle registers a handler for a specific protobuf message type.
func (s *Server) Handle(typ string, f func(message *protocol.Message)) {
	s.handlers[typ] = f
}

func (s *Server) handleConnection(conn *Conn) error {
	var err error
	for {
		var buf []byte
		buf, err = bufio.NewReader(conn).ReadSlice('\n')
		if err != nil {
			return err
		}
		req := &protocol.Message{}
		if err = req.Unmarshal(buf); err != nil {
			break
		}
		typ := reflect.TypeOf(req).Name()
		log.Printf("Message: type = %s, len = %d", typ, len(buf))
		handler, ok := s.handlers[typ]
		if !ok {
			err = fmt.Errorf("no handler for message type %s", typ)
			break
		}
		handler(req)

	}
	if err != nil {
		log.Printf("Connection closed. Error: %s", err.Error())
	} else {
		log.Printf("Connection closed.")
	}
	return err
}
