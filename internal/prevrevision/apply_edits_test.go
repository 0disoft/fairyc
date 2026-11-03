package prevrevision

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// Preserve the previous splice implementation as a differential oracle for
// mixed insertion/deletion/replacement plans, including rejected overlaps.
func spliceEditsReference(input []byte, edits []edit) ([]byte, error) {
	slices.SortFunc(edits, func(left, right edit) int {
		if left.start != right.start {
			return right.start - left.start
		}
		return right.end - left.end
	})
	output := slices.Clone(input)
	previousStart := len(input) + 1
	for _, current := range edits {
		if current.start < 0 || current.end < current.start || current.end > len(input) {
			return nil, fmt.Errorf("invalid migration edit %d..%d", current.start, current.end)
		}
		if current.end > previousStart {
			return nil, fmt.Errorf("overlapping migration edit %d..%d", current.start, current.end)
		}
		output = append(output[:current.start], append([]byte(current.replacement), output[current.end:]...)...)
		previousStart = current.start
	}
	return output, nil
}

func TestApplyEditsMatchesSplicing(t *testing.T) {
	random := rand.New(rand.NewPCG(17, 42))
	for sample := range 512 {
		input := make([]byte, random.IntN(65))
		for i := range input {
			input[i] = byte(random.IntN(256))
		}
		var edits []edit
		for cursor := 0; cursor <= len(input); {
			start := min(len(input), cursor+random.IntN(3))
			end := min(len(input), start+random.IntN(4))
			replacement := []string{"", "X", "YZ", "\x00\xff"}[random.IntN(4)]
			edits = append(edits, edit{start, end, replacement})
			if random.IntN(4) == 0 {
				edits = append(edits, edit{start, start, "insert"})
			}
			cursor = max(end, start+1)
		}
		random.Shuffle(len(edits), func(i, j int) { edits[i], edits[j] = edits[j], edits[i] })
		original := slices.Clone(input)
		want, wantErr := spliceEditsReference(input, slices.Clone(edits))
		got, err := applyEdits(input, slices.Clone(edits))
		if !bytes.Equal(got, want) || (got == nil) != (want == nil) || fmt.Sprint(err) != fmt.Sprint(wantErr) {
			t.Fatalf("sample=%d edits=%+v: got %q/%v, want %q/%v", sample, edits, got, err, want, wantErr)
		}
		if len(got) != 0 {
			got[0] ^= 255
		}
		if !bytes.Equal(input, original) {
			t.Fatal("output shares input storage or rendering changed the input")
		}
	}
}

func TestApplyEditsBoundaryOrderAndErrors(t *testing.T) {
	for _, test := range []struct {
		name, input, want, failure string
		edits                      []edit
	}{
		{name: "unchanged", input: "abcd", want: "abcd"},
		{name: "empty"},
		{name: "insert-empty", want: "x", edits: []edit{{0, 0, "x"}}},
		{name: "insert-ends", input: "ab", want: "[ab]", edits: []edit{{0, 0, "["}, {2, 2, "]"}}},
		{name: "replace-all", input: "abcd", want: "xy", edits: []edit{{0, 4, "xy"}}},
		{name: "delete-all", input: "abcd", edits: []edit{{0, 4, ""}}},
		{name: "adjacent", input: "abcd", want: "XY", edits: []edit{{0, 2, "X"}, {2, 4, "Y"}}},
		{name: "same-point", input: "ab", want: "aYXb", edits: []edit{{1, 1, "X"}, {1, 1, "Y"}}},
		{name: "shared-boundaries", input: "abcd", want: "[X]Y", edits: []edit{{0, 0, "["}, {0, 2, "X"}, {2, 2, "]"}, {2, 4, "Y"}}},
		{name: "binary", input: "a\x00\xffz", want: "a\xff\x00z", edits: []edit{{1, 3, "\xff\x00"}}},
		{name: "negative", input: "abcd", failure: "invalid migration edit -1..0", edits: []edit{{-1, 0, "x"}}},
		{name: "reversed", input: "abcd", failure: "invalid migration edit 3..2", edits: []edit{{3, 2, "x"}}},
		{name: "past-end", input: "abcd", failure: "invalid migration edit 3..5", edits: []edit{{3, 5, "x"}}},
		{name: "overlap", input: "abcd", failure: "overlapping migration edit 0..3", edits: []edit{{0, 3, "x"}, {2, 4, "y"}}},
		{name: "error-order", input: "abcd", failure: "invalid migration edit 5..6", edits: []edit{{-1, 0, "x"}, {5, 6, "y"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.input)
			original := slices.Clone(input)
			got, err := applyEdits(input, slices.Clone(test.edits))
			if test.failure != "" {
				if err == nil || err.Error() != test.failure || got != nil {
					t.Fatalf("got %q, %v; want error %q", got, err, test.failure)
				}
			} else if err != nil || string(got) != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
			if !bytes.Equal(input, original) {
				t.Fatal("modified input bytes")
			}
		})
	}
}

func TestApplyEditsKeepsEmptyInputNilness(t *testing.T) {
	for _, edits := range [][]edit{nil, {{0, 0, ""}}} {
		for _, input := range [][]byte{nil, {}} {
			got, err := applyEdits(input, slices.Clone(edits))
			if err != nil || len(got) != 0 || (got == nil) != (input == nil) {
				t.Fatalf("input=%#v edits=%+v: output=%#v err=%v", input, edits, got, err)
			}
		}
	}
}

func BenchmarkApplyEditsBatch(b *testing.B) {
	for _, count := range []int{1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			line := []byte("let i32 value = 1;\n")
			input := bytes.Repeat(line, count)
			edits := make([]edit, count)
			for i := range edits {
				offset := i*len(line) + len("let i32 value = ")
				edits[i] = edit{offset, offset + 1, "42"}
			}
			want := bytes.Repeat([]byte("let i32 value = 42;\n"), count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				got, err := applyEdits(input, slices.Clone(edits))
				if err != nil || !bytes.Equal(got, want) {
					b.Fatalf("edit batch failed: %v", err)
				}
			}
		})
	}
}
