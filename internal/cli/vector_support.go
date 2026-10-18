package cli

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func workspaceSDKStorage(workspace workspaceloader.Workspace) (bool, error) {
	owners, err := workspace.SDKStorageOwners()
	if err != nil {
		return false, err
	}
	if len(owners) != 0 {
		if err := validateVectorRuntimeOwners(workspace); err != nil {
			return false, err
		}
	}
	return len(owners) != 0, nil
}

func validateVectorRuntimeOwners(workspace workspaceloader.Workspace) error {
	for _, pkg := range workspace.Packages {
		if pkg.Path == sdkstdlib.PackagePath {
			continue
		}
		for _, source := range pkg.Sources {
			for _, function := range source.Module.ForeignFunctions {
				name := function.Name
				for _, attribute := range function.Attributes {
					if attribute.Name == "link_name" && len(attribute.Arguments) == 1 {
						name = attribute.Arguments[0].Cooked
					}
				}
				if strings.HasPrefix(name, "fy_private_vector_") {
					return fmt.Errorf("private vector runtime declarations are reserved for the embedded SDK")
				}
			}
		}
	}
	return nil
}

// Legacy source-probe helpers share native support selection with workspace builds.
func windowsVectorSupportSource(symbols []string, runReceipt bool) (string, error) {
	return nativesupport.Source(target.X8664WindowsMSVC, symbols, nativesupport.Options{SDKStorage: true, TrapReceipt: runReceipt})
}

func linuxVectorSupportSource(symbols []string) (string, error) {
	return nativesupport.Source(target.X8664LinuxGNU, symbols, nativesupport.Options{SDKStorage: true})
}
