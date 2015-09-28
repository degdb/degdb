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
	"strings"
	"sync"

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

	handlers map[string]protocolHandler
	listener *httpListener
	*log.Logger
}

type protocolHandler func(conn *Conn, msg *protocol.Message)

var (
	stunOnce sync.Once
	stunWG   sync.WaitGroup
	nat      stun.NATType
	host     *stun.Host
)

func init() {
	stunWG.Add(1)
}

func stunResults() {
	stunOnce.Do(func() {
		var err error
		// TODO(d4l3k): Fetch IP by talking to other nodes.
		stunc := stun.NewClient()
		nat, host, err = stunc.Discover()
		if err != nil {
			log.Fatal(err)
		}
		stunWG.Done()
	})
	stunWG.Wait()
}

// NewServer creates a new server with routing information.
func NewServer(log *log.Logger) (*Server, error) {
	s := &Server{
		Logger:   log,
		handlers: make(map[string]protocolHandler),
	}

	stunResults()

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
	tcpConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	conn := &Conn{tcpConn}

	conn.Send(&protocol.Message{
		Message: &protocol.Message_PeerRequest{
			PeerRequest: &protocol.PeerRequest{
				Limit: -1,
			},
		},
	})

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
		addr := conn.RemoteAddr()
		s.Printf("New connection from %s", addr)
		go s.handleConnection(&Conn{conn})
	}
}

// Handle registers a handler for a specific protobuf message type.
func (s *Server) Handle(typ string, f protocolHandler) {
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

		req := &protocol.Message{}
		if err = req.Unmarshal(buf); err != nil {
			break
		}
		s.Printf("Message: <- %s, %#v", conn.RemoteAddr(), req)
		rawType := reflect.TypeOf(req.GetMessage()).Elem().Name()
		typ := strings.TrimPrefix(rawType, "Message_")
		handler, ok := s.handlers[typ]
		if !ok {
			err = fmt.Errorf("no handler for message type %s", typ)
			break
		}
		handler(conn, req)

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
	if _, err := c.Write(packet); err != nil {
		return err
	}
	return nil
}
