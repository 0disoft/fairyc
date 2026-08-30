package packagecache

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestKeyRoundTripAndInvalidation(t *testing.T) {
	t.Parallel()
	baseline := testInputs(t)
	key, err := Compute(baseline)
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if got, want := key.String(), "fycache2-sha256-b27788473412f220dd6d393bf934f88fc7149d8c3d8718f48fa342f9d185aaa6"; got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
	parsed, err := ParseKey(key.String())
	if err != nil || parsed != key {
		t.Fatalf("ParseKey() = %v, %v; want %v", parsed, err, key)
	}

	changed := baseline
	changed.CompilerBuild = "compiler:b"
	other, err := Compute(changed)
	if err != nil {
		t.Fatalf("Compute(changed) error = %v", err)
	}
	if other == key {
		t.Fatal("compiler build change preserved cache key")
	}
}

func TestKeyRejectsNoncanonicalSets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*Inputs)
		want   string
	}{
		{name: "profile flags", mutate: func(value *Inputs) { value.ProfileFlags = []string{"optimized", "debug"} }, want: "profileFlags"},
		{name: "features", mutate: func(value *Inputs) { value.Features = []string{"tls", "base"} }, want: "features"},
		{name: "dependencies", mutate: func(value *Inputs) { value.Dependencies = append(value.Dependencies, value.Dependencies[0]) }, want: "dependencies"},
		{name: "missing package", mutate: func(value *Inputs) { value.Package = packagegraph.InstanceID{} }, want: "package must be set"},
		{name: "missing source", mutate: func(value *Inputs) { value.Source = packagecontent.Digest{} }, want: "source must be set"},
		{name: "missing dependency hash", mutate: func(value *Inputs) { value.Dependencies[0].Interface = moduleinterface.Hash{} }, want: "dependency 0"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			value := testInputs(t)
			test.mutate(&value)
			if _, err := Compute(value); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Compute() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParseKeyRejectsNoncanonicalText(t *testing.T) {
	t.Parallel()
	valid, err := Compute(testInputs(t))
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	for _, value := range []string{
		strings.TrimPrefix(valid.String(), KeyPrefix),
		valid.String()[:len(valid.String())-1],
		strings.ToUpper(valid.String()),
		KeyPrefix + strings.Repeat("g", 64),
	} {
		if _, err := ParseKey(value); err == nil {
			t.Fatalf("ParseKey(%q) accepted", value)
		}
	}
}

func TestCanonicalizeDependenciesSortsWithoutMutating(t *testing.T) {
	t.Parallel()
	inputs := testInputs(t)
	first := inputs.Dependencies[0]
	second := Dependency{Coordinate: mustCoordinate(t, "fypc1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91"), Interface: first.Interface}
	values := []Dependency{second, first}
	ordered, err := CanonicalizeDependencies(values)
	if err != nil {
		t.Fatalf("CanonicalizeDependencies() error = %v", err)
	}
	if ordered[0].Coordinate != first.Coordinate || values[0].Coordinate != second.Coordinate {
		t.Fatalf("ordered=%v input=%v", ordered, values)
	}
	if _, err := CanonicalizeDependencies([]Dependency{first, first}); err == nil {
		t.Fatal("duplicate dependencies accepted")
	}
}

func testInputs(t *testing.T) Inputs {
	t.Helper()
	return Inputs{
		CompilerBuild:        "compiler:a",
		LanguageRevision:     "fairyc-language/1",
		MIRVersion:           "fairyc-mir/0",
		TargetSchema:         "fairyc.target-spec/v0:x86_64-windows-msvc",
		ABIProfile:           "microsoft-x64/v0",
		ToolchainFingerprint: "msvc:19.44:c11",
		Backend:              "constrained-c/v0",
		ProfileFlags:         []string{"checked", "low"},
		RuntimeABI:           "fairyc-runtime/0",
		StandardLibrary:      mustInstance(t, "fypi1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91"),
		Package:              mustInstance(t, "fypi1-sha256-04fc4400f5e875c5c22ab2d50765d721df38a12a19ef3508deb432c7f4da92f3"),
		Source:               mustSource(t, "fysrc1-sha256-dd19fc7a8f93795ca7bf1abb70fa2f7bbbb0a8849fe410f6b4c080b55817e9a3"),
		Dependencies: []Dependency{{
			Coordinate: mustCoordinate(t, "fypc1-sha256-903a7b9974986cccf6166e8978ca9cb26273d4776af6d1a5dd3b04e7060e2f41"),
			Interface:  mustInterface(t, "fyif4-sha256-a5e79db9b105ffac6d57d0ff054b75813344fdf4e9f9a5c1873da8907ae281b9"),
		}},
		Features: []string{"base", "tls"},
	}
}

func mustCoordinate(t *testing.T, value string) packagegraph.CoordinateID {
	t.Helper()
	parsed, err := packagegraph.ParseCoordinateID(value)
	if err != nil {
		t.Fatalf("ParseCoordinateID(%q): %v", value, err)
	}
	return parsed
}

func mustInstance(t *testing.T, value string) packagegraph.InstanceID {
	t.Helper()
	parsed, err := packagegraph.ParseInstanceID(value)
	if err != nil {
		t.Fatalf("ParseInstanceID(%q): %v", value, err)
	}
	return parsed
}

func mustSource(t *testing.T, value string) packagecontent.Digest {
	t.Helper()
	parsed, err := packagecontent.ParseDigest(value)
	if err != nil {
		t.Fatalf("ParseDigest(%q): %v", value, err)
	}
	return parsed
}

func mustInterface(t *testing.T, value string) moduleinterface.Hash {
	t.Helper()
	parsed, err := moduleinterface.ParseHash(value)
	if err != nil {
		t.Fatalf("ParseHash(%q): %v", value, err)
	}
	return parsed
}
