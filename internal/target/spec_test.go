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
	if spec.Pointer != (Layout{SizeBytes: 8, AlignmentBytes: 8}) || spec.Stack.CallAlignmentBytes != 16 || spec.ObjectFormat != "elf64" {
		t.Fatalf("target core layout = %+v", spec)
	}
	var sizeT, ssizeT Scalar
	for _, scalar := range spec.CScalars {
		switch scalar.Name {
		case "c.size_t":
			sizeT = scalar
		case "c.ssize_t":
			ssizeT = scalar
		}
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

func TestLookupReturnsIsolatedCopies(t *testing.T) {
	t.Parallel()
	first, _ := Lookup(X8664LinuxGNU)
	first.LanguageScalars[0].Name = "mutated"
	first.ReprCFeatures[0] = "mutated"
	first.Evidence.Missing[0] = "mutated"
	second, _ := Lookup(X8664LinuxGNU)
	if second.LanguageScalars[0].Name == "mutated" || second.ReprCFeatures[0] == "mutated" || second.Evidence.Missing[0] == "mutated" {
		t.Fatalf("Lookup() leaked mutable target state: %+v", second)
	}
}

func TestLookupRejectsUnknownTarget(t *testing.T) {
	t.Parallel()
	if _, found := Lookup("x86_64-windows-msvc"); found {
		t.Fatal("unimplemented target was reported as known")
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
