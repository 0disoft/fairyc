package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	// DialectC11 identifies the C11 compiler dialect; target-qualified typedefs may require
	// an explicit feature-test macro and provider header within that dialect.
	DialectC11 = "c11"
	// InstrumentationDropTraceV0 identifies the closed native-conformance drop observer.
	InstrumentationDropTraceV0 = "drop-trace-v0"
)

// Profile names the target, C dialect, and closed instrumentation accepted by one emission run.
type Profile struct {
	Target          target.ID
	Dialect         string
	Instrumentation string
}

// Stage0Profile returns the Linux GNU Stage 0 C emission profile.
func Stage0Profile() Profile {
	return Profile{Target: target.X8664LinuxGNU, Dialect: DialectC11}
}

// Stage0WindowsMSVCProfile returns the provisional Windows MSVC Stage 0 C
// emission profile. It has no validation-only instrumentation.
func Stage0WindowsMSVCProfile() Profile {
	return Profile{Target: target.X8664WindowsMSVC, Dialect: DialectC11}
}

// Stage0DropTraceProfile returns the native-conformance profile that observes compiler-owned drops.
func Stage0DropTraceProfile() Profile {
	return Profile{
		Target: target.X8664LinuxGNU, Dialect: DialectC11,
		Instrumentation: InstrumentationDropTraceV0,
	}
}

// Stage0WindowsMSVCDropTraceProfile observes compiler-owned drops in the closed
// Windows conformance lane. Production profiles remain uninstrumented.
func Stage0WindowsMSVCDropTraceProfile() Profile {
	return Profile{Target: target.X8664WindowsMSVC, Dialect: DialectC11,
		Instrumentation: InstrumentationDropTraceV0}
}

// DropSite identifies one source-linked resource leaf that an instrumented drop may observe.
type DropSite struct {
	ID           uint32
	Package      hir.PackageID
	ResourceType string
	Source       string
	StartByte    int
	EndByte      int
}

// Unit is one deterministic generated-C translation unit plus bounded stage facts.
type Unit struct {
	Target          string
	Dialect         string
	Instrumentation string
	Entry           string
	Functions       []string
	ForeignSymbols  []string
	Features        []string
	DropSites       []DropSite
	Source          string
}

// CapabilityError reports verified MIR outside the implemented C transport subset.
type CapabilityError struct {
	Function string
	Feature  string
}

func (e CapabilityError) Error() string {
	if e.Function == "" {
		return fmt.Sprintf("C backend does not support %s", e.Feature)
	}
	return fmt.Sprintf("C backend does not support %s in %s", e.Feature, e.Function)
}
