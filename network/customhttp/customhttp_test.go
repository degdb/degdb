package customhttp

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type mockFileSystem struct{}

func (f *mockFileSystem) Open(name string) (http.File, error) {
	dirFoo := &mockFile{
		name: "foo",
		body: []byte("foobody"),
	}
	switch name {
	case "/dir/foo":
		return dirFoo, nil
	case "/dir/":
		return &mockFile{
			name: "/dir/",
			files: []os.FileInfo{
				dirFoo,
				&mockFile{
					name:  "bar",
					files: []os.FileInfo{&mockFile{}},
				},
			},
		}, nil
	}
	return nil, errors.New("file not found")
}

type mockFile struct {
	name   string
	body   []byte
	files  []os.FileInfo
	offset int64
}

// http.File
func (f *mockFile) Readdir(count int) ([]os.FileInfo, error) { return f.files, nil }
func (f *mockFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.offset = offset
	case 1:
		f.offset += offset
	case 2:
		f.offset = int64(len(f.body)) - offset
	}
	return f.offset, nil
}
func (f *mockFile) Stat() (os.FileInfo, error) { return f, nil }
func (f *mockFile) Close() error               { return nil }
func (f *mockFile) Read(p []byte) (n int, err error) {
	return bytes.NewBuffer(f.body[f.offset:]).Read(p)
}

// os.FileInfo
func (f *mockFile) Name() string       { return f.name }
func (f *mockFile) Size() int64        { return int64(len(f.body)) }
func (f *mockFile) Mode() os.FileMode  { return 0 }
func (f *mockFile) ModTime() time.Time { return time.Now() }
func (f *mockFile) IsDir() bool        { return len(f.files) > 0 }
func (f *mockFile) Sys() interface{}   { return nil }

func TestFileServer(t *testing.T) {
	t.Parallel()

	sys := &mockFileSystem{}
	s := NewFileServer(sys)

	// /404
	resp := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "404", nil)
	if err != nil {
		t.Fatal(err)
	}
	s.ServeHTTP(resp, req)
	if resp.Code != 404 {
		t.Errorf("/404 should return 404 code; got %d", resp.Code)
	}
	want := `<h1>File Not Found (404) /404</h1>`
	got := resp.Body.String()
	if got != want {
		t.Errorf("/404 should return %s; got %s", want, got)
	}

	// /dir/
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/dir/", nil)
	if err != nil {
		t.Fatal(err)
	}
	s.ServeHTTP(resp, req)
	if resp.Code == 202 {
		t.Errorf("/dir/ should return 200 code; got %d", resp.Code)
	}
	want = `<h1>/dir/</h1>
<pre>
<a href="..">..</a>
<a href="bar/">bar/</a>
<a href="foo">foo</a>
</pre>`
	got = resp.Body.String()
	if got != want {
		t.Errorf("/dir/ should return %s; got %s", want, got)
	}

	// /dir/foo
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/dir/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	s.ServeHTTP(resp, req)
	if resp.Code == 202 {
		t.Errorf("/dir/foo should return 200 code; got %d", resp.Code)
	}
	want = `foobody`
	got = resp.Body.String()
	if got != want {
		t.Errorf("/dir/foo should return %s; got %s", want, got)
	}
}
