// Package target owns target-semantic facts that must not depend on the Go host.
package target

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// SchemaV0 is the current prototype target-description schema.
	SchemaV0 = "fairyc.target-spec/v0"
	// X8664LinuxGNU identifies the first planned conformance target.
	X8664LinuxGNU ID = "x86_64-linux-gnu"
)

// ID is one canonical target identity.
type ID string

// Layout is a target-owned size and natural alignment in bytes.
type Layout struct {
	SizeBytes      uint32 `json:"sizeBytes"`
	AlignmentBytes uint32 `json:"alignmentBytes"`
}

// Scalar records one closed scalar layout and signedness rule.
type Scalar struct {
	Name       string `json:"name"`
	Layout     Layout `json:"layout"`
	Signedness string `json:"signedness"`
}

// StackRules records the fixed SysV AMD64 stack facts needed by later lowering.
type StackRules struct {
	CallAlignmentBytes uint32 `json:"callAlignmentBytes"`
	RedZoneBytes       uint32 `json:"redZoneBytes"`
}

// Evidence reports target maturity without promoting a narrow backend subset to target support.
type Evidence struct {
	EmissionSubset string   `json:"emissionSubset"`
	Emittable      bool     `json:"emittable"`
	Conformant     bool     `json:"conformant"`
	Supported      bool     `json:"supported"`
	Missing        []string `json:"missing"`
}

// Spec is the target-semantic contract. It deliberately contains no local executable or SDK path.
type Spec struct {
	SchemaVersion     string     `json:"schemaVersion"`
	ID                ID         `json:"id"`
	Architecture      string     `json:"architecture"`
	OperatingSystem   string     `json:"operatingSystem"`
	Environment       string     `json:"environment"`
	Endianness        string     `json:"endianness"`
	DataModel         string     `json:"dataModel"`
	Pointer           Layout     `json:"pointer"`
	FunctionPointer   Layout     `json:"functionPointer"`
	LanguageScalars   []Scalar   `json:"languageScalars"`
	CScalars          []Scalar   `json:"cScalars"`
	Stack             StackRules `json:"stack"`
	CallingConvention string     `json:"callingConvention"`
	AggregateLayout   string     `json:"aggregateLayout"`
	ReturnClassifier  string     `json:"returnClassifier"`
	ObjectFormat      string     `json:"objectFormat"`
	SymbolModel       string     `json:"symbolModel"`
	UnwindModel       string     `json:"unwindModel"`
	DebugContract     string     `json:"debugContract"`
	ReprCFeatures     []string   `json:"reprCFeatures"`
	Evidence          Evidence   `json:"evidence"`
}

var x8664LinuxGNUSpec = Spec{
	SchemaVersion:   SchemaV0,
	ID:              X8664LinuxGNU,
	Architecture:    "x86_64",
	OperatingSystem: "linux",
	Environment:     "gnu",
	Endianness:      "little",
	DataModel:       "lp64",
	Pointer:         Layout{SizeBytes: 8, AlignmentBytes: 8},
	FunctionPointer: Layout{SizeBytes: 8, AlignmentBytes: 8},
	LanguageScalars: []Scalar{
		{Name: "bool", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "not-applicable"},
		{Name: "f32", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "not-applicable"},
		{Name: "f64", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "not-applicable"},
		{Name: "i16", Layout: Layout{SizeBytes: 2, AlignmentBytes: 2}, Signedness: "signed"},
		{Name: "i32", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "signed"},
		{Name: "i64", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "signed"},
		{Name: "i8", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "signed"},
		{Name: "isize", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "signed"},
		{Name: "u16", Layout: Layout{SizeBytes: 2, AlignmentBytes: 2}, Signedness: "unsigned"},
		{Name: "u32", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "unsigned"},
		{Name: "u64", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "unsigned"},
		{Name: "u8", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "unsigned"},
		{Name: "usize", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "unsigned"},
	},
	CScalars: []Scalar{
		{Name: "c.bool", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "not-applicable"},
		{Name: "c.char", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "signed"},
		{Name: "c.double", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "not-applicable"},
		{Name: "c.float", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "not-applicable"},
		{Name: "c.int", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "signed"},
		{Name: "c.long", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "signed"},
		{Name: "c.longlong", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "signed"},
		{Name: "c.schar", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "signed"},
		{Name: "c.short", Layout: Layout{SizeBytes: 2, AlignmentBytes: 2}, Signedness: "signed"},
		{Name: "c.uchar", Layout: Layout{SizeBytes: 1, AlignmentBytes: 1}, Signedness: "unsigned"},
		{Name: "c.uint", Layout: Layout{SizeBytes: 4, AlignmentBytes: 4}, Signedness: "unsigned"},
		{Name: "c.ulong", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "unsigned"},
		{Name: "c.ulonglong", Layout: Layout{SizeBytes: 8, AlignmentBytes: 8}, Signedness: "unsigned"},
		{Name: "c.ushort", Layout: Layout{SizeBytes: 2, AlignmentBytes: 2}, Signedness: "unsigned"},
	},
	Stack:             StackRules{CallAlignmentBytes: 16, RedZoneBytes: 128},
	CallingConvention: "sysv-amd64",
	AggregateLayout:   "sysv-amd64-lp64",
	ReturnClassifier:  "sysv-amd64",
	ObjectFormat:      "elf64",
	SymbolModel:       "elf",
	UnwindModel:       "dwarf-cfi",
	DebugContract:     "generated-c-dwarf",
	ReprCFeatures: []string{
		"by-value-struct",
		"c-scalar",
		"explicit-c-enum",
		"f32",
		"f64",
		"fixed-array",
		"fixed-width-integer",
		"nested-struct",
		"raw-pointer",
	},
	Evidence: Evidence{
		EmissionSubset: "scalar-and-flat-i32-resource-c11",
		Missing: []string{
			"bidirectional-c-abi-fixtures",
			"c-layout-probes",
			"complete-runtime",
			"complete-standard-library",
			"hosted-linux-receipt",
			"toolchain-baseline",
			"toolchain-provisioning",
		},
	},
}

