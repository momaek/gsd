package gsd

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/printer"
	"go/token"
	"html/template"
	"io"
	"log"
	"os"
	pathpkg "path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/miclle/gsd/static"
	"github.com/yosssi/gohtml"
)

// Fake relative package path for built-ins. Documentation for all globals
// (not just exported ones) will be shown for packages in this directory,
// and there will be no association of consts, vars, and factory functions
// with types (see issue 6645).
const builtinPkgPath = "builtin"

// PageType page type
type PageType string

const (
	// PackagePage package page type
	PackagePage PageType = "package"
	// TypePage type page type
	TypePage PageType = "type"
	// FuncPage func page type
	FuncPage PageType = "func"
)

// Page generates output from a corpus.
type Page struct {
	Corpus  *Corpus
	Package *Package
	Type    *doc.Type
	Func    *doc.Func

	LayoutHTML  *template.Template
	SidebarHTML *template.Template
	PackageHTML *template.Template
	TypeHTML    *template.Template
	FuncHTML    *template.Template
	ExampleHTML *template.Template

	Title string

	// TabWidth optionally specifies the tab width.
	TabWidth int

	ShowPlayground bool
	DeclLinks      bool

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
		Corpus:    c,
		Package:   pkg,
		Title:     pkg.Name,
		TabWidth:  4,
		DeclLinks: true,
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
	page.TypeHTML = page.readTemplate("type.html")
	page.FuncHTML = page.readTemplate("func.html")
}

// FuncMap defines template functions used in godoc templates.
func (page *Page) FuncMap() template.FuncMap {
	page.initFuncMapOnce.Do(page.initFuncMap)
	return page.funcMap
}

func (page *Page) initFuncMap() {
	page.funcMap = template.FuncMap{
		"repeat":    strings.Repeat,
		"since":     page.Corpus.pkgAPIInfo.sinceVersionFunc,
		"unescaped": unescaped,
		"srcID":     srcIDFunc,

		"display_private_indent": page.Corpus.DisplayPrivateIndent,
		"indent_filter":          page.Corpus.IndentFilter,

		// access to FileInfos (directory listings)
		"fileInfoName": fileInfoNameFunc,
		"fileInfoTime": fileInfoTimeFunc,

		// formatting of AST nodes
		"node":         page.nodeFunc,
		"node_html":    page.nodeHTMLFunc,
		"comment_html": commentHTMLFunc,
		"sanitize":     sanitizeFunc,

		// support for URL attributes
		"pkgLink":       pkgLinkFunc,
		"srcLink":       srcLinkFunc,
		"posLink_url":   newPosLinkURLFunc(srcPosLinkFunc),
		"docLink":       docLinkFunc,
		"queryLink":     queryLinkFunc,
		"srcBreadcrumb": srcBreadcrumbFunc,
		"srcToPkgLink":  srcToPkgLinkFunc,

		// formatting of Examples
		"example_html":   page.exampleHTMLFunc,
		"example_name":   page.exampleNameFunc,
		"example_suffix": page.exampleSuffixFunc,

		//
		"type_fields": TypeFields,

		// formatting of Notes
		"noteTitle": noteTitle,

		// Number operation
		"multiply": multiply,

		"inc": inc,
	}
}

