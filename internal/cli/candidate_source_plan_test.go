package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
)

const sourcePlanProbe = `module main;
import project_remote_plan as plan;
import std::bytes as bytes;
import std::io as io;
@effects(console) fn void write(ref u8[] value){assert(io::write_bytes(value).is_ok);}
@effects(console) fn i32 main(){
    var bytes::Buffer data=bytes::empty();var u8[4096] buffer=[0;4096];
    while(true){let usize n=switch(io::read_stdin(ref mut buffer)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] view=ref buffer;assert(bytes::append(ref mut data,view[..n]).is_ok);if(bytes::len(ref data)>65536){break;}}
    var usize meter=0;switch(plan::parse(data,ref mut meter)){
        case .err(_):{write(b"reject\n");}
        case .ok(value):{write(b"ok\n");var usize i=0;while(i<plan::count(ref value)){var u32 field=0;while(field<5){write(plan::value(ref value,i,field));write(b"\n");field+=1;}i+=1;}}
    };return 0;
}
`

func TestCandidateSourcePlanParity(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("source plan native parity requires Windows AMD64; Linux validation is deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_remote_plan")
	writeTestSource(t, root, "src/main.fy", sourcePlanProbe)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	cases := sourcePlanCases(t)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "source-plan-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, item := range cases {
				t.Run(item.name, func(t *testing.T) {
					want := "reject\n"
					if parsed, err := packagesource.Parse(item.data); err == nil {
						want = "ok\n"
						for _, pin := range parsed.Packages {
							want += strings.Join([]string{pin.Path, pin.PackageUID, pin.Version, pin.URL, pin.SHA256}, "\n") + "\n"
						}
					}
					command := exec.CommandContext(t.Context(), path)
					command.Stdin = bytes.NewReader(item.data)
					var out, diagnostics bytes.Buffer
					command.Stdout, command.Stderr = &out, &diagnostics
					if err := command.Run(); err != nil || diagnostics.Len() != 0 || out.String() != want {
						t.Fatalf("Go/Fairy mismatch: %v; output=%q want=%q stderr=%q", err, &out, want, &diagnostics)
					}
				})
			}
		})
	}
}

type sourcePlanCase struct {
	name string
	data []byte
}

