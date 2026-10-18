package mir

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestOperationKindsPreserveStringWireIdentities(t *testing.T) {
	seen := make(map[string]bool)
	for _, kind := range AllOperationKinds() {
		name := kind.String()
		if !kind.Valid() || name == "" || seen[name] {
			t.Fatalf("missing or duplicated operation name: %d %q", kind, name)
		}
		seen[name] = true
		encoded, err := json.Marshal(kind)
		if err != nil || string(encoded) != `"`+name+`"` {
			t.Fatalf("wire identity changed: %d %s %v", kind, encoded, err)
		}
		var decoded OperationKind
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != kind {
			t.Fatalf("round trip: %s %v", encoded, err)
		}
	}
	if encoded, _ := json.Marshal(OperationStorageTake); string(encoded) != `"storage-take"` {
		t.Fatalf("legacy wire name changed: %s", encoded)
	}
	if OperationInvalid.Valid() || OperationKind(255).Valid() {
		t.Fatal("invalid opcode admitted")
	}
	for _, invalid := range []string{`"unknown-opcode"`, `"Move"`, `1`} {
		kind := OperationMove
		if err := json.Unmarshal([]byte(invalid), &kind); err == nil || kind != OperationMove {
			t.Fatalf("unknown encoding accepted or changed destination: %s %d", invalid, kind)
		}
	}
	if _, err := json.Marshal(OperationKind(255)); err == nil {
		t.Fatal("unknown opcode encoded")
	}
	first := AllOperationKinds()
	first[0] = OperationInvalid
	if slices.Contains(AllOperationKinds(), OperationInvalid) {
		t.Fatal("inventory exposes mutable state")
	}
}

func TestVerifyRejectsUnknownOperationKinds(t *testing.T) {
	for _, kind := range []OperationKind{OperationInvalid, OperationKind(255)} {
		program := Program{Functions: []Function{{Name: "main", Entry: 0,
			Blocks: []Block{{ID: 0, Operations: []Operation{{Kind: kind, Entity: "bad-op"}}, Terminator: Terminator{Kind: TerminatorUnreachable, UnwindSuccessor: NoBlock}}},
		}}}
		failures := Verify(program)
		if len(failures) != 1 || failures[0].Code != CodeOperationKind || failures[0].Entity != "bad-op" {
			t.Fatalf("unknown opcode accepted: %+v", failures)
		}
	}
}
