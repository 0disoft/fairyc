package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDependencyWorkloadIsolation(t *testing.T) {
	for _, selection := range []struct{ languages, workloads string }{
		{"fairy,rust", "rust-dependencies"},
		{"rust", "tokenizer,rust-dependencies"},
		{"c", "rust-dependencies"},
	} {
		output := filepath.Join(t.TempDir(), "must-not-exist")
		err := entry(context.Background(), []string{"run", "--output", output, "--languages", selection.languages, "--workloads", selection.workloads})
		if err == nil || !strings.Contains(err.Error(), "must run separately") {
			t.Fatalf("accepted incomparable matrix: %+v, %v", selection, err)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatal("invalid matrix created experiment state")
		}
	}
	if err := validateWorkloadSelection([]string{"rust"}, []string{"rust-dependencies"}); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyReportPreparation(t *testing.T) {
	r := fixtureReceipt()
	r.Options.Languages = []string{"rust"}
	r.Options.Workloads = []string{"rust-dependencies"}
	r.Cases[0].Language, r.Cases[0].Workload = "rust", "rust-dependencies"
	if err := validateReceipt(r); err == nil {
		t.Fatal("passed dependency case without lock or preparation was accepted")
	}
	r.Cases[0].DependencyLock = "version = 4\n"
	r.Cases[0].Preparation = []Command{{Argv: []string{"cargo", "fetch", "--locked"}}}
	if err := validateReceipt(r); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.html")
	if err := renderHTML(r, path, "en", []string{"en"}); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(html), "Downloads finish before timing") || strings.Contains(string(html), "with no third-party dependencies") {
		t.Fatal("dependency report has inaccurate cache/dependency notes", err)
	}
}
