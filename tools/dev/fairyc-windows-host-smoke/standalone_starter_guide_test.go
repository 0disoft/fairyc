package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/projectscaffold"
)

func TestStandaloneStarterGuideMatchesGeneratedProject(t *testing.T) {
	parent := t.TempDir()
	const name = "starter-smoke"
	if err := projectscaffold.Create(parent, projectscaffold.Spec{Name: name, UID: "smoke.starter"}); err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join(parent, name, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !validStandaloneStarterGuide(readme) {
		t.Fatal("standalone smoke rejects the current generated guide")
	}
	withoutNativeTest := bytes.Replace(readme, []byte("fairyc test --native --locked\n"), nil, 1)
	if validStandaloneStarterGuide(withoutNativeTest) {
		t.Fatal("standalone smoke accepted a missing platform's native test command")
	}
}
