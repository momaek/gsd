package gsd

import (
	"fmt"
	"strings"
	"testing"
)

var text = `nomral description content

    @GSD:NOTE @GSD:NOTE note content

@GSD:IGNORE ignore content

	@GSD:ABC abc tag content

	end content`

func TestAnnotation(t *testing.T) {

	fmt.Println("input", text)

	lines := strings.Split(text, "\n")

	for _, line := range lines {
		output, marker, match := Annotation(line)

		fmt.Println(output, marker, match)
	}

	output := MarkdownConvert(text)

	fmt.Println(string(output))
}
