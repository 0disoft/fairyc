package main

import (
	"os"
	"strings"
	"testing"
)

func TestStdlibQuickstartProofsHaveDedicatedSourceOwner(t *testing.T) {
	t.Parallel()

	mainSource, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	proofSource, err := os.ReadFile("stdlib_quickstart_proofs.go")
	if err != nil {
		t.Fatal(err)
	}

	mainText := string(mainSource)
	proofText := string(proofSource)
	for _, function := range []string{
		"runStdlibQuickstartProof",
		"runStdIoScalarQuickstartProof",
		"runProcessArgsQuickstartProof",
	} {
		declaration := "func " + function + "("
		if strings.Contains(mainText, declaration) {
			t.Fatalf("main.go retained %s declaration", function)
		}
		if strings.Count(proofText, declaration) != 1 {
			t.Fatalf("dedicated source must declare %s exactly once", function)
		}
		if !strings.Contains(mainText, function+"(") {
			t.Fatalf("main.go lost %s orchestration call", function)
		}
	}
	if strings.Contains(mainText, "var processArgsProofArguments") {
		t.Fatal("main.go retained processArgsProofArguments")
	}
	if strings.Count(proofText, "var processArgsProofArguments") != 1 {
		t.Fatal("dedicated source must own processArgsProofArguments exactly once")
	}
}
