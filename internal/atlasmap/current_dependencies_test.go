package atlasmap

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestCurrentDependencyMembership(t *testing.T) {
	for _, count := range []int{0, 1, 32, 128} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			value := currentDependencyFixture(count)
			before := cloneCurrentPackage(value)
			if err := validateCurrentPackage(value); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value, before) {
				t.Fatal("current-source validation changed caller-owned input")
			}
		})
	}
	for _, position := range []int{0, 16, 31} {
		t.Run(fmt.Sprintf("unlisted-reference-%d", position), func(t *testing.T) {
			value := currentDependencyFixture(32)
			target := value.Dependencies[position]
			for i := range value.PackageReferences {
				if value.PackageReferences[i].Instance == target {
					value.PackageReferences[i].Instance = packagegraph.InstanceID(sha256.Sum256([]byte("unlisted"))).String()
				}
			}
			sortCurrentDependencyReferences(&value)
			if err := validateCurrentPackage(value); err == nil || err.Error() != "current package reference is not a direct dependency" {
				t.Fatalf("unlisted reference accepted or error changed: %v", err)
			}
		})
	}
	for _, test := range []struct {
		name   string
		mutate func(*CurrentPackage)
		want   string
	}{
		{"unsorted", func(v *CurrentPackage) { slices.Reverse(v.Dependencies) }, "current dependencies are not sorted"},
		{"duplicate", func(v *CurrentPackage) { v.Dependencies[1] = v.Dependencies[0] }, "current dependencies contain invalid or duplicate text"},
		{"malformed", func(v *CurrentPackage) { v.Dependencies[0] = "bad-id" }, `current dependency "bad-id" is invalid`},
		{"missing", func(v *CurrentPackage) { v.PackageReferences = v.PackageReferences[1:] }, "current package references do not cover self and direct dependencies"},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := currentDependencyFixture(32)
			test.mutate(&value)
			if err := validateCurrentPackage(value); err == nil || err.Error() != test.want {
				t.Fatalf("invalid dependency input: got %v, want %s", err, test.want)
			}
		})
	}
}

func currentDependencyFixture(count int) CurrentPackage {
	seed := sha256.Sum256([]byte("current-dependency-fixture"))
	value := CurrentPackage{
		UID: "root", Language: "stage0", SourceKind: "path",
		Instance: packagegraph.InstanceID(seed), Coordinate: packagegraph.CoordinateID(seed), Source: packagecontent.Digest(seed),
		Files: []CurrentFile{{Path: "main.fy", Digest: packagecontent.FileDigest(seed), LineMap: packagecontent.LineMapID(seed), Kind: semanticsnapshot.FileKindSource}},
	}
	value.PackageReferences = append(value.PackageReferences, PackageReference{UID: value.UID, Instance: value.Instance.String(), Coordinate: value.Coordinate.String()})
	for i := range count {
		uid := fmt.Sprintf("dependency-%d", i)
		digest := sha256.Sum256([]byte(uid))
		instance := packagegraph.InstanceID(digest).String()
		value.Dependencies = append(value.Dependencies, instance)
		value.PackageReferences = append(value.PackageReferences, PackageReference{UID: uid, Instance: instance, Coordinate: packagegraph.CoordinateID(digest).String()})
	}
	slices.Sort(value.Dependencies)
	sortCurrentDependencyReferences(&value)
	return value
}

func sortCurrentDependencyReferences(value *CurrentPackage) {
	slices.SortFunc(value.PackageReferences, func(a, b PackageReference) int { return cmp.Compare(a.Instance, b.Instance) })
}

func BenchmarkValidateCurrentDependencies(b *testing.B) {
	for _, count := range []int{32, 1024, 8192} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			value := currentDependencyFixture(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := validateCurrentPackage(value); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
