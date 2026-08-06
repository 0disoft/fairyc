package main

import (
	"os"
	"strings"
	"testing"
)

func TestRawStdlibNativeProbeRunnersHaveDedicatedOwner(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	ownerSource, err := os.ReadFile("native_stdlib_probes.go")
	if err != nil {
		t.Fatal(err)
	}
	mainText := string(mainSource)
	ownerText := string(ownerSource)
	for _, name := range []string{
		"runWindowsFileProbe",
		"runOwnedTextStorageProbe",
		"runBufferFailureProbe",
		"runArenaFailureProbe",
	} {
		declaration := "func " + name + "("
		if strings.Contains(mainText, declaration) {
			t.Errorf("main.go still owns %s", name)
		}
		if count := strings.Count(ownerText, declaration); count != 1 {
			t.Errorf("native_stdlib_probes.go declaration count for %s = %d; want 1", name, count)
		}
		if count := strings.Count(mainText, name+"("); count != 1 {
			t.Errorf("main.go call count for %s = %d; want 1", name, count)
		}
	}
	for _, sourceFactory := range []string{
		"cstdlib.WindowsFileSource()",
		"cstdlib.WindowsFileProbeSource()",
		"cstdlib.WindowsMSVCOwnedTextProbeImplementationSource()",
		"cstdlib.OwnedTextProbeSource()",
		"cstdlib.WindowsMSVCBufferProbeImplementationSource()",
		"cstdlib.BufferProbeSource()",
		"cstdlib.WindowsMSVCArenaProbeImplementationSource()",
		"cstdlib.ArenaProbeSource()",
	} {
		if count := strings.Count(ownerText, sourceFactory); count != 1 {
			t.Errorf("native_stdlib_probes.go count for %s = %d; want 1", sourceFactory, count)
		}
	}
	if count := strings.Count(ownerText, "runWindowsSeparateTranslationUnitProbe("); count != 4 {
		t.Errorf("native_stdlib_probes.go separate-TU execution count = %d; want 4", count)
	}
}
