package atlasmap

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

func TestOverviewHeaderSurvivesMapTransport(t *testing.T) {
	t.Parallel()
	f := localCallGraphFixture(t)
	// Real source preparation can retain nonnil empty slices (for example on
	// dependency packages), while the map decoder reconstructs nil empty slices.
	p := f.partition
	p.data.header.Features = []string{}
	if len(p.data.header.Dependencies) != 0 {
		t.Fatal("expected an independent source package")
	}
	p.data.header.Dependencies = []string{}
	encoded, err := Encode(p, 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(encoded, 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	if p.data.header.Features == nil || restored.data.header.Features != nil {
		t.Fatal("regression no longer exercises empty-slice transport")
	}
	var original []byte
	var mandatory uint64
	for _, current := range []Partition{p, restored} {
		index, err := NewQueryIndex(current, f.expectation)
		if err != nil {
			t.Fatal(err)
		}
		view, err := index.Overview(generousQueryBudget(0))
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(view)
		if err != nil {
			t.Fatal(err)
		}
		if mandatory == 0 {
			row, err := json.Marshal(view.Package)
			if err != nil {
				t.Fatal(err)
			}
			mandatory = uint64(len(row) + 1)
		}
		for _, limit := range []uint64{mandatory - 1, mandatory, mandatory + 1} {
			_, err := index.Overview(QueryBudget{MaxRecords: 1, MaxSelectionBytes: limit})
			if (err != nil) != (limit < mandatory) {
				t.Fatalf("map round trip changed mandatory boundary %d/%d: %v", limit, mandatory, err)
			}
		}
		if original == nil {
			original = data
		} else if !bytes.Equal(original, data) {
			t.Fatalf("unchanged map transport changed overview facts or selection usage:\noriginal=%s\nrestored=%s", original, data)
		}
	}
}

func TestQueryHeaderCanonicalArraysStayDetached(t *testing.T) {
	t.Parallel()
	nilHeader, emptyHeader := Header{}, Header{Features: []string{}, Dependencies: []string{}, PackageReferences: []PackageReference{}, Omissions: []string{}}
	a, _ := json.Marshal(cloneQueryHeader(nilHeader))
	b, _ := json.Marshal(cloneQueryHeader(emptyHeader))
	if !bytes.Equal(a, b) || !bytes.Contains(a, []byte(`"Features":[]`)) || !bytes.Contains(a, []byte(`"Dependencies":[]`)) || !bytes.Contains(a, []byte(`"PackageReferences":[]`)) || !bytes.Contains(a, []byte(`"Omissions":[]`)) {
		t.Fatalf("empty query-header arrays are not canonical: %s / %s", a, b)
	}
	original := Header{Features: []string{"feature"}, Dependencies: []string{"dependency"}, PackageReferences: []PackageReference{{UID: "example.reference"}}, Omissions: []string{"omission"}}
	cloned := cloneQueryHeader(original)
	cloned.Features[0], cloned.Dependencies[0], cloned.PackageReferences[0].UID, cloned.Omissions[0] = "changed", "changed", "changed", "changed"
	if !slices.Equal(original.Features, []string{"feature"}) || !slices.Equal(original.Dependencies, []string{"dependency"}) || original.PackageReferences[0].UID != "example.reference" || !slices.Equal(original.Omissions, []string{"omission"}) {
		t.Fatal("canonical clone mutated its caller-owned header")
	}
}
