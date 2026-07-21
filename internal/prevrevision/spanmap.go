package prevrevision

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// SpanMapper maps byte spans through the exact two-pass previous-revision edit plan.
type SpanMapper struct {
	original     []byte
	intermediate []byte
	final        []byte
	declarations []edit
	surface      []edit
}

// BuildSpanMapper returns the migrated source and its exact old-to-new byte mapping.
func BuildSpanMapper(path string, input []byte) ([]byte, SpanMapper, error) {
	oldTree := parseMigrationSource(path, input)
	if len(oldTree.Diagnostics) != 0 {
		return nil, SpanMapper{}, fmt.Errorf("previous revision parse produced %d diagnostics", len(oldTree.Diagnostics))
	}
	declarations, err := declarationEdits(oldTree)
	if err != nil {
		return nil, SpanMapper{}, err
	}
	intermediate, err := applyEdits(input, slices.Clone(declarations))
	if err != nil {
		return nil, SpanMapper{}, err
	}
	intermediateTree := syntax.ParsePreviousRevision(source.NewFile(migrationVirtualPath(path, intermediate), intermediate), source.DefaultLimits())
	if len(intermediateTree.Diagnostics) != 0 {
		return nil, SpanMapper{}, fmt.Errorf("intermediate parse produced %d diagnostics", len(intermediateTree.Diagnostics))
	}
	surface, err := surfaceEdits(intermediateTree)
	if err != nil {
		return nil, SpanMapper{}, err
	}
	final, err := applyEdits(intermediate, slices.Clone(surface))
	if err != nil {
		return nil, SpanMapper{}, err
	}
	finalTree := syntax.Parse(source.NewFile(migrationVirtualPath(path, final), final), source.DefaultLimits())
	if len(finalTree.Diagnostics) != 0 {
		return nil, SpanMapper{}, fmt.Errorf("final parse produced %d diagnostics", len(finalTree.Diagnostics))
	}
	return final, SpanMapper{
		original: slices.Clone(input), intermediate: intermediate, final: final,
		declarations: declarations, surface: surface,
	}, nil
}

// Map maps one half-open source span through both migration passes.
func (m SpanMapper) Map(start, end int) (int, int, error) {
	start, end, err := mapSpan(m.original, m.intermediate, m.declarations, start, end)
	if err != nil {
		return 0, 0, err
	}
	return mapSpan(m.intermediate, m.final, m.surface, start, end)
}

func mapSpan(before, after []byte, edits []edit, start, end int) (int, int, error) {
	if start < 0 || end < start || end > len(before) {
		return 0, 0, fmt.Errorf("invalid source span %d..%d", start, end)
	}
	ascending := slices.Clone(edits)
	slices.SortFunc(ascending, func(left, right edit) int {
		if left.start != right.start {
			return left.start - right.start
		}
		return left.end - right.end
	})
	for _, current := range ascending {
		if start >= current.start && end <= current.end && start != end {
			needle := before[start:end]
			replacement := []byte(current.replacement)
			if offset := bytes.Index(replacement, needle); offset >= 0 && bytes.Count(replacement, needle) == 1 {
				mappedStart := mappedEditStart(ascending, current) + offset
				return mappedStart, mappedStart + len(needle), nil
			}
		}
	}
	mappedStart := mapOffset(ascending, start, false)
	mappedEnd := mapOffset(ascending, end, true)
	if mappedStart < 0 || mappedEnd < mappedStart || mappedEnd > len(after) {
		return 0, 0, fmt.Errorf("mapped span %d..%d is outside %d bytes", mappedStart, mappedEnd, len(after))
	}
	return mappedStart, mappedEnd, nil
}

func mappedEditStart(edits []edit, target edit) int {
	shift := 0
	for _, current := range edits {
		if current.start == target.start && current.end == target.end && current.replacement == target.replacement {
			return current.start + shift
		}
		shift += len(current.replacement) - (current.end - current.start)
	}
	return target.start + shift
}

func mapOffset(edits []edit, position int, endBias bool) int {
	shift := 0
	for _, current := range edits {
		newStart := current.start + shift
		newEnd := newStart + len(current.replacement)
		switch {
		case position < current.start:
			return position + shift
		case position > current.end:
			shift += len(current.replacement) - (current.end - current.start)
		case current.start == current.end:
			// An insertion sits between the byte on its left and the byte on its
			// right. A span ending at that boundary stays before the inserted
			// text, while a span starting there follows the inserted prefix.
			if endBias {
				return newStart
			}
			return newEnd
		case position == current.end:
			return newEnd
		case position == current.start:
			return newStart
		default:
			if endBias {
				return newEnd
			}
			return newStart
		}
	}
	return position + shift
}
