package gsd

import (
	"fmt"
	"time"
)

// Module go mod info type
type Module struct {
	Path      string       `json:",omitempty"` // module path
	Version   string       `json:",omitempty"` // module version
	Versions  []string     `json:",omitempty"` // available module versions
	Replace   *Module      `json:",omitempty"` // replaced by this module
	Time      *time.Time   `json:",omitempty"` // time version was created
	Update    *Module      `json:",omitempty"` // available update (with -u)
	Main      bool         `json:",omitempty"` // is this the main module?
	Indirect  bool         `json:",omitempty"` // module is only indirectly needed by main module
	Dir       string       `json:",omitempty"` // directory holding local copy of files, if any
	GoMod     string       `json:",omitempty"` // path to go.mod file describing module, if any
	GoVersion string       `json:",omitempty"` // go version used in module
	Error     *ModuleError `json:",omitempty"` // error loading module
}

// ModuleError go mod error type
type ModuleError struct {
	Err string // error text
}

func (m *Module) String() string {
	s := m.Path
	if m.Version != "" {
		s += " " + m.Version
		if m.Update != nil {
			s += " [" + m.Update.Version + "]"
		}
	}
	if m.Replace != nil {
		s += " => " + m.Replace.Path
		if m.Replace.Version != "" {
			s += " " + m.Replace.Version
			if m.Replace.Update != nil {
				s += " [" + m.Replace.Update.Version + "]"
			}
		}
	}
	return s
}

// A Package describes a single package found in a directory.
type Package struct {
	Dir           string   `json:",omitempty"` // directory containing package sources
	ImportPath    string   `json:",omitempty"` // import path of package in dir
	ImportComment string   `json:",omitempty"` // path in import comment on package statement
	Name          string   `json:",omitempty"` // package name
	Doc           string   `json:",omitempty"` // package documentation string
	Target        string   `json:",omitempty"` // installed target for this package (may be executable)
	Shlib         string   `json:",omitempty"` // the shared library that contains this package (only set when -linkshared)
	Root          string   `json:",omitempty"` // Go root, Go path dir, or module root dir containing this package
	ConflictDir   string   `json:",omitempty"` // Dir is hidden by this other directory
	ForTest       string   `json:",omitempty"` // package is only for use in named test
	Export        string   `json:",omitempty"` // file containing export data (set by go list -export)
	Module        *Module  `json:",omitempty"` // info about package's module, if any
	Match         []string `json:",omitempty"` // command-line patterns matching this package
	Goroot        bool     `json:",omitempty"` // is this package found in the Go root?
	Standard      bool     `json:",omitempty"` // is this package part of the standard Go library?
	DepOnly       bool     `json:",omitempty"` // package is only as a dependency, not explicitly listed
	BinaryOnly    bool     `json:",omitempty"` // package cannot be recompiled
	Incomplete    bool     `json:",omitempty"` // was there an error loading this package or dependencies?

	// Stale and StaleReason remain here *only* for the list command.
	// They are only initialized in preparation for list execution.
	// The regular build determines staleness on the fly during action execution.
	Stale       bool   `json:",omitempty"` // would 'go install' do anything for this package?
	StaleReason string `json:",omitempty"` // why is Stale true?

	// Source files
	// If you add to this list you MUST add to p.AllFiles (below) too.
	// Otherwise file name security lists will not apply to any new additions.
	GoFiles         []string `json:",omitempty"` // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles        []string `json:",omitempty"` // .go source files that import "C"
	CompiledGoFiles []string `json:",omitempty"` // .go output from running cgo on CgoFiles
	IgnoredGoFiles  []string `json:",omitempty"` // .go source files ignored due to build constraints
	CFiles          []string `json:",omitempty"` // .c source files
	CXXFiles        []string `json:",omitempty"` // .cc, .cpp and .cxx source files
	MFiles          []string `json:",omitempty"` // .m source files
	HFiles          []string `json:",omitempty"` // .h, .hh, .hpp and .hxx source files
	FFiles          []string `json:",omitempty"` // .f, .F, .for and .f90 Fortran source files
	SFiles          []string `json:",omitempty"` // .s source files
	SwigFiles       []string `json:",omitempty"` // .swig files
	SwigCXXFiles    []string `json:",omitempty"` // .swigcxx files
	SysoFiles       []string `json:",omitempty"` // .syso system object files added to package

	// Dependency information
	Imports   []string          `json:",omitempty"` // import paths used by this package
	ImportMap map[string]string `json:",omitempty"` // map from source import to ImportPath (identity entries omitted)
	Deps      []string          `json:",omitempty"` // all (recursively) imported dependencies

	// Test information
	// If you add to this list you MUST add to p.AllFiles (below) too.
	// Otherwise file name security lists will not apply to any new additions.
	TestGoFiles  []string `json:",omitempty"` // _test.go files in package
	TestImports  []string `json:",omitempty"` // imports from TestGoFiles
	XTestGoFiles []string `json:",omitempty"` // _test.go files outside package
	XTestImports []string `json:",omitempty"` // imports from XTestGoFiles

	// Extension fields

	ParentImportPath string   // parent package ImportPath
	Parent           *Package `json:"-"` // parent package, important: json must ignore, prevent cycle parsing
	SubPackages      Packages // subpackages
}

// AllFiles returns the names of all the files considered for the package.
// This is used for sanity and security checks, so we include all files,
// even IgnoredGoFiles, because some subcommands consider them.
// The go/build package filtered others out (like foo_wrongGOARCH.s)
// and that's OK.
func (p *Package) AllFiles() []string {
	return StringList(
		p.GoFiles,
		p.CgoFiles,
		// no p.CompiledGoFiles, because they are from GoFiles or generated by us
		p.IgnoredGoFiles,
		p.CFiles,
		p.CXXFiles,
		p.MFiles,
		p.HFiles,
		p.FFiles,
		p.SFiles,
		p.SwigFiles,
		p.SwigCXXFiles,
		p.SysoFiles,
		p.TestGoFiles,
		p.XTestGoFiles,
	)
}

// Desc returns the package "description", for use in b.showOutput.
func (p *Package) Desc() string {
	if p.ForTest != "" {
		return p.ImportPath + " [" + p.ForTest + ".test]"
	}
	return p.ImportPath
}

// Packages with package array
type Packages []*Package

// StringList flattens its arguments into a single []string.
// Each argument in args must have type string or []string.
func StringList(args ...interface{}) []string {
	var x []string
	for _, arg := range args {
		switch arg := arg.(type) {
		case []string:
			x = append(x, arg...)
		case string:
			x = append(x, arg)
		default:
			panic("stringList: invalid argument of type " + fmt.Sprintf("%T", arg))
		}
	}
	return x
}
