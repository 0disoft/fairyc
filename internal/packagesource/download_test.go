package packagesource

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const packageManifest = `schema="fairyc.manifest/v1"
[package]
uid="example.math"
name="math"
version="1.0.0"
language="fairyc-language/20"
source="src"
`

type zipEntry struct {
	name, body string
	mode       os.FileMode
}

func archiveBytes(t *testing.T, extras ...zipEntry) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	entries := append([]zipEntry{{"fairy.toml", packageManifest, 0644}, {"src/math.fy", "module math;pub fn i32 value(){return 42;}", 0644}}, extras...)
	for _, entry := range entries {
		h := zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		h.SetMode(entry.mode)
		f, err := w.CreateHeader(&h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestRemoteSourcesHTTPSCacheAndOffline(t *testing.T) {
	data := archiveBytes(t, zipEntry{".gitignore", "build/\n", 0644})
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = w.Write(data) }))
	defer server.Close()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	pin := testPin()
	pin.URL = server.URL + "/math.zip"
	pin.SHA256 = hash(data)
	f := Fetcher{Client: server.Client()}
	b, raw, hit, err := f.prepare(context.Background(), root, pin, "fairyc-language/20", false)
	if err != nil || hit || len(b.files) != 3 || !bytes.Equal(raw, data) {
		t.Fatalf("cold: hit=%v err=%v", hit, err)
	}
	if err := storeArchive(root, pin, raw); err != nil {
		t.Fatal(err)
	}
	for _, offline := range []bool{false, true} {
		if _, _, hit, err := f.prepare(context.Background(), root, pin, "fairyc-language/20", offline); err != nil || !hit {
			t.Fatalf("warm/offline: %v %v", hit, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("cache hit performed network: %d", calls.Load())
	}
	if err := root.WriteFile(CachePath+"/"+pin.SHA256+".zip", []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, offline := range []bool{false, true} {
		if _, _, _, err := f.prepare(context.Background(), root, pin, "fairyc-language/20", offline); err == nil {
			t.Fatal("corrupt cache accepted")
		}
	}
	pin.SHA256 = strings.Repeat("0", 64)
	if _, _, _, err := f.prepare(context.Background(), root, pin, "fairyc-language/20", true); err == nil {
		t.Fatal("offline miss accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("cache corruption or offline miss triggered network")
	}
}

func TestRemoteSourcesRedirectTLSHashAndBounds(t *testing.T) {
	data := archiveBytes(t)
	var targetCalls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			w.Header().Set("Location", "/target")
			w.WriteHeader(302)
		case "/big":
			w.Header().Set("Content-Length", fmt.Sprint(MaxArchiveBytes+1))
		case "/bad-status":
			w.WriteHeader(503)
		default:
			targetCalls.Add(1)
			_, _ = w.Write(data)
		}
	}))
	defer server.Close()
	f := Fetcher{Client: server.Client()}
	pin := testPin()
	for _, name := range []string{"redirect", "big", "bad-status"} {
		pin.URL = server.URL + "/" + name
		if _, err := f.download(context.Background(), pin); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	if targetCalls.Load() != 0 {
		t.Fatal("redirect followed")
	}
	pin.URL = server.URL + "/target"
	if _, err := f.downloadWithin(context.Background(), pin, 8); err == nil {
		t.Fatal("remaining compressed budget ignored")
	}
	if _, err := (Fetcher{}).download(context.Background(), pin); err == nil {
		t.Fatal("untrusted TLS accepted")
	}
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, _, _, err := f.prepare(context.Background(), root, pin, "fairyc-language/20", false); err == nil {
		t.Fatal("bad hash accepted")
	}
	if _, err := root.Lstat(CachePath); !os.IsNotExist(err) {
		t.Fatal("bad hash populated cache")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.download(ctx, pin); err == nil {
		t.Fatal("cancelled download succeeded")
	}
}

func TestRemoteSourcesArchiveContainment(t *testing.T) {
	for _, entry := range []zipEntry{
		{"../escape", "x", 0644}, {"/absolute", "x", 0644}, {"C:/outside", "x", 0644}, {"src\\bad.fy", "x", 0644},
		{"src/../bad.fy", "x", 0644}, {"src/NUL.fy", "x", 0644}, {"Src/other.fy", "x", 0644},
		{".git/config", "x", 0644}, {"src/.GiT/hooks/data", "x", 0644},
		{"src/math.fy", "duplicate", 0644}, {"src", "file over implicit directory", 0644}, {"src/link", "target", os.ModeSymlink | 0777},
		{"src/link/", "", os.ModeSymlink | 0777}, {"src/too-big", strings.Repeat("x", MaxFileBytes+1), 0644},
	} {
		t.Run(entry.name, func(t *testing.T) {
			if _, err := unpack(context.Background(), archiveBytes(t, entry), testPin(), "fairyc-language/20"); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	data := archiveBytes(t)
	pin := testPin()
	pin.PackageUID = "example.other"
	if _, err := unpack(context.Background(), data, pin, "fairyc-language/20"); err == nil {
		t.Fatal("UID mismatch accepted")
	}
	pin = testPin()
	pin.Version = "2.0.0"
	if _, err := unpack(context.Background(), data, pin, "fairyc-language/20"); err == nil {
		t.Fatal("version mismatch accepted")
	}
	if _, err := unpack(context.Background(), data, testPin(), "fairyc-language/999"); err == nil {
		t.Fatal("language mismatch accepted")
	}
	if _, err := unpackWithin(context.Background(), data, testPin(), "fairyc-language/20", 8, MaxFiles); err == nil {
		t.Fatal("remaining expanded budget ignored")
	}
	if _, err := unpackWithin(context.Background(), data, testPin(), "fairyc-language/20", MaxTotalBytes, 1); err == nil {
		t.Fatal("remaining file budget ignored")
	}
}

func TestRemoteSourcesRejectSymlinkCache(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(base, "build")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := storeArchive(root, testPin(), []byte("data")); err == nil {
		t.Fatal("symlink cache accepted")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("escaped root")
	}
}
