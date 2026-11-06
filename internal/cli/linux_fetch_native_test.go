package cli

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
)

// Called by the existing real Linux full-CLI replay for each profile. This adds
// no second compiler build, companion launch, CI job or certificate installation.
func checkLinuxFetch(t *testing.T, artifact cdriver.NativeArtifact) {
	t.Helper()
	image := filepath.Join(t.TempDir(), "fairyc")
	if err := os.WriteFile(image, artifact.Artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	help := linuxFetchHelp(t)
	for _, c := range []struct {
		args       []string
		code       int
		diagnostic string
	}{
		{[]string{"experimental"}, 2, "fairyc: an experimental command is required\n\n" + help},
		{[]string{"experimental", "fetch-packages", "--bad"}, 2, "fairyc: experimental fetch-packages accepts only optional --offline\n\n" + help},
		{[]string{"experimental", "fetch-packages", "--offline", "extra"}, 2, "fairyc: experimental fetch-packages accepts only optional --offline\n\n" + help},
	} {
		root := t.TempDir()
		code, out, diagnostic := fetchCLIProcess(t, image, root, c.args, "")
		if code != c.code || out != "" || diagnostic != c.diagnostic || len(linuxOfflineTree(t, root)) != 0 {
			t.Fatalf("syntax %v: (%d,%q,%q)", c.args, code, out, diagnostic)
		}
	}
	for _, args := range [][]string{{"--help"}, {"experimental", "--help"}, {"experimental", "fetch-packages", "--help"}} {
		code, out, diagnostic := fetchCLIProcess(t, image, t.TempDir(), args, "")
		if code != 0 || out != help || diagnostic != "" {
			t.Fatalf("help %v: (%d,%q,%q)", args, code, out, diagnostic)
		}
	}
	seed := remoteCLIWorkspace(t)
	tree := linuxOfflineTree(t, seed)
	reference, actual := linuxOfflineMaterialize(t, tree), linuxOfflineMaterialize(t, tree)
	for _, args := range [][]string{{"experimental", "fetch-packages", "--offline"}, {"experimental", "fetch-packages", "--offline"}, {"experimental", "fetch-packages"}} {
		want, stdout, stderr := runForTest(args, reference)
		code, out, diagnostic := fetchCLIProcess(t, image, actual, args, "")
		if want != 0 || code != want || out != stdout || diagnostic != stderr || !reflect.DeepEqual(linuxOfflineTree(t, actual), linuxOfflineTree(t, reference)) {
			t.Fatalf("cached fetch %v: (%d,%q,%q) want (%d,%q,%q)", args, code, out, diagnostic, want, stdout, stderr)
		}
	}
	for _, args := range [][]string{{"check"}, {"check", "--locked"}} {
		code, out, diagnostic := fetchCLIProcess(t, image, actual, args, "")
		if code != 0 || !strings.HasPrefix(out, "check ok:") || diagnostic != "" {
			t.Fatalf("restored workspace %v: (%d,%q,%q)", args, code, out, diagnostic)
		}
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
	cold := linuxOfflineMaterialize(t, tree)
	var plan packagesource.Plan
	if err := json.Unmarshal(tree[packagesource.FileName], &plan); err != nil {
		t.Fatal(err)
	}
	for index := range plan.Packages {
		plan.Packages[index].URL = server.URL + "/source.zip"
		if err := os.Remove(filepath.Join(cold, filepath.FromSlash(packagesource.CachePath), plan.Packages[index].SHA256+".zip")); err != nil {
			t.Fatal(err)
		}
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cold, packagesource.FileName), data, 0600); err != nil {
		t.Fatal(err)
	}
	before := linuxOfflineTree(t, cold)
	code, out, diagnostic := fetchCLIProcess(t, image, cold, []string{"experimental", "fetch-packages"}, "")
	if code != 7 || out != "" || diagnostic != "package fetch failed: source HTTPS request failed or timed out\n" || connections.Load() != 1 || requests.Load() != 0 || !reflect.DeepEqual(before, linuxOfflineTree(t, cold)) {
		t.Fatalf("untrusted TLS: (%d,%q,%q), connections=%d requests=%d", code, out, diagnostic, connections.Load(), requests.Load())
	}
}
