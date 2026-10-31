package main

import (
	"io"
	"strings"
	"testing"
)

func TestRetainedDebugLookupRejectsInvalidArguments(t *testing.T) {
	base := []string{"--store", "unused", "--image-sha", strings.Repeat("a", 64)}
	for _, args := range [][]string{
		nil, {"--unknown"}, append(append([]string{}, base...), "--rva", "0"),
		append(append([]string{}, base...), "--rva", "0x100000000"),
		append(append([]string{}, base...), "--rva", "-1"),
		{"--store", "unused", "--image-sha", "../escape", "--rva", "0x1000"},
		append(append([]string{}, base...), "--rva", "0x1000", "unexpected"),
	} {
		if err := runRetainedDebugLookup(args, io.Discard); err == nil {
			t.Fatalf("invalid lookup arguments accepted: %v", args)
		}
	}
}
