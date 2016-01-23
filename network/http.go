package network

import (
	"html/template"
	"net"
	"net/http"
	"sort"

	"github.com/GeertJohan/go.rice"
	"github.com/degdb/degdb/network/customhttp"
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
	s.listener.accept <- &httpConn{Conn: conn, initial: initial}
}

// httpConn is an intermediate that can append some initial bytes to a Conn
type httpConn struct {
	*Conn
	initial []byte
}

func (c *httpConn) Read(b []byte) (int, error) {
	i := len(c.initial)
	if i > 0 {
		copy(b, c.initial)
		c.initial = nil
	}
	n, err := c.Conn.Read(b[i:])
	return n + i, err
}

type httpListener struct {
	addr   net.Addr
	accept chan *httpConn
}

func (h *httpListener) Accept() (net.Conn, error) {
	return <-h.accept, nil
}

func (h *httpListener) Close() error   { return nil }
func (h *httpListener) Addr() net.Addr { return h.addr }
