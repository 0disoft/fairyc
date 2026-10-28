package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxHostPathsEmissionAndPolicy(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.linux-host-paths"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	paths, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "linux_host_paths.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/linux_host_paths.fy", string(paths))
	writeTestSource(t, root, "src/main.fy", linuxHostPathClient)
	units := map[target.ID]string{}
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		code, emitted, diagnostics := runForTest([]string{"experimental", experimentalWorkspaceCCommand, "--target", string(id)}, root)
		if code != ExitSuccess || diagnostics != "" || !strings.Contains(emitted, "fairyc_private_linux_host_path_v1(") {
			t.Fatalf("%s emission: code=%d diagnostics=%s", id, code, diagnostics)
		}
		units[id] = emitted
	}
	if runtime.GOARCH != "amd64" || runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("both-target emission passed; policy execution requires Windows or Linux amd64")
	}
	id := target.X8664WindowsMSVC
	if runtime.GOOS == "linux" {
		id = target.X8664LinuxGNU
	}
	unit := units[id]
	var symbols []string
	for _, m := range regexp.MustCompile(`\b(fy_std_[A-Za-z0-9_]+)\s*\(`).FindAllStringSubmatch(unit, -1) {
		symbols = append(symbols, m[1])
	}
	storageSource := cstdlib.WindowsMSVCSourceForSymbols
	if runtime.GOOS == "linux" {
		storageSource = cstdlib.SourceForSymbols
	}
	storage, err := storageSource(symbols)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(100)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			support := scalar + "\n" + storage + "\n" + linuxHostPathStub
			var image []byte
			if runtime.GOOS == "windows" {
				tc, resolveErr := cdriver.ResolveWindowsHostMSVC()
				if resolveErr != nil {
					t.Fatal(resolveErr)
				}
				artifact, buildErr := cdriver.BuildWindowsMSVCTarget(tc, profile, unit, support)
				image, err = artifact.Bytes, buildErr
			} else {
				tc, resolveErr := cdriver.Resolve(id, "gcc")
				if resolveErr != nil {
					t.Fatal(resolveErr)
				}
				artifact, buildErr := cdriver.Build(tc, profile, unit, support)
				image, err = artifact.Bytes, buildErr
			}
			if err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(t.TempDir(), "paths.exe")
			if err := os.WriteFile(exe, image, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if output, err := exec.CommandContext(ctx, exe).CombinedOutput(); err != nil || len(output) != 0 {
				t.Fatalf("policy execution: %v %s", err, output)
			}
		})
	}
}

const linuxHostPathClient = `module main;
import linux_host_paths as paths;
import std::bytes as bytes;
@link_name("probe_mode") unsafe extern C fn void mode(u32 value);
fn void invalid(ref u8[] value,u32 kind) {
    switch(paths::decode(value)) { case .ok(_): { assert(false); } case .err(e): { assert(e.kind==kind); } };
}
@effects(filesystem,environment) fn i32 main() {
    invalid(b"",1); invalid(b"/a\x00b",1); invalid(b"\xff",1); invalid(b"\xe2\x82",1);
    var u8[4096] long_path=[0;4096]; var usize at=0;
    while(at<4096) { long_path[at]=47; at+=1; }
    let ref u8[] view=ref long_path; invalid(view,4);
    let owned=switch(paths::decode(view[..4095])) { case .ok(p): { p } case .err(_): { assert(false);return 1; } };
    assert(bytes::len(ref owned)==4095);
    let cwd=switch(paths::current()) { case .ok(p): { p } case .err(_): { assert(false);return 1; } };
    assert(bytes::equal(bytes::view(ref cwd),b"/work/한 😀"));
    let image=switch(paths::executable()) { case .ok(p): { p } case .err(_): { assert(false);return 1; } };
    assert(bytes::equal(bytes::view(ref image),b"/bin/fairyc"));
    let temp=switch(paths::temporary()) { case .ok(p): { p } case .err(_): { assert(false);return 1; } };
    assert(bytes::equal(bytes::view(ref temp),b"relative temp"));
    unsafe { mode(1); }
    let fallback=switch(paths::temporary()) { case .ok(p): { p } case .err(_): { assert(false);return 1; } };
    assert(bytes::equal(bytes::view(ref fallback),b"/tmp"));
    var u32 scenario=2;
    while(scenario<=6) {
        unsafe { mode(scenario); }
        switch(paths::current()) { case .ok(_): { assert(false); } case .err(e): {
            assert(e.kind==if(scenario==2){3}else if(scenario==3){4}else{1});
            if(scenario==2) { assert(e.os_code==2); }
        } };
        scenario+=1;
    }
    return 0;
}
`

const linuxHostPathStub = `
#include <stdint.h>
#include <string.h>
static uint32_t probe_scenario;
void probe_mode(uint32_t value) { probe_scenario=value; }
uint32_t fairyc_private_linux_host_path_v1(uint32_t kind,uint8_t *data,uint32_t capacity,uint32_t *length) {
    if(capacity!=4096) { return 22; }
    *length=0;
    if(probe_scenario==1) { return 0; }
    if(probe_scenario==2) { return 2; }
    if(probe_scenario==3) { *length=capacity; return 0; }
    if(probe_scenario==4) { data[0]='/'; data[1]='x'; *length=1; return 0; }
    if(probe_scenario==5) { data[0]=255; *length=1; return 0; }
    if(probe_scenario==6) { return 0; }
    const char *value=kind==1 ? "/work/\xed\x95\x9c \xf0\x9f\x98\x80" : kind==2 ? "relative temp" : "/bin/fairyc";
    *length=(uint32_t)strlen(value); memcpy(data,value,*length+1); return 0;
}
`
