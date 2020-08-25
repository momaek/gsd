package gsd

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
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

	Imports   []string // import paths used by this package
	Filenames []string
	Notes     map[string][]*doc.Note

	// declarations
	Consts []*doc.Value
	Types  []*doc.Type
	Vars   []*doc.Value
	Funcs  []*doc.Func

	// Examples is a sorted list of examples associated with
	// the package. Examples are extracted from _test.go files provided to NewFromFiles.
	Examples []*doc.Example

	ParentImportPath string   // parent package ImportPath
	Parent           *Package `json:"-"` // parent package, important: json must ignore, prevent cycle parsing
	SubPackages      Packages // subpackages
}

// --------------------------------------------------------------------

// Packages with package array
type Packages []*Package

// TODO: Packages impl sorting func

// Analyze the package
func (p *Package) Analyze() (err error) {

	var fset = token.NewFileSet() // positions are relative to fset

	pkgs, err := parser.ParseDir(fset, p.Dir, nil, parser.ParseComments)
	if err != nil {
		return
	}

	var (
		name       string
		astPackage *ast.Package
	)

	for name, astPackage = range pkgs {
		if strings.HasSuffix(name, "_test") { // skip test package
			continue
		}
	}

	d := doc.New(astPackage, p.ImportPath, 0)

	p.Doc = d.Doc
	p.Name = d.Name
	p.ImportPath = d.ImportPath
	p.Imports = d.Imports
	p.Filenames = d.Filenames
	p.Notes = d.Notes
	p.Consts = d.Consts
	p.Types = d.Types
	p.Vars = d.Vars
	p.Funcs = d.Funcs
	p.Examples = d.Examples

	// for _, t := range d.Types {
	// 	Structs(t)
	// }

	// for _, f := range p.Funcs {
	// 	fmt.Println("func", f.Name)
	// 	fmt.Println("docs:\n", f.Doc)
	// }

	return
}

// Structs get struct comments
func Structs(t *doc.Type) {

	fmt.Println("\nType", t.Name)

	fmt.Println("\ndocs:")
	fmt.Println(strings.TrimSpace(t.Doc))

	// fmt.Printf("t.Methods: %#v\n", t.Methods)
	// fmt.Printf("t.Consts: %#v\n", t.Consts)
	// fmt.Printf("t.Vars: %#v\n", t.Vars)

	// fmt.Printf("t.Decl.Tok: %#v, %#v\n", t.Decl.Tok, token.TYPE)

	fmt.Println()

	var data [][]string

	for _, spec := range t.Decl.Specs {

		typeSpec := spec.(*ast.TypeSpec)

		if str, ok := typeSpec.Type.(*ast.StructType); ok {

			for _, field := range str.Fields.List {
				var fieldInfo []string

				var names []string
				for _, name := range field.Names {
					names = append(names, name.Name)
				}

				fieldInfo = append(fieldInfo, strings.Join(names, ","))                // 字段名
				fieldInfo = append(fieldInfo, fmt.Sprintf("%+v", field.Type))          // 类型
				fieldInfo = append(fieldInfo, strings.TrimSpace(field.Doc.Text()))     // 字段文档（字段上方注释）
				fieldInfo = append(fieldInfo, strings.TrimSpace(field.Comment.Text())) // 字段行尾注释

				// fmt.Printf("field.Type: %+v\n", field.Type) // 类型

				data = append(data, fieldInfo)
			}

		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Type", "Doc", "Line comments"})
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()
}
