package gsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"golang.org/x/xerrors"
)

// PackageList return packages
func PackageList(path string) ([]Package, error) {

	if path == "" {
		path = "./..."
	}

	out, err := exec.Command("go", "list", "-json", path).Output()
	if ee := (*exec.ExitError)(nil); xerrors.As(err, &ee) {
		return nil, fmt.Errorf("go command exited unsuccessfully: %v\n%s", ee.ProcessState.String(), ee.Stderr)
	} else if err != nil {
		return nil, err
	}

	var packages []Package
	for dec := json.NewDecoder(bytes.NewReader(out)); ; {
		var m Package
		err := dec.Decode(&m)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		packages = append(packages, m)
	}

	return packages, nil
}
