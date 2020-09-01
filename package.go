package gsd

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"strings"
	"time"
	"unicode"
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

// A PackagePublic describes a single package found in a directory.
// go/libexec/src/cmd/go/internal/load/pkg.go
type PackagePublic struct {
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
}

// Package type
type Package struct {
	Dir           string  // !important: directory containing package sources
	ImportPath    string  // !important: import path of package in dir
	ImportComment string  // path in import comment on package statement
	Name          string  // package name
	Doc           string  // package documentation string
	Module        *Module // info about package's module, if any
	Stale         bool    // would 'go install' do anything for this package?
	StaleReason   string  // why is Stale true?

	// declarations
	Imports   []string               // import paths used by this package
	Filenames []string               // all files
	Notes     map[string][]*doc.Note // nil if no package Notes, or contains Buts, etc...
	Consts    []*doc.Value
	Types     []*doc.Type
	Vars      []*doc.Value
	Funcs     []*doc.Func

	// Examples is a sorted list of examples associated with
	// the package. Examples are extracted from _test.go files provided to NewFromFiles.
	Examples []*doc.Example // nil if no example code

	ParentImportPath string     // parent package ImportPath
	Parent           *Package   `json:"-"` // parent package, important: json must ignore, prevent cycle parsing
	SubPackages      []*Package // subpackages

	// ------------------------------------------------------------------

	Dirname string // directory containing the package
	Err     error  // error or nil

	// package info
	FSet       *token.FileSet       // nil if no package documentation
	DocPackage *doc.Package         // nil if no package documentation
	PAst       map[string]*ast.File // nil if no AST with package exports
	IsMain     bool                 // true for package main
}

// IsEmpty return package is empty
func (p *Package) IsEmpty() bool {
	return p.Err != nil || p.PAst == nil && p.DocPackage == nil && len(p.SubPackages) == 0
}

// --------------------------------------------------------------------

// Packages with package array
type Packages []*Package

// TODO Packages impl sorting func

// Analyze the package
func (p *Package) Analyze() (err error) {

	p.FSet = token.NewFileSet() // positions are relative to fset

	pkgs, err := parser.ParseDir(p.FSet, p.Dir, nil, parser.ParseComments)
	if err != nil {
		return
	}

	var astPackage *ast.Package
	for name, apkg := range pkgs {
		if strings.HasSuffix(name, "_test") { // skip test package
			continue
		}
		astPackage = apkg
	}

	d := doc.New(astPackage, p.ImportPath, doc.AllDecls)

	p.DocPackage = d

	p.Doc = d.Doc
	p.Name = d.Name
	p.ImportPath = d.ImportPath
	p.Imports = d.Imports
	p.Filenames = d.Filenames
	p.Notes = d.Notes
	p.Consts = d.Consts
	// p.Types = d.Types
	p.Vars = d.Vars
	p.Funcs = d.Funcs
	p.Examples = d.Examples

	// TODO(miclle) check type is public
	for _, t := range d.Types {
		for _, r := range t.Name {
			if unicode.IsUpper(r) {
				p.Types = append(p.Types, t)
			}
		}
	}

	// for _, f := range p.Funcs {
	// 	fmt.Println("func", f.Name)
	// 	fmt.Println("docs:\n", f.Doc)
	// }

	return
}

// TypeFields get type fields
func TypeFields(t *doc.Type) (fields []*ast.Field) {

	for _, spec := range t.Decl.Specs {

		typeSpec := spec.(*ast.TypeSpec)

		if str, ok := typeSpec.Type.(*ast.StructType); ok {
			return str.Fields.List
		}

	}

	return
}
