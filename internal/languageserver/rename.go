package languageserver

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

var errRenameUnavailable = errors.New("rename requires a verified local binding, function parameter, workspace callable, field, or enum variant")
var errRenameChangesResolution = errors.New("rename changes source bindings or fails compiler checks")

type renamePlan struct {
	target semantic.SymbolCoordinate
	name   string
	files  []renameFile
	cursor tokenSpan
}

type renameFile struct {
	sourceView
	id    editableDocumentID
	spans []tokenSpan
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
	declarationFile := sourceView{Path: d.Path, Text: d.Text, Module: module}
	if found {
		file, ok := view.file(target.Declaration)
		if !ok || file.Path == "" || file.Module == nil {
			return renamePlan{}, errRenameUnavailable
		}
		declarationFile, module = file, file.Module
		at = target.Declaration.StartByte
	}
	name, owner, kind := "", "", ""
	decl := hir.Span{Source: key.Path, Package: key.Package}
	for _, binding := range module.Bindings {
		if binding.NameSpan.StartByte <= at && at < binding.NameSpan.EndByte {
			name, kind = binding.Name, "local"
			decl.StartByte, decl.EndByte = binding.NameSpan.StartByte, binding.NameSpan.EndByte
			break
		}
	}
	for _, fn := range module.Functions {
		if fn.Owner == "" && len(fn.TypeParameters) == 0 {
			for _, parameter := range fn.Parameters {
				if parameter.NameSpan.StartByte <= at && at < parameter.NameSpan.EndByte {
					name, kind = parameter.Name, "parameter"
					decl.StartByte, decl.EndByte = parameter.NameSpan.StartByte, parameter.NameSpan.EndByte
					break
				}
			}
		}
		if fn.NameSpan.StartByte <= at && at < fn.NameSpan.EndByte && (!fn.Public || view.root != "") &&
			(fn.Name != "main" || fn.Owner != "") && len(fn.Attributes) == 0 && len(fn.TypeParameters) == 0 {
			name, owner, kind = fn.Name, fn.Owner, "function"
			decl.StartByte, decl.EndByte = fn.NameSpan.StartByte, fn.NameSpan.EndByte
			break
		}
	}
	if name == "" {
		name, owner, kind, decl = renameMemberDeclaration(module, at, decl)
	}
	qualified := name
	if owner != "" {
		qualified = owner + "::" + name
	}
	if name == "" || found && (target.Name != qualified || target.Kind != kind) {
		return renamePlan{}, errRenameUnavailable
	}
	if !found {
		target = semantic.SymbolCoordinate{Package: key.Package, Module: module.Path, Name: qualified, Kind: kind, Declaration: decl}
	}
	files := make(map[string]sourceView)
	spans := make(map[string]map[tokenSpan]bool)
	count := 0
	add := func(file sourceView, span tokenSpan, selector bool) error {
		if file.Path == "" || (view.root != "" && !contained(view.root, file.Path)) ||
			!identifierSpan(file.Text, span) || string(file.Text[span.start:span.end]) != name {
			return errRenameUnavailable
		}
		if spans[file.Path] == nil {
			files[file.Path] = file
			spans[file.Path] = make(map[tokenSpan]bool)
		}
		if _, exists := spans[file.Path][span]; !exists {
			if count == maxReferenceLocations {
				return errReferenceLimit
			}
			count++
		}
		spans[file.Path][span] = spans[file.Path][span] || selector
		return nil
	}
	if err := add(declarationFile, tokenSpan{decl.StartByte, decl.EndByte}, false); err != nil {
		return renamePlan{}, err
	}
	selectors := make(map[hir.Span]bool)
	for _, selector := range view.result.SelectorResolutions {
		if selector.Target == target {
			selectors[selector.Use] = true
		}
	}
	for i, resolution := range view.result.Resolutions {
		if i%256 == 0 && s.ctx != nil && s.ctx.Err() != nil {
			return renamePlan{}, s.ctx.Err()
		}
		if resolution.Target != target {
			if selectors[resolution.Use] {
				// A shorthand initializer is both a field and a value use.
				return renamePlan{}, errRenameUnavailable
			}
			continue
		}
		file, ok := view.file(resolution.Use)
		end := resolution.Use.EndByte
		start := end - len(name)
		if !ok || end > len(file.Text) || start < 0 || start < resolution.Use.StartByte ||
			(start != resolution.Use.StartByte && (start < 2 || string(file.Text[start-2:start]) != "::")) {
			return renamePlan{}, errRenameUnavailable
		}
		if err := add(file, tokenSpan{start, end}, selectors[resolution.Use]); err != nil {
			return renamePlan{}, err
		}
	}
	plan := renamePlan{target: target, name: name}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		file := files[path]
		id := editableDocumentID{URI: fileURI(path)}
		for _, opened := range s.docs {
			if sameDocumentPath(opened.Path, path) {
				if !bytes.Equal(opened.Text, file.Text) {
					return renamePlan{}, errRenameUnavailable
				}
				version := opened.Version
				id = editableDocumentID{URI: opened.URI, Version: &version}
				break
			}
		}
		entry := renameFile{sourceView: file, id: id}
		tokens := lexer.Lex(source.NewFile(path, file.Text), source.NewBudget(source.DefaultLimits(), len(file.Text)))
		if len(tokens.Diagnostics) > 0 {
			return renamePlan{}, errRenameUnavailable
		}
		previous := ""
		for _, token := range tokens.Tokens {
			if token.Kind == lexer.Trivia || token.Kind == lexer.EOF {
				continue
			}
			span := tokenSpan{token.Span.StartByte, token.Span.EndByte}
			if selector, matches := spans[path][span]; matches {
				// A shorthand field shares its spelling with the value.
				if previous == "." && !selector {
					return renamePlan{}, errRenameUnavailable
				}
				entry.spans = append(entry.spans, span)
				if sameDocumentPath(d.Path, path) && span.start <= offset && offset < span.end {
					plan.cursor = span
				}
			}
			previous = token.Text(file.Text)
		}
		if len(entry.spans) != len(spans[path]) {
			return renamePlan{}, errRenameUnavailable
		}
		plan.files = append(plan.files, entry)
	}
	if plan.cursor.start == plan.cursor.end {
		return renamePlan{}, errRenameUnavailable
	}
	return plan, nil
}

