package utils_test

import (
	"fmt"
	"strings"
)

func ExampleStruct() {
	fmt.Printf("Fields are: %q", strings.Fields("  foo bar  baz   "))
	// Output: Fields are: ["foo" "bar" "baz"]
}

func ExampleFoo_String() {
	fmt.Println("ExampleFoo_String")
	// Output: ExampleFoo_String
}
