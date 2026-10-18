package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHostSmokeModeDefaultsToCorrectness(t *testing.T) {
	t.Parallel()

	mode, err := parseHostSmokeMode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if mode != hostSmokeCorrectness {
		t.Fatalf("mode = %d; want correctness", mode)
	}
}

func TestParseHostSmokeModeAdmitsExplicitMeasurements(t *testing.T) {
	t.Parallel()

	mode, err := parseHostSmokeMode([]string{"--with-measurements"})
	if err != nil {
		t.Fatal(err)
	}
	if mode != hostSmokeWithMeasurements {
		t.Fatalf("mode = %d; want measurements", mode)
	}
}

func TestParseHostSmokeModeRejectsUnknownOrCombinedArguments(t *testing.T) {
	t.Parallel()

	for _, arguments := range [][]string{
		{"--measurements"},
		{"--with-measurements", "extra"},
	} {
		if _, err := parseHostSmokeMode(arguments); err == nil {
			t.Fatalf("arguments %q were accepted", arguments)
		}
	}
}

func TestWorkflowRoutesCorrectnessAndMeasurementsSeparately(t *testing.T) {
	t.Parallel()

	frontend := readWorkflow(t, "frontend.yml")
	measurements := readWorkflow(t, "measurements.yml")
	const command = "go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke"
	if strings.Count(frontend, command) != 1 || strings.Contains(frontend, "--with-measurements") {
		t.Fatalf("frontend workflow does not own exactly one correctness-only smoke command: %q", frontend)
	}
	if strings.Count(measurements, command+" --with-measurements") != 1 {
		t.Fatalf("measurement workflow does not own exactly one explicit measurement command: %q", measurements)
	}
	for _, required := range []string{"workflow_dispatch:", "schedule:", `CGO_ENABLED: "0"`} {
		if !strings.Contains(measurements, required) {
			t.Fatalf("measurement workflow is missing %q", required)
		}
	}
	for _, forbidden := range []string{"push:", "pull_request:"} {
		if strings.Contains(measurements, forbidden) {
			t.Fatalf("measurement workflow unexpectedly contains %q", forbidden)
		}
	}
}

func readWorkflow(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", ".github", "workflows", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow %s: %v", name, err)
	}
	return string(content)
}
