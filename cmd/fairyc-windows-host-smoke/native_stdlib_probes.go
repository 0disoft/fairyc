package main

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

func runWindowsFileProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsFileSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows File implementation: %w", err)
	}
	probe, err := cstdlib.WindowsFileProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows File probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Windows File", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runOwnedTextStorageProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCOwnedTextProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC owned-text implementation: %w", err)
	}
	probe, err := cstdlib.OwnedTextProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load owned-text storage probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "owned-text storage", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runBufferFailureProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCBufferProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC Buffer implementation: %w", err)
	}
	probe, err := cstdlib.BufferProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Buffer failure probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Buffer failure", probe, implementation, 0, cdriver.C11ConsumerMode)
}

func runArenaFailureProbe(toolchain cdriver.WindowsHostToolchain) (int, error) {
	implementation, err := cstdlib.WindowsMSVCArenaProbeImplementationSource()
	if err != nil {
		return 0, fmt.Errorf("load Windows MSVC Arena implementation: %w", err)
	}
	probe, err := cstdlib.ArenaProbeSource()
	if err != nil {
		return 0, fmt.Errorf("load Arena failure probe: %w", err)
	}
	return runWindowsSeparateTranslationUnitProbe(toolchain, "Arena failure", probe, implementation, 0, cdriver.C11ConsumerMode)
}
