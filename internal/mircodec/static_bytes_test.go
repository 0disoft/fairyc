package mircodec_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func TestMIRArtifactPreservesAllLiteralBytes(t *testing.T) {
	var literal strings.Builder
	values := make([]string, 256)
	for i := range values {
		fmt.Fprintf(&literal, "\\x%02x", i)
		values[i] = fmt.Sprint(i)
	}
	text := fmt.Sprintf(`module main;
fn i32 main() {
    let ref u8[] view = b"%s";
    let u8[256] expected = [%s];
    assert(view.len == 256);
    var usize i = 0;
    while (i < view.len) { assert(view[i] == expected[i]); i += 1; }
    return 0;
}`, literal.String(), strings.Join(values, ", "))
	program := verifiedSourceProgram(t, text)
	for _, codec := range []struct {
		name   string
		encode func(mir.Program, uint64) ([]byte, error)
		decode func([]byte, uint64) (mir.Program, error)
	}{
		{"ordinary", mircodec.Encode, mircodec.Decode},
		{"storage-transfer", mircodec.EncodeStorageTransfers, mircodec.DecodeStorageTransfers},
	} {
		t.Run(codec.name, func(t *testing.T) {
			encoded, err := codec.encode(program, artifactLimit)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := codec.decode(encoded, artifactLimit)
			if err != nil || !reflect.DeepEqual(decoded, program) {
				t.Fatalf("byte-preserving round trip: %v", err)
			}
			outcome, err := interpreter.Execute(decoded, decoded.Functions[0].Name)
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
				t.Fatalf("literal execution: %v %+v", err, outcome)
			}
			old := bytes.Replace(encoded, []byte(mircodec.MIRVersion), []byte("fairyc-mir/1"), 1)
			if _, err := codec.decode(old, artifactLimit); err == nil {
				t.Fatal("accepted previous lossy MIR version")
			}
		})
	}
}
