//go:build windows && amd64

package cdriver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestWindowsDefaultLibrarySelection(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MSVC %s; LIB=%s", tc.Version, environmentValue(tc.Environment, "LIB"))
	source := `#include <stdlib.h>
#include <string.h>
__declspec(noinline) int observe(unsigned char *p,size_t n){memset(p,42,n);return p[n-1];}
int main(void){unsigned char *p=malloc(37);if(!p)return 1;int result=observe(p,37);free(p);return result;}
`
	t.Logf("source sha256=%x", sha256.Sum256([]byte(source)))
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := BuildWindowsMSVCProbeWithLibraryLog(tc, profile, source)
			if err != nil {
				t.Fatal(err)
			}
			objects := make([][]byte, len(a.Objects))
			for i, obj := range a.Objects {
				objects[i] = obj.Bytes
				t.Logf("object %s sha256=%x", obj.Name, sha256.Sum256(obj.Bytes))
			}
			receipt, err := pecoff.InspectMSVCDefaultLibraries(objects, a.Bytes, a.LinkMap, a.LinkerVerbose)
			if err != nil {
				t.Fatal(err)
			}
			paths := make([]string, len(receipt.Libraries))
			hashes := map[string]string{}
			for i, lib := range receipt.Libraries {
				paths[i] = lib.Path
				hashes[lib.Path] = crtInputHash(t, lib.Path)
				allowed := false
				for _, dir := range filepath.SplitList(environmentValue(tc.Environment, "LIB")) {
					if strings.EqualFold(filepath.Clean(dir), filepath.Dir(lib.Path)) {
						allowed = true
					}
				}
				if !allowed {
					t.Fatalf("unrecorded library search path %s", lib.Path)
				}
				t.Logf("library %s path=%s sha256=%s loaded=%d", lib.Name, lib.Path, hashes[lib.Path], len(lib.Members))
			}
			t.Logf("defaults=%v selected=%+v image=%x", receipt.ObjectDefaults, receipt.Symbols, sha256.Sum256(a.Bytes))
			run, err := ExecuteWindowsMSVCTarget(a)
			if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("%+v %v", run, err)
			}
			// Identical objects, absolute libraries, no default search or LIB fallback.
			explicit, output, linkMap, err := linkCRTInputs(t, tc, a.Objects, paths, true, "")
			if err != nil {
				t.Fatalf("explicit link: %v\n%s", err, output)
			}
			if _, err := pecoff.InspectMSVCDefaultLibraries(objects, explicit, linkMap, []byte(output)); err != nil {
				t.Fatal(err)
			}
			if _, err := pecoff.InspectAMD64Unwind(explicit); err != nil {
				t.Fatal(err)
			}
			run, err = ExecuteWindowsMSVCTarget(WindowsHostArtifact{Bytes: explicit, Toolchain: tc, Profile: profile, ConsumerMode: C11ConsumerMode})
			if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("explicit execution %+v %v", run, err)
			}
			for _, scenario := range []struct{ name, member, symbol string }{{"libcmt.lib", "exe_main.obj", "mainCRTStartup"}, {"libucrt.lib", "malloc.obj", "malloc"}, {"libvcruntime.lib", "memset.obj", "memset"}} {
				t.Run(scenario.name, func(t *testing.T) {
					index := -1
					for i, path := range paths {
						if strings.EqualFold(filepath.Base(path), scenario.name) {
							index = i
						}
					}
					if index < 0 {
						t.Fatal("unobserved CRT input")
					}
					missing := slices.Clone(paths)
					missing[index] = filepath.Join(t.TempDir(), scenario.name)
					_, output, _, err := linkCRTInputs(t, tc, a.Objects, missing, true, "")
					requireCRTLinkFailure(t, err, output, "LNK1181", scenario.name)
					_, output, _, err = linkCRTInputs(t, tc, a.Objects, nil, false, scenario.name)
					requireCRTLinkFailure(t, err, output, "LNK1120", scenario.symbol)
					matches := 0
					for _, line := range strings.Split(string(a.LinkMap), "\n") {
						fields := strings.Fields(line)
						if len(fields) >= 4 && fields[1] == scenario.symbol {
							matches++
							if !strings.EqualFold(fields[len(fields)-1], strings.TrimSuffix(scenario.name, ".lib")+":"+scenario.member) {
								t.Fatal("unexpected demanded symbol owner")
							}
						}
					}
					if matches != 1 {
						t.Fatal("missing demanded symbol")
					}
					stripped := removeCRTMember(t, tc, paths[index], scenario.member, scenario.symbol)
					reduced := slices.Clone(paths)
					reduced[index] = stripped
					_, output, _, err = linkCRTInputs(t, tc, a.Objects, reduced, true, "")
					requireCRTLinkFailure(t, err, output, "LNK1120", scenario.symbol)
				})
			}
			// OLDNAMES is searched but contributes no member in this workload.
			curated := tc
			curatedDir := t.TempDir()
			for _, path := range paths {
				if strings.EqualFold(filepath.Base(path), "oldnames.lib") {
					continue
				}
				src, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				dst, err := os.Create(filepath.Join(curatedDir, filepath.Base(path)))
				if err != nil {
					src.Close()
					t.Fatal(err)
				}
				_, copyErr := io.Copy(dst, src)
				src.Close()
				closeErr := dst.Close()
				if copyErr != nil {
					t.Fatal(copyErr)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
			curated.Environment = replaceWindowsEnvironment(curated.Environment, "LIB", curatedDir)
			curated.Environment = replaceWindowsEnvironment(curated.Environment, "LIBPATH", "")
			_, output, _, err = linkCRTInputs(t, curated, a.Objects, nil, false, "")
			requireCRTLinkFailure(t, err, output, "LNK1104", "oldnames.lib")
			withoutOldNames, _, _, err := linkCRTInputs(t, curated, a.Objects, nil, false, "oldnames.lib")
			if err != nil {
				t.Fatal(err)
			}
			run, err = ExecuteWindowsMSVCTarget(WindowsHostArtifact{Bytes: withoutOldNames, Toolchain: tc, Profile: profile, ConsumerMode: C11ConsumerMode})
			if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("suppressed unused library %+v %v", run, err)
			}
			t.Log("absent OLDNAMES fails lookup; suppressing that unused default links and exits 42")
			for path, hash := range hashes {
				if crtInputHash(t, path) != hash {
					t.Fatalf("installed archive changed: %s", path)
				}
			}
			wrong := bytes.ReplaceAll(a.LinkMap, []byte("libucrt:initialization.obj"), []byte("libucrt:wrong.obj"))
			if _, err := pecoff.InspectMSVCDefaultLibraries(objects, a.Bytes, wrong, a.LinkerVerbose); err == nil {
				t.Fatal("accepted wrong CRT owner")
			}
			wrongLog := bytes.ReplaceAll(a.LinkerVerbose, []byte("libvcruntime.lib(initialization.obj)"), []byte("libvcruntime.lib(wrong.obj)"))
			if _, err := pecoff.InspectMSVCDefaultLibraries(objects, a.Bytes, a.LinkMap, wrongLog); err == nil {
				t.Fatal("accepted missing loaded CRT member")
			}
		})
	}
}

