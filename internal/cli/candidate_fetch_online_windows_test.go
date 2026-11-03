package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
)

func candidateOnlineWorkspace(t *testing.T, address string, archive []byte) (string, string) {
	t.Helper()
	root := remoteCLIWorkspace(t)
	cache := filepath.Join(root, packagesource.CachePath)
	items, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if err = os.Remove(filepath.Join(cache, item.Name())); err != nil {
			t.Fatal(err)
		}
	}
	digest := sha256.Sum256(archive)
	name := hex.EncodeToString(digest[:])
	plan := packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: address, SHA256: name}}}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, packagesource.FileName, string(data))
	writeWorkspaceFile(t, root, "fairy.lock", "keep existing graph lock\n")
	return root, name
}

func verifyCandidateFetchOnline(t *testing.T, image string) string {
	t.Helper()
	// Use the actual runnable CLI workspace package, byte-for-byte in both roots.
	seed := remoteCLIWorkspace(t)
	items, err := os.ReadDir(filepath.Join(seed, packagesource.CachePath))
	if err != nil {
		t.Fatal(err)
	}
	good, err := os.ReadFile(filepath.Join(seed, packagesource.CachePath, items[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	bad := []byte("not a ZIP")
	identity := remoteArchiveBytes(t, 8, remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.Replace(remoteLibraryManifest, "example.math", "example.wrong", 1))}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;pub fn i32 value(){return 42;}")})
	var calls, target atomic.Int32
	server := candidateHTTPSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("ambient credentials")
		}
		switch r.URL.EscapedPath() {
		case "/ok", "/a%2Fb.zip":
			w.Header().Set("Set-Cookie", "ambient=forbidden")
			_, _ = w.Write(good)
		case "/chunked":
			w.(http.Flusher).Flush()
			_, _ = w.Write(good)
		case "/redirect":
			http.Redirect(w, r, "/target", 302)
		case "/target":
			target.Add(1)
			_, _ = w.Write(good)
		case "/auth":
			w.Header().Set("WWW-Authenticate", "Negotiate")
			w.WriteHeader(401)
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(good)
		case "/identity":
			_, _ = w.Write(identity)
		case "/bad":
			_, _ = w.Write(bad)
		case "/large":
			w.Header().Set("Content-Length", fmt.Sprint(packagesource.MaxArchiveBytes+1))
			_, _ = w.Write(good)
		default:
			w.WriteHeader(404)
		}
	}))
	online := []string{"experimental", "fetch-packages"}
	offline := append(append([]string{}, online...), "--offline")
	for _, fixture := range []struct {
		name, path string
		payload    []byte
		setup      string
		success    bool
		requests   int
	}{
		{"cold", "/ok", good, "", true, 2}, {"escaped-path", "/a%2Fb.zip", good, "", true, 2}, {"chunked", "/chunked", good, "", true, 2},
		{"redirect", "/redirect", good, "", false, 2}, {"auth", "/auth", good, "", false, 2}, {"encoding", "/gzip", good, "", false, 2}, {"oversized", "/large", good, "", false, 2}, {"wrong-hash", "/bad", good, "", false, 2}, {"invalid-zip", "/bad", bad, "", false, 2}, {"identity", "/identity", identity, "", false, 2}, {"create-cache-parents", "/ok", good, "parents", true, 2},
		{"corrupt-cache", "/ok", good, "corrupt", false, 0}, {"lock-held", "/ok", good, "lock", false, 0}, {"stale-stage", "/ok", good, "stage", false, 2}, {"cache-conflict", "/ok", good, "conflict", false, 0},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			roots := []string{}
			for i := 0; i < 2; i++ {
				root, digest := candidateOnlineWorkspace(t, server.URL+fixture.path, fixture.payload)
				roots = append(roots, root)
				switch fixture.setup {
				case "parents":
					if err = os.Remove(filepath.Join(root, packagesource.CachePath)); err != nil {
						t.Fatal(err)
					}
					if err = os.Remove(filepath.Join(root, "build")); err != nil {
						t.Fatal(err)
					}
				case "corrupt":
					writeWorkspaceFile(t, root, packagesource.CachePath+"/"+digest+".zip", "corrupt")
				case "lock":
					if err = os.Mkdir(filepath.Join(root, ".fairyc-fetch.lock"), 0700); err != nil {
						t.Fatal(err)
					}
				case "stage":
					if i == 1 {
						writeWorkspaceFile(t, root, packagesource.CachePath+"/.fairyc-download-"+digest+".tmp", "preserve stale file")
					}
				case "conflict":
					if err = os.Mkdir(filepath.Join(root, packagesource.CachePath, digest+".zip"), 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			before := calls.Load()
			if fixture.setup == "stage" {
				tree := remoteTree(t, roots[1])
				code, out, diagnostic := fetchCLIProcess(t, image, roots[1], online, "")
				if code != 7 || out != "" || !strings.Contains(diagnostic, "cannot publish immutable source archive cache") || !reflect.DeepEqual(tree, remoteTree(t, roots[1])) {
					t.Fatalf("stale stage: %d %q %q", code, out, diagnostic)
				}
				if calls.Load()-before != 1 {
					t.Fatal("stage request count")
				}
				return
			}
			reference, goErr := (packagesource.Fetcher{Client: server.Client()}).Fetch(t.Context(), roots[0], false)
			code, out, diagnostic := fetchCLIProcess(t, image, roots[1], online, "")
			if fixture.success {
				want := fmt.Sprintf("packages fetched: %d package(s), %d downloaded, %d cache hit(s), %d installed, %d reused\n", reference.Packages, reference.Downloaded, reference.CacheHits, reference.Installed, reference.Reused)
				if goErr != nil || code != 0 || out != want || diagnostic != "" {
					t.Fatalf("online: Go=%+v %v Fairy=%d %q %q", reference, goErr, code, out, diagnostic)
				}
			} else if goErr == nil || code != 7 || out != "" || !strings.HasPrefix(diagnostic, "package fetch failed:") {
				t.Fatalf("failure: Go=%v Fairy=%d %q %q", goErr, code, out, diagnostic)
			}
			if calls.Load()-before != int32(fixture.requests) || target.Load() != 0 {
				t.Fatal("unexpected HTTPS request count")
			}
			if !reflect.DeepEqual(remoteTree(t, roots[0]), remoteTree(t, roots[1])) {
				t.Fatalf("Go/Fairy final trees differ: %v / %v", remoteTree(t, roots[0]), remoteTree(t, roots[1]))
			}
			if !fixture.success {
				return
			}
			for _, args := range [][]string{online, offline} {
				code, out, diagnostic = fetchCLIProcess(t, image, roots[1], args, "")
				if code != 0 || out != "packages fetched: 1 package(s), 0 downloaded, 1 cache hit(s), 0 installed, 1 reused\n" || diagnostic != "" {
					t.Fatalf("warm: %d %q %q", code, out, diagnostic)
				}
			}
			if calls.Load()-before != int32(fixture.requests) {
				t.Fatal("warm/offline performed network request")
			}
		})
	}

	// A later invalid package leaves earlier verified cache/package publications.
	t.Run("partial-progress", func(t *testing.T) {
		roots := []string{}
		for i := 0; i < 2; i++ {
			root, _ := candidateOnlineWorkspace(t, server.URL+"/ok", good)
			roots = append(roots, root)
			manifest, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
			if err != nil {
				t.Fatal(err)
			}
			writeWorkspaceFile(t, root, "fairy.toml", strings.Replace(string(manifest), `members=["vendor/math"]`, `members=["vendor/math","vendor/zzbad"]`, 1))
			data, err := os.ReadFile(filepath.Join(root, packagesource.FileName))
			if err != nil {
				t.Fatal(err)
			}
			var plan packagesource.Plan
			if err = json.Unmarshal(data, &plan); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(bad)
			plan.Packages = append(plan.Packages, packagesource.Pin{Path: "vendor/zzbad", PackageUID: "example.zzbad", Version: "1.0.0", URL: server.URL + "/bad", SHA256: hex.EncodeToString(digest[:])})
			data, err = json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			writeWorkspaceFile(t, root, packagesource.FileName, string(data))
		}
		before := calls.Load()
		_, err := (packagesource.Fetcher{Client: server.Client()}).Fetch(t.Context(), roots[0], false)
		code, out, diagnostic := fetchCLIProcess(t, image, roots[1], online, "")
		if err == nil || code != 7 || out != "" || !strings.HasPrefix(diagnostic, "package fetch failed:") || calls.Load()-before != 4 {
			t.Fatalf("partial progress: %v %d %q %q", err, code, out, diagnostic)
		}
		if !reflect.DeepEqual(remoteTree(t, roots[0]), remoteTree(t, roots[1])) {
			t.Fatal("partial Go/Fairy trees differ")
		}
		if _, err = os.Stat(filepath.Join(roots[1], "vendor/math/fairy.toml")); err != nil {
			t.Fatal("earlier completed package lost", err)
		}
	})
	// Completed online publication survives a subsequent console output failure.
	root, digest := candidateOnlineWorkspace(t, server.URL+"/ok", good)
	code, out, diagnostic := fetchCLIProcess(t, image, root, online, "stdout")
	if code != 4 || out != "" || diagnostic != "fairyc: write output failed\n" {
		t.Fatalf("output failure: %d %q %q", code, out, diagnostic)
	}
	for _, name := range []string{packagesource.CachePath + "/" + digest + ".zip", "vendor/math/fairy.toml"} {
		if _, err = os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal("completed publication lost", err)
		}
	}
	return root
}

