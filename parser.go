package gsd

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"strings"
)

// Parser dir return ast packages map
func Parser(pkg *Package) (pkgs map[string]*ast.Package, err error) {

	var fset = token.NewFileSet() // positions are relative to fset

	var packages = map[string]*ast.Package{}

	pkgs, err = parser.ParseDir(fset, pkg.Dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for name, astPackage := range pkgs {
		packages[name] = astPackage

		// ==================================================================
		fmt.Println(strings.Repeat("=", 72))
		fmt.Printf("package name: %s\n", name)
		fmt.Printf("package ast: \t%#v\n", astPackage)
		fmt.Println(strings.Repeat("=", 72))
		// ==================================================================

		p := doc.New(astPackage, pkg.ImportPath, 0)
		for _, t := range p.Types {
			Structs(t)
		}
	}

	// fmt.Printf("p.Funcs: %#v\n", p.Funcs)

	// for _, f := range p.Funcs {
	// 	fmt.Println("func", f.Name)
	// 	fmt.Println("docs:\n", f.Doc)
	// }

	return packages, nil
}
