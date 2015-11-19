package network

import (
	"io"
	"time"

	"github.com/fatih/color"

	"github.com/degdb/degdb/protocol"
)

func (s *Server) handlePeerNotify(conn *Conn, msg *protocol.Message) {
	conn.peerRequest <- true
	peers := msg.GetPeerNotify().Peers
	for _, peer := range peers {
		if _, ok := s.Peers[peer.Id]; !ok {
			s.Peers[peer.Id] = nil
			if err := s.Connect(peer.Id); err != nil {
				s.Printf("ERR failed to connect to peer %s", err)
			}
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
	wrapper := &protocol.Message{Message: &protocol.Message_PeerNotify{
		PeerNotify: &protocol.PeerNotify{
			Peers: peers,
		}}}
	if err := conn.Send(wrapper); err != nil {
		s.Printf("ERR sending PeerNotify: %s", err)
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
	for _ = range ticker.C {
		if conn.Closed {
			ticker.Stop()
			break
		}
		conn.peerRequestRetries = 0
		err := s.sendPeerRequest(conn)
		if err == io.EOF {
			ticker.Stop()
			break
		} else if err != nil {
			s.Printf("ERR sendPeerRequest %s", err)
		}
	}
}

func (s *Server) sendPeerRequest(conn *Conn) error {
	msg := &protocol.Message{Message: &protocol.Message_PeerRequest{
		PeerRequest: &protocol.PeerRequest{
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
			msg := color.RedString("Peer timed out! %s %+v", conn.PrettyID(), conn)
			conn.peerRequestRetries++
			if conn.peerRequestRetries >= 3 {
				s.Printf(msg)
				delete(s.Peers, conn.Peer.Id)
				conn.Close()
			} else {
				s.Printf("%s. Retrying...", msg)
				s.sendPeerRequest(conn)
			}
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
