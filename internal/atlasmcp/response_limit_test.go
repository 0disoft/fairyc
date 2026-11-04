package atlasmcp

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestMCPResponseLimitAfterIDEscaping(t *testing.T) {
	for _, maximum := range []int{2048, 4096} {
		t.Run(strconv.Itoa(maximum), func(t *testing.T) {
			id := strings.Repeat("<", 510) // Exactly 512 raw ID bytes, before HTML escaping.
			input := `{"jsonrpc":"2.0","id":"` + id + `","method":"ping"}`
			var output bytes.Buffer
			if err := Serve(strings.NewReader(input), &output, 1, maximum, func(string, string) (json.RawMessage, error) {
				t.Fatal("ping must not call the query host")
				return nil, nil
			}); err != nil {
				t.Fatal(err)
			}
			line := bytes.TrimSuffix(output.Bytes(), []byte("\n"))
			if len(line) > maximum || !json.Valid(line) {
				t.Fatalf("response policy bypassed: %d > %d", len(line), maximum)
			}
			var r struct {
				ID    *string   `json:"id"`
				Error *rpcError `json:"error"`
			}
			if err := json.Unmarshal(line, &r); err != nil {
				t.Fatal(err)
			}
			if maximum == 2048 {
				if r.ID != nil || r.Error == nil || r.Error.Code != -32603 {
					t.Fatal("oversized ID survived the bounded fallback", string(line))
				}
			} else if r.ID == nil || *r.ID != id || r.Error != nil {
				t.Fatal("admitted ID was changed", string(line))
			}
		})
	}
}
