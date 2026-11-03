package cli

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCandidateRemoteOffline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows offline restoration; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_remote_offline")
	writeTestSource(t, root, "src/main.fy", `module main;import project_remote_offline as offline;import std::process as process;import std::io as io;
 @effects(filesystem,clock,console) fn i32 main(){var u8[4096] buffer=[0;4096];let usize n=switch(process::arg(1,ref mut buffer)){case .ok(value):{value}case .err(_):{return 2;}};let ref u8[] path=ref buffer;
 switch(offline::restore(path[..n])){case .ok(value):{discard io::write_bytes(b"ok ");discard io::write_usize(value.packages);discard io::write_bytes(b" ");discard io::write_usize(value.cache_hits);discard io::write_bytes(b" ");discard io::write_usize(value.installed);discard io::write_bytes(b" ");discard io::write_usize(value.reused);discard io::write_bytes(b"\n");return 0;}
 case .err(p):{discard io::write_bytes(b"error ");discard io::write_u32(p.stage);discard io::write_bytes(b" ");discard io::write_u32(p.code);discard io::write_bytes(b" ");discard io::write_u32(p.cleanup_code);discard io::write_bytes(b"\n");return 7;}};return 7;}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range build.Unit.ForeignSymbols {
		if strings.Contains(strings.ToLower(symbol), "http") {
			t.Fatalf("offline code imports network: %s", symbol)
		}
	}
	type fixture struct {
		name    string
		archive []byte
		setup   func(string, string)
	}
	stored, deflated := remoteArchiveValid(t, zip.Store), remoteArchiveValid(t, zip.Deflate)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("outside stays unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	junction := func(link string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), "cmd", "/c", "mklink", "/J", link, outside)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("create test junction: %v %s", err, out)
		}
	}
	write := func(root, name string, data []byte) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	library := func(root string) {
		write(root, "vendor/math/fairy.toml", []byte(remoteLibraryManifest))
		write(root, "vendor/math/src/math.fy", []byte("module math;pub fn i32 answer(){return 42;}\n"))
		write(root, "vendor/math/.review", []byte{0, 255, 1, 2})
	}
	fixtures := []fixture{
		{name: "stored", archive: stored}, {name: "deflated", archive: deflated},
		{name: "zip64", archive: remoteArchiveZIP64(t, deflated)},
		{"exact-reuse", deflated, func(root, _ string) { library(root) }},
		{"empty-extra-directory", deflated, func(root, _ string) {
			library(root)
			if err := os.MkdirAll(filepath.Join(root, "vendor/math/empty"), 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"cache-miss", deflated, func(root, cache string) {
			if err := os.Remove(filepath.Join(root, cache)); err != nil {
				t.Fatal(err)
			}
		}},
		{"corrupt-cache", deflated, func(root, cache string) { write(root, cache, []byte("broken")) }},
		{"corrupt-warm-cache", deflated, func(root, cache string) { library(root); write(root, cache, []byte("broken")) }},
		{"missing-cache-directory", deflated, func(root, cache string) {
			if err := os.Remove(filepath.Join(root, cache)); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, "build/package-sources")); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-junction", deflated, func(root, _ string) { junction(filepath.Join(root, "vendor/math")) }},
		{"cache-junction", deflated, func(root, cache string) {
			if err := os.Remove(filepath.Join(root, cache)); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(root, "build/package-sources")
			if err := os.Remove(parent); err != nil {
				t.Fatal(err)
			}
			junction(parent)
		}},
		{"changed-existing", deflated, func(root, _ string) { library(root); write(root, "vendor/math/src/math.fy", []byte("changed")) }},
		{"extra-existing", deflated, func(root, _ string) { library(root); write(root, "vendor/math/extra", []byte("extra")) }},
		{"incomplete-existing", deflated, func(root, _ string) { write(root, "vendor/math/fairy.toml", []byte(remoteLibraryManifest)) }},
		{"destination-file", deflated, func(root, _ string) { write(root, "vendor/math", []byte("keep")) }},
		{"lock-held", deflated, func(root, _ string) {
			if err := os.Mkdir(filepath.Join(root, ".fairyc-fetch.lock"), 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"cache-case", deflated, func(root, cache string) {
			path := filepath.Join(root, cache)
			if err := os.Rename(path, path+".tmp"); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(path+".tmp", strings.TrimSuffix(path, ".zip")+".ZIP"); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-case", deflated, func(root, _ string) {
			if err := os.MkdirAll(filepath.Join(root, "vendor/Math"), 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"stale-stage", deflated, func(root, cache string) {
			write(root, "vendor/.fairyc-source-"+strings.TrimSuffix(filepath.Base(cache), ".zip")+"/keep", []byte("preserve stale stage"))
		}},
	}
	longPath := remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(remoteLibraryManifest)}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")}, remoteArchiveEntry{name: "src/" + strings.Repeat("a", 260), data: []byte("cannot publish this component on Windows")})
	fixtures = append(fixtures, fixture{name: "stage-write-failure-cleanup", archive: longPath})
	secondArchive := remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.Replace(remoteLibraryManifest, "example.math", "example.answer", 1))}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")})
	secondPackage := func(root string) {
		manifestPath := filepath.Join(root, "fairy.toml")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		write(root, "fairy.toml", []byte(strings.Replace(string(data), "members=['vendor/math']", "members=['vendor/answer','vendor/math']", 1)))
		data, err = os.ReadFile(filepath.Join(root, "fairy.sources.json"))
		if err != nil {
			t.Fatal(err)
		}
		var plan packagesource.Plan
		if err = json.Unmarshal(data, &plan); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(secondArchive)
		digest := hex.EncodeToString(sum[:])
		plan.Packages = append(plan.Packages, packagesource.Pin{Path: "vendor/answer", PackageUID: "example.answer", Version: "1.0.0", URL: "https://packages.invalid/answer.zip", SHA256: digest})
		data, err = json.Marshal(plan)
		if err != nil {
			t.Fatal(err)
		}
		write(root, "fairy.sources.json", data)
		write(root, "build/package-sources/"+digest+".zip", secondArchive)
	}
	fixtures = append(fixtures, fixture{"two-packages", deflated, func(root, _ string) { secondPackage(root) }}, fixture{"partial-success", deflated, func(root, cache string) { secondPackage(root); write(root, cache, []byte("corrupt second package")) }})
	for _, name := range []string{"../escape", ".git/config", "src/con"} {
		fixtures = append(fixtures, fixture{name: name, archive: remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(remoteLibraryManifest)}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")}, remoteArchiveEntry{name: name, data: []byte("bad")})})
	}
	fixtures = append(fixtures, fixture{name: "identity", archive: remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.Replace(remoteLibraryManifest, "example.math", "example.other", 1))}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")})})
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "offline-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, f := range fixtures {
				t.Run(f.name, func(t *testing.T) {
					roots := []string{t.TempDir(), t.TempDir()}
					sum := sha256.Sum256(f.archive)
					digest := hex.EncodeToString(sum[:])
					cache := "build/package-sources/" + digest + ".zip"
					pins, err := json.Marshal(packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.invalid/math.zip", SHA256: digest}}})
					if err != nil {
						t.Fatal(err)
					}
					for _, r := range roots {
						write(r, "fairy.toml", []byte("schema='fairyc.manifest/v1'\n[package]\nuid='example.app'\nname='app'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n[workspace]\nmembers=['vendor/math']\n[dependencies.math]\npath='vendor/math'\nversion='1.0.0'\nfeatures=[]\n"))
						write(r, "fairy.sources.json", pins)
						write(r, "fairy.lock", []byte("unchanged lock\n"))
						write(r, cache, f.archive)
						if f.setup != nil {
							f.setup(r, cache)
						}
					}
					ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
					defer cancel()
					if f.name == "stale-stage" {
						before := remoteTree(t, roots[1])
						out, err := exec.CommandContext(ctx, path, roots[1]).CombinedOutput()
						e, ok := err.(*exec.ExitError)
						if !ok || e.ExitCode() != 7 || !reflect.DeepEqual(before, remoteTree(t, roots[1])) {
							t.Fatalf("stale stage changed: err=%v output=%s", err, out)
						}
						return
					}
					requests := 0
					fetcher := packagesource.Fetcher{Client: &http.Client{Transport: remoteOfflineTransport(func(*http.Request) (*http.Response, error) {
						requests++
						return nil, fmt.Errorf("offline transport invoked")
					})}}
					goResult, goErr := fetcher.Fetch(ctx, roots[0], true)
					cmd := exec.CommandContext(ctx, path, roots[1])
					out, nativeErr := cmd.CombinedOutput()
					if (goErr == nil) != (nativeErr == nil) {
						t.Fatalf("Go=%v candidate=%v output=%s", goErr, nativeErr, out)
					}
					if goErr == nil {
						want := fmt.Sprintf("ok %d %d %d %d\n", goResult.Packages, goResult.CacheHits, goResult.Installed, goResult.Reused)
						if string(out) != want {
							t.Fatalf("result=%q want=%q", out, want)
						}
					} else {
						e, ok := nativeErr.(*exec.ExitError)
						if !ok || e.ExitCode() != 7 {
							t.Fatalf("candidate error=%v output=%s", nativeErr, out)
						}
						var stage, code, cleanup uint32
						if _, err := fmt.Sscanf(string(out), "error %d %d %d\n", &stage, &code, &cleanup); err != nil || cleanup != 0 {
							t.Fatalf("cleanup failed: %v output=%s", err, out)
						}
					}
					a, b := remoteTree(t, roots[0]), remoteTree(t, roots[1])
					if !reflect.DeepEqual(a, b) {
						t.Fatalf("post-operation tree differs: Go=%v candidate=%v output=%s", a, b, out)
					}
					if goErr == nil {
						again, err := fetcher.Fetch(ctx, roots[0], true)
						if err != nil || again.Reused != again.Packages || again.Installed != 0 {
							t.Fatalf("warm Go restore: %+v %v", again, err)
						}
						out, err := exec.CommandContext(ctx, path, roots[1]).CombinedOutput()
						want := fmt.Sprintf("ok %d %d 0 %d\n", again.Packages, again.CacheHits, again.Reused)
						if err != nil || string(out) != want || !reflect.DeepEqual(remoteTree(t, roots[0]), remoteTree(t, roots[1])) {
							t.Fatalf("fresh-process reuse: %v output=%s", err, out)
						}
					}
					if requests != 0 {
						t.Fatalf("offline fetch made %d HTTP requests", requests)
					}
				})
			}
		})
	}
	if data, err := os.ReadFile(filepath.Join(outside, "keep")); err != nil || string(data) != "outside stays unchanged" {
		t.Fatalf("junction target changed: %v %q", err, data)
	}
}

type remoteOfflineTransport func(*http.Request) (*http.Response, error)

func (f remoteOfflineTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func remoteTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			result[relative+"/"] = "directory"
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			result[relative] = "link:" + target
			return err
		}
		if !entry.Type().IsRegular() {
			result[relative] = "special:" + entry.Type().String()
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		result[relative] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
