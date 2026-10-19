package candidatebuild

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"testing"
)

// Export complete Go runtime C bytes; the authored test compares every byte.
// Deduplicate identical outputs, retaining both real compiler-header cases.
func TestCandidateRuntimePackagingOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_RUNTIME_PACKAGING_ORACLE")
	if path == "" {
		t.Skip("explicit runtime packaging oracle only")
	}
	t.Setenv("FAIRYC_RUNTIME_ORACLE", path+".selection")
	TestCandidateRuntimeSelectionOracle(t)
	data, err := os.ReadFile(path + ".selection")
	if err != nil {
		t.Fatal(err)
	}
	input := bytes.NewReader(data)
	magic := make([]byte, 12)
	if _, err := io.ReadFull(input, magic); err != nil || string(magic) != "FYCRUNTIME1\n" {
		t.Fatal("selection fixture magic", err)
	}
	number := func() uint32 {
		var n uint32
		if err := binary.Read(input, binary.LittleEndian, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	part := func() string {
		n := number()
		if n > 2097152 {
			t.Fatal("unbounded part")
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(input, b); err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	type record struct {
		trap             uint32
		header, expected string
		accepted         bool
	}
	var records []record
	seen := map[string]bool{}
	count := number()
	for i := uint32(0); i < count; i++ {
		mode, trap := number(), number()
		header := part()
		_ = part()
		if mode != 0 {
			continue
		}
		source, err := runtimeSource(header, trap == 1)
		expected := source
		accepted := err == nil
		if err != nil {
			expected = err.Error()
		}
		key := fmt.Sprint(accepted) + expected
		if seen[key] && i < count-2 {
			continue
		}
		seen[key] = true
		records = append(records, record{trap, header, expected, accepted})
	}
	if input.Len() != 0 {
		t.Fatal("selection trailing data")
	}
	var output bytes.Buffer
	output.WriteString("FYCRTPACKTEST1\n")
	put := func(n uint32) {
		if err := binary.Write(&output, binary.LittleEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	write := func(s string) { put(uint32(len(s))); output.WriteString(s) }
	put(uint32(len(records)))
	for _, r := range records {
		put(r.trap)
		if r.accepted {
			put(1)
		} else {
			put(0)
		}
		write(r.header)
		write(r.expected)
	}
	if err := os.WriteFile(path, output.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported %d distinct complete runtime outputs/errors, %d bytes", len(records), output.Len())
}
