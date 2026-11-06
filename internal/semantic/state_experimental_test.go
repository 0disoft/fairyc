package semantic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeExperimentalStateCodecsAreIsolated(t *testing.T) {
	file := parsePackageFile(t, 7, "state.fy", `module state;
@contract_uid("test.dynamic-state")
state { cell bytes blob; cell text label; map bytes text labels; map text bytes blobs; }`)
	before, err := json.Marshal(file.Module)
	if err != nil {
		t.Fatal(err)
	}
	input := semantic.PackageInput{Files: []semantic.File{file}}
	for range 2 {
		ordinary, err := semantic.AnalyzePackages(input)
		if err != nil || ordinary.CompletedThrough == semantic.PhaseFlow || len(ordinary.HIR.StateLayouts) != 0 || len(ordinary.Diagnostics) == 0 {
			t.Fatal("ordinary codec admission widened", ordinary, err)
		}
		for _, diagnostic := range ordinary.Diagnostics {
			if diagnostic.Code != "E7203" {
				t.Fatal(diagnostic)
			}
		}
		experimental, err := semantic.AnalyzeExperimentalStatePackages(input)
		if err != nil || experimental.CompletedThrough != semantic.PhaseFlow || len(experimental.Diagnostics)+len(experimental.Unsupported) != 0 || len(experimental.HIR.StateLayouts) != 1 {
			t.Fatal("explicit codec admission", experimental, err)
		}
		fields := experimental.HIR.StateLayouts[0].Fields
		if fields[0].ValueTypeID != contractstate.Bytes.TypeID() || fields[1].ValueTypeID != contractstate.Text.TypeID() || fields[2].KeyTypeID != contractstate.Bytes.TypeID() || fields[2].ValueTypeID != contractstate.Text.TypeID() || fields[3].KeyTypeID != contractstate.Text.TypeID() {
			t.Fatal("codec identities", fields)
		}
	}
	after, err := json.Marshal(file.Module)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("codec selection mutated source", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := semantic.AnalyzeExperimentalStatePackagesContext(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled semantic analysis", err)
	}
	if _, err := semantic.AnalyzeExperimentalStatePackages(semantic.PackageInput{Files: []semantic.File{file, file}}); err == nil {
		t.Fatal("explicit entry skipped package validation")
	}
}

func TestAnalyzeExperimentalStateCodecsKeepSchemaChecks(t *testing.T) {
	for _, field := range []string{"cell bytes<u64> value;", "cell custom::text value;", "cell text[] value;", "map text String values;", "cell bytes value; cell text value;"} {
		text := `module state; @contract_uid("test.dynamic-state") state { ` + field + ` }`
		file := parsePackageFile(t, 7, "state.fy", text)
		checked, err := semantic.AnalyzeExperimentalStatePackages(semantic.PackageInput{Files: []semantic.File{file}})
		if err != nil || checked.CompletedThrough == semantic.PhaseFlow || len(checked.Diagnostics) == 0 || len(checked.HIR.StateLayouts) != 0 {
			t.Fatalf("invalid schema %s: %+v %v", field, checked, err)
		}
	}
	file := parsePackageFile(t, 7, "state.fy", strings.Replace(`module state; @contract_uid("test.dynamic-state") state {cell bytes value;}`, `@contract_uid("test.dynamic-state")`, "", 1))
	checked, err := semantic.AnalyzeExperimentalStatePackages(semantic.PackageInput{Files: []semantic.File{file}})
	if err != nil || checked.CompletedThrough == semantic.PhaseFlow || len(checked.Diagnostics) == 0 {
		t.Fatal("explicit codecs bypassed UID", checked, err)
	}
}
