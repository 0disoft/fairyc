package formatter

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func TestFormatTreeMatchesReturnedBytes(t *testing.T) {
	for _, test := range []struct {
		name, input string
		malformed   bool
	}{
		{"rewritten", "module main;  \r\nfn i32 main() { return 0; }  \r\n", false},
		{"unchanged", "module main;\nfn i32 main() { return 0; }\n", false},
		{"empty", "", true},
		{"malformed", "module main;\r\nfn i32 main( { return 0; }   \r\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.input)
			got, err := Format("main.fy", input, source.DefaultLimits())
			if test.malformed {
				if !errors.Is(err, ErrMalformedSource) || !bytes.Equal(got.Bytes, input) {
					t.Fatalf("malformed input changed or lost error: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if got.Malformed != test.malformed {
				t.Fatalf("Malformed = %v, want %v", got.Malformed, test.malformed)
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", got.Bytes), source.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			want, err := BuildGreen(parsed.Tree)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Tree, want) {
				t.Fatal("published green tree does not describe returned bytes")
			}
		})
	}
}
