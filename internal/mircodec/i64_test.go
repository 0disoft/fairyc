package mircodec_test

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func TestI64CodecPreservesExtremesAndRejectsOldVersion(t *testing.T) {
	for _, value := range []int64{math.MinInt64, math.MaxInt64, 0} {
		program := verifiedProgram(t)
		program.Functions[0].ResultType = hir.TypeI64
		program.Functions[0].Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandI64, I64: value}
		encoded, err := mircodec.Encode(program, artifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := mircodec.Decode(encoded, artifactLimit)
		if err != nil || !reflect.DeepEqual(decoded, program) {
			t.Fatalf("roundtrip %d: %v", value, err)
		}
		old := bytes.Replace(encoded, []byte(mircodec.MIRVersion), []byte("fairyc-mir/0"), 1)
		if _, err := mircodec.Decode(old, artifactLimit); err == nil {
			t.Fatal("accepted old MIR version")
		}
		program.Functions[0].Blocks[0].Terminator.ReturnValue.Type = hir.TypeUSize
		if _, err := mircodec.Encode(program, artifactLimit); err == nil {
			t.Fatal("accepted mismatched signed operand type")
		}
	}
}
