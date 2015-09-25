// Package network takes care of networking for degdb.
package network

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
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

	HTTP *http.Server
	Mux  *http.ServeMux

	handlers map[string]func(*protocol.Message)
	listener *httpListener
	*log.Logger
}

// NewServer creates a new server with routing information.
func NewServer(log *log.Logger) (*Server, error) {
	s := &Server{Logger: log}

	// TODO(d4l3k): Fetch IP by talking to other nodes.
	nat, host, err := stun.NewClient().Discover()
	if err != nil {
		return nil, err
	}
	s.IP = host.IP()
	s.NAT = nat

	tcpAddr, err := net.ResolveTCPAddr("tcp", host.TransportAddr())
	if err != nil {
		return nil, err
	}
	s.listener = &httpListener{
		addr:   tcpAddr,
		accept: make(chan *httpConn, 10),
	}
	s.Mux = http.NewServeMux()
	s.HTTP = &http.Server{Handler: s.Mux}
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
	go s.listenHTTP()
	s.Printf("Listening: 0.0.0.0:%d, ip: %s", s.Port, s.IP)
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
		header := make([]byte, 4)
		_, err = conn.Read(header)
		if err != nil {
			break
		}
		if string(header) == "GET " {
			s.Printf("Incoming HTTP connection.")
			s.handleHTTPConnection(header, conn)
			return nil
		}
		length := binary.BigEndian.Uint32(header)
		if length > 10000000 {
			err = fmt.Errorf("Packet larger than 10MB! len = %d", length)
			break
		}
		buf := make([]byte, length)
		_, err = conn.Read(buf)
		if err != nil {
			break
		}

		s.Printf("message %s", buf)
		req := &protocol.Message{}
		if err = req.Unmarshal(buf); err != nil {
			break
		}
		typ := reflect.TypeOf(req).Name()
		s.Printf("Message: type = %s, len = %d", typ, len(buf))
		handler, ok := s.handlers[typ]
		if !ok {
			err = fmt.Errorf("no handler for message type %s", typ)
			break
		}
		handler(req)

	}
	if err != nil {
		s.Printf("Connection closed. Error: %s", err.Error())
	} else {
		s.Printf("Connection closed.")
	}
	conn.Close()
	return err
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
	packet := make([]byte, len(msg)+4)
	binary.BigEndian.PutUint32(packet, uint32(len(msg)))
	copy(packet[4:], msg)
	_, err = c.Write(packet)
	if err != nil {
		return err
	}
	return nil
}
