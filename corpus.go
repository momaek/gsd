package gsd

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
	packages, err := PackageList(c.Path)
	if err != nil {
		return
	}

	for _, pkg := range packages {
		c.Packages[pkg.ImportPath] = pkg
	}

	c.Tree = PackageTree(packages)

	// for _, p := range c.Packages {
	// 	Parser(p)
	// }

	return nil
}
