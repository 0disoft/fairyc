package cdriver

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0disoft/fairyc/internal/pecoff"
)

// BuildWindowsMSVCTargetCached reuses exact-input runtime objects, plus generated
// units for release builds, in bounded user caches. Linking always stays fresh.
func BuildWindowsMSVCTargetCached(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return buildWindowsObjectsCached(toolchain, profile, unitSource, runtimeSource, openRuntimeObjectStore, openUnitObjectStore)
}

func buildWindowsRuntimeCached(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string, open runtimeStoreOpener) (WindowsHostArtifact, error) {
	return buildWindowsObjectsCached(toolchain, profile, unitSource, runtimeSource, open, nil)
}

func buildWindowsObjectsCached(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string, open, openUnit runtimeStoreOpener) (WindowsHostArtifact, error) {
	// Development compilation is already cheap. Its measured time/space tradeoff
	// does not justify retaining generated objects or another preprocessor pass.
	if profile != OptimizedProfile {
		openUnit = nil
	}
	uncached := func() (WindowsHostArtifact, error) {
		artifact, err := BuildWindowsMSVCTarget(toolchain, profile, unitSource, runtimeSource)
		artifact.RuntimeCache = RuntimeCacheBypass
		artifact.UnitCache = RuntimeCacheBypass
		return artifact, err
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" ||
		(profile != LowProfile && profile != OptimizedProfile) || toolchain.Name != "msvc" ||
		unitSource == "" || runtimeSource == "" {
		return uncached()
	}
	// CL/_CL_ can inject compile/link switches and additional inputs outside
	// this closed lane. Preserve their uncached behavior rather than guessing.
	effectiveEnvironment := canonicalRuntimeEnvironment(toolchain.Environment)
	if environmentValue(effectiveEnvironment, "CL") != "" || environmentValue(effectiveEnvironment, "_CL_") != "" {
		return uncached()
	}
	compiler, err := readArtifact(toolchain.Path)
	if err != nil {
		return uncached()
	}
	directory, err := os.MkdirTemp("", "fairyc-runtime-object-")
	if err != nil {
		return uncached()
	}
	defer os.RemoveAll(directory)
	sourcePath := filepath.Join(directory, "runtime.c")
	if err := os.WriteFile(sourcePath, []byte(runtimeSource), 0o600); err != nil {
		return uncached()
	}
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	flags := []string{"/nologo", "/std:c11", "/TC", "/W4", "/WX", "/MT", optimization}
	preprocessed, err := preprocessRuntime(toolchain.Path, toolchain.Environment, directory, append(append([]string{}, flags...), "/EP", "runtime.c"))
	if err != nil {
		return uncached()
	}
	if err := os.WriteFile(sourcePath, []byte(preprocessed), 0o600); err != nil {
		return uncached()
	}
	identity := runtimeObjectIdentity{CompilerPath: toolchain.Path, CompilerVersion: toolchain.Version,
		CompilerDigest: sha256.Sum256(compiler), Target: "x86_64-windows-msvc", Mode: C11ConsumerMode,
		Profile: profile, Flags: flags, Environment: toolchain.Environment, Source: sha256.Sum256([]byte(runtimeSource)),
		Preprocessed: sha256.Sum256([]byte(preprocessed))}
	key := identity.key()
	store, _ := open()
	if store != nil {
		defer store.Close()
	}
	var artifact WindowsHostArtifact
	status, err := useRuntimeObject(store, key, func() ([]byte, error) {
		args := append(append([]string{}, flags...), "/c", "runtime.c", "/Fo:runtime.obj")
		if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, directory, args...); err != nil {
			return nil, fmt.Errorf("compile runtime: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
		}
		return readArtifact(filepath.Join(directory, "runtime.obj"))
	}, func(object []byte) error {
		_, err := pecoff.InspectObject(object)
		return err
	}, func(object []byte) error {
		var err error
		artifact, err = buildWindowsUnitCached(toolchain, profile, unitSource, object, directory, identity, openUnit)
		return err
	})
	artifact.RuntimeCache = status
	return artifact, err
}
