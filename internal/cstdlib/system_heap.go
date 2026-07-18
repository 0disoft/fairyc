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

// SystemHeapSource returns the provisional x86_64-linux-gnu SystemHeap implementation.
func SystemHeapSource() (string, error) {
	return canonicalSource("SystemHeap", stage0SystemHeap)
}

// SystemHeapProbeSource returns the closed native conformance probe for SystemHeap.
func SystemHeapProbeSource() (string, error) {
	return canonicalSource("SystemHeap probe", stage0SystemHeapProbe)
}

func canonicalSource(name, source string) (string, error) {
	if source == "" || strings.Contains(source, "\r") || !strings.HasSuffix(source, "\n") {
		return "", fmt.Errorf("%s C source is empty or non-canonical", name)
	}
	return source, nil
}
