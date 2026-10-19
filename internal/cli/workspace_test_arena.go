package cli

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

var referenceArenaSDK = sync.OnceValues(sdkstdlib.Load)

func workspaceArenaInterpreterOptions(workspace workspaceloader.Workspace) (interpreter.Options, error) {
	loaded, found := workspace.Package(sdkstdlib.PackagePath)
	if !found {
		return interpreter.Options{}, nil
	}
	if loaded.SourceKind != workspaceloader.SourceKindSDK || loaded.SDKIdentity != sdkstdlib.SDKIdentity || loaded.RuntimeABI != sdkstdlib.RuntimeABI {
		return interpreter.Options{}, fmt.Errorf("semantic Arena adapter requires the bundled SDK owner")
	}
	definition, err := referenceArenaSDK()
	if err != nil {
		return interpreter.Options{}, err
	}
	var authentic bool
	for _, actual := range loaded.Sources {
		if actual.Path != "std/alloc.fy" {
			continue
		}
		for _, expected := range definition.Sources {
			if expected.Path == actual.Path {
				authentic = !actual.Test && bytes.Equal(actual.Bytes, expected.Bytes) && reflect.DeepEqual(actual.Module, expected.Module)
			}
		}
		if !authentic {
			return interpreter.Options{}, fmt.Errorf("semantic Arena adapter source differs from the bundled SDK")
		}
	}
	if !authentic {
		return interpreter.Options{}, nil
	}
	owner, err := workspace.PackageID(sdkstdlib.PackagePath)
	if err != nil {
		return interpreter.Options{}, err
	}
	return interpreter.Options{AbstractArenaSDKPackages: []hir.PackageID{owner}}, nil
}
