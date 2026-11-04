package cbackend

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func fanoutDropProgram(t *testing.T, depth int) mir.Program {
	t.Helper()
	var text strings.Builder
	text.WriteString("module main; fn void audit() {} resource Node0 { i32 value; cleanup { audit(); } } fn Node0 make0() { return Node0 { .value = 1 }; }\n")
	for index := 1; index <= depth; index++ {
		fmt.Fprintf(&text, "resource Node%d { Node%d first; Node%d last; cleanup { audit(); } }\n", index, index-1, index-1)
		fmt.Fprintf(&text, "fn Node%d make%d() { return Node%d { .first = make%d(), .last = make%d() }; }\n", index, index, index, index-1, index-1)
	}
	fmt.Fprintf(&text, "fn i32 main() { let Node%d tree = make%d(); return 42; }\n", depth, depth)
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("syntax: %v, %+v", err, parsed.Tree.Diagnostics)
	}
	analyzed := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if len(analyzed.Diagnostics) != 0 || len(analyzed.Unsupported) != 0 {
		t.Fatalf("semantic: %+v, %+v", analyzed.Diagnostics, analyzed.Unsupported)
	}
	program, err := mir.Lower(analyzed.HIR)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestNestedDropFanoutEmitsLinearHelpers(t *testing.T) {
	const depth = 20 // A million leaf paths must not become a million C fragments.
	program := fanoutDropProgram(t, depth)
	for _, profile := range []Profile{Stage0Profile(), Stage0WindowsMSVCProfile(), Stage0DropTraceProfile(), Stage0WindowsMSVCDropTraceProfile()} {
		t.Run(fmt.Sprintf("%s/%s", profile.Target, profile.Instrumentation), func(t *testing.T) {
			unit, err := Emit(program, "main::main", profile)
			if err != nil {
				t.Fatal(err)
			}
			if count := strings.Count(unit.Source, "*fy_value) {"); count != depth {
				t.Fatalf("helper definitions=%d, want %d", count, depth)
			}
			if len(unit.Source) > 150_000 {
				t.Fatalf("expanded nested cleanup output: %d bytes", len(unit.Source))
			}
			t.Logf("depth=%d leaf-paths=%d helpers=%d C-bytes=%d", depth, 1<<depth, depth, len(unit.Source))
			if profile.Instrumentation != "" && len(unit.DropSites) != depth+1 {
				t.Fatalf("trace sites expanded field paths: %d", len(unit.DropSites))
			}
			parts, err := EmitBuildContext(context.Background(), program, "main::main", profile, false, 1024)
			if err != nil {
				t.Fatal(err)
			}
			if parts.Header == "" || strings.Count(strings.Join(parts.Sources, "\n"), "*fy_value) {") != depth {
				t.Fatal("partitioning duplicated or omitted cleanup definitions")
			}
			again, err := Emit(program, "main::main", profile)
			if err != nil || again.Source != unit.Source {
				t.Fatal("nondeterministic cleanup helpers", err)
			}
		})
	}
}
