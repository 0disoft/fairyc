package mircodec_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const artifactLimit = 1 << 20

func TestVerifiedMIRArtifactRoundTripIsDeterministic(t *testing.T) {
	t.Parallel()
	program := verifiedProgram(t)
	first, err := mircodec.Encode(program, artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	second, err := mircodec.Encode(program, artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same verified MIR program produced different artifact bytes")
	}
	wantPrefix := `{"schema":"fairyc.mir-artifact/v0","mirVersion":"fairyc-mir/0","program":`
	if !bytes.HasPrefix(first, []byte(wantPrefix)) {
		t.Fatalf("artifact prefix = %q", first[:min(len(first), len(wantPrefix))])
	}
	decoded, err := mircodec.Decode(first, artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, program) {
		t.Fatalf("decoded MIR differs from source\n got: %+v\nwant: %+v", decoded, program)
	}
}

func TestMIRArtifactRejectsInvalidLimitsAndBytes(t *testing.T) {
	t.Parallel()
	program := verifiedProgram(t)
	encoded, err := mircodec.Encode(program, artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.Encode(program, 0); err == nil {
		t.Fatal("zero encode limit accepted")
	}
	if _, err := mircodec.Decode(encoded, 0); err == nil {
		t.Fatal("zero decode limit accepted")
	}
	if _, err := mircodec.Encode(program, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
		t.Fatalf("small encode limit error = %v", err)
	}
	if _, err := mircodec.Decode(encoded, uint64(len(encoded)-1)); !errors.Is(err, mircodec.ErrArtifactTooLarge) {
		t.Fatalf("small decode limit error = %v", err)
	}
	if _, err := mircodec.Decode(nil, artifactLimit); err == nil {
		t.Fatal("empty artifact accepted")
	}
	withWhitespace := append(append([]byte(nil), encoded...), '\n')
	if _, err := mircodec.Decode(withWhitespace, artifactLimit); !errors.Is(err, mircodec.ErrNonCanonical) {
		t.Fatalf("noncanonical whitespace error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := mircodec.Decode(unknown, artifactLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestMIRArtifactRejectsIncompatibleAndUnverifiedPrograms(t *testing.T) {
	t.Parallel()
	program := verifiedProgram(t)
	encoded, err := mircodec.Encode(program, artifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	wrongSchema := bytes.Replace(encoded, []byte(mircodec.Schema), []byte("fairyc.mir-artifact/v9"), 1)
	if _, err := mircodec.Decode(wrongSchema, artifactLimit); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("schema error = %v", err)
	}
	wrongVersion := bytes.Replace(encoded, []byte(mircodec.MIRVersion), []byte("fairyc-mir/9"), 1)
	if _, err := mircodec.Decode(wrongVersion, artifactLimit); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("version error = %v", err)
	}

	program.Functions[0].Entry = 99
	if _, err := mircodec.Encode(program, artifactLimit); err == nil || !strings.Contains(err.Error(), "verify MIR") {
		t.Fatalf("unverified encode error = %v", err)
	}
	corrupt := bytes.Replace(encoded, []byte(`"Entry":0`), []byte(`"Entry":99`), 1)
	if bytes.Equal(corrupt, encoded) {
		t.Fatal("test artifact did not contain the expected entry block")
	}
	if _, err := mircodec.Decode(corrupt, artifactLimit); err == nil || !strings.Contains(err.Error(), "verify decoded MIR") {
		t.Fatalf("unverified decode error = %v", err)
	}
}

func verifiedProgram(t *testing.T) mir.Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(
		source.NewFile("main.fy", []byte("module main;\nfn i32 main() { return 42; }\n")),
		source.DefaultLimits(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("syntax result = %+v", parsed.Tree.Diagnostics)
	}
	result, err := compiler.Check([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}})
	if err != nil || !result.Verified {
		t.Fatalf("compiler.Check() = %+v, %v", result, err)
	}
	return result.MIR
}