func sourcePlanCases(t *testing.T) []sourcePlanCase {
	t.Helper()
	pin := packagesource.Pin{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.example/math.zip", SHA256: strings.Repeat("a", 64)}
	encode := func(pins ...packagesource.Pin) []byte {
		data, err := json.Marshal(packagesource.Plan{Schema: packagesource.Schema, Packages: pins})
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	valid := encode(pin)
	cases := []sourcePlanCase{{"canonical", valid}, {"empty", nil}, {"empty-array", encode()}, {"root-array", []byte("[]")}, {"trailing", append(bytes.Clone(valid), []byte(" {}")...)}}
	for name, edit := range map[string]func(*packagesource.Pin){
		"traversal":        func(p *packagesource.Pin) { p.Path = "../math" },
		"reserved-path":    func(p *packagesource.Pin) { p.Path = "vendor/con" },
		"nested-path":      func(p *packagesource.Pin) { p.Path = "vendor/math/src" },
		"path-case":        func(p *packagesource.Pin) { p.Path = "Vendor/math" },
		"uid":              func(p *packagesource.Pin) { p.PackageUID = "Math" },
		"range":            func(p *packagesource.Pin) { p.Version = "^1.0.0" },
		"prerelease-zero":  func(p *packagesource.Pin) { p.Version = "1.0.0-01" },
		"prerelease-build": func(p *packagesource.Pin) { p.Version = "1.2.3-alpha.1+build.001" },
		"digest-case":      func(p *packagesource.Pin) { p.SHA256 = strings.Repeat("A", 64) },
		"digest-short":     func(p *packagesource.Pin) { p.SHA256 = strings.Repeat("0", 63) },
		"digest-nonhex":    func(p *packagesource.Pin) { p.SHA256 = strings.Repeat("z", 64) },
	} {
		p := pin
		edit(&p)
		cases = append(cases, sourcePlanCase{name, encode(p)})
	}
	for name, data := range map[string]string{
		"duplicate-key":     strings.Replace(string(valid), `"path":`, `"path":"vendor/x","path":`, 1),
		"duplicate-root":    strings.Replace(string(valid), `"schema":`, `"schema":"x","schema":`, 1),
		"unknown":           strings.Replace(string(valid), `"url":`, `"URL":`, 1),
		"missing":           strings.Replace(string(valid), `"version":"1.0.0",`, "", 1),
		"wrong-type":        strings.Replace(string(valid), `"version":"1.0.0"`, `"version":3`, 1),
		"null":              strings.Replace(string(valid), `"version":"1.0.0"`, `"version":null`, 1),
		"schema":            strings.Replace(string(valid), "fairyc.sources/v1", "fairyc.sources/v2", 1),
		"escaped-key":       strings.Replace(string(valid), `"schema"`, `"\u0073chema"`, 1),
		"escaped-duplicate": strings.Replace(string(valid), `"schema":`, `"\u0073chema":"x","schema":`, 1),
		"surrogate-path":    strings.Replace(string(valid), "math.zip", `\ud800.zip`, 1),
		"raw-invalid-utf8":  strings.Replace(string(valid), "math.zip", "\xff.zip", 1),
		"trailing-comma":    strings.Replace(string(valid), `}]}`, `},]}`, 1),
	} {
		cases = append(cases, sourcePlanCase{name, []byte(data)})
	}
	p := pin
	p.Path, p.PackageUID = "vendor/answer", "example.answer"
	cases = append(cases, sourcePlanCase{"sorted", encode(pin, p)}, sourcePlanCase{"duplicate-pin", encode(pin, pin)})
	p.PackageUID = pin.PackageUID
	cases = append(cases, sourcePlanCase{"duplicate-uid", encode(pin, p)})
	p = pin
	p.PackageUID = "example.other"
	cases = append(cases, sourcePlanCase{"duplicate-path", encode(pin, p)})
	var pins []packagesource.Pin
	for i := 0; i < 33; i++ {
		p = pin
		p.Path, p.PackageUID = fmt.Sprintf("vendor/p%d", i), fmt.Sprintf("example.p%d", i)
		pins = append(pins, p)
	}
	cases = append(cases, sourcePlanCase{"32-pins", encode(pins[:32]...)}, sourcePlanCase{"33-pins", encode(pins...)})
	for _, size := range []int{65536, 65537} {
		cases = append(cases, sourcePlanCase{fmt.Sprintf("bytes-%d", size), append(bytes.Clone(valid), bytes.Repeat([]byte(" "), size-len(valid))...)})
	}
	urls := []string{
		"https://host/", "HTTPS://host/x", "https://host:/x", "https://host:1/x", "https://host:65535/x", "https://host:00001/x",
		"https://host:0/x", "https://host:65536/x", "https://host:-1/x", "https://host:abc/x", "https://:443/x", "https:///x", "https://host", "https:host/x",
		"http://host/x", "https://user:pass@host/x", "https://host/x?", "https://host/x?q", "https://host/x#", "https://host/x#f", "https://host/x y", "https://host/x\\y", "https://host/x\x00y",
		"https://host/%20%00%2F%23%3f", "https://host/%", "https://host/%zz", "https://host/a@b", "https://é.example/한글.zip", "https://%ff/x", "https://%25/x", "https://%41/x",
		"https://[::1]/x", "https://[::]/x", "https://[1:2:3:4:5:6:7:8]:443/x", "https://[::ffff:192.0.2.1]/x", "https://[fe80::1%25eth0]/x", "https://[fe80::1%25eth%200]/x",
		"https://[fe80::1%25]/x", "https://[fe80::1%25eth%ff]/x", "https://[1::2::3]/x", "https://[1:2:3:4:5:6:7]/x", "https://[127.0.0.1]/x", "https://[::ffff:192.00.2.1]/x", "https://a:b:443/x", "https://[bad]/x",
	}
	// Compare host/path byte classification and every %-escape against the Go
	// policy, rather than blessing a second handwritten URL acceptance table.
	for c := 0; c < 256; c++ {
		urls = append(urls, fmt.Sprintf("https://ho%cst/x", rune(c)), fmt.Sprintf("https://host/%%%02x", c), fmt.Sprintf("https://ho%%%02xst/x", c))
	}
	for i, value := range urls {
		p = pin
		p.URL = value
		cases = append(cases, sourcePlanCase{fmt.Sprintf("url-%03d", i), encode(p)})
	}
	return cases
}
