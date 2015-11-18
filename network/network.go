// Package network takes care of networking for degdb.
package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ccding/go-stun/stun"
	"github.com/degdb/degdb/protocol"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/spaolacci/murmur3"
)

// Server handles all network traffic.
type Server struct {
	IP   string
	Port int
	NAT  stun.NATType

	HTTP          *http.Server
	mux           *http.ServeMux
	httpEndpoints []string

	Peers map[string]*Conn

	handlers map[string]protocolHandler
	listener *httpListener
	*log.Logger
}

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

func (s *Server) handlePeerNotify(conn *Conn, msg *protocol.Message) {
	conn.peerRequest <- true
	s.Printf("PeerNotify %+v", conn)
	peers := msg.GetPeerNotify().Peers
	for _, peer := range peers {
		if _, ok := s.Peers[peer.Id]; !ok {
			s.Peers[peer.Id] = nil
			s.Connect(peer.Id)
		}
	}
}

func (s *Server) handlePeerRequest(conn *Conn, msg *protocol.Message) {
	// TODO(d4l3k): Handle keyspace check.
	req := msg.GetPeerRequest()

	var peers []*protocol.Peer
	for id, v := range s.Peers {
		if conn.Peer.Id == id || v == nil {
			continue
		}
		peers = append(peers, v.Peer)
		if req.Limit > 0 && int32(len(peers)) >= req.Limit {
			break
		}
	}
	err := conn.Send(&protocol.Message{Message: &protocol.Message_PeerNotify{
		PeerNotify: &protocol.PeerNotify{
			Peers: peers,
		}}})
	if err != nil {
		log.Printf("ERR sending PeerNotify: %s", err)
	}
}

func (s *Server) handleHandshake(conn *Conn, msg *protocol.Message) {
	handshake := msg.GetHandshake()
	conn.Peer = handshake.GetSender()
	if peer := s.Peers[conn.Peer.Id]; peer != nil {
		s.Printf("Ignoring duplicate peer %s.", conn.PrettyID())
		if err := conn.Close(); err != nil && err != io.EOF {
			s.Printf("ERR closing connection %s", err)
		}
		return
	}
	s.Peers[conn.Peer.Id] = conn
	s.Print(color.GreenString("New peer %s", conn.PrettyID()))
	if !handshake.Response {
		if err := s.sendHandshake(conn, true); err != nil {
			s.Printf("ERR sendHandshake %s", err)
		}
	} else {
		if err := s.sendPeerRequest(conn); err != nil {
			s.Printf("ERR sendPeerRequest %s", err)
		}
	}
	go s.connHeartbeat(conn)
}

func (s *Server) connHeartbeat(conn *Conn) {
	ticker := time.NewTicker(time.Second * 60)
	for range ticker.C {
		err := s.sendPeerRequest(conn)
		if err == io.EOF {
			ticker.Stop()
		} else if err != nil {
			s.Printf("ERR sendPeerRequest %s", err)
		}
	}
}

func (s *Server) sendPeerRequest(conn *Conn) error {
	msg := &protocol.Message{Message: &protocol.Message_PeerRequest{
		PeerRequest: &protocol.PeerRequest{
			Limit: -1,
			//Keyspace: s.LocalPeer().Keyspace,
		}}}
	conn.peerRequest = make(chan bool, 1)
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()
	go func() {
		select {
		case <-conn.peerRequest:
		case <-timeout:
			id := conn.Peer.Id
			s.Printf(color.RedString("Peer timed out! %s %+v", conn.PrettyID(), conn))
			delete(s.Peers, id)
			conn.Close()
		}
	}()
	if err := conn.Send(msg); err != nil {
		return err
	}
	return nil
}

func (s *Server) sendHandshake(conn *Conn, response bool) error {
	return conn.Send(&protocol.Message{
		Message: &protocol.Message_Handshake{
			Handshake: &protocol.Handshake{
				Response: response,
				Sender:   s.LocalPeer(),
			},
		},
	})
}

// Connect to another server. `addr` should be in the format "google.com:80".
func (s *Server) Connect(addr string) error {
	tcpConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	conn := s.NewConn(tcpConn)

	if err := s.sendHandshake(conn, false); err != nil {
		return err
	}

	return s.handleConnection(conn)
}

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
		addr := conn.RemoteAddr()
		s.Printf("New connection from %s", addr)
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
	if conn.Peer != nil {
		delete(s.Peers, conn.Peer.Id)
	}
	return err
}

// Conn is a net.Conn with extensions.
type Conn struct {
	Peer *protocol.Peer

	// Notify channel for heartbeats
	peerRequest chan bool
	server      *Server

	net.Conn
}

// NewConn creates a new Conn with the specified net.Conn.
func (s *Server) NewConn(c net.Conn) *Conn {
	return &Conn{
		Conn:   c,
		server: s,
	}
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
	c.server.Printf("Message: -> %s, %+v", c.PrettyID(), m.GetMessage())
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
