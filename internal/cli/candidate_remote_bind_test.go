package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/projectmanifest"
)

func TestCandidateRemoteSourceBinding(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("source binding native parity requires Windows AMD64; Linux validation is deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_remote_bind")
	pin := packagesource.Pin{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.example/math.zip", SHA256: strings.Repeat("a", 64)}
	source, err := json.Marshal(packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{pin}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := packagesource.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	var program strings.Builder
	program.WriteString(`module main;
import project_remote_plan as plan;
import project_remote_bind as bind;
import project_manifest as manifest;
import std::bytes as bytes;
import std::io as io;
@effects(console) fn i32 main(){var bytes::Buffer input=bytes::empty();
`)
	fmt.Fprintf(&program, "assert(bytes::append(ref mut input,b%s).is_ok);var usize meter=0;let sources=switch(plan::parse(input,ref mut meter)){case .ok(value):{value}case .err(_):{return 2;}};\n", strconv.Quote(string(source)))
	accepted, rejected := 0, 0
	for _, item := range remoteBindingCases() {
		// The same TOML must first be admitted by each existing manifest parser.
		rootManifest, err := projectmanifest.Parse([]byte(item.source))
		if err != nil {
			t.Fatalf("invalid binding fixture %s: %v", item.name, err)
		}
		want := packagesource.Bind(parsed, rootManifest) == nil
		if want {
			accepted++
		} else {
			rejected++
		}
		fmt.Fprintf(&program, "// %s\n{let root=switch(manifest::parse(b%s)){case .ok(value):{value}case .err(_):{return 3;}};var usize meter=0;assert((bind::validate(ref sources,ref root,ref mut meter)==0)==%t);var usize exhausted=16777216;assert(bind::validate(ref sources,ref root,ref mut exhausted)==2);}\n", item.name, strconv.Quote(item.source), want)
	}
	program.WriteString(`assert(io::write_bytes(b"bound\n").is_ok);return 0;}`)
	writeTestSource(t, root, "src/main.fy", program.String())
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "remote-bind-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), path).CombinedOutput()
			if err != nil || !bytes.Equal(out, []byte("bound\n")) {
				t.Fatalf("bind Go/Fairy comparison: %v; output=%q", err, out)
			}
			t.Logf("%d accepted, %d rejected roots and %d exhausted-budget controls", accepted, rejected, accepted+rejected)
		})
	}
}

func remoteBindingCases() []struct{ name, source string } {
	root := "schema='fairyc.manifest/v1'\n[package]\nuid='example.app'\nname='app'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
	member := "[workspace]\nmembers=['vendor/math']\n"
	dependency := "[dependencies.math]\npath='vendor/math'\nversion='1.0.0'\nfeatures=[]\n"
	valid := root + member + dependency
	return []struct{ name, source string }{
		{"exact", valid},
		{"no-dependency", root + member},
		{"missing-workspace", root},
		{"empty-members", root + "[workspace]\nmembers=[]\n"},
		{"member-case", strings.Replace(valid, "members=['vendor/math']", "members=['Vendor/math']", 1)},
		{"root-uid", strings.Replace(valid, "example.app", "example.math", 1)},
		{"version", strings.Replace(valid, "path='vendor/math'\nversion='1.0.0'", "path='vendor/math'\nversion='2.0.0'", 1)},
		{"root-source-parent", strings.Replace(valid, "source='src'", "source='vendor'", 1)},
		{"root-source-child", strings.Replace(valid, "source='src'", "source='vendor/math/src'", 1)},
		{"root-source-case", strings.Replace(valid, "source='src'", "source='VENDOR/Math'", 1)},
		{"source-prefix-only", strings.Replace(valid, "source='src'", "source='vendor/mathematics'", 1)},
		{"member-parent", strings.Replace(valid, "['vendor/math']", "['vendor','vendor/math']", 1)},
		{"member-child", strings.Replace(valid, "['vendor/math']", "['vendor/math','vendor/math/src']", 1)},
		{"member-case-collision", strings.Replace(valid, "['vendor/math']", "['Vendor/Math','vendor/math']", 1)},
		{"member-prefix-only", strings.Replace(valid, "['vendor/math']", "['vendor/math','vendor/mathematics']", 1)},
		{"cache-source", strings.Replace(valid, "source='src'", "source='build/package-sources'", 1)},
		{"cache-source-parent", strings.Replace(valid, "source='src'", "source='build'", 1)},
		{"cache-source-child", strings.Replace(valid, "source='src'", "source='BUILD/package-sources/src'", 1)},
		{"cache-source-prefix-only", strings.Replace(valid, "source='src'", "source='build/package-sources-extra'", 1)},
		{"cache-member", strings.Replace(valid, "['vendor/math']", "['build/package-sources','vendor/math']", 1)},
		{"cache-member-parent", strings.Replace(valid, "['vendor/math']", "['build','vendor/math']", 1)},
		{"cache-member-child", strings.Replace(valid, "['vendor/math']", "['BUILD/package-sources/other','vendor/math']", 1)},
		{"cache-member-prefix-only", strings.Replace(valid, "['vendor/math']", "['build/package-sources-extra','vendor/math']", 1)},
		{"other-dependency", strings.Replace(valid, "path='vendor/math'\nversion='1.0.0'", "path='vendor/other'\nversion='2.0.0'", 1)},
	}
}
