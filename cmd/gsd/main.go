package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/miclle/gsd"
)

// Main docs
func main() {
	// fset := token.NewFileSet() // positions are relative to fset

	pkgs, err := gsd.PackageList("./...")

	// pkgs, err := parser.ParseDir(fset, "./", nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(strings.Repeat("=", 72))

	for _, p := range pkgs {
		gsd.Parser(p)
	}

	tree := gsd.PackageTrees(pkgs)

	for _, pkg := range tree {
		fmt.Printf("pkg: %#v\n", pkg.SubPackages)
	}

	file, err := json.Marshal(tree)

	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("docs/packages.json", file, 0644)
	if err != nil {
		panic(err)
	}

}
