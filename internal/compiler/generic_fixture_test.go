package compiler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

const genericFixtureRoot = "../../spec/fixtures/generic-source"

type genericFixture struct {
	ID               string         `json:"id"`
	Source           string         `json:"source"`
	Outcome          string         `json:"outcome"`
	Value            int32          `json:"value"`
	Specializations  int            `json:"specializations"`
	Code             string         `json:"code"`
	Phase            semantic.Phase `json:"phase"`
	PrimaryText      string         `json:"primary_text"`
	ExpectedTypeText string         `json:"expected_type_text"`
}

func genericFixtures(t *testing.T) []genericFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(genericFixtureRoot, "manifest-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema    string           `json:"schema"`
		Admission string           `json:"admission"`
		Cases     []genericFixture `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing manifest data")
	}
	if manifest.Schema != "fairyc.generic-source-fixtures/v0" || manifest.Admission != "compiler-private-local-scalars" || len(manifest.Cases) == 0 {
		t.Fatal("invalid generic fixture manifest")
	}
	seen := map[string]bool{}
	for _, fixture := range manifest.Cases {
		if fixture.ID == "" || seen[fixture.ID] || !filepath.IsLocal(fixture.Source) || filepath.Base(fixture.Source) != "main.fy" {
			t.Fatal("invalid generic fixture identity/path")
		}
		seen[fixture.ID] = true
		switch fixture.Outcome {
		case "return":
			if fixture.Specializations <= 0 || fixture.Code != "" {
				t.Fatal("invalid return expectation")
			}
		case "planner":
			if fixture.Code == "" || fixture.Phase != "" {
				t.Fatal("missing planner rejection category")
			}
		case "semantic":
			if fixture.Code == "" || fixture.Phase == "" || fixture.PrimaryText == "" || fixture.ExpectedTypeText == "" {
				t.Fatal("missing structured semantic expectation")
			}
		default:
			t.Fatalf("unknown outcome %q", fixture.Outcome)
		}
	}
	return manifest.Cases
}

func genericFixtureSource(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(genericFixtureRoot, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func genericProgram(t *testing.T) string {
	t.Helper()
	return genericFixtureSource(t, "pass/transitive/main.fy")
}

func TestVersionedGenericSourceFixtures(t *testing.T) {
	for _, fixture := range genericFixtures(t) {
		t.Run(fixture.ID, func(t *testing.T) {
			text := genericFixtureSource(t, fixture.Source)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			result, plan, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			switch fixture.Outcome {
			case "return":
				if err != nil || !result.Verified || len(plan.Specializations) != fixture.Specializations {
					t.Fatalf("compile: %v, plan=%+v semantic=%+v", err, plan, result.Semantic)
				}
				outcome, err := interpreter.Execute(result.MIR, "main::main")
				if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != fixture.Value {
					t.Fatalf("execute: %+v %v", outcome, err)
				}
			case "planner":
				if result.Verified || string(genericplan.ErrorCode(err)) != fixture.Code {
					t.Fatalf("expected planner %s, got %v", fixture.Code, err)
				}
			case "semantic":
				if err != nil || result.Verified || result.Semantic.FailedPhase != fixture.Phase || len(result.Semantic.Unsupported) != 0 || len(result.Semantic.Diagnostics) != 1 {
					t.Fatalf("expected semantic failure: %v %+v", err, result.Semantic)
				}
				d := result.Semantic.Diagnostics[0]
				if d.Code != fixture.Code || fixtureSpanText(t, text, d.Primary) != fixture.PrimaryText || len(d.Related) != 1 || d.Related[0].Role != "expected-type" || fixtureSpanText(t, text, d.Related[0].Span) != fixture.ExpectedTypeText {
					t.Fatalf("diagnostic origin changed: %+v", d)
				}
			}
			// The fixture declares a private path, not silent stable-language promotion.
			ordinary, err := compiler.Check([]semantic.File{file})
			if err == nil && ordinary.Verified {
				t.Fatal("ordinary compiler admitted private generic fixture")
			}
		})
	}
}

func fixtureSpanText(t *testing.T, text string, span hir.Span) string {
	t.Helper()
	if span.Source != "main.fy" || span.StartByte < 0 || span.EndByte < span.StartByte || span.EndByte > len(text) {
		t.Fatalf("invalid diagnostic span: %+v", span)
	}
	return text[span.StartByte:span.EndByte]
}
