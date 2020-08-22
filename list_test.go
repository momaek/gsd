package gsd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageList(t *testing.T) {
	assert := assert.New(t)

	packages, err := PackageList()

	assert.Nil(err)

	for _, p := range packages {

		// ==================================================================
		fmt.Println(strings.Repeat("=", 72))
		fmt.Printf("p.Dir: %#v\n", p.Dir)
		fmt.Printf("p.ImportPath: %#v\n", p.ImportPath)
		fmt.Printf("p.Name: %#v\n", p.Name)
		fmt.Printf("p.Doc: %#v\n", p.Doc)
		fmt.Printf("p.Root: %#v\n", p.Root)
		fmt.Printf("p.GoFiles: %#v\n", p.GoFiles)
	}
}
