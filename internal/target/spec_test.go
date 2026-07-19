package target

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestX8664LinuxGNUSpecIsValidAndHostIndependent(t *testing.T) {
	t.Parallel()
	spec, found := Lookup(X8664LinuxGNU)
	if !found {
		t.Fatal("x86_64-linux-gnu target is missing")
	}
	if err := Validate(spec); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if spec.Pointer != (Layout{SizeBytes: 8, AlignmentBytes: 8}) || spec.Stack != (StackRules{CallAlignmentBytes: 16, RedZoneBytes: 128, ShadowSpaceBytes: 0}) || spec.ObjectFormat != "elf64" {
		t.Fatalf("target core layout = %+v", spec)
	}
	var cLong, sizeT, ssizeT Scalar
	for _, scalar := range spec.CScalars {
		switch scalar.Name {
		case "c.long":
			cLong = scalar
		case "c.size_t":
			sizeT = scalar
		case "c.ssize_t":
			ssizeT = scalar
		}
	}
	if cLong.Name == "" || cLong.Layout != (Layout{SizeBytes: 8, AlignmentBytes: 8}) || cLong.Signedness != "signed" || cLong.Header != "" || cLong.Availability != "" {
		t.Fatalf("target c.long = %+v", cLong)
	}
	if sizeT.Name == "" || sizeT.Layout != (Layout{SizeBytes: 8, AlignmentBytes: 8}) || sizeT.Signedness != "unsigned" || sizeT.Header != "stddef.h" || sizeT.Availability != "iso-c11" {
		t.Fatalf("target c.size_t = %+v", sizeT)
	}
	if ssizeT.Name == "" || ssizeT.Layout != (Layout{SizeBytes: 8, AlignmentBytes: 8}) || ssizeT.Signedness != "signed" || ssizeT.Header != "sys/types.h" || ssizeT.Availability != "posix.1-2008" {
		t.Fatalf("target c.ssize_t = %+v", ssizeT)
	}
	if spec.Evidence.EmissionSubset != "scalar-and-flat-i32-resource-tagged-c11" {
		t.Fatalf("target emission subset = %q", spec.Evidence.EmissionSubset)
	}
	if spec.Evidence.Emittable || spec.Evidence.Conformant || spec.Evidence.Supported || len(spec.Evidence.Missing) == 0 {
		t.Fatalf("target evidence overclaims maturity = %+v", spec.Evidence)
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"executablePath", "sdkPath", "sysrootPath", "C:\\\\", "/usr/bin/"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("target spec contains host-local data %q: %s", forbidden, encoded)
		}
	}
}

func TestX8664WindowsMSVCSpecIsValidLLP64AndFailClosed(t *testing.T) {
	t.Parallel()
	spec, found := Lookup(X8664WindowsMSVC)
	if !found {
		t.Fatal("x86_64-windows-msvc target is missing")
	}
	if err := Validate(spec); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if spec.DataModel != "llp64" || spec.Pointer != (Layout{SizeBytes: 8, AlignmentBytes: 8}) ||
		spec.Stack != (StackRules{CallAlignmentBytes: 16, RedZoneBytes: 0, ShadowSpaceBytes: 32}) ||
		spec.ObjectFormat != "coff-x86-64" || spec.UnwindModel != "windows-x64-pdata-xdata" {
		t.Fatalf("Windows target core facts = %+v", spec)
	}
	var cLong, sizeT Scalar
	for _, scalar := range spec.CScalars {
		switch scalar.Name {
		case "c.long":
			cLong = scalar
		case "c.size_t":
			sizeT = scalar
		case "c.ssize_t":
			t.Fatalf("Windows target exposes unavailable POSIX scalar: %+v", scalar)
		}
	}
	if cLong.Layout != (Layout{SizeBytes: 4, AlignmentBytes: 4}) || sizeT.Layout != (Layout{SizeBytes: 8, AlignmentBytes: 8}) {
		t.Fatalf("Windows LLP64 scalar facts: c.long=%+v c.size_t=%+v", cLong, sizeT)
	}
	if spec.Evidence.EmissionSubset != "scalar-and-llp64-c-scalar-c11" || spec.Evidence.Emittable || spec.Evidence.Conformant || spec.Evidence.Supported {
		t.Fatalf("Windows target overclaims maturity = %+v", spec.Evidence)
	}
	for _, missing := range spec.Evidence.Missing {
		if missing == "windows-x64-abi-classifier" || missing == "target-c-emission-profile" {
			t.Fatalf("implemented Windows evidence remains marked missing: %+v", spec.Evidence)
		}
	}
}

