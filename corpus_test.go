package gsd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/miclle/gsd"
)

func TestPackageList(t *testing.T) {
	assert := assert.New(t)

	corpus := gsd.NewCorpus()

	packages, err := corpus.ParsePackageList()

	assert.Nil(err)
	assert.NotEmpty(packages)

	for _, p := range packages {
		assert.NotEmpty(p.Dir)
		assert.NotEmpty(p.ImportPath)
		assert.NotEmpty(p.Name)
	}
}
