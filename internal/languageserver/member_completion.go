package languageserver

import (
	"bytes"
	"context"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

func (s *server) memberCompletion(d *document, dot int, prefix string) []Completion {
	view := d.View
	if view == nil || view.unsynchronized || view.budgetExceeded || dot == 0 || d.Text[dot-1] == '.' {
		return []Completion{}
	}
	end := len(bytes.TrimRight(d.Text[:dot], " \t\r\n"))
	if end == 0 {
		return []Completion{}
	}
	module, key := view.completionLocal(d.Path)
	if module == nil && prefix == "" && (dot+1 == len(d.Text) || !identifierByte(d.Text[dot+1])) {
		view = s.probeMemberReceiver(d, dot+1)
		if view != nil {
			module, key = view.completionLocal(d.Path)
		}
	}
	if module == nil {
		return []Completion{}
	}
	owner := receiverTypeAt(view, key, end)
	if element, _, borrowed := hir.BorrowElement(owner); borrowed {
		owner = element
	}
	if owner == hir.TypeInvalid {
		return []Completion{}
	}
	members := completionCandidates{}
	add := func(name string, kind int, detail string) {
		if strings.HasPrefix(name, prefix) {
			members.add(Completion{name, kind, detail})
		}
	}
	for _, typed := range view.result.HIR.Modules {
		local := typed.Package == key.Package && typed.Path == module.Path
		for _, structure := range typed.Structs {
			if structure.Type == owner {
				for _, field := range structure.Fields {
					add(field.Name, 5, "field")
				}
			}
		}
		if local {
			for _, resource := range typed.Resources {
				if resource.Type == owner {
					for _, field := range resource.Fields {
						add(field.Name, 5, "field")
					}
				}
			}
		}
		for _, fn := range typed.Functions {
			if fn.OwnerType == owner && fn.Receiver != "" && fn.Receiver != "invalid" && (local || fn.Public) {
				_, name, qualified := strings.Cut(fn.Name, "::")
				if qualified {
					add(name, 2, "method")
				}
			}
		}
	}
	return members.sorted()
}

// The receiver ends immediately before the dot. Prefer the outermost typed
// expression at that boundary and reject different types at its exact span.
func receiverTypeAt(view *analysis, key sourceKey, end int) hir.Type {
	best := end
	owner := hir.TypeInvalid
	ambiguous := false
	for _, fact := range view.result.Types {
		span := fact.Expression
		if span.EndByte != end || span.StartByte > best || span.StartByte >= end || fact.CanonicalType == hir.TypeInvalid {
			continue
		}
		pkg, relative := hir.SourcePathParts(span.Source)
		if span.Package != 0 {
			pkg, relative = span.Package, span.Source
		}
		if pkg != key.Package || relative != key.Path {
			continue
		}
		if span.StartByte < best {
			best, owner, ambiguous = span.StartByte, fact.CanonicalType, false
		} else if owner != fact.CanonicalType {
			ambiguous = true
		}
	}
	if ambiguous {
		return hir.TypeInvalid
	}
	return owner
}

// Insert an identifier after the incomplete dot so normal checking can type
// the unchanged receiver. The probe owns its document wrappers so admission
// cannot evict live analyses or publish diagnostics for speculative text.
func (s *server) probeMemberReceiver(d *document, offset int) *analysis {
	if len(d.Text) >= s.documentByteLimit(d.URI) {
		return nil
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	probe := server{docs: make(map[string]*document, len(s.docs)), analysisBudget: s.analysisBudget}
	for uri, opened := range s.docs {
		probe.docs[uri] = &document{URI: uri, Path: opened.Path, Text: opened.Text, needsFullText: opened.needsFullText}
	}
	text := make([]byte, 0, len(d.Text)+1)
	text = append(text, d.Text[:offset]...)
	text = append(text, '_')
	text = append(text, d.Text[offset:]...)
	focus := &document{URI: d.URI, Path: d.Path, Text: text}
	probe.docs[d.URI] = focus
	view, err := probe.analyzeContext(ctx, focus, d.View.root)
	if err != nil {
		return nil
	}
	return view
}