func renameMemberDeclaration(module *ast.Module, offset int, decl hir.Span) (string, string, string, hir.Span) {
	match := func(owner string, fields []ast.FieldDeclaration) (string, string, hir.Span) {
		for _, field := range fields {
			if field.NameSpan.StartByte <= offset && offset < field.NameSpan.EndByte {
				decl.StartByte, decl.EndByte = field.NameSpan.StartByte, field.NameSpan.EndByte
				return field.Name, owner, decl
			}
		}
		return "", "", decl
	}
	for _, structure := range module.Structs {
		if len(structure.Attributes) == 0 && len(structure.TypeParameters) == 0 {
			if name, owner, span := match(structure.Name, structure.Fields); name != "" {
				return name, owner, "field", span
			}
		}
	}
	for _, resource := range module.Resources {
		if len(resource.Attributes) == 0 && len(resource.TypeParameters) == 0 {
			if name, owner, span := match(resource.Name, resource.Fields); name != "" {
				return name, owner, "field", span
			}
		}
	}
	for _, enumeration := range module.Enums {
		if len(enumeration.Attributes) != 0 {
			continue
		}
		for _, variant := range enumeration.Variants {
			if variant.NameSpan.StartByte <= offset && offset < variant.NameSpan.EndByte {
				decl.StartByte, decl.EndByte = variant.NameSpan.StartByte, variant.NameSpan.EndByte
				return variant.Name, enumeration.Name, "enum-variant", decl
			}
			if name, owner, span := match(enumeration.Name, variant.Fields); name != "" {
				return name, owner, "field", span
			}
		}
	}
	return "", "", "", decl
}

