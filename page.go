package gsd

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/miclle/gsd/static"
	"github.com/yosssi/gohtml"
)

// Page generates output from a corpus.
type Page struct {
	Corpus  *Corpus
	Package *Package

	LayoutHTML  *template.Template
	SidebarHTML *template.Template
	PackageHTML *template.Template

	Title string

	Sidebar []byte // Sidebar content
	Body    []byte // Main content

	initFuncMapOnce sync.Once
	funcMap         template.FuncMap
}

// NewPage returns a new Presentation from a corpus.
func NewPage(c *Corpus, pkg *Package) *Page {
	if c == nil {
		panic("nil Corpus")
	}
	page := &Page{
		Corpus:  c,
		Package: pkg,
		Title:   pkg.Name,
	}

	page.readTemplates()

	return page
}

func (page *Page) readTemplate(name string) *template.Template {

	data, exists := static.Files[name]
	if !exists {
		panic("file not found")
	}

	t, err := template.New(name).Funcs(page.FuncMap()).Parse(data)
	if err != nil {
		panic(err)
	}

	return t
}

func (page *Page) readTemplates() {
	page.LayoutHTML = page.readTemplate("layout.html")
	page.SidebarHTML = page.readTemplate("sidebar.html")
	page.PackageHTML = page.readTemplate("package.html")
}

// FuncMap defines template functions used in godoc templates.
func (page *Page) FuncMap() template.FuncMap {
	page.initFuncMapOnce.Do(page.initFuncMap)
	return page.funcMap
}

func (page *Page) initFuncMap() {
	page.funcMap = template.FuncMap{
		"repeat":    strings.Repeat,
		"unescaped": unescaped,
	}
}

// Render package page
func (page *Page) Render(writer io.Writer) (err error) {

	if page.Corpus == nil || page.Package == nil {
		panic("page corpuus, package is nil")
	}

	if page.Sidebar, err = applyTemplate(page.SidebarHTML, "sidebar", page); err != nil {
		return err
	}

	if page.Body, err = applyTemplate(page.PackageHTML, "body", page); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := page.LayoutHTML.Execute(&buf, page); err != nil {
		log.Printf("%s.Execute: %s", "layout", err)
		return err
	}

	// format template html
	gohtml.Condense = true
	_, err = writer.Write(gohtml.FormatBytes(buf.Bytes()))

	return err
}

func applyTemplate(t *template.Template, name string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Printf("%s.Execute: %s", name, err)
		return nil, err
	}
	return buf.Bytes(), nil
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
