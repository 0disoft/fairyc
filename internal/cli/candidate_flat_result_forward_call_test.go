package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

const flatResultForwardMinimal = `module root;
fn Result<T,u32> tail<T>(Result<T,u32> value){return value;}
fn Result<T,u32> relay<T>(Result<T,u32> value){
let Result<T,u32> copied=value;discard value;
var Result<T,u32> replaced=.err(9);discard replaced;
replaced=copied;discard copied;return tail<T>(replaced);
}
fn i32 main(){let Result<i32,u32> value=relay<i32>(.ok(42));return switch(value){case .ok(x):{x}case .err(_):{0}};}`

func TestCandidateFlatResultForwardCall(t *testing.T) {
	candidateResultNativeGate(t)
	if checked := candidateResultGoCheck(t, flatResultForwardMinimal); !checked.Verified {
		t.Fatalf("independent original-source Go admission: %+v", checked.Semantic)
	}
	root, err := os.MkdirTemp(filepath.Join("..", "..", ".cache"), "flat-result-forward-")
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained forward-call evidence=%s source-sha256=%x", root, sha256.Sum256([]byte(flatResultForwardMinimal)))
	copyLexerExample(t, root)
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "composite_result_forward_call.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/composite_result_forward_call.fy", fixture), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("parse authored fixture: %v %+v", err, parsed.Tree.Diagnostics)
	}
	var imports []string
	for _, imported := range parsed.AST.Imports {
		imports = append(imports, imported.Path)
	}
	program := strings.Replace(string(fixture), "module tests::composite_result_forward_call;", "module main;", 1)
	program += "\n@effects(console) fn i32 main(){return stdin_forward();}\n"

	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, imports...)
	inputs := map[string]string{}
	files, err := filepath.Glob(filepath.Join(root, "src", "*.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inputs[filepath.Base(path)] = fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	encoded, err := json.MarshalIndent(inputs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source-inputs.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("current authored closure files=%d inventory-sha256=%x fixture-sha256=%x", len(inputs), sha256.Sum256(encoded), sha256.Sum256(fixture))
	build, support := candidateParityBuildParts(t, root)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(producer.Name, func(t *testing.T) {
			digest := sha256.New()
			for _, part := range append([]string{producer.Name, build.Header, support}, build.Sources...) {
				digest.Write([]byte(part))
				digest.Write([]byte{0})
			}
			t.Logf("source pipeline producer=%s C-parts=%d generated-C-digest=%x", producer.Name, len(build.Sources), digest.Sum(nil))
			artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "producer-"+producer.Name+".exe")
			if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for _, vector := range []struct{ name, raw string }{
				{"minimal", flatResultForwardMinimal},
				{"full-baseline", "module root;" + strings.TrimPrefix(compositeResultCLIProvider, "module answer;") + strings.ReplaceAll(strings.TrimPrefix(compositeResultCLIMain(), "module main;import answer as a;"), "a::", "")},
				{"full-helper", "module root;" + strings.TrimPrefix(compositeResultCLIEdits(true)[0].provider, "module answer;") + strings.ReplaceAll(strings.TrimPrefix(compositeResultCLIMain(), "module main;import answer as a;"), "a::", "")},
			} {
				t.Run(vector.name, func(t *testing.T) {
					checked := candidateResultGoCheck(t, vector.raw)
					if !checked.Verified {
						t.Fatalf("Go vector %s: %+v", vector.name, checked.Semantic)
					}
					t.Logf("vector=%s source-sha256=%x Go-MIR-functions=%d", vector.name, sha256.Sum256([]byte(vector.raw)), len(checked.MIR.Functions))
					if err := os.WriteFile(filepath.Join(root, vector.name+".fy"), []byte(vector.raw), 0600); err != nil {
						t.Fatal(err)
					}
					ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
					command := exec.CommandContext(ctx, path)
					command.Stdin = strings.NewReader(vector.raw)
					errout := new(strings.Builder)
					command.Stderr = errout
					out, err := command.Output()
					cancel()
					if command.Process == nil {
						t.Fatalf("start source producer: %v", err)
					}
					t.Logf("producer=%s pid=%d image-sha256=%x stderr=%q", producer.Name, command.Process.Pid, sha256.Sum256(artifact.Bytes), errout.String())
					// Preserve the complete helper input as a capacity-negative control.
					// Its 68 generic instances exceed the documented 64-per-unit cap.
					// Minimal forwarding and the unchanged 60-instance baseline remain
					// independent positive checks, including generated C execution.
					if vector.name == "full-helper" {
						exit, ok := err.(*exec.ExitError)
						if !ok || exit.ExitCode() != 4 || len(out) != 0 || errout.String() != "phase=3 stage=3 code=4503 start=638\n" {
							t.Fatalf("exact capacity refusal: error=%v stdout=%q stderr=%q", err, out, errout)
						}
						t.Log("capacity-negative PASS: exact phase3/stage3/code4503; no generated packet, supported per-unit limit unchanged")
						return
					}
					if err != nil || errout.Len() != 0 {
						t.Fatalf("uncached Fairy source pipeline: %v stderr=%s", err, errout)
					}
					packet, err := candidatebuild.Decode(out)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(root, "packet-"+producer.Name+"-"+vector.name+".bin"), out, 0600); err != nil {
						t.Fatal(err)
					}
					for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
						image, err := cdriver.BuildWindowsMSVCTargetParts(tc, consumer, packet.Header, packet.Literals, packet.Units, rt)
						if err != nil {
							t.Fatal(err)
						}
						result, err := cdriver.ExecuteWindowsMSVCABIProbe(image)
						if err != nil || result.ExitCode != 42 || result.Stdout != "" || result.Stderr != "" {
							t.Fatalf("generated C %s/%s: %+v %v", producer.Name, consumer.Name, result, err)
						}
						t.Logf("generated C %s/%s exit42 empty streams image-sha256=%x", producer.Name, consumer.Name, sha256.Sum256(image.Bytes))
					}
				})
			}
		})
	}
	for name, before := range inputs {
		if name == "main.fy" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != before {
			t.Fatalf("authored compiler input changed during frozen replay: %s %v", name, err)
		}
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("all current-source compiler inputs unchanged; Go executable=%s", identity.String())
}
