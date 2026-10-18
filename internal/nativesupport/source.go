// Package nativesupport links the closed standard-library and runtime sources
// for a native target, independently of CLI parsing and local toolchain discovery.
package nativesupport

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

type Options struct {
	SDKStorage  bool
	TrapReceipt bool
}

type provider struct {
	standard   func([]string) (string, error)
	storage    func() (string, error)
	runRuntime func(uint8) (string, error)
}

var providers = map[target.ID]provider{
	target.X8664WindowsMSVC: {cstdlib.WindowsMSVCSourceForSymbols, cstdlib.WindowsMSVCVectorStorageSource, cruntime.WindowsRunSource},
	target.X8664LinuxGNU:    {cstdlib.SourceForSymbols, cstdlib.LinuxGNUVectorStorageSource, cruntime.LinuxRunSource},
}

func Source(id target.ID, symbols []string, options Options) (string, error) {
	selected, found := providers[id]
	if !found {
		return "", fmt.Errorf("unsupported native support target %q", id)
	}
	standard, vector := symbols, false
	if options.SDKStorage {
		var err error
		standard, vector, err = splitVectorRuntimeSymbols(symbols)
		if err != nil {
			return "", err
		}
		if vector {
			standard = append(standard, "fy_std_system_heap_allocate_v0")
		}
	}
	support, err := selected.standard(standard)
	if err != nil {
		return "", err
	}
	if vector {
		heap, err := selected.standard([]string{"fy_std_system_heap_allocate_v0"})
		if err != nil {
			return "", err
		}
		storage, err := selected.storage()
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(storage, heap) {
			return "", fmt.Errorf("private vector heap source boundary changed")
		}
		support += strings.TrimPrefix(storage, heap)
	}
	renderRuntime := cruntime.ScalarSource
	if options.TrapReceipt {
		renderRuntime = selected.runRuntime
	}
	runtimeSource, err := renderRuntime(cruntime.NativeTrapExitBase)
	if err != nil {
		return "", err
	}
	if support == "" {
		return runtimeSource, nil
	}
	return support + "\n" + runtimeSource, nil
}

func splitVectorRuntimeSymbols(symbols []string) ([]string, bool, error) {
	standard := make([]string, 0, len(symbols)+1)
	vector := false
	for _, symbol := range symbols {
		if !strings.HasPrefix(symbol, "fy_private_vector_") {
			standard = append(standard, symbol)
			continue
		}
		switch symbol {
		case "fy_private_vector_initialize_v0", "fy_private_vector_release_empty_v0", "fy_private_vector_reserve_v0",
			"fy_private_vector_prepare_push_v0", "fy_private_vector_commit_push_v0", "fy_private_vector_pop_slot_v0", "fy_private_vector_element_slot_v0", "fy_private_vector_element_slot_mut_v0":
			vector = true
		default:
			return nil, false, fmt.Errorf("unrecognized private vector runtime symbol %q", symbol)
		}
	}
	return standard, vector, nil
}
