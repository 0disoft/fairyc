package languageserver

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"sort"
	"strings"
)

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}
type Completion struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

func (s *server) definition(d *document, offset int) []Location {
	locations := []Location{}
	if d.View == nil {
		return locations
	}
	for _, r := range d.View.result.Resolutions {
		use, ok := d.View.file(r.Use)
		if !ok || use.Path != d.Path || offset < r.Use.StartByte || offset >= r.Use.EndByte {
			continue
		}
		target, ok := d.View.file(r.Target.Declaration)
		if !ok || target.Path == "" {
			continue
		}
		locations = append(locations, Location{fileURI(target.Path), textRange(target.Text, r.Target.Declaration.StartByte, r.Target.Declaration.EndByte)})
	}
	return locations
}
func identifierByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
func (s *server) completion(d *document, offset int) []Completion {
	out := []Completion{}
	scanned := lexer.Lex(source.NewFile("main.fy", d.Text), source.NewBudget(source.DefaultLimits(), len(d.Text)))
	for _, token := range scanned.Tokens {
		if token.Span.StartByte < offset && offset < token.Span.EndByte && (token.Kind == lexer.String || token.Kind == lexer.ByteString || token.Kind == lexer.CString || token.Trivia == lexer.TriviaLineComment || token.Trivia == lexer.TriviaDocumentationLine || token.Trivia == lexer.TriviaBlockComment) {
			return out
		}
	}
	start := offset
	for start > 0 && identifierByte(d.Text[start-1]) {
		start--
	}
	prefix := string(d.Text[start:offset])
	if start > 0 && d.Text[start-1] == '.' {
		return out
	}
	symbols := map[string]Completion{}
	add := func(label string, kind int, detail string) {
		if strings.HasPrefix(label, prefix) {
			symbols[label] = Completion{label, kind, detail}
		}
	}
	words, _ := lexer.EditorVocabulary()
	for _, word := range words {
		add(word, 14, "keyword")
	}
	for _, word := range []string{"void", "bool", "u8", "i32", "u32", "usize", "str", "CStr"} {
		add(word, 7, "built-in type")
	}
	view := d.View
	if view != nil {
		var local *ast.Module
		var localKey sourceKey
		for key, file := range view.files {
			if file.Path == d.Path {
				local = file.Module
				localKey = key
				break
			}
		}
		if local == nil && d.LastClean != nil {
			for key, file := range d.LastClean.files {
				if file.Path == d.Path {
					local = file.Module
					localKey = key
					break
				}
			}
		}
		if start >= 2 && string(d.Text[start-2:start]) == "::" {
			// Only exact source import aliases are offered; no guessed member/type completion.
			aliasEnd := start - 2
			aliasStart := aliasEnd
			for aliasStart > 0 && identifierByte(d.Text[aliasStart-1]) {
				aliasStart--
			}
			alias := string(d.Text[aliasStart:aliasEnd])
			symbols = map[string]Completion{}
			if local != nil {
				for _, imported := range local.Imports {
					if imported.Alias == alias {
						provider, exact := view.imports[sourceKey{localKey.Package, imported.Path}]
						for key, file := range view.files {
							if exact && key.Package == provider && file.Module != nil && file.Module.Path == imported.Path {
								moduleSymbols(file.Module, true, add)
							}
						}
					}
				}
			}
		} else if local != nil {
			moduleSymbols(local, false, add)
			for _, imported := range local.Imports {
				add(imported.Alias, 9, "module")
			}
			for _, fn := range local.Functions {
				if fn.Body != nil && fn.Body.Span.StartByte <= offset && offset <= fn.Body.Span.EndByte {
					for _, p := range fn.Parameters {
						add(p.Name, 6, "parameter")
					}
					blockSymbols(fn.Body, offset, add)
				}
			}
		}
	}
	for _, item := range symbols {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	if len(out) > 256 {
		out = out[:256]
	}
	return out
}
func moduleSymbols(m *ast.Module, publicOnly bool, add func(string, int, string)) {
	for _, f := range m.Functions {
		if !publicOnly || f.Public {
			if f.Owner == "" {
				add(f.Name, 3, "function")
			}
		}
	}
	for _, f := range m.ForeignFunctions {
		if !publicOnly || f.Public {
			add(f.Name, 3, "foreign function")
		}
	}
	for _, t := range m.Structs {
		if !publicOnly || t.Public {
			add(t.Name, 22, "struct")
		}
	}
	for _, t := range m.Resources {
		if !publicOnly || t.Public {
			add(t.Name, 7, "resource")
		}
	}
	for _, t := range m.Enums {
		if !publicOnly || t.Public {
			add(t.Name, 13, "enum")
		}
	}
	for _, c := range m.Constants {
		if !publicOnly || c.Public {
			add(c.Name, 21, "constant")
		}
	}
}
func blockSymbols(b *ast.Block, pos int, add func(string, int, string)) {
	if b == nil || pos < b.Span.StartByte || pos > b.Span.EndByte {
		return
	}
	for _, stmt := range b.Statements {
		if stmt.Span.StartByte >= pos {
			break
		}
		if stmt.Binding != nil && stmt.Span.EndByte <= pos {
			add(stmt.Binding.Name, 6, "local")
		}
		if stmt.For != nil && stmt.For.Span.StartByte <= pos && pos <= stmt.For.Span.EndByte {
			if init := stmt.For.Initializer; init != nil && init.Binding != nil {
				add(init.Binding.Name, 6, "loop local")
			}
			blockSymbols(stmt.For.Body, pos, add)
		}
		if stmt.ByteFor != nil && stmt.ByteFor.Body != nil && stmt.ByteFor.Body.Span.StartByte <= pos && pos <= stmt.ByteFor.Body.Span.EndByte {
			add(stmt.ByteFor.Name, 6, "loop local")
			blockSymbols(stmt.ByteFor.Body, pos, add)
		}
		expressionSymbols(stmt.Expression, pos, add)
	}
	expressionSymbols(b.Tail, pos, add)
}
func expressionSymbols(e *ast.Expression, pos int, add func(string, int, string)) {
	if e == nil || pos < e.Span.StartByte || pos > e.Span.EndByte {
		return
	}
	blockSymbols(e.Block, pos, add)
	for _, child := range e.Children {
		expressionSymbols(child, pos, add)
	}
}
