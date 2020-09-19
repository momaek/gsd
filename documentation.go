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

// Documentation with comments
type Documentation struct {
	Doc  string // original content
	Body string // markdown content

	Summary Markdown // summary annotation content
}

// Markdown type
type Markdown struct {
	Text   string // original content
	HTML   string // markdown content
	Marker string // annotation mark
}

func (d *Documentation) String() string {
	return d.Body
}

// NewDocumentation return documentation with doc comments
func NewDocumentation(text string) Documentation {
	doc := Documentation{
		Doc: text,
	}

	var (
		blocks = strings.Split(strings.Trim(text, " "), "\n\n")
		body   = new(bytes.Buffer)
	)

	for i, block := range blocks {
		output, marker, match := Annotation(block)

		segment := new(bytes.Buffer)

		if match {
			fmt.Fprintf(segment, `<div class="marker marker-%s">`, marker)
		}

		if err := md.Convert([]byte(output), segment); err != nil {
			log.Println("markdown convert error", err.Error())
			fmt.Fprintf(segment, block)
		}

		if match {
			fmt.Fprintf(segment, "</div>\n\n")
		}

		if i == 0 {
			doc.Summary = Markdown{
				Text: output,
				HTML: autocorrect.Format(segment.String()),
			}
		}

		// set summary
		if marker == "summary" && doc.Summary.Marker == "" {
			doc.Summary = Markdown{
				Text: output,
				HTML: autocorrect.Format(segment.String()),
			}
		}

		segment.WriteTo(body)
	}

	doc.Body = autocorrect.Format(body.String())

	return doc
}

// MarkdownConvert parse markdown text to HTML
func MarkdownConvert(text string) string {

	var (
		blocks = strings.Split(strings.Trim(text, " "), "\n\n")
		buf    = new(bytes.Buffer)
	)

	for _, block := range blocks {
		output, marker, match := Annotation(block)

		if !match {
			if err := md.Convert([]byte(block), buf); err != nil {
				log.Println("markdown convert error", err.Error())
				fmt.Fprintf(buf, block)
			}
			continue
		}

		fmt.Fprintf(buf, `<div class="marker marker-%s">`, marker)

		if err := md.Convert([]byte(output), buf); err != nil {
			log.Println("markdown convert error", err.Error())
			fmt.Fprintf(buf, block)
		}

		fmt.Fprintf(buf, "</div>\n\n")
	}

	return autocorrect.Format(buf.String())
}

// --------------------------------------------------------------------

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
