package main

import (
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestStudyCorpusCompilesAndChecksOracles(t *testing.T) {
	for _, w := range workloads() {
		t.Run(w.Name, func(t *testing.T) {
			r, _, err := compileWorkload(w)
			if err != nil {
				t.Fatal(err)
			}
			status, _, err := oracle(w, r.MIR)
			if err != nil {
				t.Fatal(err)
			}
			if status == "" {
				t.Fatal("missing oracle classification")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
				u, err := cbackend.Emit(r.MIR, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				if w.Name == "128-function-emission" && len(u.Source) < 16000 {
					t.Fatal("growth workload collapsed to a small C unit")
				}
			}
		})
	}
}
func TestStudyRejectsTrapExitWithoutReceipt(t *testing.T) {
	w := workload{Trap: "integer-overflow"}
	if checkNative(w, cdriver.NativeOutcome{ExitCode: 131}) == nil {
		t.Fatal("ordinary reserved exit accepted as trap proof")
	}
	if checkNative(workload{}, cdriver.NativeOutcome{ExitCode: 42, TrapCategory: "integer-overflow"}) == nil {
		t.Fatal("trap accepted as normal return")
	}
	if checkNative(workload{}, cdriver.NativeOutcome{ExitCode: 42, Stdout: "extra"}) == nil {
		t.Fatal("unexpected output accepted")
	}
}
func TestStudyRejectsUnboundedSamplesBeforeDiscovery(t *testing.T) {
	for _, n := range []int{0, 6} {
		if _, err := run(n, ""); err == nil {
			t.Fatal("unbounded study accepted")
		}
	}
}
