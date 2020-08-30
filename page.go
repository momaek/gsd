package gsd

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"strings"
	"sync"
)

// Page generates output from a corpus.
type Page struct {
	Corpus *Corpus

	SidebarHTML *template.Template
	PackageHTML *template.Template

	initFuncMapOnce sync.Once
	funcMap         template.FuncMap
}

// NewPage returns a new Presentation from a corpus.
func NewPage(c *Corpus) *Page {
	if c == nil {
		panic("nil Corpus")
	}
	p := &Page{
		Corpus: c,
	}

	p.readTemplates()

	return p
}

func (p *Page) readTemplate(name string) *template.Template {

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

func (p *Page) readTemplates() {

	p.SidebarHTML = p.readTemplate("static/sidebar.html")
	p.PackageHTML = p.readTemplate("static/package.html")
}

// FuncMap defines template functions used in godoc templates.
func (p *Page) FuncMap() template.FuncMap {
	p.initFuncMapOnce.Do(p.initFuncMap)
	return p.funcMap
}

func (p *Page) initFuncMap() {
	if p.Corpus == nil {
		panic("nil Presentation.Corpus")
	}

	p.funcMap = template.FuncMap{
		"repeat":    strings.Repeat,
		"unescaped": unescaped,
	}
}

// unescaped 取消模板转义
func unescaped(x string) interface{} { return template.HTML(x) }

// RenderPackage render package html
func RenderPackage(p *Page, pkg *Package) ([]byte, error) {

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
