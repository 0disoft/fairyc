package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

type candidateClosureCCase struct {
	name, raw string
	exit      int
}

func candidateClosureCCases() []candidateClosureCCase {
	return []candidateClosureCCase{
		{"scalar-unchanged", `module main;fn i32 main(){return 42;}`, 42},
		{"dead-typed-literal-after-return", `module main;fn i32 main(){return 42;let fn i32(i32) unused=fn [] i32(i32 x){let z=1/0;return x;};}`, 42},
		{"signature-only-no-literal", `module main;fn fn i32(i32) identity(fn i32(i32) action){return action;}fn i32 main(){return 42;}`, 42},
		{"ordinary-generic-beside-closure", `module main;fn T identity<T>(T x){return x;}
fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}
fn i32 main(){let action=make(40);return identity<i32>(action(2));}`, 42},
		{"returned-four-captures-struct-snapshot", `module main;
struct Holder{fn i32(i32) action;}
fn fn i32(i32) make(i32 seed){var i32 a=seed;let i32 b=3;let i32 c=2;let i32 d=4;
let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){let reduced=a-b*x;return reduced+c-d;};a=999;return action;}
fn Holder boxed(i32 seed){let action=make(seed);return Holder{.action=action};}
fn fn i32(i32) keep(fn i32(i32) action){let copied=action;return copied;}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn i32 main(){let original=make(50);let copy=keep(original);let holder=boxed(50);let stored=holder;let extracted=stored.action;
let separate=make(60);assert(apply(copy,2)==42);assert(apply(separate,2)==52);assert(apply(extracted,2)==42);assert(apply(copy,3)==39);return apply(copy,2);}`, 42},
		{"different-targets-one-parameter", `module main;
fn fn i32(i32) first(){let i32 base=40;let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}
fn fn i32(i32) second(){let fn i32(i32) action=fn [] i32(i32 x){return x*2;};return action;}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn i32 main(){let a=first();let b=second();assert(apply(a,2)==42);assert(apply(b,21)==42);return apply(a,2);}`, 42},
		{"unused-trapping-body-copy", `module main;struct Holder{fn i32(i32) action;}
fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){let unused=1/0;return x;};return action;}
fn i32 main(){let action=make();let copied=action;let holder=Holder{.action=copied};let stored=holder;return 42;}`, 42},
		{"unused-local-traps-on-invocation", `module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let unused=42/x;return x;};discard action(0);return 42;}`, 82},
		{"argument-trap-precedes-body-overflow", `module main;fn i32 main(){let i32 base=2147483647;let fn i32(i32) action=fn [i32 base] i32(i32 x){let unused=base+1;return x;};return action(42/0);}`, 82},
		{"unused-body-overflow", `module main;fn i32 main(){let i32 base=2147483647;let fn i32(i32) action=fn [i32 base] i32(i32 x){let unused=base+1;return x;};return action(42);}`, 81},
		{"runtime-signed-divide-overflow", `module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let unused=-2147483648/x;return x;};return action(-1);}`, 83},
		{"runtime-divide-success", `module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let answer=42/x;return answer;};return action(1);}`, 42},
		{"signed-boundary-success", `module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){let answer=-2147483648/x;return answer;};assert(action(1)==-2147483648);return 42;}`, 42},
		{"argument-once-body-locals-repeat", `module main;fn i32 main(){var i32 base=40;
let fn i32(i32) action=fn [i32 base] i32(i32 x){let sum=base+x;let twice=sum*2;let answer=twice/2;return answer;};base=999;
var i32 count=0;let copied=action;let result=copied({count=count+1;2});assert(count==1);assert(action(3)==43);assert(copied(2)==42);return result;}`, 42},
	}
}

func candidateClosureCRefusals() []string {
	return []string{
		`module main;fn T keep<T>(T action){return action;}fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let copy=keep<fn i32(i32)>(action);return copy(1);}`,
		`module main;struct Holder{fn i32(i32) action;}struct Box{Holder item;}fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let holder=Holder{.action=action};let box=Box{.item=holder};return 0;}`,
		`module main;@repr(C) struct Holder{fn i32(i32) action;}fn i32 main(){return 0;}`,
		`module main;struct Holder{fn i32(i32) action;}unsafe extern C fn void consume(Holder h);fn i32 main(){return 0;}`,
		`module main;struct Holder{fn i32(i32) action;}unsafe extern C fn ptr Holder expose();fn i32 main(){return 0;}`,
		`module main;fn void consume(Option<fn i32(i32)> action){}fn i32 main(){return 0;}`,
		`module main;struct Holder{fn i32(i32) action;}fn void consume(Option<Holder> action){}fn i32 main(){return 0;}`,
		`module main;fn void consume(Result<fn i32(i32),u32> action){}fn i32 main(){return 0;}`,
		`module main;struct Holder{(fn i32(i32))[2] actions;}fn i32 main(){return 0;}`,
		`module main;unsafe extern C fn i32 consume(fn i32(i32) action);fn i32 main(){return 0;}`,
		`module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};let copied=fn [fn i32(i32) action] i32(i32 x){return action(x);};return copied(1);}`,
		`module main;fn i32 main(){var fn i32(i32) action=fn [] i32(i32 x){return x;};return action(1);}`,
		`module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};discard action==action;return 0;}`,
		`module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x;};discard action as usize;return 0;}`,
	}
}

