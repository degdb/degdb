package network

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestHTTPProxy(t *testing.T) {
	s, err := NewServer(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := s.Listen(); err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// Test endpoint
	expectedResp := "foo"
	s.HTTPHandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedResp))
	})
	baseURL := "http://localhost:" + strconv.Itoa(s.Port)
	url := baseURL + "/test"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != expectedResp {
		t.Errorf("http.Get(%s) = %s; not %s", url, body, expectedResp)
	}
}
