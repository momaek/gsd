package document

import (
	"bytes"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"strings"
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

// A PackagePublic describes a single package found in a directory.
// go/libexec/src/cmd/go/internal/load/pkg.go
type PackagePublic struct {
	Dir           string   `json:",omitempty"` // directory containing package sources
	ImportPath    string   `json:",omitempty"` // import path of package in dir
	ImportComment string   `json:",omitempty"` // path in import comment on package statement
	Name          string   `json:",omitempty"` // package name
	Doc           string   `json:",omitempty"` // package document string
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
	Doc           string  // package document string
	Module        *Module // info about package's module, if any
	Stale         bool    // would 'go install' do anything for this package?
	StaleReason   string  // why is Stale true?

	// declarations
	Imports   []string               // import paths used by this package
	Filenames []string               // all files
	Notes     map[string][]*doc.Note // nil if no package Notes, or contains Buts, etc...
	Consts    []*doc.Value
	Types     []*Type
	Vars      []*doc.Value
	Funcs     []*Func

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
	FSet       *token.FileSet       // nil if no package document
	DocPackage *doc.Package         // nil if no package document
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
	p.Vars = d.Vars
	p.Examples = d.Examples

	// set package types
	for _, t := range d.Types {
		p.Types = append(p.Types, NewTypeWithDoc(t))
	}

	// set package funcs
	for _, fn := range d.Funcs {
		p.Funcs = append(p.Funcs, NewFuncWithDoc(fn))
	}

	return
}

// --------------------------------------------------------------------

// TypeFields get type fields
func TypeFields(t *Type) (fields []*Field) {

	if t == nil {
		return
	}

	for _, spec := range t.Decl.Specs {

		typeSpec := spec.(*ast.TypeSpec)

		// struct type
		if str, ok := typeSpec.Type.(*ast.StructType); ok {

			for _, f := range str.Fields.List {
				fields = append(fields, &Field{
					Field: f,
					Type:  t,
				})
			}

			return
		}

		// interface type methods
		if str, ok := typeSpec.Type.(*ast.InterfaceType); ok {
			for _, field := range str.Methods.List {
				if ident, ok := field.Type.(*ast.Ident); ok && ident.Obj != nil {
					field.Names = []*ast.Ident{ident}
				}
			}

			for _, f := range str.Methods.List {
				fields = append(fields, &Field{
					Field: f,
					Type:  t,
				})
			}

			return
		}
	}

	return
}

// TypeSpec type spec
type TypeSpec string

const (
	// StructType struct type spec
	StructType TypeSpec = "struct"

	// InterfaceType interface type spec
	InterfaceType TypeSpec = "interface"
)

// Type type
type Type struct {

	// doc.Type
	Doc  string
	Name string
	Decl *ast.GenDecl

	Documentation Documentation

	// associated declarations
	Consts []*doc.Value // sorted list of constants of (mostly) this type
	Vars   []*doc.Value // sorted list of variables of (mostly) this type

	Funcs   []*Func // sorted list of functions returning this type
	Methods []*Func // sorted list of methods (including embedded ones) of this type

	// Examples is a sorted list of examples associated with
	// this type. Examples are extracted from _test.go files
	// provided to NewFromFiles.
	Examples []*doc.Example

	// Fields *ast.FieldList

	Fields []*Field

	TypeSpec TypeSpec // type spec
}

// NewTypeWithDoc return type with doc.Type
func NewTypeWithDoc(t *doc.Type) *Type {

	var _t = &Type{
		Doc:      t.Doc,
		Name:     t.Name,
		Decl:     t.Decl,
		Consts:   t.Consts,
		Vars:     t.Vars,
		Examples: t.Examples,
	}

	_t.Documentation = NewDocumentation(t.Doc)

	_t.Fields = TypeFields(_t)

	for _, spec := range t.Decl.Specs {
		typeSpec := spec.(*ast.TypeSpec)

		if _, ok := typeSpec.Type.(*ast.StructType); ok {
			_t.TypeSpec = StructType
		}

		// interface type methods
		if str, ok := typeSpec.Type.(*ast.InterfaceType); ok {
			_t.TypeSpec = InterfaceType

			for _, field := range str.Methods.List {
				// interface funcs
				if fn, ok := field.Type.(*ast.FuncType); ok {
					var f = &Func{
						Doc:  field.Doc.Text(),
						Name: field.Names[0].Name,

						Params:  fn.Params,
						Results: fn.Results,
					}

					f.Documentation = NewDocumentation(field.Doc.Text())

					_t.Funcs = append(_t.Funcs, f)
				}
			}
		}
	}

	for _, fn := range t.Funcs {
		_t.Funcs = append(_t.Funcs, NewFuncWithDoc(fn))
	}

	for _, fn := range t.Methods {
		_t.Methods = append(_t.Methods, NewFuncWithDoc(fn))
	}

	return _t
}

// ------------------------------------------------------------------

// Func type
type Func struct {

	// doc.Func
	Doc  string
	Name string
	Decl *ast.FuncDecl

	// methods
	// (for functions, these fields have the respective zero value)
	Recv  string // actual   receiver "T" or "*T"
	Orig  string // original receiver "T" or "*T"
	Level int    // embedding level; 0 means not embedded

	Examples []*doc.Example

	// interface type
	Field    *ast.Field
	FuncType *ast.FuncType

	// ast.FuncType fields
	Params  *ast.FieldList // (incoming) parameters; non-nil
	Results *ast.FieldList // (outgoing) results; or nil

	Documentation Documentation
}

// NewFuncWithDoc return func with doc.Func
func NewFuncWithDoc(f *doc.Func) *Func {

	var fn = &Func{
		Doc:      f.Doc,
		Name:     f.Name,
		Decl:     f.Decl,
		Recv:     f.Recv,
		Orig:     f.Orig,
		Level:    f.Level,
		Examples: f.Examples,

		Params:  f.Decl.Type.Params,
		Results: f.Decl.Type.Results,

		Documentation: NewDocumentation(f.Doc),
	}
	return fn
}

// ------------------------------------------------------------------

// Field type
type Field struct {
	*ast.Field
	Type *Type
}

// JoinNames return names array
func (f Field) JoinNames() (names []string) {
	if f.Field == nil {
		return
	}

	for _, name := range f.Field.Names {
		names = append(names, name.Name)
	}
	return
}

// Documentation parse markdown doc
func (f *Field) Documentation() string {

	buf := new(bytes.Buffer)

	if f.Doc != nil {
		buf.WriteString(f.Doc.Text())
	}
	if f.Comment != nil {
		buf.WriteString(f.Comment.Text())
	}

	return MarkdownConvert(buf.String())
}
