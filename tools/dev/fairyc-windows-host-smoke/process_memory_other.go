//go:build !windows

package main

import "fmt"

type processMemoryProbe struct{}

func openProcessMemoryProbe(int) (processMemoryProbe, error) {
	return processMemoryProbe{}, fmt.Errorf("process memory counters require Windows")
}

func (processMemoryProbe) snapshot() (uint64, uint64, error) {
	return 0, 0, fmt.Errorf("process memory counters require Windows")
}

func (processMemoryProbe) close() error {
	return nil
}