func (s *server) rename(d *document, plan renamePlan, name string) ([]documentEdit, error) {
	keywords, _ := lexer.EditorVocabulary()
	if len(name) > 255 || !source.ValidIdentifier(name) || slices.Contains(keywords, name) {
		return nil, errors.New("new name must be a Fairy C identifier of up to 255 bytes")
	}
	if plan.target.Kind == "function" && plan.target.Name == plan.name && name == "main" {
		return nil, errors.New("main is reserved for the program entry point")
	}
	if name == plan.name {
		return []documentEdit{}, nil
	}
	delta := len(name) - len(plan.name)
	qualified := strings.TrimSuffix(plan.target.Name, plan.name) + name
	replacements := make(map[string][]byte, len(plan.files))
	edits := make([]documentEdit, 0, len(plan.files))
	responseBudget := maxMessage - 1024
	openBytes := 0
	for _, opened := range s.docs {
		openBytes += len(opened.Text)
	}
	for _, file := range plan.files {
		size := len(file.Text) + delta*len(file.spans)
		if file.id.Version != nil {
			openBytes += size - len(file.Text)
		}
		if size > source.DefaultLimits().MaxBytes || openBytes > maxOpenDocumentBytes {
			return nil, errors.New("renamed document exceeds the editor byte limit")
		}
		responseBudget -= 6*len(file.id.URI) + 128 + len(file.spans)*(6*len(name)+128)
		if responseBudget < 0 {
			return nil, errReferenceLimit
		}
		text := make([]byte, 0, size)
		edit := documentEdit{TextDocument: file.id, Edits: make([]textEdit, 0, len(file.spans))}
		index := newPositionIndex(file.Text)
		previousEnd := 0
		for _, span := range file.spans {
			text = append(text, file.Text[previousEnd:span.start]...)
			text = append(text, name...)
			edit.Edits = append(edit.Edits, textEdit{Range: Range{index.bytePosition(span.start), index.bytePosition(span.end)}, NewText: name})
			previousEnd = span.end
		}
		replacements[file.Path] = append(text, file.Text[previousEnd:]...)
		edits = append(edits, edit)
	}
	rechecked, err := s.probeDocuments(d, replacements)
	if err != nil {
		return nil, err
	}
	if rechecked == nil || !rechecked.clean || !sameRenameBindings(d.View, rechecked, replacements, plan, qualified, delta) {
		return nil, errRenameChangesResolution
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if root := d.View.root; root != "" {
		original := make(map[string][]byte)
		for _, opened := range s.docs {
			if contained(root, opened.Path) {
				original[opened.Path] = opened.Text
			}
		}
		current, err := d.View.snapshot.Matches(ctx, original)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil || !current {
			return nil, errRenameChangesResolution
		}
	}
	return edits, ctx.Err()
}

func sameRenameBindings(before, after *analysis, replacements map[string][]byte, plan renamePlan, name string, delta int) bool {
	if len(before.files) != len(after.files) || len(before.result.Resolutions) != len(after.result.Resolutions) {
		return false
	}
	for key, file := range before.files {
		current, ok := after.files[key]
		expected := file.Text
		if text, changed := replacements[file.Path]; changed {
			expected = text
		}
		if !ok || file.Path != current.Path || !bytes.Equal(expected, current.Text) {
			return false
		}
	}
	spans := make(map[string][]tokenSpan, len(plan.files))
	for _, file := range plan.files {
		spans[file.Path] = file.spans
	}
	translate := func(span hir.Span) hir.Span {
		if file, ok := before.file(span); ok && len(spans[file.Path]) != 0 {
			edits := spans[file.Path]
			shift := func(offset int) int {
				return offset + delta*sort.Search(len(edits), func(i int) bool { return edits[i].end > offset })
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
