package gsd

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	autocorrect "github.com/huacnlee/go-auto-correct"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/miclle/gsd/lazyregexp"
)

var md goldmark.Markdown

func init() {

	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			// don’t using html.WithHardWraps
		),
	)

}

// MarkdownConvert parse markdown text to HTML
func MarkdownConvert(text string) string {

	var input = MarkBlockParse(strings.Trim(text, " "))

	var buf bytes.Buffer
	if err := md.Convert(input, &buf); err != nil {
		log.Println("markdown convert error", err.Error())
		return text
	}

	return autocorrect.Format(buf.String())
}

var markerRx = lazyregexp.New(`^[ \t]*\@(GSD|gsd):([\w]+)?`)

// Annotation extracts the expected output and whether there was a valid output comment
func Annotation(text string) (output, marker string, match bool) {

	// test that it begins with the prefix
	loc := markerRx.FindStringSubmatchIndex(text)

	if loc == nil {
		return text, "", false // no suitable comment found
	}

	output = text[loc[1]:]

	// Strip zero or more spaces followed by \n or a single space.
	output = strings.TrimLeft(output, " ")
	if len(output) > 0 && output[0] == '\n' {
		output = output[1:]
	}

	marker = clean(text[loc[0]:loc[1]], keepNL)
	marker = strings.ToLower(marker)
	marker = strings.TrimPrefix(marker, "@gsd:")

	return output, marker, true
}

// MarkBlockParse parse text
func MarkBlockParse(text string) []byte {

	buf := new(bytes.Buffer)

	lines := strings.Split(text, "\n")

	for _, line := range lines {

		output, marker, match := Annotation(line)

		if match {

			var markdown bytes.Buffer

			if err := md.Convert([]byte(output), &markdown); err != nil {
				log.Println("markdown convert error", err.Error())
			}

			fmt.Fprintf(buf, `<div class="marker marker-%s">`, marker)
			buf.Write(markdown.Bytes())
			fmt.Fprintf(buf, "</div>\n")
		} else {
			buf.WriteString(line + "\n")
		}
	}

	return buf.Bytes()
}

const (
	keepNL = 1 << iota
)

// clean replaces each sequence of space, \n, \r, or \t characters
// with a single space and removes any trailing and leading spaces.
// If the keepNL flag is set, newline characters are passed through
// instead of being change to spaces.
func clean(s string, flags int) string {
	var b []byte
	p := byte(' ')
	for i := 0; i < len(s); i++ {
		q := s[i]
		if (flags&keepNL) == 0 && q == '\n' || q == '\r' || q == '\t' {
			q = ' '
		}
		if q != ' ' || p != ' ' {
			b = append(b, q)
			p = q
		}
	}
	// remove trailing blank, if any
	if n := len(b); n > 0 && p == ' ' {
		b = b[0 : n-1]
	}
	return string(b)
}
