package mir

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestOwningPayloadSharedInertGraph(t *testing.T) {
	value := hir.TypeI32
	for range 32 {
		value = hir.Result(value, value)
	}
	v := operationVerifier{}
	if v.payloadNeedsDrop(value) {
		t.Fatal("inert graph gained ownership")
	}
	// This graph has 33 types but over four billion expanded paths.
	if len(v.payloadOwnership) > 33 {
		t.Fatal("shared type identities lost")
	}
}
