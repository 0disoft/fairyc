package packagesource

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func testPin() Pin {
	return Pin{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.example/math.zip", SHA256: strings.Repeat("a", 64)}
}

func TestSourcePlanExactPins(t *testing.T) {
	pin := testPin()
	data, _ := json.Marshal(Plan{Schema: Schema, Packages: []Pin{pin}})
	plan, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	root := projectmanifest.Manifest{Package: projectmanifest.Package{UID: "example.app", Source: "src"}, Workspace: projectmanifest.Workspace{Members: []string{pin.Path}}, Dependencies: []projectmanifest.Dependency{{Path: pin.Path, Version: pin.Version}}}
	if err := Bind(plan, root); err != nil {
		t.Fatal(err)
	}
	root.Dependencies[0].Version = "2.0.0"
	if err := Bind(plan, root); err == nil {
		t.Fatal("version mismatch admitted")
	}
	root.Dependencies = nil
	root.Workspace.Members = nil
	if err := Bind(plan, root); err == nil {
		t.Fatal("implicit member admitted")
	}
	for _, bad := range []string{
		strings.Replace(string(data), `"schema":`, `"schema":"other","schema":`, 1),
		strings.Replace(string(data), `"path":`, `"path":"vendor/other","path":`, 1),
		strings.Replace(string(data), `"url":`, `"URL":`, 1),
		string(data) + ` {}`, strings.Replace(string(data), `"version":"1.0.0",`, "", 1),
	} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatalf("invalid JSON accepted: %s", bad)
		}
	}
}

func TestSourcePlanRejectsUntrustedCoordinates(t *testing.T) {
	for _, edit := range []func(*Pin){
		func(p *Pin) { p.Path = "../outside" }, func(p *Pin) { p.Path = "vendor/con" }, func(p *Pin) { p.Path = "vendor/a/b" },
		func(p *Pin) { p.Version = "^1.0.0" }, func(p *Pin) { p.Version = "1.0.0-01" }, func(p *Pin) { p.SHA256 = strings.Repeat("A", 64) },
		func(p *Pin) { p.URL = "http://packages.example/math.zip" }, func(p *Pin) { p.URL = "https://user:password@packages.example/math.zip" },
		func(p *Pin) { p.URL += "?key=value" }, func(p *Pin) { p.URL += "#fragment" }, func(p *Pin) { p.URL = "file:///math.zip" },
	} {
		p := testPin()
		edit(&p)
		if err := Validate(Plan{Schema: Schema, Packages: []Pin{p}}); err == nil {
			t.Fatalf("bad pin accepted: %+v", p)
		}
	}
	p := testPin()
	if err := Validate(Plan{Schema: Schema, Packages: []Pin{p, p}}); err == nil {
		t.Fatal("duplicate accepted")
	}
}
