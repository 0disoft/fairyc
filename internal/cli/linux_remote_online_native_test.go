package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/target"
)

// Authored actual filesystem + parent/worker/libcurl replay. Cached operations
// must never open a socket; an absent-cache request must reject untrusted TLS
// without publishing bytes. No system trust store is modified by this test.
func TestLinuxRemoteOnlineNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux amd64 and libcurl development/runtime prerequisites; execution remains batched")
	}
	var connections, requests atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.StartTLS()
	defer server.Close()
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_online")
	writeTestSource(t, root, "src/main.fy", linuxOnlineNativeEntry(t))
	build, support := linuxRemoteBuild(t, root, target.X8664LinuxGNU, true)
	support += linuxOnlineNativeObserver
	for _, compiler := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(compiler+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildLinuxTargetParts(newLinuxRemoteBuildStage(t), compiler, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, false, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "online")
				if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				run := func(tree map[string][]byte) (string, []byte, error) {
					t.Helper()
					workspace := linuxOfflineMaterialize(t, tree)
					cmd := exec.CommandContext(t.Context(), path)
					cmd.Dir, cmd.Env = workspace, append(os.Environ(), "PATH=")
					out, err := cmd.CombinedOutput()
					return workspace, out, err
				}
				for _, fixture := range linuxOfflineFixtures(t) {
					if fixture.name == "cache-miss" || fixture.name == "cache-directory-miss" {
						continue // The explicit missing-cache trust probe below owns this path.
					}
					t.Run("cache/"+fixture.name, func(t *testing.T) {
						tree := linuxOnlineClone(fixture.initial)
						var plan packagesource.Plan
						if json.Unmarshal(tree[packagesource.FileName], &plan) == nil && plan.Schema == packagesource.Schema {
							for index := range plan.Packages {
								plan.Packages[index].URL = fmt.Sprintf("%s/package-%d.zip", server.URL, index)
							}
							data, err := json.Marshal(plan)
							if err != nil {
								t.Fatal(err)
							}
							tree[packagesource.FileName] = data
						}
						reference := linuxOfflineMaterialize(t, tree)
						calls := 0
						fetcher := packagesource.Fetcher{Client: &http.Client{Transport: remoteOfflineTransport(func(*http.Request) (*http.Response, error) {
							calls++
							return nil, fmt.Errorf("cache operation attempted transport")
						})}}
						result, wantErr := fetcher.Fetch(t.Context(), reference, false)
						before := connections.Load()
						actual, out, err := run(tree)
						if wantErr == nil {
							want := fmt.Sprintf("ok %d %d %d %d %d\n", result.Packages, result.Downloaded, result.CacheHits, result.Installed, result.Reused)
							if err != nil || string(out) != want {
								t.Fatalf("success: %v %q want %q", err, out, want)
							}
						} else if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 || !bytes.HasPrefix(out, []byte("error ")) {
							t.Fatalf("failure: %v %q", err, out)
						}
						if calls != 0 || connections.Load() != before || !reflect.DeepEqual(linuxOfflineTree(t, reference), linuxOfflineTree(t, actual)) {
							t.Fatal("cache operation used network or produced a different final tree")
						}
					})
				}
				t.Run("missing-cache-untrusted-TLS", func(t *testing.T) {
					fixture := linuxOnlineCold(t, []byte("never accepted untrusted payload"), server.URL+"/cold.zip")
					before := connections.Load()
					actual, out, err := run(fixture.initial)
					if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 || string(out) != "error 10 3 65596 0\n" {
						t.Fatalf("untrusted TLS: %v %q", err, out)
					}
					if connections.Load()-before != 1 || requests.Load() != 0 || !reflect.DeepEqual(fixture.initial, linuxOfflineTree(t, actual)) {
						t.Fatal("rejected TLS retried, reached HTTP or changed the namespace")
					}
				})
			})
		}
	}
}

func TestLinuxRemoteOnlineNativeInputs(t *testing.T) {
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_online")
	entry := linuxOnlineNativeEntry(t)
	writeTestSource(t, root, "src/main.fy", entry)
	_, support := linuxRemoteBuild(t, root, target.X8664LinuxGNU, true)
	for _, forbidden := range []string{"online_probe_setup", "FAIRYC_ONLINE_CASE", "remote_nodes[", "online_payloads["} {
		if strings.Contains(entry+support+linuxOnlineNativeObserver, forbidden) {
			t.Fatalf("model input in native replay: %s", forbidden)
		}
	}
	for _, required := range []string{"O_NOFOLLOW", "O_TMPFILE", "renameat2(", "PR_SET_PDEATHSIG", "poll(", "dlopen(", "CURLOPT_SSL_VERIFYPEER"} {
		if !strings.Contains(support, required) {
			t.Fatalf("native support missing %s", required)
		}
	}
}

func linuxOnlineNativeEntry(t *testing.T) string {
	t.Helper()
	entry := strings.NewReplacer(
		`@link_name("online_probe_setup") unsafe extern C fn void setup();`+"\n", "",
		`unsafe{setup();}let result=online::fetch(b"/work");unsafe{finish();}`, `let result=online::fetch(b".");unsafe{finish();}`,
	).Replace(linuxOnlineEntry)
	if strings.Contains(entry, "online_probe_setup") || !strings.Contains(entry, `online::fetch(b".")`) {
		t.Fatal("native online entry substitution missing")
	}
	return entry
}

// Observation only, after the real Fairy owner has released its resources.
const linuxOnlineNativeObserver = `
#include <assert.h>
void online_probe_finish(void){int status=0;errno=0;assert(waitpid(-1,&status,WNOHANG)==-1 && errno==ECHILD);}
`
