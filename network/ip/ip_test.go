package ip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServer() (string, *httptest.Server) {
	mux := http.NewServeMux()
	mux.HandleFunc("/good", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("127.0.0.1"))
	})
	mux.HandleFunc("/mal", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("8.8.8.8"))
	})
	mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("err"))
	})
	s := httptest.NewServer(mux)
	addr := "http://" + s.Listener.Addr().String()
	return addr, s
}

func TestIP(t *testing.T) {
	t.Parallel()

	addr, s := newServer()
	defer s.Close()

	testData := []struct {
		servers []string
		want    string
	}{
		{
			[]string{
				addr + "/good",
				addr + "/err",
			},
			"127.0.0.1",
		},
		{
			[]string{
				addr + "/mal",
				addr + "/good",
				addr + "/good",
			},
			"127.0.0.1",
		},
	}
	for i, td := range testData {
		ipServers = td.servers
		ip, err := IP()
		if err != nil {
			t.Error(err)
		}
		if ip != td.want {
			t.Errorf("%d. IP() = %s; not %s", i, ip, td.want)
		}
	}
}
