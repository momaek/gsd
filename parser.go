package gsd

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
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
