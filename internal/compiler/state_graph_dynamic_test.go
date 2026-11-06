package compiler_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

func dynamicStateGraphSource(codec, body string) string {
	return fmt.Sprintf(`module transaction; @contract_uid("test.dynamic-source")
state {cell %s value;cell %s target;map %s %s values;}
@effects(storage_read,storage_write) fn void transaction() {%s}`, codec, codec, codec, codec, body)
}

func TestExperimentalStateGraphSourceDynamicCodecs(t *testing.T) {
	data, err := os.ReadFile("../../examples/contract_state_graph/dynamic/transaction.fy")
	if err != nil {
		t.Fatal(err)
	}
	compiled := compileStateGraph(t, string(data))
	p := compiled.StateTransactions[0].Program
	store, err := contractstatemir.NewMemoryStore(p.Layout)
	if err != nil {
		t.Fatal(err)
	}
	out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
	if err != nil || out.Exit != contractstatemir.ExitCommit || len(out.Reads) != 7 || out.Cost != 25 {
		t.Fatalf("dynamic source %+v %v", out, err)
	}
	if !bytes.Equal(out.Reads[5].Value.Encoded, []byte{'A', 'B', 26}) || string(out.Reads[6].Value.Encoded) != "나" {
		t.Fatal("nested/UTF-8 copies lost bytes", out.Reads)
	}
	for _, trial := range []struct {
		codec, literal string
		want           []byte
	}{
		{"bytes", `b""`, []byte{}}, {"bytes", `b"\x00\xff\r\n\x1a"`, []byte{0, 255, 13, 10, 26}},
		{"text", `""`, []byte{}}, {"text", `"가\0나"`, []byte("가\x00나")},
	} {
		t.Run(trial.codec+"/"+trial.literal, func(t *testing.T) {
			body := fmt.Sprintf(`storage::read("value"); storage::write("value",%s);let %s value=storage::read("value"); storage::write("values",value,value);storage::read("values",value);storage::commit();`, trial.literal, trial.codec)
			compiled := compileStateGraph(t, dynamicStateGraphSource(trial.codec, body))
			p := compiled.StateTransactions[0].Program
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if err != nil || len(out.Reads) != 3 || out.Reads[0].Found || len(out.Reads[0].Value.Encoded) != 0 || !out.Reads[1].Found || !out.Reads[2].Found || !bytes.Equal(out.Reads[1].Value.Encoded, trial.want) || !bytes.Equal(out.Reads[2].Value.Encoded, trial.want) {
				t.Fatalf("empty/presence/source bytes %+v %v", out, err)
			}
		})
	}
}

func TestExperimentalStateGraphSourceDynamicRefusals(t *testing.T) {
	for _, trial := range []struct{ codec, body string }{
		{"bytes", `storage::write("value","text");storage::commit();`},
		{"text", `storage::write("value",b"bytes");storage::commit();`},
		{"bytes", `storage::write("value",1);storage::commit();`},
		{"text", `storage::write("value",true);storage::commit();`},
		{"bytes", `let text value=storage::read("value");storage::commit();`},
		{"text", `let bytes value=storage::read("value");storage::commit();`},
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", dynamicStateGraphSource(trial.codec, trial.body))
		compiled, err := compiler.CheckExperimentalStateGraph(file)
		if compiled.Verified || len(compiled.MIR.Functions)+len(compiled.StateTransactions) != 0 {
			t.Fatal("invalid dynamic operand published", err)
		}
	}
	// The parser normally rejects ill-formed UTF-8 strings too. The explicit
	// source adapter independently validates cooked bytes at its own boundary.
	file := parseCompilerFile(t, hir.DefaultPackage, "transaction.fy", dynamicStateGraphSource("text", `storage::write("value","a");storage::commit();`))
	file.Module.Functions[0].Body.Statements[0].Expression.Children[2].Cooked = "ff"
	compiled, err := compiler.CheckExperimentalStateGraph(file)
	if compiled.Verified || len(compiled.StateTransactions)+len(compiled.MIR.Functions) != 0 || err == nil {
		t.Fatal("noncanonical cooked text published", err)
	}
}

func TestExperimentalStateGraphSourceTextViewCanonicality(t *testing.T) {
	for _, trial := range []struct {
		name           string
		offset, length int
		valid          bool
		want           string
	}{
		{"scalar", 3, 3, true, "나"}, {"whole", 0, 9, true, "가나다"}, {"split-scalar", 1, 1, false, ""}, {"trailing-fragment", 3, 2, false, ""},
	} {
		t.Run(trial.name, func(t *testing.T) {
			body := fmt.Sprintf(`storage::write("value","가나다");let b=storage::borrow("value");let v=storage::view(b,%d,%d);let text copied=storage::copy(v);storage::write("target",copied);storage::read("target");storage::commit();`, trial.offset, trial.length)
			compiled := compileStateGraph(t, dynamicStateGraphSource("text", body))
			p := compiled.StateTransactions[0].Program
			store, err := contractstatemir.NewMemoryStore(p.Layout)
			if err != nil {
				t.Fatal(err)
			}
			out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
			if (err == nil) != trial.valid {
				t.Fatal("UTF-8 view result", err)
			}
			if trial.valid && string(out.Reads[1].Value.Encoded) != trial.want {
				t.Fatal("UTF-8 view bytes", out.Reads)
			}
			if !trial.valid && len(store.Dump()) != 0 {
				t.Fatal("noncanonical UTF-8 published")
			}
		})
	}
	// Text addressing preserves exact UTF-8 bytes, without Unicode normalization.
	body := `storage::write("values","가","one");storage::write("values","가","two");storage::read("values","가");storage::read("values","가");storage::commit();`
	compiled := compileStateGraph(t, dynamicStateGraphSource("text", body))
	p := compiled.StateTransactions[0].Program
	store, err := contractstatemir.NewMemoryStore(p.Layout)
	if err != nil {
		t.Fatal(err)
	}
	out, err := contractstatemir.Execute(store, p, contractstate.Limits{MaxWrites: 10, MaxWriteBytes: 4096, MaxValueBytes: 64, MaxMapKeyBytes: 64})
	if err != nil || len(store.Dump()) != 2 || string(out.Reads[0].Value.Encoded) != "one" || string(out.Reads[1].Value.Encoded) != "two" {
		t.Fatal("distinct text keys collapsed", out, err)
	}
	if strings.EqualFold("가", "가") {
		t.Fatal("test keys unexpectedly equal")
	}
}
