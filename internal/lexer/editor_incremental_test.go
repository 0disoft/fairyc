package lexer

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestEditorLexMatchesBatchAcrossRevisions(t *testing.T) {
	padding := strings.Repeat("alpha 1.25e2 /* nested /* block */ comment */ b\"hi\"\r\n", 100)
	limits := source.DefaultLimits()
	original := "\ufeff" + padding + "before 1. x > > =\r\n/// note\r\n" + padding
	var previous *EditorLexSnapshot
	for i, step := range []struct {
		text string
		both bool
	}{
		{original, false},
		{strings.Replace(original, "1. x > > =", "1.5 >>=\n 0xA", 1), true},
		{strings.Replace(original, "before", "after_replacement", 1), true},
		{strings.Replace(original, "before", "/* unterminated", 1), false},
		{original, false},
		{strings.TrimPrefix(original, "\ufeff"), false},
	} {
		file := source.NewFile("main.fy", []byte(step.text))
		currentBudget := source.NewBudget(limits, len(file.Bytes))
		batchBudget := source.NewBudget(limits, len(file.Bytes))
		result, next := LexEditor(file, currentBudget, previous)
		if want := Lex(file, batchBudget); !reflect.DeepEqual(result, want) || currentBudget.WorkUnits() != batchBudget.WorkUnits() {
			t.Fatalf("revision %d differs from batch: %v/%v", i, result.Diagnostics, want.Diagnostics)
		}
		if step.both && (next.ReusedPrefix() == 0 || next.reusedSuffix == 0) {
			t.Fatalf("revision %d did not reuse both token regions", i)
		}
		previous = next
	}
}

func TestEditorLexVerifiesLookaheadAtCheckpoint(t *testing.T) {
	// The checkpoint follows a token whose classification depends on the
	// second following byte, e.g. integer 1 followed by . and a digit.
	start := strings.Repeat(";", editorCheckpointTokens-1)
	end := strings.Repeat(";tick ", editorCheckpointTokens)
	for _, spelling := range [][2]string{{"1.x", "1.5"}, {">> ", ">>="}, {"b x", "b\"x\""}, {"\r\n ", "\rX "}} {
		old, updated := source.NewFile("main.fy", []byte(start+spelling[0]+end)), source.NewFile("main.fy", []byte(start+spelling[1]+end))
		limits := source.DefaultLimits()
		_, previous := LexEditor(old, source.NewBudget(limits, len(old.Bytes)), nil)
		got, _ := LexEditor(updated, source.NewBudget(limits, len(updated.Bytes)), previous)
		want := Lex(updated, source.NewBudget(limits, len(updated.Bytes)))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("checkpoint affected by %q -> %q", spelling[0], spelling[1])
		}
	}
}

func TestEditorLexMatchesBatchWhenSuffixExceedsBudget(t *testing.T) {
	padding := strings.Repeat("; 1.25e2 tick ", editorCheckpointTokens)
	old := source.NewFile("main.fy", []byte(padding+"1.0"+padding))
	current := source.NewFile("main.fy", []byte(padding+"1e100"+padding))
	limits := source.DefaultLimits()
	budget := source.NewBudget(limits, len(old.Bytes))
	_, previous := LexEditor(old, budget, nil)
	limits.MaxWorkUnits = budget.WorkUnits() + 15
	got, next := LexEditor(current, source.NewBudget(limits, len(current.Bytes)), previous)
	want := Lex(current, source.NewBudget(limits, len(current.Bytes)))
	if next != nil || len(want.Diagnostics) == 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("exhaustion differs: %v/%v", got.Diagnostics, want.Diagnostics)
	}
}
