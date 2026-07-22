package packagegraph

import (
	"bytes"
	"strings"
	"testing"
)

func TestCoordinateIDIsSourceIndependentAndCoordinateSensitive(t *testing.T) {
	t.Parallel()
	left := Identity{PackageUID: "example.net", SourceIdentity: "source-a", VersionIdentity: "1.2.0", Features: []string{"tls"}}
	right := left
	right.SourceIdentity = "source-b"
	leftID, err := ComputeCoordinateID(CoordinateOf(left))
	if err != nil {
		t.Fatal(err)
	}
	rightID, err := ComputeCoordinateID(CoordinateOf(right))
	if err != nil {
		t.Fatal(err)
	}
	if leftID != rightID {
		t.Fatalf("source-only change altered coordinate: %s != %s", leftID, rightID)
	}
	changed := CoordinateOf(right)
	changed.VersionIdentity = "1.3.0"
	changedID, err := ComputeCoordinateID(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedID == leftID {
		t.Fatal("version change preserved package coordinate")
	}
	parsed, err := ParseCoordinateID(leftID.String())
	if err != nil || parsed != leftID {
		t.Fatalf("ParseCoordinateID() = %s, %v", parsed, err)
	}
}

func TestCoordinateEncodingIsFramedAndRejectsNoncanonicalInput(t *testing.T) {
	t.Parallel()
	left, err := EncodeCoordinate(Coordinate{PackageUID: "ab", VersionIdentity: "c", Features: []string{"d"}})
	if err != nil {
		t.Fatal(err)
	}
	right, err := EncodeCoordinate(Coordinate{PackageUID: "a", VersionIdentity: "bc", Features: []string{"d"}})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(left, right) {
		t.Fatal("coordinate encoding is ambiguous")
	}
	for _, value := range []Coordinate{
		{VersionIdentity: "1"},
		{PackageUID: "pkg", VersionIdentity: "1", Features: []string{"z", "a"}},
		{PackageUID: "pkg", VersionIdentity: "1", Features: []string{"a", "a"}},
	} {
		if _, err := EncodeCoordinate(value); err == nil {
			t.Fatalf("EncodeCoordinate(%+v) accepted", value)
		}
	}
	for _, value := range []string{
		strings.TrimPrefix(CoordinateIDPrefix+strings.Repeat("a", 64), CoordinateIDPrefix),
		CoordinateIDPrefix + strings.Repeat("A", 64),
		CoordinateIDPrefix + strings.Repeat("g", 64),
	} {
		if _, err := ParseCoordinateID(value); err == nil {
			t.Fatalf("ParseCoordinateID(%q) accepted", value)
		}
	}
}
