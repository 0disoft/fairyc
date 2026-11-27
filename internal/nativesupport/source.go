// Package nativesupport links the closed standard-library and runtime sources
// for a native target, independently of CLI parsing and local toolchain discovery.
package nativesupport

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

type Options struct {
	// ExternalC is selected only by explicit Windows native project settings.
	ExternalC   bool
	SDKStorage  bool
	TrapReceipt bool
	// CandidateHost admits the closed compiler-private adapters for this target.
	CandidateHost bool
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

const windowsRuntimeHeaderOptions = `#ifndef NOGDI
#define NOGDI
#endif
#ifndef NOUSER
#define NOUSER
#endif

`

func Source(id target.ID, symbols []string, options Options) (string, error) {
	selected, found := providers[id]
	if !found {
		return "", fmt.Errorf("unsupported native support target %q", id)
	}
	standard, vector := symbols, false
	if options.ExternalC {
		if id != target.X8664WindowsMSVC {
			return "", fmt.Errorf("project C linkage requires Windows")
		}
		standard = nil
		for _, symbol := range symbols {
			if strings.HasPrefix(symbol, "fy_") || strings.HasPrefix(symbol, "fairyc_") {
				standard = append(standard, symbol)
			}
		}
	}
	var windowsHost string
	if options.CandidateHost && id == target.X8664WindowsMSVC {
		var err error
		standard, windowsHost, err = splitWindowsHostSymbols(standard)
		if err != nil {
			return "", err
		}
	}
	if options.SDKStorage {
		var err error
		standard, vector, err = splitVectorRuntimeSymbols(standard)
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
		support = runtimeSource
	} else {
		support += "\n" + runtimeSource
	}
	if options.CandidateHost && id == target.X8664LinuxGNU {
		// LinuxSource selects declarations, not bare symbol spellings.
		header := strings.Join(symbols, "(\n") + "("
		host, err := candidatehost.LinuxSource(header)
		if err != nil {
			return "", err
		}
		if host != "" {
			// GNU/POSIX feature selection must precede every libc header,
			// including headers selected by the standard runtime components.
			support = host + "\n" + support
		}
	}
	if windowsHost != "" {
		support += "\n" + windowsHost
	}
	if id == target.X8664WindowsMSVC && strings.Contains(support, "#include <windows.h>\n") {
		// The private runtime uses no GDI or USER APIs. Keep these options in
		// its translation unit so user C and resource files get the full SDK.
		support = windowsRuntimeHeaderOptions + support
	}
	return support, nil
}

func splitWindowsHostSymbols(symbols []string) ([]string, string, error) {
	const prefix = "fairyc_private_windows_"
	standard := make([]string, 0, len(symbols))
	var declarations strings.Builder
	for _, symbol := range symbols {
		if !strings.HasPrefix(symbol, prefix) {
			standard = append(standard, symbol)
			continue
		}
		// WindowsSource accepts C declaration text. Our input is a symbol list:
		// validate the entire spelling before adapting it, so regex selection
		// cannot silently ignore a bare prefix or parse only an injected suffix.
		if len(symbol) == len(prefix) {
			return nil, "", fmt.Errorf("invalid private Windows host symbol %q", symbol)
		}
		for i := range symbol {
			c := symbol[i]
			if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') {
				return nil, "", fmt.Errorf("invalid private Windows host symbol %q", symbol)
			}
		}
		declarations.WriteString(symbol)
		declarations.WriteString("(\n")
	}
	// Reuse the existing exact closed allowlist; never maintain a second list.
	host, err := candidatehost.WindowsSource(declarations.String())
	if err != nil {
		return nil, "", err
	}
	return standard, host, nil
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
