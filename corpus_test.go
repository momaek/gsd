package gsd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/miclle/gsd"
)

func TestPackageList(t *testing.T) {
	assert := assert.New(t)

	corpus := gsd.NewCorpus(".")

	err := corpus.ParsePackages()
	assert.Nil(err)
	assert.NotEmpty(corpus.Packages)

	for _, p := range corpus.Packages {
		assert.NotEmpty(p.Dir)
		assert.NotEmpty(p.ImportPath)
		assert.NotEmpty(p.Name)
	}
}
