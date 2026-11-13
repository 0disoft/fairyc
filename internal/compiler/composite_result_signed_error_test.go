package compiler_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCompositeResultSignedErrorAdmission(t *testing.T) {
	for _, scalar := range compositeResultScalars {
		t.Run(scalar, func(t *testing.T) {
			text := fmt.Sprintf(`module main;
fn Result<T,E> relay<T,E>(Result<T,E> value){let Result<T,E> copied=value;discard value;return copied;}
fn Result<T,i32> propagate<T>(Result<T,i32> value){let T actual=relay<T,i32>(value)?;return .ok(actual);}
fn i32 main(){let Result<%[1]s,i32> original=.err(-77);let Result<%[1]s,i32> copied=original;discard original;let Result<%[1]s,i32> result=propagate<%[1]s>(copied);return switch(result){case .ok(_):{0}case .err(code):{code}};}`, scalar)
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
			before, _ := json.Marshal(file.Module)
			result, err := compiler.CheckExplicitFiles([]semantic.File{file})
			if err != nil || !result.Verified {
				t.Fatalf("signed error %s: %v %+v", scalar, err, result.Semantic)
			}
			after, _ := json.Marshal(file.Module)
			if string(before) != string(after) {
				t.Fatal("signed error admission changed authored AST")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(result.MIR, "main::main", profile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCompositeResultSignedErrorNativePayloads(t *testing.T) {
	text, observer := compositeResultNativeProgram()
	text = strings.NewReplacer(",u32>", ",i32>", "let u32 error", "let i32 error", "77", "-77").Replace(text)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("signed error native fixture: %v %+v", err, result.Semantic)
	}
	t.Run("source", func(t *testing.T) { runExpandedLanguageNative(t, result, "main::main", observer) })
	encoded, err := mircodec.Encode(result.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(encoded, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("restored", func(t *testing.T) {
		runExpandedLanguageNative(t, compiler.Result{MIR: restored}, "main::main", observer)
	})
}
