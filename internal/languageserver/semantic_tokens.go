package languageserver

import (
	"cmp"
	"errors"
	"maps"
	"slices"
	"strings"
)

type tokenKind int

const (
	tokenNamespace tokenKind = iota
	tokenType
	tokenEnum
	tokenStruct
	tokenParameter
	tokenVariable
	tokenProperty
	tokenEnumMember
	tokenFunction
	tokenConstant
)

var semanticTokenTypes = []string{"namespace", "type", "enum", "struct", "parameter", "variable", "property", "enumMember", "function"}
var semanticTokenModifiers = []string{"declaration", "readonly"}

const (
	tokenDeclaration = 1 << iota
	tokenReadonly
	maxSemanticTokens = 32768
)

var errSemanticTokenLimit = errors.New("semantic tokens exceed the server limit")

type semanticTokens struct {
	Data []int `json:"data"`
}

type tokenSpan struct{ start, end int }
type tokenClass struct {
	kind      tokenKind
	modifiers int
	ambiguous bool
}

func (s *server) semanticTokens(d *document) (semanticTokens, error) {
	result := semanticTokens{Data: []int{}}
	view := d.View
	if view == nil || !view.clean {
		return result, nil
	}
	module, _ := view.completionLocal(d.Path)
	if module == nil {
		return result, nil
	}
	s.touchAnalysis(view)
	candidates := make(map[tokenSpan]tokenClass)
	add := func(span tokenSpan, kind tokenKind, modifiers int) bool {
		if !identifierSpan(d.Text, span) {
			return true
		}
		if kind == tokenConstant {
			kind, modifiers = tokenVariable, modifiers|tokenReadonly
		}
		if prior, found := candidates[span]; found {
			if prior.kind != kind {
				prior.ambiguous = true
			}
			prior.modifiers |= modifiers
			candidates[span] = prior
			return true
		}
		if len(candidates) >= maxSemanticTokens {
			return false
		}
		candidates[span] = tokenClass{kind: kind, modifiers: modifiers}
		return true
	}
	check := func() error {
		if s.ctx != nil {
			return s.ctx.Err()
		}
		return nil
	}
	for declaration := range authoredDeclarations(module) {
		if err := check(); err != nil {
			return result, err
		}
		if !add(tokenSpan{declaration.span.StartByte, declaration.span.EndByte}, declaration.kind, tokenDeclaration) {
			return result, errSemanticTokenLimit
		}
	}
	for i, resolution := range view.result.Resolutions {
		if i%256 == 0 {
			if err := check(); err != nil {
				return result, err
			}
		}
		file, ok := view.file(resolution.Use)
		if !ok || file.Path != d.Path {
			continue
		}
		kind, ok := resolvedTokenKind(resolution.Target.Kind)
		if !ok {
			continue
		}
		name := resolution.Target.Name
		if last := strings.LastIndex(name, "::"); last >= 0 {
			name = name[last+2:]
		}
		end := resolution.Use.EndByte
		start := end - len(name)
		if start < resolution.Use.StartByte || start < 0 || end > len(d.Text) || start >= end || string(d.Text[start:end]) != name {
			continue
		}
		if start != resolution.Use.StartByte && (start < 2 || string(d.Text[start-2:start]) != "::") {
			continue
		}
		if !add(tokenSpan{start, end}, kind, 0) {
			return result, errSemanticTokenLimit
		}
	}
	spans := slices.Collect(maps.Keys(candidates))
	slices.SortFunc(spans, func(a, b tokenSpan) int {
		if order := cmp.Compare(a.start, b.start); order != 0 {
			return order
		}
		return cmp.Compare(a.end, b.end)
	})
	index := newPositionIndex(d.Text)
	previous := Position{}
	previousEnd := 0
	for _, span := range spans {
		class := candidates[span]
		if class.ambiguous || span.start < previousEnd {
			continue
		}
		start, end := index.bytePosition(span.start), index.bytePosition(span.end)
		deltaColumn := start.Character
		if start.Line == previous.Line {
			deltaColumn -= previous.Character
		}
		result.Data = append(result.Data, start.Line-previous.Line, deltaColumn, end.Character-start.Character, int(class.kind), class.modifiers)
		previous, previousEnd = start, span.end
	}
	return result, check()
}

func identifierSpan(text []byte, span tokenSpan) bool {
	if span.start < 0 || span.end > len(text) || span.start >= span.end {
		return false
	}
	if (span.start > 0 && identifierByte(text[span.start-1])) || (span.end < len(text) && identifierByte(text[span.end])) {
		return false
	}
	for _, b := range text[span.start:span.end] {
		if !identifierByte(b) {
			return false
		}
	}
	return true
}

func resolvedTokenKind(kind string) (tokenKind, bool) {
	switch kind {
	case "function", "foreign-function":
		return tokenFunction, true
	case "constant":
		return tokenConstant, true
	case "parameter":
		return tokenParameter, true
	case "local", "match-binding", "region":
		return tokenVariable, true
	case "struct":
		return tokenStruct, true
	case "resource", "capability":
		return tokenType, true
	case "enum":
		return tokenEnum, true
	default:
		return 0, false
	}
}
