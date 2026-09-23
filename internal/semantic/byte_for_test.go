package semantic_test

import "testing"

func TestByteViewForFailsClosedUntilSemanticLowering(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn void show(ref u8[] view) { for (let u8 byte in view) { assert(byte == 7); } }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "byte-view iteration awaits semantic and MIR lowering" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
}
