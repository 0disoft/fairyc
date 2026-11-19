package languageserver

import (
	"cmp"
	"errors"
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const maxReferenceLocations = 4096

var (
	errReferenceLimit     = errors.New("reference result exceeds the server limit")
	errAmbiguousReference = errors.New("reference target is ambiguous")
)

func (s *server) references(d *document, offset int, includeDeclaration bool) ([]Location, error) {
	locations := []Location{}
	view := d.View
	if view == nil || !view.clean {
		return locations, nil
	}
	s.touchAnalysis(view)
	target, found, err := s.referenceTarget(d, offset)
	if errors.Is(err, errAmbiguousReference) {
		return locations, nil
	}
	if err != nil {
		return nil, err
	}
	if !found {
		if includeDeclaration {
			if span, ok := view.authoredDeclaration(d.Path, offset); ok {
				if location, ok := view.referenceLocation(span); ok {
					locations = append(locations, location)
				}
			}
		}
		return locations, nil
	}

	seen := make(map[Location]struct{})
	budget := maxMessage - 1024
	add := func(span hir.Span) error {
		location, ok := view.referenceLocation(span)
		if _, duplicate := seen[location]; !ok || duplicate {
			return nil
		}
		// Allow for JSON escaping, coordinates, and framing.
		budget -= 6*len(location.URI) + 128
		if budget < 0 || len(locations) == maxReferenceLocations {
			return errReferenceLimit
		}
		seen[location] = struct{}{}
		locations = append(locations, location)
		return nil
	}
	if includeDeclaration {
		if err := add(target.Declaration); err != nil {
			return nil, err
		}
	}
	for i, resolution := range view.result.Resolutions {
		if i%256 == 0 && s.ctx != nil && s.ctx.Err() != nil {
			return nil, s.ctx.Err()
		}
		if resolution.Target == target && resolution.Use != target.Declaration {
			if err := add(resolution.Use); err != nil {
				return nil, err
			}
		}
	}
	slices.SortFunc(locations, compareReferenceLocations)
	return locations, nil
}

func (s *server) referenceTarget(d *document, offset int) (semantic.SymbolCoordinate, bool, error) {
	view := d.View
	var target semantic.SymbolCoordinate
	found := false
	var scratch [8]int
	for _, index := range view.definitionCandidates(d.Path, offset, scratch[:0]) {
		candidate := view.result.Resolutions[index].Target
		if found && candidate != target {
			return target, false, errAmbiguousReference
		}
		target, found = candidate, true
	}
	if found {
		return target, true, nil
	}
	for i, resolution := range view.result.Resolutions {
		if i%256 == 0 && s.ctx != nil && s.ctx.Err() != nil {
			return target, false, s.ctx.Err()
		}
		candidate := resolution.Target
		span := candidate.Declaration
		if offset < span.StartByte || offset >= span.EndByte {
			continue
		}
		if file, ok := view.file(span); !ok || file.Path != d.Path {
			continue
		}
		if found && candidate != target {
			return target, false, errAmbiguousReference
		}
		target, found = candidate, true
	}
	return target, found, nil
}

// A named declaration can have no recorded uses. Its exact AST name span
// still identifies the declaration itself in a clean source snapshot.
func (a *analysis) authoredDeclaration(path string, offset int) (hir.Span, bool) {
	module, key := a.completionLocal(path)
	if module == nil {
		return hir.Span{}, false
	}
	match := func(span diagnostic.Span) bool {
		return span.StartByte <= offset && offset < span.EndByte
	}
	makeSpan := func(span diagnostic.Span) hir.Span {
		return hir.Span{Package: key.Package, Source: key.Path, StartByte: span.StartByte, EndByte: span.EndByte}
	}
	for _, fn := range module.Functions {
		if match(fn.NameSpan) {
			return makeSpan(fn.NameSpan), true
		}
		for _, parameter := range fn.Parameters {
			if match(parameter.NameSpan) {
				return makeSpan(parameter.NameSpan), true
			}
		}
	}
	for _, fn := range module.ForeignFunctions {
		if match(fn.NameSpan) {
			return makeSpan(fn.NameSpan), true
		}
	}
	for _, binding := range module.Bindings {
		if match(binding.NameSpan) {
			return makeSpan(binding.NameSpan), true
		}
	}
	for _, constant := range module.Constants {
		if match(constant.NameSpan) {
			return makeSpan(constant.NameSpan), true
		}
	}
	return hir.Span{}, false
}

func (a *analysis) referenceLocation(span hir.Span) (Location, bool) {
	file, ok := a.file(span)
	if !ok || file.Path == "" || span.StartByte < 0 || span.EndByte <= span.StartByte || span.EndByte > len(file.Text) {
		return Location{}, false
	}
	return Location{fileURI(file.Path), a.sourceRange(file.Path, file.Text, span.StartByte, span.EndByte)}, true
}

func compareReferenceLocations(a, b Location) int {
	if order := cmp.Compare(a.URI, b.URI); order != 0 {
		return order
	}
	if order := cmp.Compare(a.Range.Start.Line, b.Range.Start.Line); order != 0 {
		return order
	}
	if order := cmp.Compare(a.Range.Start.Character, b.Range.Start.Character); order != 0 {
		return order
	}
	if order := cmp.Compare(a.Range.End.Line, b.Range.End.Line); order != 0 {
		return order
	}
	return cmp.Compare(a.Range.End.Character, b.Range.End.Character)
}
