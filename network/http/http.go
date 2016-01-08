// Package customhttp is a reimplementation of http.FileServer that supports custom HTML templates for errors and directory listings.
package customhttp

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"
	"strings"
)

var (
	defaultIndexTemplate = template.Must(template.New("").Parse(`<h1>{{.Path}}</h1>
<pre>
{{range .Files}}<a href="{{.}}">{{.}}</a>
{{end}}</pre>`))
	default404Template   = template.Must(template.New("").Parse(`File Not Found {{.}}`))
	defaultErrorTemplate = template.Must(template.New("").Parse(`<h1>{{.}}</h1>`))
)

type FileServer struct {
	// IndexTemplate is a html template that accepts an array of files.
	IndexTemplate *template.Template

	// Error404Template is a html template that accepts the URL path.
	Error404Template *template.Template

	// ErrorTemplate is a html template that accepts the error message.
	ErrorTemplate *template.Template

	// PathPrefix is the prefix to apply to the displayed URL.
	PathPrefix string

	root http.FileSystem
}

func (f *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	file, err := f.root.Open(upath)
	if err != nil {
		f.handle404(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		f.handle404(w, r)
		return
	}
	if info.IsDir() {
		files, err := file.Readdir(-1)
		if err != nil {
			f.handleError(w, err.Error(), 404)
			return
		}
		paths := []string{".."}
		for _, file := range files {
			path := file.Name()
			if file.IsDir() {
				path += "/"
			}
			paths = append(paths, path)
		}
		sort.Strings(paths)
		w.Header().Add("Content-Type", "text/html")
		f.IndexTemplate.Execute(w, DirectoryListing{f.PathPrefix + upath, paths})
	} else {
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

type DirectoryListing struct {
	Path  string
	Files []string
}

func (f *FileServer) handle404(w http.ResponseWriter, r *http.Request) {
	buf := bytes.NewBuffer(nil)
	f.Error404Template.Execute(buf, f.PathPrefix+r.URL.Path)
	f.handleError(w, string(buf.Bytes()), 404)
}
func (f *FileServer) handleError(w http.ResponseWriter, err string, code int) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(code)
	f.ErrorTemplate.Execute(w, err)
}

func NewFileServer(root http.FileSystem) *FileServer {
	f := &FileServer{
		root:             root,
		IndexTemplate:    defaultIndexTemplate,
		Error404Template: default404Template,
		ErrorTemplate:    defaultErrorTemplate,
		PathPrefix:       "",
	}
	return f
}
