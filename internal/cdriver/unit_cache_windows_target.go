package cdriver

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func buildWindowsUnitCached(toolchain WindowsHostToolchain, profile Profile, source string,
	runtimeObject []byte, directory string, identity runtimeObjectIdentity, open runtimeStoreOpener,
) (WindowsHostArtifact, error) {
	runtimeInput := WindowsHostObject{Name: "runtime.obj", Bytes: runtimeObject}
	uncached := func() (WindowsHostArtifact, error) {
		artifact, err := buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode,
			[]windowsMSVCSource{{Name: "unit.c", Contents: source}},
			WindowsMSVCBuildOptions{PrebuiltObjects: []WindowsHostObject{runtimeInput}})
		artifact.UnitCache = RuntimeCacheBypass
		return artifact, err
	}
	if open == nil {
		return uncached()
	}
	store, _ := open()
	if store == nil {
		return uncached()
	}
	defer store.Close()
	path := filepath.Join(directory, "unit.c")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		return uncached()
	}
	flags := identity.Flags
	snapshot, err := preprocessRuntime(toolchain.Path, toolchain.Environment, directory,
		append(append([]string{}, flags...), "/EP", "unit.c"))
	if err != nil {
		return uncached()
	}
	if err := os.WriteFile(path, []byte(snapshot), 0o600); err != nil {
		return uncached()
	}
	identity.Source = sha256.Sum256([]byte(source))
	identity.Preprocessed = sha256.Sum256([]byte(snapshot))
	var artifact WindowsHostArtifact
	status, err := useRuntimeObject(store, identity.unitKey(), func() ([]byte, error) {
		args := append(append([]string{}, flags...), "/c", "unit.c", "/Fo:unit.obj")
		if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, directory, args...); err != nil {
			return nil, fmt.Errorf("compile generated unit: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
		}
		return readArtifact(filepath.Join(directory, "unit.obj"))
	}, func(object []byte) error {
		_, err := pecoff.InspectObject(object)
		return err
	}, func(object []byte) error {
		var err error
		artifact, err = buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, nil,
			WindowsMSVCBuildOptions{PrebuiltObjects: []WindowsHostObject{
				{Name: "unit.obj", Bytes: object}, runtimeInput,
			}})
		return err
	})
	artifact.UnitCache = status
	return artifact, err
}
