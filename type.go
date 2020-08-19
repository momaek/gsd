package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"strings"
)

// Struct get struct comments
func Struct(t *doc.Type) {
	fmt.Println("type", t.Name)
	fmt.Println("docs:\n", t.Doc)
	fmt.Printf("t.Methods: %#v\n", t.Methods)
	fmt.Printf("t.Consts: %#v\n", t.Consts)
	fmt.Printf("t.Vars: %#v\n", t.Vars)

	// fmt.Printf("t.Decl.Tok: %#v, %#v\n", t.Decl.Tok, token.TYPE)

	fmt.Println()

	// for _, spec := range t.Decl.Specs {
	// 	fmt.Printf("t.Decl spec: %#v\n", spec)
	// }

	for _, v := range t.Decl.Specs {

		tSpec := v.(*ast.TypeSpec)

		str := tSpec.Type.(*ast.StructType)

		for _, field := range str.Fields.List {

			fmt.Printf("Field name: %+v\n", field.Names)

			fmt.Printf("field.Type: %#v\n", field.Type)
			fmt.Println()

			fmt.Printf("Field %+v doc:\n%s\n", field.Names, field.Doc.Text())
			fmt.Printf("Field %+v comments:\n%+v \n", field.Names, field.Comment.Text())

			if field.Comment != nil {
				for _, comment := range field.Comment.List {
					fmt.Printf("Field comment: %+v ", comment.Text)
				}
				fmt.Println()
			}

			fmt.Println(strings.Repeat("-", 72))
		}
	}

	fmt.Println("\n", strings.Repeat("-", 72))
}
