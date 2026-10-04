package compiler_test

import (
	"bytes"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"testing"
)

func TestArrayForGenericExpansion(t *testing.T) {
	const source = `module main;fn T identity<T>(T value){return value;}
 fn i32 main(){let i64[2] values=[7,4000000000];var i64 total=0;for(let i64 item in values){total+=identity<i64>(item);}assert(total==4000000007);return 0;}`
	encode := func() []byte {
		a, err := analyzeStorageBorrow(t, source)
		if err != nil {
			t.Fatal(err)
		}
		r, err := compiler.LowerAnalyzedStorageTransfers(a.Analysis)
		if err != nil || !r.Verified {
			t.Fatalf("expanded array loop: %v %+v", err, r.Semantic)
		}
		data, err := mircodec.EncodeStorageTransfers(r.MIR, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	data := encode()
	if !bytes.Equal(data, encode()) {
		t.Fatal("expanded array owner identity changed MIR")
	}
	program, err := mircodec.DecodeStorageTransfers(data, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(program, hir.QualifiedModulePath(1, "main")+"::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("expanded array execution: %+v %v", outcome, err)
	}
}
