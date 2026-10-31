//go:build windows && amd64

package cdriver_test

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func recordGeneratedDefaultLibraries(t *testing.T, a cdriver.WindowsHostArtifact) {
	t.Helper()
	objects := make([][]byte, len(a.Objects))
	for i, obj := range a.Objects {
		objects[i] = obj.Bytes
	}
	r, err := pecoff.InspectMSVCDefaultLibraries(objects, a.Bytes, a.LinkMap, a.LinkerVerbose)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("production defaults=%v selected=%+v", r.ObjectDefaults, r.Symbols)
	var directories []string
	for _, entry := range a.Toolchain.Environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "LIB") {
			directories = filepath.SplitList(value)
		}
	}
	for _, lib := range r.Libraries {
		allowed := false
		for _, dir := range directories {
			if strings.EqualFold(filepath.Clean(dir), filepath.Dir(lib.Path)) {
				allowed = true
			}
		}
		if !allowed {
			t.Fatalf("unrecorded production library path %s", lib.Path)
		}
		file, err := os.Open(lib.Path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		_, err = io.Copy(hash, file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("production library %s path=%s sha256=%x loaded=%d", lib.Name, lib.Path, hash.Sum(nil), len(lib.Members))
	}
}
