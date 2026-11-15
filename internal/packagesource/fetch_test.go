package packagesource

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func fixtureWorkspace(t *testing.T, pin Pin) string {
	t.Helper()
	dir := t.TempDir()
	manifest := strings.Replace(packageManifest, `uid="example.math"`, `uid="example.app"`, 1) + "[workspace]\nmembers=[\"vendor/math\"]\n[dependencies.math]\npath=\"vendor/math\"\nversion=\"1.0.0\"\nfeatures=[]\n"
	if err := os.WriteFile(filepath.Join(dir, "fairy.toml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Plan{Schema: Schema, Packages: []Pin{pin}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRemoteSourcesFetchTLSInstallReuseRestore(t *testing.T) {
	data := archiveBytes(t)
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); _, _ = w.Write(data) }))
	defer server.Close()
	pin := testPin()
	pin.URL = server.URL + "/math.zip"
	pin.SHA256 = hash(data)
	dir := fixtureWorkspace(t, pin)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lock := []byte("preserve graph lock bytes\n")
	if err := root.WriteFile("fairy.lock", lock, 0644); err != nil {
		t.Fatal(err)
	}
	manifestBefore, _ := root.ReadFile("fairy.toml")
	pinsBefore, _ := root.ReadFile(FileName)
	f := Fetcher{Client: server.Client()}
	r, err := f.Fetch(context.Background(), dir, false)
	if err != nil || r.Downloaded != 1 || r.Installed != 1 {
		t.Fatalf("cold: %+v %v", r, err)
	}
	r, err = f.Fetch(context.Background(), dir, false)
	if err != nil || r.CacheHits != 1 || r.Reused != 1 {
		t.Fatalf("warm: %+v %v", r, err)
	}
	if err := root.RemoveAll(pin.Path); err != nil {
		t.Fatal(err)
	}
	server.Close()
	r, err = f.Fetch(context.Background(), dir, true)
	if err != nil || r.CacheHits != 1 || r.Installed != 1 || requests.Load() != 1 {
		t.Fatalf("restore: %+v %v, requests %d", r, err, requests.Load())
	}
	for name, want := range map[string][]byte{"fairy.lock": lock, "fairy.toml": manifestBefore, FileName: pinsBefore} {
		got, err := root.ReadFile(name)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("modified input %s", name)
		}
	}
	entries, err := root.Open("vendor")
	if err != nil {
		t.Fatal(err)
	}
	defer entries.Close()
	names, err := entries.Readdirnames(-1)
	if err != nil || len(names) != 1 || names[0] != "math" {
		t.Fatalf("staging leaked: %v %v", names, err)
	}
	if _, err := root.Lstat(fetchLock); !os.IsNotExist(err) {
		t.Fatal("lock leaked")
	}

	for _, edit := range []string{"changed", "extra", "missing"} {
		t.Run(edit, func(t *testing.T) {
			source := "vendor/math/src/math.fy"
			original, _ := root.ReadFile(source)
			switch edit {
			case "changed":
				err = root.WriteFile(source, []byte("user edit"), 0644)
			case "extra":
				err = root.WriteFile("vendor/math/extra.txt", []byte("user data"), 0644)
			case "missing":
				err = root.Remove(source)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.Fetch(context.Background(), dir, true); err == nil {
				t.Fatal("existing conflict accepted")
			}
			if edit == "changed" {
				got, _ := root.ReadFile(source)
				if string(got) != "user edit" {
					t.Fatal("overwrote user edit")
				}
			}
			if edit == "extra" {
				got, _ := root.ReadFile("vendor/math/extra.txt")
				if string(got) != "user data" {
					t.Fatal("deleted extra file")
				}
				if err := root.Remove("vendor/math/extra.txt"); err != nil {
					t.Fatal(err)
				}
			}
			if edit == "missing" {
				if _, err := root.Lstat(source); !os.IsNotExist(err) {
					t.Fatal("merged incomplete existing tree")
				}
			}
			if err := root.WriteFile(source, original, 0644); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := root.Mkdir(fetchLock, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Fetch(context.Background(), dir, true); err == nil {
		t.Fatal("concurrent/stale fetch lock ignored")
	}
	if _, err := root.Lstat(fetchLock); err != nil {
		t.Fatal("removed another fetch lock")
	}
	if err := root.Remove(fetchLock); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteSourcesCancelledStageCleanup(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := publish(ctx, root, testPin(), bundle{files: map[string][]byte{"data": []byte("x")}}); err == nil {
		t.Fatal("cancelled publication succeeded")
	}
	if _, err := root.Lstat("vendor/math"); !os.IsNotExist(err) {
		t.Fatal("published cancelled package")
	}
	if _, err := root.Lstat("vendor"); !os.IsNotExist(err) {
		t.Fatalf("pre-cancelled publication created vendor: %v", err)
	}
}
