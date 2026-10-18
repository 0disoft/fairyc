package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPipelineFailureUsesTypesAcrossWrapping(t *testing.T) {
	for _, test := range []struct {
		err  error
		code int
		text string
	}{
		{&compiler.Error{Stage: compiler.StageMIRVerify, Cause: errors.New("detail")}, ExitInternalFailure, "internal compiler invariant"},
		{&cdriver.Error{Kind: cdriver.ErrorBuild, Cause: errors.New("detail")}, ExitBuildFailure, "build failed"},
		{&workspaceloader.LoadError{Reason: workspaceloader.ReasonManifest, Detail: "detail"}, ExitPackageFailure, "package workspace failure"},
		{&packagecache.Error{Kind: packagecache.ErrorConfiguration, Cause: errors.New("detail")}, ExitPackageFailure, "package workspace failure"},
		{errors.New("build failed: arbitrary text cannot choose a category"), ExitInternalFailure, "internal compiler invariant"},
	} {
		var stderr strings.Builder
		if code := pipelineFailure(&stderr, fmt.Errorf("outer: %w", test.err)); code != test.code || !strings.Contains(stderr.String(), test.text) {
			t.Fatalf("%T: code=%d stderr=%q", test.err, code, stderr.String())
		}
	}
}

func TestWorkspaceLoadFailureRetainsFilesystemCause(t *testing.T) {
	_, err := workspaceloader.Load(filepath.Join(t.TempDir(), "missing"))
	var failure *workspaceloader.LoadError
	if !errors.As(err, &failure) || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("lost load cause: %v", err)
	}
}

func TestCacheAndDriverFailureKinds(t *testing.T) {
	_, err := packagecache.NewMemoryStore(packagecache.Limits{})
	var cached *packagecache.Error
	if !errors.As(err, &cached) || cached.Kind != packagecache.ErrorConfiguration {
		t.Fatalf("cache configuration: %v", err)
	}
	_, err = packagecache.ParseKey("invalid")
	if !errors.As(err, &cached) || cached.Kind != packagecache.ErrorKey {
		t.Fatalf("cache key: %v", err)
	}
	_, err = cdriver.ResolveNativeToolchain("unknown", "")
	var driver *cdriver.Error
	if !errors.As(err, &driver) || driver.Kind != cdriver.ErrorToolchain {
		t.Fatalf("native resolver: %v", err)
	}
}
