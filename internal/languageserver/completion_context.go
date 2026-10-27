package languageserver

import (
	"sort"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

type completionSpan struct {
	start, end int
}

// Keep only literal/comment byte ranges for the current document snapshot, not
// the source copy, cooked strings or complete lexer token stream. Accepted full
// document changes invalidate the index even if subsequent analysis is canceled.
type completionContextIndex struct {
	blocked []completionSpan
}

func (d *document) blocksCompletion(offset int) bool {
	if d.completionIndex == nil {
		index := &completionContextIndex{}
		scanned := lexer.Lex(source.NewFile("main.fy", d.Text), source.NewBudget(source.DefaultLimits(), len(d.Text)))
		for _, token := range scanned.Tokens {
			if token.Kind == lexer.String || token.Kind == lexer.ByteString || token.Kind == lexer.CString ||
				token.Trivia == lexer.TriviaLineComment || token.Trivia == lexer.TriviaDocumentationLine || token.Trivia == lexer.TriviaBlockComment {
				index.blocked = append(index.blocked, completionSpan{start: token.Span.StartByte, end: token.Span.EndByte})
			}
		}
		d.completionIndex = index
	}
	spans := d.completionIndex.blocked
	index := sort.Search(len(spans), func(i int) bool { return spans[i].end > offset })
	// Preserve the existing strict token-boundary policy, including adjacent
	// literal tokens; merging their ranges would suppress completion at the join.
	return index < len(spans) && spans[index].start < offset
}
