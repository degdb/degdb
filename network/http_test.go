package network

import (
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

func TestHTTPProxy(t *testing.T) {
	s, err := NewServer(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		go wg.Done()
		if err := s.Listen(); err != nil {
			t.Fatal(err)
		}
	}()
	wg.Wait()

	// Test endpoint
	expectedResp := "foo"
	s.HTTPHandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedResp))
	})
	baseURL := "http://" + s.LocalID()
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
