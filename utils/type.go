package utils

import (
	"fmt"
	"go/ast"
	"go/doc"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
)

// Struct get struct comments
func Struct(t *doc.Type) {

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

		str := typeSpec.Type.(*ast.StructType)

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

			fmt.Printf("field.Type: %+v\n", field.Type) // 类型

			data = append(data, fieldInfo)
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

type Foo struct {
}

func (f Foo) String() {

}
