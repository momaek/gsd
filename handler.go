package gsd

import (
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

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

	// parse packages
	if err := c.ParsePackages(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

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
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("document not found"))
		return
	}

	var page = NewPage(c, pkg)
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
