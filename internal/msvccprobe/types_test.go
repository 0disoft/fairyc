package msvccprobe

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

func TestTypeEmitterProducesDeterministicLLP64Record(t *testing.T) {
	t.Parallel()
	spec, found := target.Lookup(target.X8664WindowsMSVC)
	if !found {
		t.Fatal("Windows MSVC target is missing")
	}
	value := layout.Type{Kind: layout.KindStruct, Name: "Record", Fields: []layout.Field{
		{Name: "number", Type: layout.Type{Kind: layout.KindCScalar, Name: "c.long"}},
		{Name: "pointer", Type: layout.Type{Kind: layout.KindPointer}},
	}}
	first, err := NewTypeEmitter(spec)
	if err != nil {
		t.Fatal(err)
	}
	name, err := first.EmitType(value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTypeEmitter(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.EmitType(value); err != nil {
		t.Fatal(err)
	}
	if name != "fy_windows_probe_type_0" || first.Source() != second.Source() {
		t.Fatalf("deterministic emission failed: name=%q", name)
	}
	for _, required := range []string{
		"typedef long fy_windows_probe_type_1;",
		"typedef void *fy_windows_probe_type_2;",
		"typedef struct fy_windows_probe_tag_0",
		"sizeof(fy_windows_probe_type_0) == UINT64_C(16)",
		"offsetof(fy_windows_probe_type_0, f_1) == UINT64_C(8)",
	} {
		if !strings.Contains(first.Source(), required) {
			t.Errorf("emitted source is missing %q", required)
		}
	}
}

func TestTypeEmitterRejectsForeignTargetAndUnavailableScalar(t *testing.T) {
	t.Parallel()
	linux, found := target.Lookup(target.X8664LinuxGNU)
	if !found {
		t.Fatal("Linux GNU target is missing")
	}
	if _, err := NewTypeEmitter(linux); err == nil || !strings.Contains(err.Error(), "want \"x86_64-windows-msvc\"") {
		t.Fatalf("NewTypeEmitter() error = %v", err)
	}
	windows, _ := target.Lookup(target.X8664WindowsMSVC)
	emitter, err := NewTypeEmitter(windows)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emitter.EmitType(layout.Type{Kind: layout.KindCScalar, Name: "c.ssize_t"}); err == nil || !strings.Contains(err.Error(), "unsupported Windows C scalar") {
		t.Fatalf("EmitType() error = %v", err)
	}
	if _, found := CScalarSpelling("c.ssize_t"); found {
		t.Fatal("CScalarSpelling() admitted unavailable Windows c.ssize_t")
	}
}

func TestNilTypeEmitterFailsClosed(t *testing.T) {
	t.Parallel()
	var emitter *TypeEmitter
	if _, err := emitter.EmitType(layout.Type{Kind: layout.KindPointer}); err == nil || !strings.Contains(err.Error(), "is nil") {
		t.Fatalf("EmitType() error = %v", err)
	}
	if emitter.Source() != "" {
		t.Fatal("nil TypeEmitter returned source")
	}
}
