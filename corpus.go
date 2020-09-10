package gsd

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/xerrors"

	"github.com/miclle/gsd/static"
)

// A Corpus holds all the package documentation
//
type Corpus struct {
	Path string

	// Packages is all packages cache
	Packages map[string]*Package

	// Tree is packages tree struct
	Tree Packages

	// pkgAPIInfo contains the information about which package API
	// features were added in which version of Go.
	pkgAPIInfo apiVersions

	EnablePrivateIndent bool

	store sync.Map
}

// NewCorpus return a new Corpus
func NewCorpus(path string) *Corpus {

	c := &Corpus{
		Path:     path,
		Packages: map[string]*Package{},
	}

	return c
}

// Export store documents
func (c *Corpus) Export() error {

	if err := c.Build(); err != nil {
		return err
	}

	var (
		buf        = new(bytes.Buffer)
		compressor = tar.NewWriter(buf)
	)

	c.store.Range(func(key, value interface{}) bool {

		// path := strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)
		// TODO(m) set path prefix with Corpus

		fmt.Printf("key: %#v\n", key)

		var (
			name    = key.(string)
			content = value.([]byte)
			hdr     = &tar.Header{
				Name: "docs/" + name,
				Mode: 0600,
				Size: int64(len(content)),
			}
		)

		if err := compressor.WriteHeader(hdr); err != nil {
			return false
		}

		if _, err := compressor.Write([]byte(content)); err != nil {
			return false
		}

		return true
	})

	if err := compressor.Close(); err != nil {
		return err
	}

	f, err := os.OpenFile("docs.tar", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(f)

	return err
}

// Watch server
func (c *Corpus) Watch(address string) (err error) {

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

	log.Printf("Listening and serving HTTP on %s\n", address)

	return http.ListenAndServe(address, mux)
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

// Build parse packages and render pages
func (c *Corpus) Build() (err error) {
	clearSyncMap(&c.store)

	fmt.Print("parse packages")

	if err = c.ParsePackages(); err != nil {
		fmt.Printf(" error: %s", err.Error())
		return err
	}
	fmt.Println(" success")

	fmt.Print("render pages")
	if err = c.RenderPages(); err != nil {
		fmt.Printf(" error: %s", err.Error())
		return err
	}
	fmt.Println(" success")

	return
}

// ParsePackages return packages
func (c *Corpus) ParsePackages() error {

	path := c.Path

	if !strings.HasSuffix(path, "/...") {
		path = strings.TrimSuffix(path, "/") + "/..."
	}

	out, err := exec.Command("go", "list", "-json", path).Output()
	if ee := (*exec.ExitError)(nil); xerrors.As(err, &ee) {
		return fmt.Errorf("go command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
	} else if err != nil {
		return err
	}

	c.Packages = map[string]*Package{}

	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var dpkg PackagePublic
		err := dec.Decode(&dpkg)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		c.Packages[dpkg.ImportPath] = &Package{
			Dir:         dpkg.Dir,
			Doc:         dpkg.Doc,
			Name:        dpkg.Name,
			ImportPath:  dpkg.ImportPath,
			Module:      dpkg.Module,
			Imports:     dpkg.Imports,
			Stale:       dpkg.Stale,
			StaleReason: dpkg.StaleReason,
		}
	}

	// parse packages tree
	for _, pkg := range c.Packages {
		if pkg.ImportPath == pkg.Module.Path {
			continue
		}

		var (
			seps       = strings.Split(strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path+"/"), "/")
			parentPath = pkg.ImportPath
		)

		for i := len(seps); i > 0; i-- {
			parentPath = strings.TrimSuffix(parentPath, "/"+seps[i-1])
			if parentPkg, exists := c.Packages[parentPath]; exists {
				pkg.ParentImportPath = parentPkg.ParentImportPath
				pkg.Parent = parentPkg
				parentPkg.SubPackages = append(parentPkg.SubPackages, pkg)
				break
			}
		}
	}

	c.Tree = []*Package{}

	for _, pkg := range c.Packages {
		if pkg.Parent == nil {
			c.Tree = append(c.Tree, pkg)
		}

		if err = pkg.Analyze(); err != nil {
			return err
		}
	}

	return nil
}

// RenderPages docss
func (c *Corpus) RenderPages() (err error) {

	// storing static assets
	c.storingStaticAssets()

	for _, pkg := range c.Packages {
		if err = c.storingPackage(pkg); err != nil {
			return err
		}
	}

	return
}

// storingStaticAssets storing static assets
func (c *Corpus) storingStaticAssets() {
	for filename, content := range static.Files {
		switch filepath.Ext(filename) {
		case ".html": // ignore golang template file
			continue
		}
		path := filepath.Join("_static", filename)
		c.store.Store(path, []byte(content))
	}
}

// storingPackage storing package, types and funcs pages
func (c *Corpus) storingPackage(pkg *Package) (err error) {

	var (
		path = pkg.ImportPath
		page = NewPage(c, pkg)
	)

	// generate package page
	{
		var buf bytes.Buffer
		if err = page.Render(&buf, PackagePage); err != nil {
			return
		}
		filename := fmt.Sprintf("%s/index.html", path)
		c.store.Store(filename, buf.Bytes())
	}

	// // generate packate types page
	// for _, t := range pkg.Types {
	// 	if IsExported(t.Name) == false {
	// 		break
	// 	}

	// 	// generate packate type page
	// 	{
	// 		page.Type = t
	// 		page.Title = t.Name

	// 		var buf bytes.Buffer
	// 		if err = page.Render(&buf, TypePage); err != nil {
	// 			return
	// 		}

	// 		filename := fmt.Sprintf("%s/%s.html", path, t.Name)
	// 		c.store.Store(filename, buf.Bytes())
	// 	}

	// 	// generate packate type's funcs & methods page
	// 	var funcs []*Func
	// 	funcs = append(funcs, t.Funcs...)
	// 	funcs = append(funcs, t.Methods...)

	// 	for _, fn := range funcs {
	// 		if IsExported(fn.Name) == false {
	// 			break
	// 		}

	// 		page.Func = fn
	// 		page.Title = fn.Name

	// 		var buf bytes.Buffer
	// 		if err = page.Render(&buf, FuncPage); err != nil {
	// 			return
	// 		}

	// 		filename := fmt.Sprintf("%s/%s.%s.html", path, t.Name, fn.Name)
	// 		c.store.Store(filename, buf.Bytes())
	// 	}
	// }

	return err
}

// --------------------------------------------------------------------

// IndentFilter indent filter
func (c *Corpus) IndentFilter(nodes interface{}) (result interface{}) {

	if c.EnablePrivateIndent {
		return nodes
	}

	switch nodes.(type) {

	case []*doc.Value:
		var values []*doc.Value
		for _, node := range nodes.([]*doc.Value) {
			var names []string
			for _, name := range node.Names {
				if IsExported(name) {
					names = append(names, name)
				}
				if len(names) > 0 {
					node.Names = names
					values = append(values, node)
				}
			}
		}
		return values

	case []*Type:
		var types []*Type
		for _, node := range nodes.([]*Type) {
			if IsExported(node.Name) {
				types = append(types, node)
			}
		}
		return types

	case []*Func:
		var funcs []*Func
		for _, node := range nodes.([]*Func) {
			if IsExported(node.Name) {
				funcs = append(funcs, node)
			}
		}
		return funcs

	case []*ast.Field:
		var fields []*ast.Field
		for _, node := range nodes.([]*ast.Field) {
			var idents []*ast.Ident
			for _, name := range node.Names {
				if IsExported(name.Name) {
					idents = append(idents, name)
				}
				if len(idents) > 0 {
					node.Names = idents
					fields = append(fields, node)
				}
			}
		}
		return fields

	default:
		return nodes
	}
}

// IsExported check first letter is capital
func IsExported(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func clearSyncMap(m *sync.Map) {
	m.Range(func(k, _ interface{}) bool {
		m.Delete(k)
		return true
	})
}
