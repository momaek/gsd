package gsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/xerrors"

	"github.com/miclle/gsd/static"
	"github.com/miclle/gsd/util"
)

// Version info
const Version = "0.0.1"

// A Corpus holds all the package documentation
//
type Corpus struct {
	// source code path
	Path string

	// exclude paths
	Excludes []string

	// output docs path
	Output string

	// http server address
	Addr string

	// Packages is all packages cache
	Packages map[string]*Package

	// Tree is packages tree struct
	// - a
	// 	- a-a
	// - b
	//
	Tree Packages

	// pkgAPIInfo contains the information about which package API
	// features were added in which version of Go.
	pkgAPIInfo apiVersions

	EnablePrivateIndent bool

	excludeMatcher Matcher
}

// Config with corpus
type Config struct {
	// source code path
	Path string

	// exclude paths
	Excludes []string

	// output docs path
	Output string

	// http server address
	Addr string
}

// NewCorpus return a new Corpus
func NewCorpus(config *Config) (*Corpus, error) {

	corpus := &Corpus{
		Path:     config.Path,
		Packages: map[string]*Package{},
		Output:   config.Output,
		Addr:     config.Addr,
	}

	if corpus.Output == "" {
		corpus.Output = "docs"
	}

	directory, err := filepath.Abs(config.Path)
	if err != nil {
		return nil, err
	}

	log.Println("document source code path:", directory)
	log.Println("the documents export path:", corpus.Output)

	config.Excludes = append(config.Excludes, corpus.Output)

	matcher, err := ParseMatchers(config.Excludes)
	if err != nil {
		return nil, fmt.Errorf("Error parse match role %s", err.Error())
	}

	corpus.excludeMatcher = multiMatcher{defaultExcludeMatcher, matcher}

	return corpus, nil
}

// Export store documents
func (c *Corpus) Export() (err error) {

	if err := c.ParsePackages(); err != nil {
		return err
	}

	// write static asset files
	for filename, content := range static.Files {
		if filepath.Ext(filename) == ".html" {
			continue
		}

		path := filepath.Join("docs/_static", filepath.Dir(filename))

		fmt.Println("write static asset file:", filename)

		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}

		if err = ioutil.WriteFile("docs/_static/"+filename, []byte(content), 0644); err != nil {
			return
		}
	}

	// write documents
	for _, pkg := range c.Packages {
		if err := c.renderPackage(pkg); err != nil {
			return err
		}
	}

	return err
}

// renderPackage storing package, types and funcs pages
func (c *Corpus) renderPackage(pkg *Package) (err error) {

	// path := strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)
	path := pkg.ImportPath
	path = fmt.Sprintf("docs/%s", path)

	// auto mkdir dirs
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return
	}

	// generate package info page
	page := NewPage(c, pkg)

	{
		var buf bytes.Buffer
		if err = page.Render(&buf, PackagePage); err != nil {
			return
		}

		filename := fmt.Sprintf("%s/index.html", path)

		fmt.Printf("write package %s doc: %s\n", pkg.Name, filename)

		if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			return
		}
	}

	// generate packate types page
	for _, t := range pkg.Types {
		if IsExported(t.Name) == false {
			break
		}

		page.Type = t
		page.Title = t.Name

		var buf bytes.Buffer
		if err = page.Render(&buf, TypePage); err != nil {
			return err
		}

		filename := fmt.Sprintf("%s/%s.html", path, t.Name)
		fmt.Printf("write type %s doc: %s\n", t.Name, filename)
		if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			return err
		}

		// generate packate type's funcs & methods page
		var funcs []*Func
		funcs = append(funcs, t.Funcs...)
		funcs = append(funcs, t.Methods...)

		for _, fn := range funcs {
			if IsExported(fn.Name) == false {
				break
			}

			page.Func = fn
			page.Title = fn.Name

			var buf bytes.Buffer
			if err = page.Render(&buf, FuncPage); err != nil {
				return
			}

			filename := fmt.Sprintf("%s/%s.%s.html", path, t.Name, fn.Name)
			fmt.Printf("write func %s.%s doc: %s\n", t.Name, fn.Name, filename)
			if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
				return err
			}
		}
	}

	return err
}

// Watch server
func (c *Corpus) Watch(address string) (err error) {

	// parse packages first
	if err := c.ParsePackages(); err != nil {
		log.Fatal(err)
	}

	// file system notify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	changes := make(chan string)

	go watch(c.Path, watcher, changes, c.excludeMatcher)
	go c.broadcast(changes)

	server := &http.Server{
		Addr:    address,
		Handler: c.ServeMux(),
	}

	// cleanup
	server.RegisterOnShutdown(func() {
		log.Println("server.RegisterOnShutdown")

		watcher.Close()
	})

	log.Printf("Listening and serving HTTP on %s\n", address)

	// open browser
	// time.AfterFunc(time.Second*2, func() {
	// 	if err := util.OpenBrowser(address); err != nil {
	// 		log.Println(err.Error())
	// 	}
	// })

	// start webserver
	err = server.ListenAndServe()

	return
}

func (c *Corpus) broadcast(in <-chan string) {

	debounced := util.Debouncer(100 * time.Millisecond)

	for e := range in {
		log.Println("file changed:", e)

		debounced(func() {

			log.Print("parse packages ")

			if err := c.ParsePackages(); err != nil {
				log.Println("error", err)
			}

			log.Println("success")
		})
	}
}

// ParsePackages return packages
func (c *Corpus) ParsePackages() error {

	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = c.Path

	out, err := cmd.Output()
	if ee := (*exec.ExitError)(nil); xerrors.As(err, &ee) {
		return fmt.Errorf("go list command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
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
