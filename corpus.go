package gsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
func (c *Corpus) RenderPackage(pkg *Package) error {

	page := NewPage(c, pkg)

	var buf bytes.Buffer
	if err := page.Render(&buf); err != nil {
		return nil
	}

	path := strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path)
	path = fmt.Sprintf("docs/%s", path)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/index.html", path)
	err := ioutil.WriteFile(filename, buf.Bytes(), 0644)

	return err
}
