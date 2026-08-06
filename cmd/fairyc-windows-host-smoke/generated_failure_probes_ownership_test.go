package main

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratedFailureProbesHaveDedicatedOwner(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	ownerSource, err := os.ReadFile("generated_failure_probes.go")
	if err != nil {
		t.Fatal(err)
	}

	mainText := string(mainSource)
	ownerText := string(ownerSource)
	for _, name := range []string{
		"generatedFairyRegionFailureSource",
		"generatedFairyBufferInitialFailureSource",
		"generatedFairyBufferResizeFailureSource",
		"generatedFairyBufferAppendByteFailureSource",
		"generatedFairyStringAppendFailureSource",
		"generatedFairyCStringFailureSource",
		"generatedFairyStringCopyFailureSource",
	} {
		declaration := "const " + name + " ="
		if strings.Contains(mainText, declaration) {
			t.Errorf("main.go still owns %s", name)
		}
		if count := strings.Count(ownerText, declaration); count != 1 {
			t.Errorf("generated_failure_probes.go declaration count for %s = %d; want 1", name, count)
		}
	}

	for _, name := range []string{
		"runGeneratedFairyRegionFailureProbe",
		"generatedFairyRegionFailureProbeSource",
		"runGeneratedFairyBufferFailureProbe",
		"generatedFairyBufferInitialFailureProbeSource",
		"generatedFairyBufferResizeFailureProbeSource",
		"generatedFairyBufferAppendByteFailureProbeSource",
		"runGeneratedFairyStringAppendFailureProbe",
		"generatedFairyStringAppendFailureProbeSource",
		"runGeneratedFairyCStringFailureProbe",
		"generatedFairyCStringFailureProbeSource",
		"runGeneratedFairyStringCopyFailureProbe",
		"generatedFairyStringCopyFailureProbeSource",
	} {
		declaration := "func " + name + "("
		if strings.Contains(mainText, declaration) {
			t.Errorf("main.go still owns %s", name)
		}
		if count := strings.Count(ownerText, declaration); count != 1 {
			t.Errorf("generated_failure_probes.go declaration count for %s = %d; want 1", name, count)
		}
	}

	for _, name := range []string{
		"runGeneratedFairyRegionFailureProbe",
		"runGeneratedFairyBufferFailureProbe",
		"runGeneratedFairyStringAppendFailureProbe",
		"runGeneratedFairyCStringFailureProbe",
		"runGeneratedFairyStringCopyFailureProbe",
	} {
		call := name + "("
		if count := strings.Count(mainText, call); count != 1 {
			t.Errorf("main.go call count for %s = %d; want 1", name, count)
		}
	}
}