// Lookup returns an isolated copy of one known target description.
func Lookup(id ID) (Spec, bool) {
	if id != X8664LinuxGNU {
		return Spec{}, false
	}
	return cloneSpec(x8664LinuxGNUSpec), true
}

// All returns every known target description in canonical ID order.
func All() []Spec {
	value, _ := Lookup(X8664LinuxGNU)
	return []Spec{value}
}

// Validate checks structural invariants that prevent malformed target data and false support claims.
func Validate(spec Spec) error {
	if spec.SchemaVersion != SchemaV0 {
		return fmt.Errorf("target %q has schema %q, want %q", spec.ID, spec.SchemaVersion, SchemaV0)
	}
	if spec.ID == "" || spec.Architecture == "" || spec.OperatingSystem == "" || spec.Environment == "" || spec.Endianness == "" || spec.DataModel == "" {
		return fmt.Errorf("target identity is incomplete")
	}
	if err := validateLayout("pointer", spec.Pointer); err != nil {
		return err
	}
	if err := validateLayout("function pointer", spec.FunctionPointer); err != nil {
		return err
	}
	if err := validateScalars("language", spec.LanguageScalars); err != nil {
		return err
	}
	if err := validateScalars("C", spec.CScalars); err != nil {
		return err
	}
	if spec.Stack.CallAlignmentBytes == 0 || spec.Stack.CallAlignmentBytes&(spec.Stack.CallAlignmentBytes-1) != 0 {
		return fmt.Errorf("target %q has invalid call alignment %d", spec.ID, spec.Stack.CallAlignmentBytes)
	}
	for name, value := range map[string]string{
		"calling convention": spec.CallingConvention,
		"aggregate layout":   spec.AggregateLayout,
		"return classifier":  spec.ReturnClassifier,
		"object format":      spec.ObjectFormat,
		"symbol model":       spec.SymbolModel,
		"unwind model":       spec.UnwindModel,
		"debug contract":     spec.DebugContract,
	} {
		if value == "" {
			return fmt.Errorf("target %q has no %s", spec.ID, name)
		}
	}
	if err := validateSortedUnique("@repr(C) feature", spec.ReprCFeatures); err != nil {
		return err
	}
	if spec.Evidence.Supported && !spec.Evidence.Conformant || spec.Evidence.Conformant && !spec.Evidence.Emittable {
		return fmt.Errorf("target %q has an impossible support-state ordering", spec.ID)
	}
	if spec.Evidence.EmissionSubset == "" || !spec.Evidence.Supported && len(spec.Evidence.Missing) == 0 {
		return fmt.Errorf("target %q has incomplete maturity evidence", spec.ID)
	}
	if err := validateSortedUnique("missing evidence", spec.Evidence.Missing); err != nil {
		return err
	}
	return nil
}

func cloneSpec(spec Spec) Spec {
	spec.LanguageScalars = slices.Clone(spec.LanguageScalars)
	spec.CScalars = slices.Clone(spec.CScalars)
	spec.ReprCFeatures = slices.Clone(spec.ReprCFeatures)
	spec.Evidence.Missing = slices.Clone(spec.Evidence.Missing)
	return spec
}

func validateScalars(family string, values []Scalar) error {
	if len(values) == 0 {
		return fmt.Errorf("target has no %s scalar layouts", family)
	}
	names := make([]string, 0, len(values))
	for _, value := range values {
		if value.Name == "" {
			return fmt.Errorf("target has an unnamed %s scalar", family)
		}
		if value.Signedness != "signed" && value.Signedness != "unsigned" && value.Signedness != "not-applicable" {
			return fmt.Errorf("%s scalar %q has invalid signedness %q", family, value.Name, value.Signedness)
		}
		if err := validateLayout(family+" scalar "+value.Name, value.Layout); err != nil {
			return err
		}
		names = append(names, value.Name)
	}
	return validateSortedUnique(family+" scalar", names)
}

func validateLayout(name string, value Layout) error {
	if value.SizeBytes == 0 || value.AlignmentBytes == 0 || value.AlignmentBytes&(value.AlignmentBytes-1) != 0 || value.AlignmentBytes > value.SizeBytes {
		return fmt.Errorf("%s has invalid layout size=%d alignment=%d", name, value.SizeBytes, value.AlignmentBytes)
	}
	return nil
}

func validateSortedUnique(name string, values []string) error {
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s list contains an empty value", name)
		}
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("%s list is not strictly sorted at %q", name, value)
		}
	}
	return nil
}
