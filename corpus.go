package gsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/miclle/gsd/static"
	"golang.org/x/xerrors"
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
}

// NewCorpus return a new Corpus
func NewCorpus() *Corpus {

	c := &Corpus{
		Packages: map[string]*Package{},
	}

	return c
}

// Init initializes Corpus, once options on Corpus are set.
// It must be called before any subsequent method calls.
func (c *Corpus) Init() (err error) {

	err = c.ParsePackages()
	if err != nil {
		return
	}

	for _, p := range c.Packages {
		if err = p.Analyze(); err != nil {
			return
		}
	}

	return nil
}

// ParsePackages return packages
func (c *Corpus) ParsePackages() error {

	path := c.Path

	if path == "" {
		path = "./..."
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

	for _, pkg := range c.Packages {
		if pkg.Parent == nil {
			c.Tree = append(c.Tree, pkg)
		}
	}

	return nil
}

// RenderStaticAssets write static asset files
func (c *Corpus) RenderStaticAssets() (err error) {

	for filename, content := range static.Files {

		switch filepath.Ext(filename) {
		case ".html":
			continue
		}

		path := filepath.Join("docs/_static", filepath.Dir(filename))

		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}

		err = ioutil.WriteFile("docs/_static/"+filename, []byte(content), 0644)
		if err != nil {
			return
		}
	}

	return
}

// RenderPackage write package html page
func (c *Corpus) RenderPackage(pkg *Package) (err error) {

	// path := strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)
	path := pkg.ImportPath
	path = fmt.Sprintf("docs/%s", path)

	// auto mkdir dirs
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return
	}

	// generate package info page
	page := NewPage(c, pkg)

	var buf bytes.Buffer

	{
		if err = page.Render(&buf, PackagePage); err != nil {
			return
		}

		filename := fmt.Sprintf("%s/index.html", path)
		fmt.Printf("write package %s doc: %s", pkg.Name, filename)
		if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			fmt.Printf(" error\n")
			return
		}
		fmt.Printf(" success\n")
	}

	// generate packate types page
	for _, t := range pkg.Types {

		if c.DisplayPrivateIndent(t.Name) == false {
			break
		}

		page.Type = t
		page.Title = t.Name

		buf.Reset()
		if err = page.Render(&buf, TypePage); err != nil {
			return
		}

		filename := fmt.Sprintf("%s/%s.html", path, t.Name)
		fmt.Printf("write type %s doc: %s", t.Name, filename)
		if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			fmt.Printf(" error\n")
			return
		}
		fmt.Printf(" success\n")

		// generate packate type's funcs & methods page
		var funcs []*Func
		funcs = append(funcs, t.Funcs...)
		funcs = append(funcs, t.Methods...)

		for _, fn := range funcs {

			if c.DisplayPrivateIndent(fn.Name) == false {
				break
			}

			page.Func = fn
			page.Title = fn.Name

			buf.Reset()
			if err = page.Render(&buf, FuncPage); err != nil {
				return
			}

			filename := fmt.Sprintf("%s/%s.%s.html", path, t.Name, fn.Name)
			fmt.Printf("write func %s.%s doc: %s", t.Name, fn.Name, filename)
			if err = ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
				fmt.Printf(" error\n")
				return
			}
			fmt.Printf(" success\n")
		}
	}

	return err
}

// DisplayPrivateIndent display private indent
func (c *Corpus) DisplayPrivateIndent(name string) bool {

	if c.EnablePrivateIndent {
		return true
	}

	for _, r := range name {
		if unicode.IsUpper(r) {
			return true
		}
		return false
	}

	return false
}

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

	case []*doc.Type:
		var types []*doc.Type
		for _, node := range nodes.([]*doc.Type) {
			if IsExported(node.Name) {
				types = append(types, node)
			}
		}
		return types

	case []*doc.Func:
		var funcs []*doc.Func
		for _, node := range nodes.([]*doc.Func) {
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