func TestCandidateClosureValueCSource(t *testing.T) {
	for _, name := range []string{"src/c_closure_values.fy", "src/c_emit.fy", "src/c_flow.fy", "src/c_scalar.fy", "src/c_text.fy", "tests/closure_value_c.fy"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		path := name
		if strings.HasPrefix(path, "src/") {
			path = filepath.Base(path)
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, raw), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("authored C owner %s: %v %+v", name, err, parsed.Tree.Diagnostics)
		}
	}
	for _, tc := range candidateClosureCCases() {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(tc.raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("original source %s: %v %+v", tc.name, err, parsed.Tree.Diagnostics)
		}
	}
}

func candidateClosureCWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_c.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(fixture), "module tests::closure_value_c;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test ", "")
	main += "\n@effects(console) fn i32 main(){\n"
	cases := candidateClosureCCases()
	for _, tc := range cases {
		main += "member(b" + strconv.Quote(tc.raw) + ",b" + strconv.Quote(tc.name) + ");\n"
	}
	main += "closure_c_refuses_forged_mir_before_output();\n"
	for _, raw := range candidateClosureCRefusals() {
		main += "reject_source(b" + strconv.Quote(raw) + ");\n"
	}
	main += "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "c_emit", "c_closure_values", "mir_lower", "hir_lower", "name_resolution", "type_check", "syntax", "scanner", "hir_shapes")
	return root
}

func TestCandidateClosureValueCCheck(t *testing.T) {
	root := candidateClosureCWorkspace(t)
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("bounded C producer Go source check: %d %s %s", code, out, diagnostic)
	} else {
		t.Log(strings.TrimSpace(out))
	}
}

// One bounded Fairy producer per profile executes the original source pipeline
// for every case and all forged-MIR refusals. Go only builds the producer image
// and observes the emitted C; it never supplies pre-lowered source targets.
func TestCandidateClosureValueC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 copied closure C execution; Linux remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	root := candidateClosureCWorkspace(t)
	cases := candidateClosureCCases()
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	// This observer exposes the emitted candidate trap category without changing
	// checks, source semantics, native warning flags, or compilation timeouts.
	const observer = "#include <stdint.h>\n#include <stdlib.h>\n_Noreturn void fy_runtime_trap(uint32_t category){exit(80+(int)category);}\n"
	var first [][]byte
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("bounded %s Fairy producer: %d parts", producer.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "closure-c-producer-"+producer.Name+".exe")
		if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		command := exec.CommandContext(ctx, path)
		var diagnostics bytes.Buffer
		command.Stderr = &diagnostics
		out, err := command.Output()
		cancel()
		if err != nil || diagnostics.Len() != 0 {
			t.Fatalf("Fairy C producer %s: %v stderr=%s", producer.Name, err, &diagnostics)
		}
		reader := bufio.NewReader(bytes.NewReader(out))
		for i, tc := range cases {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			n, err := strconv.Atoi(strings.TrimSuffix(line, "\n"))
			if err != nil || n < 1 || n > 2097152 {
				t.Fatalf("invalid C member size %q: %v", line, err)
			}
			unit := make([]byte, n)
			if _, err = io.ReadFull(reader, unit); err != nil {
				t.Fatal(err)
			}
			if producer.Name == cdriver.LowProfile.Name {
				first = append(first, unit)
			} else if !bytes.Equal(unit, first[i]) {
				t.Fatalf("C bytes differ between producer profiles: %s", tc.name)
			}
			if i == 0 && bytes.Contains(unit, []byte("fy_closure_")) {
				t.Fatal("no-closure emission acquired closure declarations")
			}
			for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, consumer, string(unit), observer)
				if err != nil {
					t.Fatalf("C consumer %s/%s/%s: %v", producer.Name, consumer.Name, tc.name, err)
				}
				imagePath := filepath.Join(root, fmt.Sprintf("closure-c-%s-%s-%d.exe", producer.Name, consumer.Name, i))
				if err = os.WriteFile(imagePath, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				code, output := candidateAtlasProcess(t, imagePath, nil)
				if code != tc.exit || output != "" {
					t.Fatalf("C consumer %s/%s/%s: exit=%d want=%d stdout=%q", producer.Name, consumer.Name, tc.name, code, tc.exit, output)
				}
				t.Logf("%s/%s %s: exit=%d empty streams source-sha256=%x", producer.Name, consumer.Name, tc.name, code, sha256.Sum256(unit))
			}
		}
		if _, err := reader.ReadByte(); err != io.EOF {
			t.Fatalf("unexpected trailing producer bytes: %v", err)
		}
		t.Logf("%s producer: 28 forged MIR cases + %d source refusals", producer.Name, len(candidateClosureCRefusals()))
	}
}
