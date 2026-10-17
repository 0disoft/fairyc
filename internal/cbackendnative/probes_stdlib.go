package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func runSystemHeapSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, systemHeapSource string) (int, error) {
	if systemHeapSource == "" {
		return 0, fmt.Errorf("SystemHeap source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"system-heap-source-call",
		systemHeapSourceCallSmokeSource,
		systemHeapSource+systemHeapSourceCallBridge,
		42,
		systemHeapSourceCallExpectedCFragments,
	)
}

func runBufferSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, storageSource string) (int, error) {
	if storageSource == "" {
		return 0, fmt.Errorf("Buffer source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"buffer-source-call",
		bufferSourceCallSmokeSource,
		storageSource+bufferSourceCallBridge,
		42,
		bufferSourceCallExpectedCFragments,
	)
}

func runArenaSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, arenaSource string) (int, error) {
	if arenaSource == "" {
		return 0, fmt.Errorf("Arena source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"arena-source-call",
		arenaSourceCallSmokeSource,
		arenaSource+arenaSourceCallBridge,
		42,
		arenaSourceCallExpectedCFragments,
	)
}

func runTextValidationSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, textViewSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"text-validation-source-call",
		textValidationSourceCallSmokeSource,
		textViewSource,
		42,
		[]string{
			"extern uint32_t fy_std_text_utf8_valid_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1);",
			"fy_std_text_utf8_valid_v0(((void *)0), ((uintptr_t)UINT64_C(0)))",
		},
	)
}

func runTextViewOutputSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, textViewSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"text-view-output-source-call",
		textViewOutputSourceCallSmokeSource,
		textViewSource,
		42,
		[]string{
			"struct fy_std_text_view_v0 {",
			"extern uint32_t fy_std_text_validate_utf8_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1, struct fy_std_text_view_v0 * fy_a_2);",
			" = &fy_l_",
			"fy_std_text_validate_utf8_v0(((void *)0), ((uintptr_t)UINT64_C(0)), fy_l_",
			".length",
		},
	)
}

func runCStrValidationSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, ownedTextSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"cstr-validation-source-call",
		cStrValidationSourceCallSmokeSource,
		ownedTextSource+cStrValidationSourceCallBridge,
		42,
		cStrValidationSourceCallExpectedCFragments,
	)
}

func runOwnedTextSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, ownedTextSource string) (int, error) {
	if ownedTextSource == "" {
		return 0, fmt.Errorf("owned-text source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"owned-text-source-call",
		ownedTextSourceCallSmokeSource,
		ownedTextSource+ownedTextSourceCallBridge,
		42,
		ownedTextSourceCallExpectedCFragments,
	)
}

func runLinuxFileSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, linuxFileSource string) (int, error) {
	if linuxFileSource == "" {
		return 0, fmt.Errorf("Linux File source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"linux-file-source-call",
		linuxFileSourceCallSmokeSource,
		linuxFileSource+linuxFileSourceCallBridge,
		42,
		linuxFileSourceCallExpectedCFragments,
	)
}
