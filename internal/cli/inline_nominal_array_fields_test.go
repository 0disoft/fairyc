package cli

import (
	"os"
	"testing"
)

func TestInlineNominalArrayFieldNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/inline_nominal_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "inline-nominal-array-field", []struct{ name, source, trap string }{
		{"value-copy-matrix", string(source), ""},
	})
}

func TestInlineNominalArrayFieldCache(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/inline_nominal_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	checkStructVecViewCache(t, string(source))
}
