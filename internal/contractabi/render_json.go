package contractabi

import "encoding/json"

// RenderJSON emits deterministic, indented contract IDL with one trailing newline.
func RenderJSON(document Document) ([]byte, error) {
	if err := validateDocument(document); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
