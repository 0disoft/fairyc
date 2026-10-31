package foreignstructprobe_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestRecordArrayProbeEmitsBothTargets(t *testing.T) {
	for _, test := range []struct {
		name    string
		unit    foreignstructprobe.Unit
		profile cbackend.Profile
	}{
		{"windows", foreignstructprobe.Generate(), cbackend.Stage0WindowsMSVCProfile()},
		{"linux-emission-only", foreignstructprobe.GeneratePOSIX(), cbackend.Stage0Profile()},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(test.unit.FairySource)), source.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				t.Fatalf("parse: %+v", parsed.Tree.Diagnostics)
			}
			checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked)
			}
			unit, err := cbackend.Emit(checked.MIR, "main::main", test.profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, fragment := range test.unit.RequiredCFragments {
				if !strings.Contains(unit.Source, fragment) {
					t.Fatalf("missing C fact: %s", fragment)
				}
			}
			for _, family := range []string{"small", "max", "mixed", "arrays"} {
				if !strings.Contains(test.unit.FairySource, "check_record_array_"+family+"(2)") || !strings.Contains(test.unit.CompanionC, "struct fairyc_record_element_"+family) {
					t.Fatalf("missing record oracle family %s", family)
				}
			}
			if strings.Contains(test.unit.FairySource, "%!") || strings.Contains(test.unit.CompanionC, "%!") {
				t.Fatal("malformed template")
			}
		})
	}
}
