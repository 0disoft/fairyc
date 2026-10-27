package languageserver

import "sort"

type definitionSpan struct {
	resolution int
	maxEnd     int
}

// The index belongs to one immutable analysis, just like its source positions.
// Build only the requested file's index. Physical paths can have more
// than one package identity, so group through file() rather than Source alone.
func (a *analysis) indexDefinitions(path string) []definitionSpan {
	var spans []definitionSpan
	for index, resolution := range a.result.Resolutions {
		if resolution.Use.EndByte <= resolution.Use.StartByte {
			continue
		}
		if file, found := a.file(resolution.Use); found && file.Path == path {
			spans = append(spans, definitionSpan{resolution: index})
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		return a.result.Resolutions[spans[i].resolution].Use.StartByte < a.result.Resolutions[spans[j].resolution].Use.StartByte
	})
	for index := range spans {
		end := a.result.Resolutions[spans[index].resolution].Use.EndByte
		if index > 0 {
			end = max(end, spans[index-1].maxEnd)
		}
		spans[index].maxEnd = end
	}
	return spans
}

// Prefix maximum ends keep earlier overlapping spans in the candidate window.
// With disjoint spans the window contains only actual matches; deeply nested
// overlaps can still require a linear scan. Restore semantic order, including
// duplicates, so the wire result is identical to the previous full scan.
func (a *analysis) definitionCandidates(path string, offset int, matches []int) []int {
	if a.definitions == nil {
		a.definitions = make(map[string][]definitionSpan)
	}
	spans, found := a.definitions[path]
	if !found {
		spans = a.indexDefinitions(path)
		a.definitions[path] = spans
	}
	hi := sort.Search(len(spans), func(i int) bool {
		return a.result.Resolutions[spans[i].resolution].Use.StartByte > offset
	})
	lo := sort.Search(hi, func(i int) bool { return spans[i].maxEnd > offset })
	for _, span := range spans[lo:hi] {
		if offset < a.result.Resolutions[span.resolution].Use.EndByte {
			matches = append(matches, span.resolution)
		}
	}
	sort.Ints(matches)
	return matches
}