func TestLookupReturnsIsolatedCopies(t *testing.T) {
	t.Parallel()
	for _, id := range []ID{X8664LinuxGNU, X8664WindowsMSVC} {
		first, _ := Lookup(id)
		first.LanguageScalars[0].Name = "mutated"
		first.ReprCFeatures[0] = "mutated"
		first.Evidence.Missing[0] = "mutated"
		second, _ := Lookup(id)
		if second.LanguageScalars[0].Name == "mutated" || second.ReprCFeatures[0] == "mutated" || second.Evidence.Missing[0] == "mutated" {
			t.Fatalf("Lookup(%q) leaked mutable target state: %+v", id, second)
		}
	}
}

func TestLookupRejectsUnknownTarget(t *testing.T) {
	t.Parallel()
	if _, found := Lookup("aarch64-linux-gnu"); found {
		t.Fatal("unknown target was reported as known")
	}
}

func TestValidateRejectsMalformedLayoutOrderingAndMaturity(t *testing.T) {
	t.Parallel()
	valid, _ := Lookup(X8664LinuxGNU)
	tests := []struct {
		name   string
		mutate func(*Spec)
	}{
		{name: "non-power-of-two alignment", mutate: func(spec *Spec) { spec.Pointer.AlignmentBytes = 3 }},
		{name: "unsorted scalar names", mutate: func(spec *Spec) {
			spec.LanguageScalars[0], spec.LanguageScalars[1] = spec.LanguageScalars[1], spec.LanguageScalars[0]
		}},
		{name: "supported without conformance", mutate: func(spec *Spec) { spec.Evidence.Supported = true }},
		{name: "profile identity drift", mutate: func(spec *Spec) { spec.DataModel = "llp64" }},
		{name: "profile stack drift", mutate: func(spec *Spec) { spec.Stack.ShadowSpaceBytes = 32 }},
		{name: "missing evidence erased", mutate: func(spec *Spec) { spec.Evidence.Missing = nil }},
		{name: "language scalar with C header", mutate: func(spec *Spec) {
			spec.LanguageScalars[0].Header = "stddef.h"
			spec.LanguageScalars[0].Availability = "iso-c11"
		}},
		{name: "C typedef missing availability", mutate: func(spec *Spec) {
			spec.CScalars[cScalarIndex(spec.CScalars, "c.ssize_t")].Availability = ""
		}},
		{name: "C typedef parent header", mutate: func(spec *Spec) {
			spec.CScalars[cScalarIndex(spec.CScalars, "c.ssize_t")].Header = "../sys/types.h"
		}},
		{name: "C typedef absolute header", mutate: func(spec *Spec) {
			spec.CScalars[cScalarIndex(spec.CScalars, "c.ssize_t")].Header = "/sys/types.h"
		}},
		{name: "C typedef unknown availability", mutate: func(spec *Spec) {
			spec.CScalars[cScalarIndex(spec.CScalars, "c.ssize_t")].Availability = "posix-next"
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneSpec(valid)
			test.mutate(&candidate)
			if err := Validate(candidate); err == nil {
				t.Fatalf("Validate() accepted malformed target: %+v", candidate)
			}
		})
	}
}

func cScalarIndex(values []Scalar, name string) int {
	for index, value := range values {
		if value.Name == name {
			return index
		}
	}
	panic("test target is missing C scalar " + name)
}