// Render package page
func (page *Page) Render(writer io.Writer, t PageType) (err error) {

	if page.Corpus == nil || page.Package == nil {
		panic("page corpuus, package is nil")
	}

	if page.Sidebar, err = applyTemplate(page.SidebarHTML, "sidebar", page); err != nil {
		return err
	}

	switch t {
	case PackagePage:
		if page.Body, err = applyTemplate(page.PackageHTML, "package", page); err != nil {
			return err
		}

	case TypePage:
		if page.Body, err = applyTemplate(page.TypeHTML, "type", page); err != nil {
			return err
		}

	case FuncPage:
		if page.Body, err = applyTemplate(page.FuncHTML, "func", page); err != nil {
			return err
		}
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

// conversion / to -
func srcIDFunc(s string) string {
	s = strings.TrimPrefix(s, "/src/")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

// --------------------------------------------------------------------

func inc(a, b int) int      { return a + b }
func multiply(a, b int) int { return a * b }

func filenameFunc(path string) string {
	_, localname := pathpkg.Split(path)
	return localname
}

func fileInfoNameFunc(fi os.FileInfo) string {
	name := fi.Name()
	if fi.IsDir() {
		name += "/"
	}
	return name
}

func fileInfoTimeFunc(fi os.FileInfo) string {
	if t := fi.ModTime(); t.Unix() != 0 {
		return t.Local().String()
	}
	return "" // don't return epoch if time is obviously not set
}

// The strings in infoKinds must be properly html-escaped.
var infoKinds = [nKinds]string{
	PackageClause: "package&nbsp;clause",
	ImportDecl:    "import&nbsp;decl",
	ConstDecl:     "const&nbsp;decl",
	TypeDecl:      "type&nbsp;decl",
	VarDecl:       "var&nbsp;decl",
	FuncDecl:      "func&nbsp;decl",
	MethodDecl:    "method&nbsp;decl",
	Use:           "use",
}

func (page *Page) nodeFunc(info *Package, node interface{}) string {
	var buf bytes.Buffer
	page.writeNode(&buf, info, info.FSet, node)
	return buf.String()
}

func (page *Page) nodeHTMLFunc(info *Package, node interface{}, linkify bool) string {
	var buf1 bytes.Buffer
	page.writeNode(&buf1, info, info.FSet, node)

	var buf2 bytes.Buffer
	if n, _ := node.(ast.Node); n != nil && linkify && page.DeclLinks {
		LinkifyText(&buf2, buf1.Bytes(), n)
		if st, name := isStructTypeDecl(n); st != nil {
			addStructFieldIDAttributes(&buf2, name, st)
		}
	} else {
		FormatText(&buf2, buf1.Bytes(), -1, true, "", nil)
	}

	return buf2.String()
}

// isStructTypeDecl checks whether n is a struct declaration.
// It either returns a non-nil StructType and its name, or zero values.
func isStructTypeDecl(n ast.Node) (st *ast.StructType, name string) {
	gd, ok := n.(*ast.GenDecl)
	if !ok || gd.Tok != token.TYPE {
		return nil, ""
	}
	if gd.Lparen > 0 {
		// Parenthesized type. Who does that, anyway?
		// TODO: Reportedly gri does. Fix this to handle that too.
		return nil, ""
	}
	if len(gd.Specs) != 1 {
		return nil, ""
	}
	ts, ok := gd.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, ""
	}
	st, ok = ts.Type.(*ast.StructType)
	if !ok {
		return nil, ""
	}
	return st, ts.Name.Name
}

// addStructFieldIDAttributes modifies the contents of buf such that
// all struct fields of the named struct have <span id='name.Field'>
// in them, so people can link to /#Struct.Field.
func addStructFieldIDAttributes(buf *bytes.Buffer, name string, st *ast.StructType) {
	if st.Fields == nil {
		return
	}
	// needsLink is a set of identifiers that still need to be
	// linked, where value == key, to avoid an allocation in func
	// linkedField.
	needsLink := make(map[string]string)

	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		fieldName := f.Names[0].Name
		needsLink[fieldName] = fieldName
	}
	var newBuf bytes.Buffer
	foreachLine(buf.Bytes(), func(line []byte) {
		if fieldName := linkedField(line, needsLink); fieldName != "" {
			fmt.Fprintf(&newBuf, `<span id="%s.%s"></span>`, name, fieldName)
			delete(needsLink, fieldName)
		}
		newBuf.Write(line)
	})
	buf.Reset()
	buf.Write(newBuf.Bytes())
}

// foreachLine calls fn for each line of in, where a line includes
// the trailing "\n", except on the last line, if it doesn't exist.
func foreachLine(in []byte, fn func(line []byte)) {
	for len(in) > 0 {
		nl := bytes.IndexByte(in, '\n')
		if nl == -1 {
			fn(in)
			return
		}
		fn(in[:nl+1])
		in = in[nl+1:]
	}
}

// commentPrefix is the line prefix for comments after they've been HTMLified.
var commentPrefix = []byte(`<span class="comment">// `)

// linkedField determines whether the given line starts with an
// identifier in the provided ids map (mapping from identifier to the
// same identifier). The line can start with either an identifier or
// an identifier in a comment. If one matches, it returns the
// identifier that matched. Otherwise it returns the empty string.
func linkedField(line []byte, ids map[string]string) string {
	line = bytes.TrimSpace(line)

	// For fields with a doc string of the
	// conventional form, we put the new span into
	// the comment instead of the field.
	// The "conventional" form is a complete sentence
	// per https://golang.org/s/style#comment-sentences like:
	//
	//    // Foo is an optional Fooer to foo the foos.
	//    Foo Fooer
	//
	// In this case, we want the #StructName.Foo
	// link to make the browser go to the comment
	// line "Foo is an optional Fooer" instead of
	// the "Foo Fooer" line, which could otherwise
	// obscure the docs above the browser's "fold".
	//
	line = bytes.TrimPrefix(line, commentPrefix)

	id := scanIdentifier(line)
	if len(id) == 0 {
		// No leading identifier. Avoid map lookup for
		// somewhat common case.
		return ""
	}
	return ids[string(id)]
}

// scanIdentifier scans a valid Go identifier off the front of v and
// either returns a subslice of v if there's a valid identifier, or
// returns a zero-length slice.
func scanIdentifier(v []byte) []byte {
	var n int // number of leading bytes of v belonging to an identifier
	for {
		r, width := utf8.DecodeRune(v[n:])
		if !(isLetter(r) || n > 0 && isDigit(r)) {
			break
		}
		n += width
	}
	return v[:n]
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func commentHTMLFunc(comment string) string {
	var buf bytes.Buffer
	// TODO(gri) Provide list of words (e.g. function parameters)
	//           to be emphasized by ToHTML.
	doc.ToHTML(&buf, comment, nil) // does html-escaping
	return buf.String()
}

// sanitizeFunc sanitizes the argument src by replacing newlines with
// blanks, removing extra blanks, and by removing trailing whitespace
// and commas before closing parentheses.
func sanitizeFunc(src string) string {
	buf := make([]byte, len(src))
	j := 0      // buf index
	comma := -1 // comma index if >= 0
	for i := 0; i < len(src); i++ {
		ch := src[i]
		switch ch {
		case '\t', '\n', ' ':
			// ignore whitespace at the beginning, after a blank, or after opening parentheses
			if j == 0 {
				continue
			}
			if p := buf[j-1]; p == ' ' || p == '(' || p == '{' || p == '[' {
				continue
			}
			// replace all whitespace with blanks
			ch = ' '
		case ',':
			comma = j
		case ')', '}', ']':
			// remove any trailing comma
			if comma >= 0 {
				j = comma
			}
			// remove any trailing whitespace
			if j > 0 && buf[j-1] == ' ' {
				j--
			}
		default:
			comma = -1
		}
		buf[j] = ch
		j++
	}
	// remove trailing blank, if any
	if j > 0 && buf[j-1] == ' ' {
		j--
	}
	return string(buf[:j])
}

func pkgLinkFunc(path string) string {
	// because of the irregular mapping under goroot
	// we need to correct certain relative paths
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "src/")
	path = strings.TrimPrefix(path, "pkg/")
	return "pkg/" + path
}

// srcToPkgLinkFunc builds an <a> tag linking to the package
// documentation of relpath.
func srcToPkgLinkFunc(relpath string) string {
	relpath = pkgLinkFunc(relpath)
	relpath = pathpkg.Dir(relpath)
	if relpath == "pkg" {
		return `<a href="/pkg">Index</a>`
	}
	return fmt.Sprintf(`<a href="/%s">%s</a>`, relpath, relpath[len("pkg/"):])
}

// srcBreadcrumbFun converts each segment of relpath to a HTML <a>.
// Each segment links to its corresponding src directories.
func srcBreadcrumbFunc(relpath string) string {
	segments := strings.Split(relpath, "/")
	var buf bytes.Buffer
	var selectedSegment string
	var selectedIndex int

	if strings.HasSuffix(relpath, "/") {
		// relpath is a directory ending with a "/".
		// Selected segment is the segment before the last slash.
		selectedIndex = len(segments) - 2
		selectedSegment = segments[selectedIndex] + "/"
	} else {
		selectedIndex = len(segments) - 1
		selectedSegment = segments[selectedIndex]
	}

	for i := range segments[:selectedIndex] {
		buf.WriteString(fmt.Sprintf(`<a href="/%s">%s</a>/`,
			strings.Join(segments[:i+1], "/"),
			segments[i],
		))
	}

	buf.WriteString(`<span class="text-muted">`)
	buf.WriteString(selectedSegment)
	buf.WriteString(`</span>`)
	return buf.String()
}

func newPosLinkURLFunc(srcPosLinkFunc func(s string, line, low, high int) string) func(info *Package, n interface{}) string {
	// n must be an ast.Node or a *doc.Note
	return func(info *Package, n interface{}) string {
		var pos, end token.Pos

		switch n := n.(type) {
		case ast.Node:
			pos = n.Pos()
			end = n.End()
		case *doc.Note:
			pos = n.Pos
			end = n.End
		default:
			panic(fmt.Sprintf("wrong type for posLink_url template formatter: %T", n))
		}

		var relpath string
		var line int
		var low, high int // selection offset range

		if pos.IsValid() {
			p := info.FSet.Position(pos)
			relpath = p.Filename
			line = p.Line
			low = p.Offset
		}
		if end.IsValid() {
			high = info.FSet.Position(end).Offset
		}

		return srcPosLinkFunc(relpath, line, low, high)
	}
}

func srcPosLinkFunc(s string, line, low, high int) string {
	s = srcLinkFunc(s)
	var buf bytes.Buffer
	template.HTMLEscape(&buf, []byte(s))
	// selection ranges are of form "s=low:high"
	if low < high {
		fmt.Fprintf(&buf, "?s=%d:%d", low, high) // no need for URL escaping
		// if we have a selection, position the page
		// such that the selection is a bit below the top
		line -= 10
		if line < 1 {
			line = 1
		}
	}
	// line id's in html-printed source are of the
	// form "L%d" where %d stands for the line number
	if line > 0 {
		fmt.Fprintf(&buf, "#L%d", line) // no need for URL escaping
	}
	return buf.String()
}

func srcLinkFunc(s string) string {
	s = pathpkg.Clean("/" + s)
	if !strings.HasPrefix(s, "/src/") {
		s = "/src" + s
	}
	return s
}

// queryLinkFunc returns a URL for a line in a source file with a highlighted
// query term.
// s is expected to be a path to a source file.
// query is expected to be a string that has already been appropriately escaped
// for use in a URL query.
func queryLinkFunc(s, query string, line int) string {
	url := pathpkg.Clean("/"+s) + "?h=" + query
	if line > 0 {
		url += "#L" + strconv.Itoa(line)
	}
	return url
}

func docLinkFunc(s string, ident string) string {
	return pathpkg.Clean("/pkg/"+s) + "/#" + ident
}

func (page *Page) exampleHTMLFunc(info *Package, funcName string) string {
	var buf bytes.Buffer
	for _, eg := range info.Examples {
		name := stripExampleSuffix(eg.Name)

		if name != funcName {
			continue
		}

		// print code
		cnode := &printer.CommentedNode{Node: eg.Code, Comments: eg.Comments}
		code := page.nodeHTMLFunc(info, cnode, true)
		out := eg.Output
		wholeFile := true

		// Additional formatting if this is a function body.
		if n := len(code); n >= 2 && code[0] == '{' && code[n-1] == '}' {
			wholeFile = false
			// remove surrounding braces
			code = code[1 : n-1]
			// unindent
			code = replaceLeadingIndentation(code, strings.Repeat(" ", page.TabWidth), "")
			// remove output comment
			if loc := exampleOutputRx.FindStringIndex(code); loc != nil {
				code = strings.TrimSpace(code[:loc[0]])
			}
		}

		// Write out the playground code in standard Go style
		// (use tabs, no comment highlight, etc).
		play := ""
		if eg.Play != nil && page.ShowPlayground {
			var buf bytes.Buffer
			eg.Play.Comments = filterOutBuildAnnotations(eg.Play.Comments)
			if err := format.Node(&buf, info.FSet, eg.Play); err != nil {
				log.Print(err)
			} else {
				play = buf.String()
			}
		}

		// Drop output, as the output comment will appear in the code.
		if wholeFile && play == "" {
			out = ""
		}

		if page.ExampleHTML == nil {
			out = ""
			return ""
		}

		err := page.ExampleHTML.Execute(&buf, struct {
			Name, Doc, Code, Play, Output string
		}{eg.Name, eg.Doc, code, play, out})
		if err != nil {
			log.Print(err)
		}
	}
	return buf.String()
}

func filterOutBuildAnnotations(cg []*ast.CommentGroup) []*ast.CommentGroup {
	if len(cg) == 0 {
		return cg
	}

	for i := range cg {
		if !strings.HasPrefix(cg[i].Text(), "+build ") {
			// Found the first non-build tag, return from here until the end
			// of the slice.
			return cg[i:]
		}
	}

	// There weren't any non-build tags, return an empty slice.
	return []*ast.CommentGroup{}
}

// exampleNameFunc takes an example function name and returns its display
// name. For example, "Foo_Bar_quux" becomes "Foo.Bar (Quux)".
func (page *Page) exampleNameFunc(s string) string {
	name, suffix := splitExampleName(s)
	// replace _ with . for method names
	name = strings.Replace(name, "_", ".", 1)
	// use "Package" if no name provided
	if name == "" {
		name = "Package"
	}
	return name + suffix
}

// exampleSuffixFunc takes an example function name and returns its suffix in
// parenthesized form. For example, "Foo_Bar_quux" becomes " (Quux)".
func (page *Page) exampleSuffixFunc(name string) string {
	_, suffix := splitExampleName(name)
	return suffix
}

func noteTitle(note string) string {
	return strings.Title(strings.ToLower(note))
}

func startsWithUppercase(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

var exampleOutputRx = regexp.MustCompile(`(?i)//[[:space:]]*(unordered )?output:`)

// stripExampleSuffix strips lowercase braz in Foo_braz or Foo_Bar_braz from name
// while keeping uppercase Braz in Foo_Braz.
func stripExampleSuffix(name string) string {
	if i := strings.LastIndex(name, "_"); i != -1 {
		if i < len(name)-1 && !startsWithUppercase(name[i+1:]) {
			name = name[:i]
		}
	}
	return name
}

func splitExampleName(s string) (name, suffix string) {
	i := strings.LastIndex(s, "_")
	if 0 <= i && i < len(s)-1 && !startsWithUppercase(s[i+1:]) {
		name = s[:i]
		suffix = " (" + strings.Title(s[i+1:]) + ")"
		return
	}
	name = s
	return
}

// replaceLeadingIndentation replaces oldIndent at the beginning of each line
// with newIndent. This is used for formatting examples. Raw strings that
// span multiple lines are handled specially: oldIndent is not removed (since
// go/printer will not add any indentation there), but newIndent is added
// (since we may still want leading indentation).
func replaceLeadingIndentation(body, oldIndent, newIndent string) string {
	// Handle indent at the beginning of the first line. After this, we handle
	// indentation only after a newline.
	var buf bytes.Buffer
	if strings.HasPrefix(body, oldIndent) {
		buf.WriteString(newIndent)
		body = body[len(oldIndent):]
	}

	// Use a state machine to keep track of whether we're in a string or
	// rune literal while we process the rest of the code.
	const (
		codeState = iota
		runeState
		interpretedStringState
		rawStringState
	)
	searchChars := []string{
		"'\"`\n", // codeState
		`\'`,     // runeState
		`\"`,     // interpretedStringState
		"`\n",    // rawStringState
		// newlineState does not need to search
	}
	state := codeState
	for {
		i := strings.IndexAny(body, searchChars[state])
		if i < 0 {
			buf.WriteString(body)
			break
		}
		c := body[i]
		buf.WriteString(body[:i+1])
		body = body[i+1:]
		switch state {
		case codeState:
			switch c {
			case '\'':
				state = runeState
			case '"':
				state = interpretedStringState
			case '`':
				state = rawStringState
			case '\n':
				if strings.HasPrefix(body, oldIndent) {
					buf.WriteString(newIndent)
					body = body[len(oldIndent):]
				}
			}

		case runeState:
			switch c {
			case '\\':
				r, size := utf8.DecodeRuneInString(body)
				buf.WriteRune(r)
				body = body[size:]
			case '\'':
				state = codeState
			}

		case interpretedStringState:
			switch c {
			case '\\':
				r, size := utf8.DecodeRuneInString(body)
				buf.WriteRune(r)
				body = body[size:]
			case '"':
				state = codeState
			}

		case rawStringState:
			switch c {
			case '`':
				state = codeState
			case '\n':
				buf.WriteString(newIndent)
			}
		}
	}
	return buf.String()
}

// writeNode writes the AST node x to w.
//
// The provided fset must be non-nil. The pageInfo is optional. If
// present, the pageInfo is used to add comments to struct fields to
// say which version of Go introduced them.
func (page *Page) writeNode(w io.Writer, pageInfo *Package, fset *token.FileSet, x interface{}) {
	// convert trailing tabs into spaces using a tconv filter
	// to ensure a good outcome in most browsers (there may still
	// be tabs in comments and strings, but converting those into
	// the right number of spaces is much harder)
	//
	// TODO(gri) rethink printer flags - perhaps tconv can be eliminated
	//           with an another printer mode (which is more efficiently
	//           implemented in the printer than here with another layer)

	var pkgName, structName string
	var apiInfo pkgAPIVersions
	if gd, ok := x.(*ast.GenDecl); ok && pageInfo != nil && pageInfo.DocPackage != nil &&
		page.Corpus != nil && gd.Tok == token.TYPE && len(gd.Specs) != 0 {
		pkgName = pageInfo.DocPackage.ImportPath
		if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
			if _, ok := ts.Type.(*ast.StructType); ok {
				structName = ts.Name.Name
			}
		}
		apiInfo = page.Corpus.pkgAPIInfo[pkgName]
	}

	var out = w
	var buf bytes.Buffer
	if structName != "" {
		out = &buf
	}

	mode := printer.TabIndent | printer.UseSpaces
	err := (&printer.Config{Mode: mode, Tabwidth: page.TabWidth}).Fprint(&tconv{p: page, output: out}, fset, x)
	if err != nil {
		log.Print(err)
	}

	// Add comments to struct fields saying which Go version introduced them.
	if structName != "" {
		fieldSince := apiInfo.fieldSince[structName]
		typeSince := apiInfo.typeSince[structName]
		// Add/rewrite comments on struct fields to note which Go version added them.
		var buf2 bytes.Buffer
		buf2.Grow(buf.Len() + len(" // Added in Go 1.n")*10)
		bs := bufio.NewScanner(&buf)
		for bs.Scan() {
			line := bs.Bytes()
			field := firstIdent(line)
			var since string
			if field != "" {
				since = fieldSince[field]
				if since != "" && since == typeSince {
					// Don't highlight field versions if they were the
					// same as the struct itself.
					since = ""
				}
			}
			if since == "" {
				buf2.Write(line)
			} else {
				if bytes.Contains(line, slashSlash) {
					line = bytes.TrimRight(line, " \t.")
					buf2.Write(line)
					buf2.WriteString("; added in Go ")
				} else {
					buf2.Write(line)
					buf2.WriteString(" // Go ")
				}
				buf2.WriteString(since)
			}
			buf2.WriteByte('\n')
		}
		w.Write(buf2.Bytes())
	}
}

var slashSlash = []byte("//")

// WriteNode writes x to w.
// TODO(bgarcia) Is this method needed? It's just a wrapper for p.writeNode.
func (page *Page) WriteNode(w io.Writer, fset *token.FileSet, x interface{}) {
	page.writeNode(w, nil, fset, x)
}

// firstIdent returns the first identifier in x.
// This actually parses "identifiers" that begin with numbers too, but we
// never feed it such input, so it's fine.
func firstIdent(x []byte) string {
	x = bytes.TrimSpace(x)
	i := bytes.IndexFunc(x, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	if i == -1 {
		return string(x)
	}
	return string(x[:i])
}