func TestCandidateFetchOnlineCommand(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_HTTPS_TRUST_TESTS") != "1" {
		t.Skip("opt-in temporary loopback TLS trust")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "project_fetch_cli")
	writeTestSource(t, root, "src/main.fy", `module main;import project_fetch_cli as fetch;@effects(filesystem,console,network,clock) fn i32 main(){return fetch::arguments();}`)
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 128<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "online-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			verifyCandidateFetchOnline(t, path)
		})
	}
}

func TestCandidateFetchOnlineCLI(t *testing.T) {
	if os.Getenv("FAIRYC_RUN_HTTPS_TRUST_TESTS") != "1" {
		t.Skip("opt-in temporary loopback TLS trust")
	}
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			if err := os.Remove(filepath.Join(filepath.Dir(image), "fairyc-candidate-build.exe")); err != nil {
				t.Fatal(err)
			}
			verifyCandidateFetchCLI(t, image)
			root := verifyCandidateFetchOnline(t, image)
			// A real cold online installation can immediately feed authored check and lock.
			if err := os.Remove(filepath.Join(root, "fairy.lock")); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"check"}, {"check", "--locked"}} {
				code, out, diagnostic := fetchCLIProcess(t, image, root, args, "")
				if code != 0 || diagnostic != "" || !strings.HasPrefix(out, "check ok:") {
					t.Fatalf("online workspace %v: %d %q %q", args, code, out, diagnostic)
				}
			}
		})
	}
}
