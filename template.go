package gsd

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"strings"
	"sync"
)

// Presentation generates output from a corpus.
type Presentation struct {
	Corpus *Corpus

	SidebarHTML *template.Template
	PackageHTML *template.Template

	initFuncMapOnce sync.Once
	funcMap         template.FuncMap
}

// NewPresentation returns a new Presentation from a corpus.
func NewPresentation(c *Corpus) *Presentation {
	if c == nil {
		panic("nil Corpus")
	}
	p := &Presentation{
		Corpus: c,
	}

	p.readTemplates()

	return p
}

func (p *Presentation) readTemplate(name string) *template.Template {

	data, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}

	t, err := template.New(name).Funcs(p.FuncMap()).Parse(string(data))
	if err != nil {
		panic(err)
	}

	return t
}

func (p *Presentation) readTemplates() {

	p.SidebarHTML = p.readTemplate("static/sidebar.html")
	p.PackageHTML = p.readTemplate("static/package.html")
}

// FuncMap defines template functions used in godoc templates.
func (p *Presentation) FuncMap() template.FuncMap {
	p.initFuncMapOnce.Do(p.initFuncMap)
	return p.funcMap
}

func (p *Presentation) initFuncMap() {
	if p.Corpus == nil {
		panic("nil Presentation.Corpus")
	}

	p.funcMap = template.FuncMap{
		"repeat":      strings.Repeat,
		"unescaped":   unescaped,
		"DisplayTree": DisplayTree,
	}
}

// unescaped 取消模板转义
func unescaped(x string) interface{} { return template.HTML(x) }

// DisplayTree with template helper
func DisplayTree(pkgs Packages) interface{} {

	var buf bytes.Buffer // A Buffer needs no initialization.

	buf.Write([]byte("<ul>\n"))

	displayTree(&buf, pkgs)

	buf.Write([]byte("</ul>\n"))

	return unescaped(buf.String())
}

func displayTree(buf *bytes.Buffer, pkgs Packages) {
	for _, pkg := range pkgs {
		buf.Write([]byte("<li>"))

		fmt.Fprintf(buf, "\n<a>%s</a>\n", pkg.ImportPath)

		if len(pkg.SubPackages) > 0 {
			buf.Write([]byte("\n<ul>\n"))
			displayTree(buf, pkg.SubPackages)
			buf.Write([]byte("</ul>\n"))
		}

		buf.Write([]byte("</li>\n"))
	}
}

// RenderPackage render package html
func RenderPackage(p *Presentation, pkg *Package) ([]byte, error) {

	var buf bytes.Buffer

	data := map[string]interface{}{
		"pkg": pkg,
	}

	if err := p.PackageHTML.Execute(&buf, data); err != nil {
		log.Printf("%s.Execute: %s", p.PackageHTML.Name(), err)
		return nil, err
	}

	return buf.Bytes(), nil
}