func crtInputHash(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func linkCRTInputs(t *testing.T, tc WindowsHostToolchain, objects []WindowsHostObject, libraries []string, isolated bool, suppress string) ([]byte, string, []byte, error) {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "program.exe")
	mapPath := filepath.Join(dir, "program.map")
	args := []string{"/NOLOGO", "/INCREMENTAL:NO", "/SUBSYSTEM:CONSOLE", "/OUT:" + exe, "/MAP:" + mapPath, "/VERBOSE"}
	for _, obj := range objects {
		path := filepath.Join(dir, obj.Name)
		if err := os.WriteFile(path, obj.Bytes, 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, path)
	}
	env := slices.Clone(tc.Environment)
	if isolated {
		args = append(args, "/NODEFAULTLIB")
		env = replaceWindowsEnvironment(env, "LIB", "")
		env = replaceWindowsEnvironment(env, "LIBPATH", "")
	}
	if suppress != "" {
		args = append(args, "/NODEFAULTLIB:"+suppress)
	}
	args = append(args, libraries...)
	output, err := runWindowsHostCommandWithLimitAndTimeout(filepath.Join(filepath.Dir(tc.Path), "link.exe"), env, dir, 8*1024*1024, 10*time.Second, args...)
	if err != nil {
		return nil, output, nil, err
	}
	image, err := readArtifact(exe)
	if err != nil {
		t.Fatal(err)
	}
	linkMap, err := readArtifact(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	return image, output, linkMap, nil
}

func requireCRTLinkFailure(t *testing.T, err error, output, code, detail string) {
	t.Helper()
	if err == nil || !strings.Contains(output, code) || !strings.Contains(strings.ToLower(output), strings.ToLower(detail)) {
		t.Fatalf("expected %s/%s: %v\n%s", code, detail, err, output)
	}
	t.Logf("expected failure %s detail=%s", code, detail)
}

func removeCRTMember(t *testing.T, tc WindowsHostToolchain, archive, base, symbol string) string {
	t.Helper()
	tool := filepath.Join(filepath.Dir(tc.Path), "lib.exe")
	dir := t.TempDir()
	listing, err := runWindowsHostCommandWithLimit(tool, tc.Environment, dir, 1024*1024, "/nologo", "/list", archive)
	if err != nil {
		t.Fatal(err)
	}
	member := ""
	for _, line := range strings.Split(listing, "\n") {
		name := strings.TrimSpace(line)
		if strings.EqualFold(filepath.Base(name), base) {
			if member != "" {
				t.Fatal("ambiguous removal member")
			}
			member = name
		}
	}
	if member == "" {
		t.Fatal("missing removal member " + base)
	}
	extracted := filepath.Join(dir, "selected.obj")
	if _, err := runWindowsHostCommand(tool, tc.Environment, dir, "/nologo", archive, "/extract:"+member, "/out:"+extracted); err != nil {
		t.Fatal(err)
	}
	raw, err := readArtifact(extracted)
	if err != nil {
		t.Fatal(err)
	}
	object, err := pecoff.InspectObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	defined := 0
	for _, s := range object.SymbolRecords {
		if s.Name == symbol && s.StorageClass == 2 && s.SectionNumber > 0 {
			defined++
		}
	}
	if defined != 1 {
		t.Fatalf("selected archive member does not uniquely define %s", symbol)
	}
	t.Logf("selected member %s defines %s sha256=%x", member, symbol, sha256.Sum256(raw))
	output := filepath.Join(dir, filepath.Base(archive))
	if _, err := runWindowsHostCommand(tool, tc.Environment, dir, "/nologo", archive, "/remove:"+member, "/out:"+output); err != nil {
		t.Fatal(err)
	}
	listing, err = runWindowsHostCommandWithLimit(tool, tc.Environment, dir, 1024*1024, "/nologo", "/list", output)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(listing, "\n") {
		if strings.EqualFold(filepath.Base(strings.TrimSpace(line)), base) {
			t.Fatal("removed member remains")
		}
	}
	t.Logf("removed actual member %s; reduced archive sha256=%s", member, crtInputHash(t, output))
	return output
}
