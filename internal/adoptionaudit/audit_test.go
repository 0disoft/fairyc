package adoptionaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditRetainedEvidence(t *testing.T) {
	root := t.TempDir()
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	binary := []byte("fixture image, never executed")
	compiler := filepath.Join(root, "compiler")
	write(compiler, binary)
	harness := filepath.Join(root, "harness-executed.ps1")
	write(harness, []byte("retained harness"))
	actual := json.RawMessage(`{"exitCode":0,"stdout":"한e\u0301","stderr":""}`)
	r := receipt{Schema: "fairyc.ledger-compare-adoption/v1", Passed: 68, CasesPerImage: 17, CompilerSHA256: digest(binary), HarnessSHA256: digest([]byte("retained harness"))}
	var raw []byte
	for _, variant := range []string{"legacy", "sdk"} {
		dir := filepath.Join(root, variant)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, profile := range []string{"development", "release"} {
			exe := filepath.Join(dir, profile)
			write(exe, binary)
			r.Images = append(r.Images, image{variant, profile, exe, digest(binary)})
			for i := range 17 {
				r.Cases = append(r.Cases, entry{Variant: variant, Profile: profile, Name: fmt.Sprint(i), Actual: actual, Expected: actual})
				rec := record{Kind: "program", Executable: exe, Cwd: dir, Argv: []string{"fixture"}, Actual: actual}
				data, _ := json.Marshal(rec)
				raw = append(raw, append(data, '\n')...)
			}
		}
	}
	path := filepath.Join(root, "receipt.json")
	rawPath := filepath.Join(root, "raw-executions.jsonl")
	write(rawPath, raw)
	save := func() { data, _ := json.Marshal(r); write(path, data) }
	save()
	o := Options{Kind: "ledger", Receipt: path, Compiler: compiler}
	result, err := Audit(o)
	if err != nil || result.Cases != 68 || result.NativeExecutions != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	write(rawPath, bytes.Replace(raw, []byte(`"argv":["fixture"]`), []byte(`"argv":[null]`), 1))
	if _, err := Audit(o); err == nil {
		t.Fatal("accepted null argv")
	}
	write(rawPath, raw)
	r.Cases[0].Expected = json.RawMessage(`{"exitCode":0,"stdout":"한é","stderr":""}`)
	save()
	if _, err := Audit(o); err == nil {
		t.Fatal("accepted normalized alias")
	}
	r.Cases[0].Expected = actual
	save()
	write(rawPath, append(raw, raw...))
	if _, err := Audit(o); err == nil {
		t.Fatal("accepted duplicate raw inventory")
	}
	write(rawPath, raw)
	write(r.Images[0].Executable, []byte("changed"))
	if _, err := Audit(o); err == nil {
		t.Fatal("accepted changed image")
	}
}
