package packageartifact

import (
	"bytes"
	"encoding/json"
)

// hasTrailingJSON checks the original bytes after the first decoded value. Use
// InputOffset because the decoder may have read ahead into its own buffer. Any
// non-JSON-whitespace suffix is invalid; never materialize it as an interface tree.
func hasTrailingJSON(encoded []byte, decoder *json.Decoder) bool {
	return len(bytes.TrimLeft(encoded[decoder.InputOffset():], " \t\r\n")) != 0
}
