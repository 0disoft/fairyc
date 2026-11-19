package languageserver

import (
	"bytes"
	"errors"
	"slices"
	"sort"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

var errRenameUnavailable = errors.New("rename requires a verified local binding or private function")
var errRenameChangesResolution = errors.New("rename changes source bindings or fails compiler checks")

type renamePlan struct {
	target semantic.SymbolCoordinate
	spans  []tokenSpan
	cursor tokenSpan
}

func (s *server) planRename(d *document, offset int) (renamePlan, error) {
	view := d.View
	if view == nil || !view.clean || view.result.RequiresGenericProvenance {
		return renamePlan{}, errRenameUnavailable
	}
	module, key := view.completionLocal(d.Path)
	if module == nil {
		return renamePlan{}, errRenameUnavailable
	}
	target, found, err := s.referenceTarget(d, offset)
	if errors.Is(err, errAmbiguousReference) {
		return renamePlan{}, errRenameUnavailable
	}
	if err != nil {
		return renamePlan{}, err
	}
	at := offset
	if found {
		file, ok := view.file(target.Declaration)
		if !ok || file.Path != d.Path {
			return renamePlan{}, errRenameUnavailable
		}
		at = target.Declaration.StartByte
	}
	name, kind := "", ""
	decl := hir.Span{Source: key.Path, Package: key.Package}
	for _, binding := range module.Bindings {
		if binding.NameSpan.StartByte <= at && at < binding.NameSpan.EndByte {
			name, kind = binding.Name, "local"
			decl.StartByte, decl.EndByte = binding.NameSpan.StartByte, binding.NameSpan.EndByte
			break
		}
	}
	for _, fn := range module.Functions {
		if fn.NameSpan.StartByte <= at && at < fn.NameSpan.EndByte && !fn.Public && fn.Owner == "" &&
			fn.Name != "main" && len(fn.Attributes) == 0 && len(fn.TypeParameters) == 0 {
			name, kind = fn.Name, "function"
			decl.StartByte, decl.EndByte = fn.NameSpan.StartByte, fn.NameSpan.EndByte
			break
		}
	}
	if name == "" || found && (target.Name != name || target.Kind != kind) {
		return renamePlan{}, errRenameUnavailable
	}
	if !found {
		target = semantic.SymbolCoordinate{Package: key.Package, Module: module.Path, Name: name, Kind: kind, Declaration: decl}
	}
	declaration := tokenSpan{decl.StartByte, decl.EndByte}
	if !identifierSpan(d.Text, declaration) || string(d.Text[declaration.start:declaration.end]) != name {
		return renamePlan{}, errRenameUnavailable
	}
	spans := map[tokenSpan]struct{}{declaration: {}}
	for i, resolution := range view.result.Resolutions {
		if i%256 == 0 && s.ctx != nil && s.ctx.Err() != nil {
			return renamePlan{}, s.ctx.Err()
		}
		if resolution.Target != target {
			continue
		}
		file, ok := view.file(resolution.Use)
		span := tokenSpan{resolution.Use.StartByte, resolution.Use.EndByte}
		if !ok || file.Path != d.Path || !identifierSpan(d.Text, span) || string(d.Text[span.start:span.end]) != name {
			return renamePlan{}, errRenameUnavailable
		}
		spans[span] = struct{}{}
		if len(spans) > maxReferenceLocations {
			return renamePlan{}, errReferenceLimit
		}
	}
	// A shorthand field has one spelling for both the field and the value.
	// Neither name can be edited in isolation without an explicit expansion.
	tokens := lexer.Lex(source.NewFile(key.Path, d.Text), source.NewBudget(source.DefaultLimits(), len(d.Text)))
	if len(tokens.Diagnostics) > 0 {
		return renamePlan{}, errRenameUnavailable
	}
	var plan renamePlan
	plan.target = target
	previous := ""
	for _, token := range tokens.Tokens {
		if token.Kind == lexer.Trivia || token.Kind == lexer.EOF {
			continue
		}
		span := tokenSpan{token.Span.StartByte, token.Span.EndByte}
		if _, matches := spans[span]; matches {
			if previous == "." {
				return renamePlan{}, errRenameUnavailable
			}
			plan.spans = append(plan.spans, span)
			if span.start <= offset && offset < span.end {
				plan.cursor = span
			}
		}
		previous = token.Text(d.Text)
	}
	if len(plan.spans) != len(spans) || plan.cursor.start == plan.cursor.end {
		return renamePlan{}, errRenameUnavailable
	}
	return plan, nil
}

func (s *server) rename(d *document, plan renamePlan, name string) ([]documentEdit, error) {
	keywords, _ := lexer.EditorVocabulary()
	if len(name) > 255 || !source.ValidIdentifier(name) || slices.Contains(keywords, name) {
		return nil, errors.New("new name must be a Fairy C identifier of up to 255 bytes")
	}
	if plan.target.Kind == "function" && name == "main" {
		return nil, errors.New("main is reserved for the program entry point")
	}
	if name == plan.target.Name {
		return []documentEdit{}, nil
	}
	delta := len(name) - len(plan.target.Name)
	size := len(d.Text) + delta*len(plan.spans)
	if size > s.documentByteLimit(d.URI) {
		return nil, errors.New("renamed document exceeds the editor byte limit")
	}
	text := make([]byte, 0, size)
	edits := make([]textEdit, 0, len(plan.spans))
	index := newPositionIndex(d.Text)
	previousEnd := 0
	for _, span := range plan.spans {
		text = append(text, d.Text[previousEnd:span.start]...)
		text = append(text, name...)
		edits = append(edits, textEdit{Range: Range{index.bytePosition(span.start), index.bytePosition(span.end)}, NewText: name})
		previousEnd = span.end
	}
	text = append(text, d.Text[previousEnd:]...)
	rechecked, err := s.probeDocument(d, text)
	if err != nil {
		return nil, err
	}
	if rechecked == nil || !rechecked.clean || !sameRenameBindings(d, rechecked, text, plan, name, delta) {
		return nil, errRenameChangesResolution
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.ctx.Err()
	}
	return []documentEdit{{TextDocument: documentID{URI: d.URI, Version: d.Version}, Edits: edits}}, nil
}

func sameRenameBindings(d *document, after *analysis, text []byte, plan renamePlan, name string, delta int) bool {
	before := d.View
	if len(before.files) != len(after.files) || len(before.result.Resolutions) != len(after.result.Resolutions) {
		return false
	}
	for key, file := range before.files {
		current, ok := after.files[key]
		expected := file.Text
		if file.Path == d.Path {
			expected = text
		}
		if !ok || file.Path != current.Path || !bytes.Equal(expected, current.Text) {
			return false
		}
	}
	translate := func(span hir.Span) hir.Span {
		if file, ok := before.file(span); ok && file.Path == d.Path {
			shift := func(offset int) int {
				return offset + delta*sort.Search(len(plan.spans), func(i int) bool { return plan.spans[i].end > offset })
			}
			span.StartByte, span.EndByte = shift(span.StartByte), shift(span.EndByte)
		}
		return span
	}
	found := make(map[semantic.Resolution]int, len(after.result.Resolutions))
	for _, use := range after.result.Resolutions {
		found[use]++
	}
	for _, use := range before.result.Resolutions {
		if use.Target == plan.target {
			use.Target.Name = name
		}
		use.Use = translate(use.Use)
		use.Target.Declaration = translate(use.Target.Declaration)
		if found[use] == 0 {
			return false
		}
		found[use]--
	}
	return true
}
