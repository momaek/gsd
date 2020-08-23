package gsd

// A Corpus holds all the package documentation
//
type Corpus struct {
	Path     string
	Packages Packages
	Tree     Packages
}

// Init initializes Corpus, once options on Corpus are set.
// It must be called before any subsequent method calls.
func (c *Corpus) Init() (err error) {

	c.Packages, err = PackageList(c.Path)
	if err != nil {
		return
	}

	c.Tree = PackageTrees(c.Packages)

	return nil
}
