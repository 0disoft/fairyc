// Package cstdlib owns embedded C sources for the target standard-library implementation.
package cstdlib

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed stage0_system_heap.c
var stage0SystemHeap string

//go:embed stage0_system_heap_probe.c
var stage0SystemHeapProbe string

//go:embed stage0_buffer.c
var stage0Buffer string

//go:embed stage0_buffer_probe.c
var stage0BufferProbe string

//go:embed stage0_arena.c
var stage0Arena string

//go:embed stage0_arena_probe.c
var stage0ArenaProbe string

// SystemHeapSource returns the provisional x86_64-linux-gnu SystemHeap implementation.
func SystemHeapSource() (string, error) {
	return canonicalSource("SystemHeap", stage0SystemHeap)
}

// SystemHeapProbeSource returns the closed native conformance probe for SystemHeap.
func SystemHeapProbeSource() (string, error) {
	return canonicalSource("SystemHeap probe", stage0SystemHeapProbe)
}

// StorageSource returns the provisional SystemHeap and Buffer implementation as one C unit.
func StorageSource() (string, error) {
	heap, err := SystemHeapSource()
	if err != nil {
		return "", err
	}
	buffer, err := canonicalSource("Buffer", stage0Buffer)
	if err != nil {
		return "", err
	}
	return heap + "\n" + buffer, nil
}

// BufferProbeSource returns the closed native conformance probe for Buffer.
func BufferProbeSource() (string, error) {
	return canonicalSource("Buffer probe", stage0BufferProbe)
}

// ArenaSource returns the provisional SystemHeap and Arena implementation as one C unit.
func ArenaSource() (string, error) {
	heap, err := SystemHeapSource()
	if err != nil {
		return "", err
	}
	arena, err := canonicalSource("Arena", stage0Arena)
	if err != nil {
		return "", err
	}
	return heap + "\n" + arena, nil
}

// ArenaProbeSource returns the closed native conformance probe for Arena and Region.
func ArenaProbeSource() (string, error) {
	return canonicalSource("Arena probe", stage0ArenaProbe)
}

func canonicalSource(name, source string) (string, error) {
	if source == "" || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		return "", fmt.Errorf("%s C source is empty or non-canonical", name)
	}
	return source, nil
}
