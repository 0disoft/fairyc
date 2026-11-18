package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/target"
)

// Authored real-syscall replay. Never replace this with the Windows model pass.
func TestLinuxRemoteOfflineNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires actual Linux amd64; replay remains in the combined Linux batch")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_offline")
	writeTestSource(t, root, "src/main.fy", linuxOfflineNativeEntry(t))
	build, support := linuxOfflineBuild(t, root, target.X8664LinuxGNU)
	fixtures := linuxOfflineFixtures(t)
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildLinuxTargetParts(newLinuxRemoteBuildStage(t), compiler, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, false, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "offline")
				if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				for _, fixture := range fixtures {
					t.Run(fixture.name, func(t *testing.T) {
						reference, actual := linuxOfflineMaterialize(t, fixture.initial), linuxOfflineMaterialize(t, fixture.initial)
						result, wantErr := (&packagesource.Fetcher{}).Fetch(t.Context(), reference, true)
						cmd := exec.CommandContext(t.Context(), path)
						cmd.Dir = actual
						cmd.Env = append(os.Environ(), "PATH=")
						out, err := cmd.CombinedOutput()
						if wantErr == nil {
							want := fmt.Sprintf("ok %d %d %d %d\n", result.Packages, result.CacheHits, result.Installed, result.Reused)
							if err != nil || string(out) != want {
								t.Fatalf("outcome: %v %q want %q", err, out, want)
							}
						} else if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 {
							t.Fatalf("failure outcome: %v %q", err, out)
						}
						if !reflect.DeepEqual(linuxOfflineTree(t, reference), linuxOfflineTree(t, actual)) {
							t.Fatalf("final tree differs; %s", out)
						}
					})
				}
				for _, kind := range []string{"cache-symlink", "cache-hardlink", "member-symlink", "stale-stage"} {
					t.Run(kind, func(t *testing.T) {
						fixture := fixtures[1]
						actual := linuxOfflineMaterialize(t, fixture.initial)
						outside := t.TempDir()
						if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("outside"), 0600); err != nil {
							t.Fatal(err)
						}
						var cache string
						for name := range fixture.initial {
							if strings.HasSuffix(name, ".zip") {
								cache = name
							}
						}
						cachePath := filepath.Join(actual, filepath.FromSlash(cache))
						vendor := filepath.Join(actual, "vendor")
						if err := os.MkdirAll(vendor, 0700); err != nil {
							t.Fatal(err)
						}
						switch kind {
						case "cache-symlink":
							if err := os.Rename(cachePath, filepath.Join(outside, "cache")); err != nil {
								t.Fatal(err)
							}
							if err := os.Symlink(filepath.Join(outside, "cache"), cachePath); err != nil {
								t.Fatal(err)
							}
						case "cache-hardlink":
							if err := os.Link(cachePath, filepath.Join(outside, "linked")); err != nil {
								t.Fatal(err)
							}
						case "member-symlink":
							if err := os.Symlink(outside, filepath.Join(vendor, "math")); err != nil {
								t.Fatal(err)
							}
						case "stale-stage":
							stage := filepath.Join(vendor, ".fairyc-source-"+strings.TrimSuffix(filepath.Base(cachePath), ".zip"))
							if err := os.Mkdir(stage, 0700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(stage, "keep"), []byte("stage"), 0600); err != nil {
								t.Fatal(err)
							}
						}
						before, external := remoteTree(t, actual), remoteTree(t, outside)
						cmd := exec.CommandContext(t.Context(), path)
						cmd.Dir = actual
						cmd.Env = append(os.Environ(), "PATH=")
						out, err := cmd.CombinedOutput()
						if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 || !bytes.HasPrefix(out, []byte("error ")) {
							t.Fatalf("rejection: %v %q", err, out)
						}
						if !reflect.DeepEqual(before, remoteTree(t, actual)) || !reflect.DeepEqual(external, remoteTree(t, outside)) {
							t.Fatal("rejected namespace or outside tree changed")
						}
					})
				}
			})
		}
	}
}

func newLinuxRemoteBuildStage(t *testing.T) string {
	t.Helper()
	// TempDir owns cleanup, but its returned directory need not have mode 0700.
	stage, err := os.MkdirTemp(t.TempDir(), "stage-")
	if err != nil {
		t.Fatal(err)
	}
	return stage
}

// Check the exact deferred replay inputs on Windows without executing Linux.
func TestLinuxRemoteOfflineNativeInputs(t *testing.T) {
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_offline")
	writeTestSource(t, root, "src/main.fy", linuxOfflineNativeEntry(t))
	build, support := linuxOfflineBuild(t, root, target.X8664LinuxGNU)
	for _, symbol := range build.Unit.ForeignSymbols {
		if strings.HasPrefix(symbol, "offline_probe_") {
			t.Fatalf("test substitute in native entry: %s", symbol)
		}
	}
	for _, required := range []string{"clock_gettime(CLOCK_MONOTONIC", "renameat2(", "O_NOFOLLOW", "O_TMPFILE"} {
		if !strings.Contains(support, required) {
			t.Fatalf("native support missing %q", required)
		}
	}
}

func linuxOfflineNativeEntry(t *testing.T) string {
	t.Helper()
	entry := strings.NewReplacer(
		`@link_name("offline_probe_setup") unsafe extern C fn void setup();`+"\n", "",
		`@link_name("offline_probe_finish") unsafe extern C fn void finish();`+"\n", "",
		`unsafe{setup();}let result=offline::restore(b"/work");unsafe{finish();}`, `let result=offline::restore(b".");`,
	).Replace(linuxOfflineEntry)
	if strings.Contains(entry, "offline_probe_") || !strings.Contains(entry, `offline::restore(b".")`) {
		t.Fatal("native entry substitution missing")
	}
	return entry
}

func linuxOfflineMaterialize(t *testing.T, tree map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range linuxOfflineNames(tree) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if tree[name] == nil {
			if err := os.MkdirAll(path, 0700); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, tree[name], 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}
