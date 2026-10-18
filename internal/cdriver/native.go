package cdriver

import (
	"fmt"
	"io"
	"runtime"
	"slices"
	"time"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

// NativeTarget describes command-facing conventions without discovering a local compiler.
type NativeTarget struct {
	ID                        target.ID
	Platform                  string
	ExecutableSuffix          string
	Compilers                 []string
	ResultLabel               string
	RunInWorkspace            bool
	EmissionLabel             string
	ForbiddenExecutableSuffix string
}

type nativeAdapter struct {
	description NativeTarget
	resolve     func(string) (NativeToolchain, error)
}

var nativeAdapters = map[target.ID]nativeAdapter{
	target.X8664WindowsMSVC: {
		description: NativeTarget{ID: target.X8664WindowsMSVC, Platform: "Windows", ExecutableSuffix: ".exe", ResultLabel: "result", EmissionLabel: "Windows target"},
		resolve:     resolveWindowsNative,
	},
	target.X8664LinuxGNU: {
		description: NativeTarget{ID: target.X8664LinuxGNU, Platform: "Linux", Compilers: []string{"gcc", "clang"}, ResultLabel: "status", RunInWorkspace: true, EmissionLabel: "Linux workspace", ForbiddenExecutableSuffix: ".exe"},
		resolve:     resolveLinuxNative,
	},
}

func LookupNativeTarget(id target.ID) (NativeTarget, bool) {
	adapter, found := nativeAdapters[id]
	description := adapter.description
	description.Compilers = slices.Clone(description.Compilers)
	return description, found
}

func NativeCompilerSupported(name string) bool {
	for _, adapter := range nativeAdapters {
		if slices.Contains(adapter.description.Compilers, name) {
			return true
		}
	}
	return false
}

func HostNativeTarget() (target.ID, error) {
	return hostNativeTarget(runtime.GOOS, runtime.GOARCH)
}

func hostNativeTarget(goos, goarch string) (target.ID, error) {
	architecture := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
	for id := range nativeAdapters {
		spec, found := target.Lookup(id)
		if found && spec.OperatingSystem == goos && spec.Architecture == architecture {
			return id, nil
		}
	}
	return "", fmt.Errorf("native commands do not support host %s/%s", goos, goarch)
}

// NativeToolchain owns platform build, image validation, execution and trap receipts.
// ResolveNativeToolchain selects its implementation by target ID.
type NativeToolchain interface {
	Build(NativeBuildOptions) (NativeArtifact, error)
	Execute(NativeArtifact, NativeExecutionOptions) (NativeOutcome, error)
}

type NativeBuildOptions struct {
	Profile                   Profile
	UnitSource, RuntimeSource string
	TrapControl               bool
}

type NativeArtifact struct {
	Artifact
	TrapControl bool
}

type NativeExecutionOptions struct {
	Input            io.Reader
	Arguments        []string
	WorkingDirectory string
	Timeout          time.Duration
	RejectOutput     bool
}

type NativeOutcome struct {
	ExitCode                     int32
	Stdout, Stderr, TrapCategory string
}

func ResolveNativeToolchain(id target.ID, compiler string) (result NativeToolchain, err error) {
	defer func() { err = driverError(ErrorToolchain, "ResolveNativeToolchain", err) }()
	adapter, found := nativeAdapters[id]
	if !found {
		return nil, fmt.Errorf("unsupported native target %q", id)
	}
	return adapter.resolve(compiler)
}

func validateNativeArtifact(artifact NativeArtifact, id target.ID) error {
	if artifact.Toolchain.Target != id {
		return fmt.Errorf("native artifact target %q does not match toolchain target %q", artifact.Toolchain.Target, id)
	}
	return nil
}

func classifyNativeOutcome(label string, exit int32, stdout, stderr, receipt string) (NativeOutcome, error) {
	outcome := NativeOutcome{ExitCode: exit, Stdout: stdout, Stderr: stderr}
	if receipt == "" {
		return outcome, nil
	}
	if exit <= NativeTrapExitBase || exit > NativeTrapExitBase+7 {
		return NativeOutcome{}, fmt.Errorf("%s executable produced unexpected trap control data", label)
	}
	code := uint8(exit - NativeTrapExitBase)
	if receipt != string([]byte{code}) {
		return NativeOutcome{}, fmt.Errorf("%s executable produced invalid trap control data", label)
	}
	outcome.TrapCategory, _ = NativeTrapCategory(code)
	return outcome, nil
}

func finishNativeOutcome(label string, exit int32, stdout, stderr, receipt string, options NativeExecutionOptions) (NativeOutcome, error) {
	outcome, err := classifyNativeOutcome(label, exit, stdout, stderr, receipt)
	if err != nil {
		return NativeOutcome{}, err
	}
	if options.RejectOutput && (stdout != "" || stderr != "") {
		return NativeOutcome{}, fmt.Errorf("%s executable produced unexpected output", label)
	}
	return outcome, nil
}

// NativeTrapExitBase is shared by support-source generation and receipt validation.
const NativeTrapExitBase = cruntime.NativeTrapExitBase

func NativeTrapCategory(code uint8) (string, bool) {
	categories := [...]string{"", "bounds", "integer-overflow", "division-by-zero", "signed-division-overflow", "invalid-shift", "assertion-failed", "unreachable"}
	if code == 0 || int(code) >= len(categories) {
		return "", false
	}
	return categories[code], true
}
