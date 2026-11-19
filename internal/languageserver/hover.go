package languageserver

import (
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

type hoverResult struct {
	Contents hoverContents `json:"contents"`
	Range    Range         `json:"range"`
}

type hoverContents struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

const maxHoverBytes = 4096

func (s *server) hover(d *document, offset int) *hoverResult {
	view := d.View
	if view == nil || !view.clean {
		return nil
	}
	s.touchAnalysis(view)
	var scratch [8]int
	var selected *semantic.Resolution
	for _, index := range view.definitionCandidates(d.Path, offset, scratch[:0]) {
		resolution := &view.result.Resolutions[index]
		if selected != nil && selected.Target != resolution.Target {
			return nil
		}
		selected = resolution
	}
	if selected != nil {
		target := selected.Target
		text := target.Kind + " " + target.Name
		if signature := view.hoverSignature(target.Declaration); signature != "" {
			text = signature
		} else if typ := view.hoverType(selected.Use); typ != "" {
			text += ": " + typ
		}
		return newHover(d, selected.Use.StartByte, selected.Use.EndByte, text)
	}
	module, key := view.completionLocal(d.Path)
	if module == nil {
		return nil
	}
	for _, fn := range module.Functions {
		if fn.NameSpan.StartByte <= offset && offset < fn.NameSpan.EndByte {
			span := hir.Span{Package: key.Package, Source: key.Path, StartByte: fn.NameSpan.StartByte, EndByte: fn.NameSpan.EndByte}
			return newHover(d, span.StartByte, span.EndByte, view.hoverSignature(span))
		}
	}
	for _, fn := range module.ForeignFunctions {
		if fn.NameSpan.StartByte <= offset && offset < fn.NameSpan.EndByte {
			span := hir.Span{Package: key.Package, Source: key.Path, StartByte: fn.NameSpan.StartByte, EndByte: fn.NameSpan.EndByte}
			return newHover(d, span.StartByte, span.EndByte, view.hoverSignature(span))
		}
	}
	return nil
}

func newHover(d *document, start, end int, value string) *hoverResult {
	if value == "" || len(value) > maxHoverBytes || start < 0 || end <= start || end > len(d.Text) {
		return nil
	}
	return &hoverResult{hoverContents{"plaintext", value}, d.View.sourceRange(d.Path, d.Text, start, end)}
}

func (a *analysis) hoverSignature(declaration hir.Span) string {
	file, ok := a.file(declaration)
	if !ok || file.Module == nil {
		return ""
	}
	signature := func(start, end int) string {
		if start < 0 || end <= start || end > len(file.Text) || end-start > maxHoverBytes {
			return ""
		}
		return strings.TrimSpace(string(file.Text[start:end]))
	}
	for _, fn := range file.Module.Functions {
		if fn.NameSpan.StartByte == declaration.StartByte && fn.NameSpan.EndByte == declaration.EndByte && fn.Body != nil {
			return signature(fn.Span.StartByte, fn.Body.Span.StartByte)
		}
	}
	for _, fn := range file.Module.ForeignFunctions {
		if fn.NameSpan.StartByte == declaration.StartByte && fn.NameSpan.EndByte == declaration.EndByte {
			return signature(fn.Span.StartByte, fn.Span.EndByte)
		}
	}
	return ""
}

func (a *analysis) hoverType(use hir.Span) string {
	typ := hir.TypeInvalid
	for _, fact := range a.result.Types {
		if fact.Expression != use || fact.CanonicalType == hir.TypeInvalid {
			continue
		}
		if typ != hir.TypeInvalid && typ != fact.CanonicalType {
			return ""
		}
		typ = fact.CanonicalType
	}
	if typ == hir.TypeInvalid {
		return ""
	}
	return typ.String()
}
