package network

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
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
	testData := []struct {
		path     string
		status   int
		contains string
	}{
		{
			"/test",
			200,
			expectedResp,
		},
		{
			"/",
			200,
			`<a href="/test">/test</a>`,
		},
		{
			"/404",
			404,
			"File Not Found",
		},
	}

	for i, td := range testData {
		baseURL := "http://localhost:" + strconv.Itoa(s.Port)
		url := baseURL + td.path
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != td.status {
			t.Errorf("%d. http.Get(%#v) status code = %d; not %d", i, url, resp.StatusCode, td.status)
		}
		if !strings.Contains(string(body), td.contains) {
			t.Errorf("%d. http.Get(%#v) = %#v; missing %#v", i, url, body, td.contains)
		}
	}
}
