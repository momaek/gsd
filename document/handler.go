package document

import (
	"bytes"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	autocorrect "github.com/huacnlee/go-auto-correct"
	"github.com/miclle/gsd/static"
)

// ServeMux return an HTTP request multiplexer.
func (c *Corpus) ServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, req *http.Request) {
		content, exists := static.Files["favicon.ico"]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("assets not found"))
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write([]byte(content))
	})

	mux.HandleFunc("/_static/", c.StaticHandler)
	mux.HandleFunc("/", c.DocumentHandler)

	return mux
}

// StaticHandler serve static assets
func (c *Corpus) StaticHandler(w http.ResponseWriter, req *http.Request) {

	var (
		filename        = strings.TrimPrefix(req.URL.Path, "/_static/")
		content, exists = static.Files[filename]
	)

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("assets not found"))
		return
	}

	var ctype string
	if ctypes, haveType := w.Header()["Content-Type"]; !haveType {
		if ctype = mime.TypeByExtension(filepath.Ext(filename)); ctype == "" {
			ctype = http.DetectContentType([]byte(content))
		}
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	w.Header().Set("Content-Type", ctype)
	w.Write([]byte(content))
}

// DocumentHandler serve documents
// The "/" pattern matches everything, so we need to check that we're at the root here.
func (c *Corpus) DocumentHandler(w http.ResponseWriter, req *http.Request) {

	// logging
	log.Printf("%s %s\n", req.RemoteAddr, req.URL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var (
		path       = strings.Trim(req.URL.Path, "/")
		importPath = path
		typeName   string
		funcName   string
	)

	// type or method page
	if strings.HasSuffix(path, ".html") {
		importPath = filepath.Dir(path)
		slices := strings.Split(filepath.Base(path), ".")
		typeName = slices[0]
		if len(slices) > 2 {
			funcName = slices[1]
		}
	}

	// get package
	pkg, exists := c.Packages[importPath]
	if !exists {
		c.ReadmeHandler(w, req)
		return
	}

	var page = NewPage(c)
	page.Package = pkg
	page.Title = pkg.Name
	page.PageType = PackagePage

	for _, t := range pkg.Types {
		if typeName != "" && typeName == t.Name {
			page.Title = t.Name
			page.Type = t
			page.PageType = TypePage

			var funcs []*Func
			funcs = append(funcs, t.Funcs...)
			funcs = append(funcs, t.Methods...)

			for _, fn := range funcs {
				if funcName != "" && funcName == fn.Name {
					page.Func = fn
					page.Title = fn.Name
					page.PageType = FuncPage
				}
			}
		}
	}

	// render page
	if err := page.Render(w, page.PageType); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

// ReadmeHandler handle the README.md file
func (c *Corpus) ReadmeHandler(w http.ResponseWriter, req *http.Request) {

	var path = strings.Trim(req.URL.Path, "/")

	var filename string

	for _, name := range ReadmeFileNames {
		if dir := filepath.Join(c.Path, path, name); fileExists(dir) {
			filename = dir
			break
		}
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var buf bytes.Buffer
	if err := md.Convert(data, &buf); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	cjk := autocorrect.Format(buf.String())

	// render page
	page := NewPage(c)
	if err := page.RenderBody(w, []byte(cjk)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
