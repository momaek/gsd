package gsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

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

	// 获取所有包列表
	c.Packages, err = ParsePackageList(c.Path)
	if err != nil {
		return
	}

	c.Tree = ParsePackageTree(c.Packages)

	for _, p := range c.Packages {
		if err = p.Analyze(); err != nil {
			return
		}
	}

	return nil
}

// ParsePackageList return packages
func ParsePackageList(path string) (map[string]*Package, error) {

	if path == "" {
		path = "./..."
	}

	out, err := exec.Command("go", "list", "-json", path).Output()
	if ee := (*exec.ExitError)(nil); xerrors.As(err, &ee) {
		return nil, fmt.Errorf("go command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
	} else if err != nil {
		return nil, err
	}

	var pkgs = map[string]*Package{}
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var dpkg PackagePublic
		err := dec.Decode(&dpkg)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		pkgs[dpkg.ImportPath] = &Package{
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

	return pkgs, nil
}

// ParsePackageTree return packages tree
func ParsePackageTree(pkgs map[string]*Package) Packages {

	for _, pkg := range pkgs {
		if pkg.ImportPath == pkg.Module.Path {
			continue
		}

		var (
			seps       = strings.Split(strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path+"/"), "/")
			parentPath = pkg.ImportPath
		)

		for i := len(seps); i > 0; i-- {
			parentPath = strings.TrimSuffix(parentPath, "/"+seps[i-1])
			if parentPkg, exists := pkgs[parentPath]; exists {
				pkg.ParentImportPath = parentPkg.ParentImportPath
				pkg.Parent = parentPkg
				parentPkg.SubPackages = append(parentPkg.SubPackages, pkg)
				break
			}
		}
	}

	var roots Packages
	for _, pkg := range pkgs {
		if pkg.Parent == nil {
			roots = append(roots, pkg)
		}
	}

	return roots
}
