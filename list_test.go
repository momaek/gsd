package gsd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageList(t *testing.T) {
	assert := assert.New(t)

	path := "./..."
	packages, err := PackageList(path)

	assert.Nil(err)
	assert.NotEmpty(packages)

	for _, p := range packages {
		assert.NotEmpty(p.Dir)
		assert.NotEmpty(p.ImportPath)
		assert.NotEmpty(p.Name)
	}
}
