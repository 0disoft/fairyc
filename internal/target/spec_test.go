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
	if spec.Evidence.EmissionSubset != "scalar-and-flat-i32-resource-c11" {
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
