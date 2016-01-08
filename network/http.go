package network

import (
	"html/template"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/degdb/degdb/network/http"
)

var staticBox = rice.MustFindBox("../static/")

var (
	IndexTemplate = template.Must(template.New("").Parse(staticBox.MustString("common/index.html")))
	ErrorTemplate = template.Must(template.New("").Parse(staticBox.MustString("common/error.html")))
)

func (s *Server) initHTTPRouting() {
	s.mux = http.NewServeMux()
	s.HTTP = &http.Server{Handler: s.mux}
	s.mux.HandleFunc("/", s.handleNotFound)
}

// handleNotFound renders a 404 page for missing pages.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	url := r.URL.String()
	w.Header().Add("Content-Type", "text/html")
	if url == "/" {
		var urls []string
		for _, path := range s.httpEndpoints {
			urls = append(urls, path)
		}
		sort.Strings(urls)
		IndexTemplate.Execute(w, customhttp.DirectoryListing{"/", urls})
	} else {
		w.WriteHeader(404)
		ErrorTemplate.Execute(w, "File Not Found (404) "+url)
	}
}

func (s *Server) handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), 500)
}

func (s *Server) HTTPHandleFunc(route string, handler func(w http.ResponseWriter, r *http.Request)) {
	s.httpEndpoints = append(s.httpEndpoints, route)
	s.mux.HandleFunc(route, handler)
}

func (s *Server) HTTPHandle(route string, handler http.Handler) {
	s.httpEndpoints = append(s.httpEndpoints, route)
	s.mux.Handle(route, handler)
}

func (s *Server) listenHTTP(addr net.Addr) {
	s.listener.addr = addr
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
