package languageserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
)

// Bound the total coordinate scans and buffer movement in one notification.
const maxDocumentChangeWork = 128 << 20

func applyDocumentChanges(original []byte, changes json.RawMessage, sizeLimit int, needsFullText bool) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(changes))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('[') {
		return nil, errors.New("invalid document changes")
	}
	text, owned := original, false
	remainingWork := maxDocumentChangeWork
	for decoder.More() {
		var change struct {
			Text  *string         `json:"text"`
			Range json.RawMessage `json:"range"`
		}
		if err := decoder.Decode(&change); err != nil || change.Text == nil {
			return nil, errors.New("invalid document change")
		}
		if len(*change.Text) > sizeLimit {
			return nil, errors.New("editor document limit exceeded")
		}
		if len(change.Range) == 0 {
			text, owned = []byte(*change.Text), true
			needsFullText = false
			continue
		}
		if needsFullText {
			return nil, errors.New("document needs full synchronization")
		}
		remainingWork -= len(text) + len(*change.Text)
		if remainingWork < 0 {
			return nil, errors.New("document change work limit exceeded")
		}
		start, end, ok := rangeOffsets(text, change.Range)
		if !ok {
			return nil, errors.New("invalid UTF-16 change range")
		}
		if len(text)-(end-start)+len(*change.Text) > sizeLimit {
			return nil, errors.New("editor document limit exceeded")
		}
		if !owned {
			text, owned = slices.Clone(text), true
		}
		text = slices.Replace(text, start, end, []byte(*change.Text)...)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, errors.New("invalid document changes")
	}
	if needsFullText {
		return nil, errors.New("document needs full synchronization")
	}
	return text, nil
}

func rangeOffsets(text []byte, raw json.RawMessage) (start, end int, ok bool) {
	type editPosition struct {
		Line      *int `json:"line"`
		Character *int `json:"character"`
	}
	var r struct {
		Start editPosition `json:"start"`
		End   editPosition `json:"end"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Start.Line == nil || r.Start.Character == nil ||
		r.End.Line == nil || r.End.Character == nil {
		return 0, 0, false
	}
	index := newPositionIndex(text)
	start, startOK := index.positionByte(Position{*r.Start.Line, *r.Start.Character})
	end, endOK := index.positionByte(Position{*r.End.Line, *r.End.Character})
	return start, end, startOK && endOK && start <= end
}
