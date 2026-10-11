package cstdlib

import _ "embed"

// WindowsMSVCCandidateSourceForSymbols composes the closed SDK runtime slice
// with optional private vector transport, including the heap exactly once.
func WindowsMSVCCandidateSourceForSymbols(symbols []string, vectors bool) (string, error) {
	selected := append([]string(nil), symbols...)
	if vectors {
		selected = append(selected, "fy_std_system_heap_allocate_v0")
	}
	source, err := WindowsMSVCSourceForSymbols(selected)
	if err != nil || !vectors {
		return source, err
	}
	storage, err := canonicalSource("private vector storage", vectorStorage)
	if err != nil {
		return "", err
	}
	return source + "\n" + storage, nil
}

//go:embed stage0_vector_storage.c
var vectorStorage string

// LinuxGNUVectorStorageSource composes the compiler-private storage transport
// with the Linux SystemHeap source. It does not add a public runtime symbol.
func LinuxGNUVectorStorageSource() (string, error) {
	heap, err := SourceForSymbols([]string{"fy_std_system_heap_allocate_v0"})
	if err != nil {
		return "", err
	}
	storage, err := canonicalSource("private vector storage", vectorStorage)
	if err != nil {
		return "", err
	}
	return heap + "\n" + storage, nil
}

// WindowsMSVCVectorStorageSource is a compiler-private storage experiment, excluded from
// product symbol/component selection and the stable runtime ABI.
func WindowsMSVCVectorStorageSource() (string, error) {
	heap, err := WindowsMSVCSourceForSymbols([]string{"fy_std_system_heap_allocate_v0"})
	if err != nil {
		return "", err
	}
	storage, err := canonicalSource("private vector storage", vectorStorage)
	if err != nil {
		return "", err
	}
	return heap + "\n" + storage, nil
}
