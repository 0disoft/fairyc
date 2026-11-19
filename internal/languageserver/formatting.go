package languageserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/formatter"
	"github.com/0disoft/fairyc/internal/source"
)

type textEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

var errFormattingLimit = errors.New("formatted document exceeds the response limit")

func (s *server) formatDocument(d *document) ([]textEdit, error) {
	if s.ctx != nil {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
	}
	result, err := formatter.Format(filepath.Base(d.Path), d.Text, source.DefaultLimits())
	if s.ctx != nil {
		if canceled := s.ctx.Err(); canceled != nil {
			return nil, canceled
		}
	}
	if errors.Is(err, formatter.ErrMalformedSource) {
		return []textEdit{}, nil
	}
	if err != nil {
		return nil, err
	}
	if bytes.Equal(result.Bytes, d.Text) {
		return []textEdit{}, nil
	}
	newText := string(result.Bytes)
	// JSON escaping can expand the edit beyond the framed protocol's budget.
	const textBudget = maxMessage - 1024
	if 6*len(newText)+2 > textBudget {
		encoded, err := json.Marshal(newText)
		if err != nil {
			return nil, err
		}
		if len(encoded) > textBudget {
			return nil, errFormattingLimit
		}
	}
	return []textEdit{{Range: Range{End: bytePosition(d.Text, len(d.Text))}, NewText: newText}}, nil
}
