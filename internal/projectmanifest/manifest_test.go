package projectmanifest

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseCanonicalLocalExactManifest(t *testing.T) {
	t.Parallel()
	manifest, err := Parse([]byte(`schema = "fairyc.manifest/v1"

[package]
uid = "example.tools.app"
name = "example-app"
version = "1.2.3-rc.1+build.7"
language = "fairyc-language/0"
source = "src"

[workspace]
members = ["libs/net", "libs/util"]

[dependencies.util]
path = "libs/util"
version = "2.0.0"
features = ["base"]

[dependencies.net]
path = "libs/net"
version = "1.4.0"
features = []
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if manifest.Schema != SchemaVersion || manifest.Package.UID != "example.tools.app" || manifest.Package.Source != "src" {
		t.Fatalf("Parse() manifest = %+v", manifest)
	}
	if !slices.Equal(manifest.Workspace.Members, []string{"libs/net", "libs/util"}) {
		t.Fatalf("members = %v", manifest.Workspace.Members)
	}
	if !manifest.Workspace.Present {
		t.Fatal("workspace table presence was not retained")
	}
	if len(manifest.Dependencies) != 2 || manifest.Dependencies[0].Binding != "net" || manifest.Dependencies[1].Binding != "util" {
		t.Fatalf("dependencies = %+v", manifest.Dependencies)
	}
}

func TestParseRetainsAbsentWorkspace(t *testing.T) {
	t.Parallel()
	manifest, err := Parse([]byte(`schema = "fairyc.manifest/v1"
[package]
uid = "example.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/0"
source = "src"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if manifest.Workspace.Present || len(manifest.Workspace.Members) != 0 {
		t.Fatalf("workspace = %+v", manifest.Workspace)
	}
}

func TestParseRejectsManifestBoundaryFailures(t *testing.T) {
	t.Parallel()
	base := `schema = "fairyc.manifest/v1"
[package]
uid = "example.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/0"
source = "src"
`
	tests := []struct {
		name   string
		input  string
		reason string
		field  string
	}{
		{name: "unknown", input: base + "mystery = true\n", reason: ReasonUnknownField, field: "package.mystery"},
		{name: "duplicate", input: strings.Replace(base, "name = \"app\"", "name = \"app\"\nname = \"again\"", 1), reason: ReasonParse},
		{name: "missing package field", input: strings.Replace(base, "source = \"src\"\n", "", 1), reason: ReasonMissingField, field: "package.source"},
		{name: "unknown schema", input: strings.Replace(base, SchemaVersion, "fairyc.manifest/v2", 1), reason: ReasonUnsupportedSchema, field: "schema"},
		{name: "uid", input: strings.Replace(base, "example.app", "Example-", 1), reason: ReasonInvalidPackageUID, field: "package.uid"},
		{name: "version", input: strings.Replace(base, "1.0.0", "01.0.0", 1), reason: ReasonInvalidVersion, field: "package.version"},
		{name: "prerelease leading zero", input: strings.Replace(base, "1.0.0", "1.0.0-01", 1), reason: ReasonInvalidVersion, field: "package.version"},
		{name: "language", input: strings.Replace(base, "fairyc-language/0", "latest", 1), reason: ReasonInvalidLanguage, field: "package.language"},
		{name: "source escape", input: strings.Replace(base, "source = \"src\"", "source = \"../src\"", 1), reason: ReasonInvalidPath, field: "package.source"},
		{name: "source control", input: strings.Replace(base, "source = \"src\"", "source = \"src\\u0001bad\"", 1), reason: ReasonInvalidPath, field: "package.source"},
		{name: "windows device", input: strings.Replace(base, "source = \"src\"", "source = \"NUL\"", 1), reason: ReasonInvalidPath, field: "package.source"},
		{name: "windows trailing dot", input: strings.Replace(base, "source = \"src\"", "source = \"src.\"", 1), reason: ReasonInvalidPath, field: "package.source"},
		{name: "workspace missing members", input: base + "\n[workspace]\n", reason: ReasonMissingField, field: "workspace.members"},
		{name: "workspace order", input: base + "\n[workspace]\nmembers = [\"z\", \"a\"]\n", reason: ReasonNotSortedUnique, field: "workspace.members"},
		{name: "dependency missing features", input: base + "\n[dependencies.dep]\npath = \"deps/dep\"\nversion = \"1.0.0\"\n", reason: ReasonMissingField, field: "dependencies.dep.features"},
		{name: "feature order", input: base + "\n[dependencies.dep]\npath = \"deps/dep\"\nversion = \"1.0.0\"\nfeatures = [\"tls\", \"base\"]\n", reason: ReasonNotSortedUnique, field: "dependencies.dep.features"},
		{name: "dependency unknown", input: base + "\n[dependencies.dep]\npath = \"deps/dep\"\nversion = \"1.0.0\"\nfeatures = []\noptional = true\n", reason: ReasonUnknownField, field: "dependencies.dep.optional"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(test.input))
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Reason != test.reason || validation.Field != test.field {
				t.Fatalf("Parse() error = %#v, want reason=%q field=%q", err, test.reason, test.field)
			}
		})
	}
}

func TestParseRejectsInvalidEncodingAndSize(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		input  []byte
		reason string
	}{
		{name: "utf8", input: []byte{0xff}, reason: ReasonInvalidUTF8},
		{name: "size", input: make([]byte, MaxBytes+1), reason: ReasonInputTooLarge},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(test.input)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Reason != test.reason {
				t.Fatalf("Parse() error = %#v, want %q", err, test.reason)
			}
		})
	}
}
