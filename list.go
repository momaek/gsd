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

// PackageList return packages
func PackageList(path string) (Packages, error) {

	if path == "" {
		path = "./..."
	}

	out, err := exec.Command("go", "list", "-json", path).Output()
	if ee := (*exec.ExitError)(nil); xerrors.As(err, &ee) {
		return nil, fmt.Errorf("go command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
	} else if err != nil {
		return nil, err
	}

	var pkgs Packages
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var pkg Package
		err := dec.Decode(&pkg)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, &pkg)
	}

	return pkgs, nil
}

// PackageTree return packages tree
func PackageTree(pkgs Packages) Packages {

	var cache = map[string]*Package{}

	for _, pkg := range pkgs {
		cache[pkg.ImportPath] = pkg
	}

	for _, pkg := range pkgs {
		if pkg.ImportPath == pkg.Module.Path {
			continue
		}

		seps := strings.Split(strings.TrimPrefix(pkg.ImportPath, pkg.Module.Path+"/"), "/")

		var parentPath = pkg.ImportPath

		for i := len(seps); i > 0; i-- {
			parentPath = strings.TrimSuffix(parentPath, "/"+seps[i-1])
			if parentPkg, exists := cache[parentPath]; exists {
				pkg.ParentImportPath = parentPkg.ParentImportPath
				pkg.Parent = parentPkg
				parentPkg.SubPackages = append(parentPkg.SubPackages, pkg)
				break
			}
		}
	}

	var roots Packages
	for _, pkg := range cache {
		if pkg.Parent == nil {
			roots = append(roots, pkg)
		}
	}

	return roots
}
