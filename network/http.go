package network

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func (s *Server) initHTTPRouting() {
	s.mux = http.NewServeMux()
	s.HTTP = &http.Server{Handler: s.mux}
	s.mux.HandleFunc("/", s.handleNotFound)
}

// handleNotFound renders a 404 page for missing pages.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() == "/" {
		html := "<pre>"
		for _, e := range s.httpEndpoints {
			html += "<a href='" + e + "'>" + e + "</a>\n"
		}
		html += "</pre>"
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(html))
	} else {
		http.Error(w, fmt.Sprintf("degdb: file not found %s", r.URL), 404)
	}
}

func (s *Server) HTTPHandleFunc(route string, handler func(w http.ResponseWriter, r *http.Request)) {
	s.httpEndpoints = append(s.httpEndpoints, route)
	s.mux.HandleFunc(route, handler)
}
func (s *Server) HTTPHandle(route string, handler http.Handler) {
	s.httpEndpoints = append(s.httpEndpoints, route)
	s.mux.Handle(route, handler)
}

func (s *Server) listenHTTP() {
	if err := s.HTTP.Serve(s.listener); err != nil {
		s.Fatal(err)
	}
}

func (s *Server) handleHTTPConnection(initial []byte, conn *Conn) {
	s.listener.accept <- &httpConn{conn: conn, initial: initial}
}

// httpConn is an intermediate that can append some initial bytes to a Conn
type httpConn struct {
	conn    *Conn
	initial []byte
}

func (c *httpConn) Read(b []byte) (int, error) {
	i := len(c.initial)
	if i > 0 {
		copy(b, c.initial)
		c.initial = nil
	}
	n, err := c.conn.Read(b[i:])
	return n + i, err
}

func (c *httpConn) Close() error                       { return c.conn.Close() }
func (c *httpConn) LocalAddr() net.Addr                { return c.conn.LocalAddr() }
func (c *httpConn) Write(b []byte) (int, error)        { return c.conn.Write(b) }
func (c *httpConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }
func (c *httpConn) SetDeadline(t time.Time) error      { return c.conn.SetDeadline(t) }
func (c *httpConn) SetReadDeadline(t time.Time) error  { return c.conn.SetReadDeadline(t) }
func (c *httpConn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

type httpListener struct {
	addr   net.Addr
	accept chan *httpConn
}

func (h *httpListener) Accept() (net.Conn, error) {
	return <-h.accept, nil
}

func (h *httpListener) Close() error   { return nil }
func (h *httpListener) Addr() net.Addr { return h.addr }
