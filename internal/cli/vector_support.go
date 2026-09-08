package cli

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
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

func windowsVectorSupportSource(symbols []string, runReceipt bool) (string, error) {
	standard := make([]string, 0, len(symbols)+1)
	vector := false
	for _, symbol := range symbols {
		if !strings.HasPrefix(symbol, "fy_private_vector_") {
			standard = append(standard, symbol)
			continue
		}
		switch symbol {
		case "fy_private_vector_initialize_v0", "fy_private_vector_release_empty_v0", "fy_private_vector_reserve_v0",
			"fy_private_vector_prepare_push_v0", "fy_private_vector_commit_push_v0", "fy_private_vector_pop_slot_v0", "fy_private_vector_element_slot_v0":
			vector = true
		default:
			return "", fmt.Errorf("unrecognized private vector runtime symbol %q", symbol)
		}
	}
	if vector {
		standard = append(standard, "fy_std_system_heap_allocate_v0")
	}
	support, err := cstdlib.WindowsMSVCSourceForSymbols(standard)
	if err != nil {
		return "", err
	}
	if vector {
		heap, err := cstdlib.WindowsMSVCSourceForSymbols([]string{"fy_std_system_heap_allocate_v0"})
		if err != nil {
			return "", err
		}
		storage, err := cstdlib.WindowsMSVCVectorStorageSource()
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(storage, heap) {
			return "", fmt.Errorf("private vector heap source boundary changed")
		}
		support += strings.TrimPrefix(storage, heap)
	}
	if runReceipt {
		return combineWindowsRunSupportSource(support)
	}
	return combineProductSupportSource(support)
}
