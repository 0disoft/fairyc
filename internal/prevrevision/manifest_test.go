package prevrevision

import (
	"bytes"
	"testing"
)

func TestManifestWithoutSpanNumbersIgnoresOnlySpanOffsets(t *testing.T) {
	t.Parallel()
	left := []byte(`{"span":{"source":"main.fy","startByte":12,"endByte":34},"count":12}`)
	right := []byte(`{"span":{"source":"main.fy","startByte":123,"endByte":4},"count":12}`)
	leftShape, err := manifestWithoutSpanNumbers(left)
	if err != nil {
		t.Fatal(err)
	}
	rightShape, err := manifestWithoutSpanNumbers(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftShape, rightShape) {
		t.Fatalf("span-only changes differ:\n%s\n%s", leftShape, rightShape)
	}
	changedPayload := bytes.ReplaceAll(right, []byte(`"count":12`), []byte(`"count":13`))
	payloadShape, err := manifestWithoutSpanNumbers(changedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(leftShape, payloadShape) {
		t.Fatal("non-span manifest change was ignored")
	}
}

func TestNonIsomorphicSpanOverridesAreExact(t *testing.T) {
	t.Parallel()
	tests := []struct {
		manifest, id, source string
		start, end           int
		wantStart, wantEnd   int
	}{
		{"spec/fixtures/result/manifest-v0.json", "fail.type.incompatible-error", "main.fy", 188, 199, 97, 108},
		{"spec/fixtures/semantic/manifest-v0.json", "fail.type.foreign-safe-reference", "main.fy", 48, 52, 45, 52},
		{"spec/fixtures/semantic/manifest-v0.json", "fail.flow.missing-resource-drop-effect", "main.fy", 164, 217, 193, 202},
	}
	for _, test := range tests {
		gotStart, gotEnd, ok := nonIsomorphicSpan(test.manifest, test.id, test.source, test.start, test.end)
		if !ok || gotStart != test.wantStart || gotEnd != test.wantEnd {
			t.Fatalf("override %s = %d..%d, %t; want %d..%d, true", test.id, gotStart, gotEnd, ok, test.wantStart, test.wantEnd)
		}
		if _, _, ok := nonIsomorphicSpan(test.manifest, test.id, test.source, test.start, test.end+1); ok {
			t.Fatalf("override %s accepted a different old span", test.id)
		}
	}
}
