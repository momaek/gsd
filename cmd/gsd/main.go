// package main

// import (
// 	"go/ast"
// 	"go/parser"
// 	"go/token"
// 	"reflect"

// 	"fmt"
// )

// // CommonBandwidthMeter commonbandwidth meter
// // aaawdaaiijaiwjawda
// type CommonBandwidthMeter struct {
// 	ID       uint64 `gorm:"primary_key"`
// 	UID      uint32 `gorm:"index:idx_uid_regionid"`
// 	RegionID string `gorm:"index:idx_uid_regionid"`
// 	UniqueID string `gorm:"unique_index"` // for upsert
// 	// UnionID ddddddaaaaa
// 	UnionID    string `gorm:"index:idx_unionid_start"` /* union_id = uid_region_spec for some search and count queryi*/
// 	InstanceID string
// 	Spec       string
// 	Size       float64
// 	Hours      float64

// 	Month string `sql:"-"`
// }

// func (m *CommonBandwidthMeter) Get(f ast.File) string {

// 	return ""
// }

// // main main
// func main() {
// 	fmt.Println("vim-go")
// 	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, parser.ParseComments)
// 	if err != nil {
// 		fmt.Println("---- parse error")
// 		return
// 	}

// 	ast.Inspect(f, func(n ast.Node) bool {
// 		switch n.(type) {
// 		case *ast.FuncDecl:
// 			fmt.Printf("func ========= %+v \n", n)
// 		case *ast.GenDecl:
// 			fmt.Printf("Gen ========= %+v \n", n)
// 			decl := n.(*ast.GenDecl)

// 			if decl.Tok == token.TYPE {
// 				for _, v := range decl.Specs {
// 					tSpec := v.(*ast.TypeSpec)
// 					fmt.Printf("Type Spec : ====== %+v ,real type: %+v\n", tSpec.Type, reflect.TypeOf(tSpec.Type).String())
// 					str := tSpec.Type.(*ast.StructType)
// 					for _, vv := range str.Fields.List {
// 						fmt.Printf("======@@@@@@@======= %+v \n", vv)
// 						if vv.Comment != nil {
// 							for _, vvv := range vv.Comment.List {
// 								fmt.Printf("======@@@@@@@======= comment: %+v\n", vvv)
// 							}
// 						}
// 					}
// 				}
// 			}

// 		case *ast.BadDecl:
// 			fmt.Printf("bad ========= %+v \n", n)
// 		default:
// 			//			fmt.Printf("default ===================== %+v \n", n)
// 		}

// 		return true
// 	})
// }

package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"log"

	"github.com/miclle/gsd"
)

// Foo type
// this is Foo type desc
type Foo struct {

	// FooString field docs 1
	//
	// FooString field docs 2-1
	// FooString field docs 2-2
	//
	// FooString field docs 3
	FooString string /* FooA field line comment */ // FooA comment2

	// FooInt field docs
	FooA, FooInt int /* FooB field line comment */ // FooB comment2

	// FooIntArray field docs
	FooIntArray []int /* FooIntArray field line comment */ // FooIntArray comment2

	// Reader field documentation
	Reader io.Reader // Reader line comment

	// Logger field documentation
	log.Logger // Logger field line comment
}

// Status return status
// this is status desc
// this is status desc
func (f Foo) Status(a string) (s string) { // this is Boo.Status method inline comment
	return f.FooString
}

// Main docs
func main() {
	fset := token.NewFileSet() // positions are relative to fset

	pkgs, err := parser.ParseDir(fset, "./", nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	for k, f := range pkgs {

		fmt.Println("package", k)

		p := doc.New(f, "./", 0)

		for _, t := range p.Types {
			gsd.Struct(t)
		}

		// fmt.Printf("p.Funcs: %#v\n", p.Funcs)

		// for _, f := range p.Funcs {
		// 	fmt.Println("func", f.Name)
		// 	fmt.Println("docs:\n", f.Doc)
		// }

		fmt.Println()

		// ================================================================

		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				fmt.Printf("\n%s:\tFunc: %s\t%s\n", fset.Position(n.Pos()), x.Name, x.Doc.Text())

			case *ast.TypeSpec:
				fmt.Printf("\n%s:\tTypeSpec %s\t%s\n", fset.Position(n.Pos()), x.Name, x.Doc.Text())

			case *ast.Field:
				fmt.Printf("%s:\tField %s\t%s\n", fset.Position(n.Pos()), x.Names, x.Doc.Text())

				if x.Comment != nil {
					fmt.Printf("%s:\tField Comment %s\t", fset.Position(n.Pos()), x.Names)
					for _, c := range x.Comment.List {
						fmt.Printf("%+v ", c.Text)
					}
					fmt.Println()
				}

			case *ast.GenDecl:
				fmt.Printf("\n%s:\tGenDecl %s\n\n", fset.Position(n.Pos()), x.Doc.Text())
			}

			return true
		})
	}
}
