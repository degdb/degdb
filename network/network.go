// Package network takes care of networking for degdb.
package network

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ccding/go-stun/stun"
	"github.com/dustin/go-humanize"
	"github.com/spaolacci/murmur3"

	"github.com/degdb/degdb/protocol"
)

type protocolHandler func(conn *Conn, msg *protocol.Message)

var (
	stunOnce sync.Once
	stunWG   sync.WaitGroup
	nat      stun.NATType
	host     string
)

func init() {
	stunWG.Add(1)
}

func stunResults() {
	stunOnce.Do(func() {
		// TODO(d4l3k): Fetch IP by talking to other nodes.
		stunc := stun.NewClient()
		natType, stunHost, err := stunc.Discover()
		if err != nil {
			log.Fatal(err)
		}
		nat = natType
		host = stunHost.IP()
		stunWG.Done()
	})
	stunWG.Wait()
}

// Server handles all network traffic.
type Server struct {
	IP   string
	Port int
	NAT  stun.NATType

	HTTP          *http.Server
	mux           *http.ServeMux
	httpEndpoints []string

	Peers     map[string]*Conn
	peersLock sync.RWMutex

	handlers map[string]protocolHandler
	listener *httpListener
	*log.Logger
}

// NewServer creates a new server with routing information.
func NewServer(log *log.Logger, port int) (*Server, error) {
	s := &Server{
		Logger:   log,
		Port:     port,
		Peers:    make(map[string]*Conn),
		handlers: make(map[string]protocolHandler),
	}

	stunResults()

	s.IP = host
	s.NAT = nat

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", s.IP, s.Port))
	if err != nil {
		return nil, err
	}
	s.listener = &httpListener{
		addr:   tcpAddr,
		accept: make(chan *httpConn, 10),
	}

	s.initHTTPRouting()

	// Handlers
	s.Handle("Handshake", s.handleHandshake)
	s.Handle("PeerRequest", s.handlePeerRequest)
	s.Handle("PeerNotify", s.handlePeerNotify)

	return s, nil
}

// Connect to another server. `addr` should be in the format "google.com:80".
func (s *Server) Connect(addr string) error {
	netConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	tcpConn := netConn.(*net.TCPConn)
	tcpConn.SetKeepAlive(true)

	conn := s.NewConn(tcpConn)

	if err := s.sendHandshake(conn, false); err != nil {
		return err
	}

	go s.handleConnection(conn)
	return nil
}

// Listen for incoming connections on the specified port.
func (s *Server) Listen() error {
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
		s.Printf("New connection from %s", conn.RemoteAddr())
		go s.handleConnection(s.NewConn(conn))
	}
}

// Handle registers a handler for a specific protobuf message type.
func (s *Server) Handle(typ string, f protocolHandler) {
	s.handlers[typ] = f
}

// Broadcast sends a message to all peers with that have the hash in their keyspace.
func (s *Server) Broadcast(hash *uint64, msg *protocol.Message) error {
	alreadySentTo := make(map[uint64]bool)
	if msg.Gossip {
		for _, to := range msg.SentTo {
			alreadySentTo[to] = true
		}
	}
	sentTo := []uint64{murmur3.Sum64([]byte(s.LocalPeer().Id))}
	var toPeers []*Conn
	for _, peer := range s.Peers {
		peerHash := murmur3.Sum64([]byte(peer.Peer.Id))
		if (hash == nil || peer.Peer.GetKeyspace().Includes(*hash)) && !alreadySentTo[peerHash] {
			sentTo = append(sentTo, peerHash)
			toPeers = append(toPeers, peer)
		}
	}
	if msg.Gossip {
		msg.SentTo = append(msg.SentTo, sentTo...)
	}
	for _, peer := range toPeers {
		s.Printf("Broadcasting to %s", peer.Peer.Id)
		if err := peer.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleConnection(conn *Conn) error {
	var err error
	for {
		header := make([]byte, 4)
		_, err = conn.Read(header)
		if err != nil {
			break
		}
		if string(header) == "GET " || string(header) == "POST" {
			s.Printf("Incoming HTTP connection.")
			s.handleHTTPConnection(header, conn)
			return nil
		}
		length := binary.BigEndian.Uint32(header)
		if length > 10000000 {
			err = fmt.Errorf("Packet larger than 10MB! len = %s", humanize.SI(float64(length), "B"))
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
		s.Printf("Message: <- %s, %+v", conn.PrettyID(), req.GetMessage())
		if req.ResponseTo != 0 {
			if c, ok := conn.expectedMessages[req.ResponseTo]; ok {
				c <- req
				continue
			} else {
				err = fmt.Errorf("response sent to invalid request %d", req.ResponseTo)
				break
			}
		}
		rawType := reflect.TypeOf(req.GetMessage()).Elem().Name()
		typ := strings.TrimPrefix(rawType, "Message_")
		handler, ok := s.handlers[typ]
		if !ok {
			err = fmt.Errorf("no handler for message type %s", typ)
			break
		}
		go handler(conn, req)
	}
	s.Printf("Connection closed. %s", err)
	conn.Close()
	if conn.Peer != nil {
		delete(s.Peers, conn.Peer.Id)
	}
	return err
}

// LocalPeer returns a peer object of the current server.
func (s *Server) LocalPeer() *protocol.Peer {
	id := fmt.Sprintf("%s:%d", s.IP, s.Port)
	center := murmur3.Sum64([]byte(id))
	keyspace := &protocol.Keyspace{
		Start: center - math.MaxUint64/4,
		End:   center + math.MaxUint64/4,
	}
	return &protocol.Peer{
		Id:       id,
		Keyspace: keyspace,
	}
}
