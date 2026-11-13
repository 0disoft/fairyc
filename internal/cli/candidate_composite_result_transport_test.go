package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

var candidateResultScalars = []string{"i8", "i16", "isize", "u16", "u64", "f32", "f64"}

type candidateResultControl struct {
	name, raw, code, token string
	unsupported            int
}

func candidateResultControls() []candidateResultControl {
	var out []candidateResultControl
	for _, scalar := range candidateResultScalars {
		for _, shape := range []string{"Result<T,u32>", "Result<T,i32>", "Result<T,i64>", "Result<u32,T>", "Result<T,T>", "Result<Option<T>,u32>", "Result<Result<T,u32>,u32>", "Option<Result<T,u32>>", "ref Result<T,u32>", "ptr Result<T,u32>", "Result<T,u32>[2]"} {
			concrete := strings.ReplaceAll(shape, "T", scalar)
			raw := fmt.Sprintf("module root; // 한글\r\nfn %[1]s transport<T>(%[1]s value){return value;}fn %[2]s forward(%[2]s value){return transport<%[3]s>(value);}", shape, concrete, scalar)
			unsupported := 2
			if shape == "Result<T,u32>" || shape == "Result<T,i32>" {
				unsupported = 0
			}
			out = append(out, candidateResultControl{name: scalar + "/" + shape, raw: raw, unsupported: unsupported})
		}
		for _, errorType := range []string{"u32", "i32", "i64"} {
			raw := fmt.Sprintf("module root; // 한글\r\nfn Result<T,E> transport<T,E>(Result<T,E> value){return value;}fn Result<%[1]s,%[2]s> forward(Result<%[1]s,%[2]s> value){return transport<%[1]s,%[2]s>(value);}", scalar, errorType)
			unsupported := 2
			if errorType == "u32" || errorType == "i32" {
				unsupported = 0
			}
			out = append(out, candidateResultControl{name: scalar + "/resolved-E-" + errorType, raw: raw, unsupported: unsupported})
		}
		out = append(out,
			candidateResultControl{name: scalar + "/pending-copy", raw: fmt.Sprintf("module root; // 한글\r\nfn void pending(Result<%[1]s,u32> original){let Result<%[1]s,u32> copied=original;discard original;}", scalar), code: "E4101", token: "original"},
			candidateResultControl{name: scalar + "/pending-original", raw: fmt.Sprintf("module root; // 한글\r\nfn void pending(Result<%[1]s,u32> original){let Result<%[1]s,u32> copied=original;discard copied;original;}", scalar), code: "E4101", token: "original"},
			// The u32/i32 error mismatch remains independent of success transport.
			candidateResultControl{name: scalar + "/error-mismatch", raw: fmt.Sprintf("module root; // 한글\r\nfn Result<u32,i32> mismatch(Result<%[1]s,u32> value){let %[1]s actual=value?;discard actual;return .ok(1);}", scalar), code: "E4103", token: "?"},
		)
	}
	return out
}

func candidateResultGoCheck(t *testing.T, raw string) compiler.Result {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("root.fy", []byte(raw)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("Result control parse: %v %+v", err, parsed.Tree)
	}
	before, err := json.Marshal(parsed.AST)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckExplicitFiles([]semantic.File{{Path: "root.fy", Module: parsed.AST}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(parsed.AST)
	if err != nil || string(before) != string(after) {
		t.Fatalf("Result control modified original AST: %v", err)
	}
	return checked
}

func candidateResultControlCheck(t *testing.T, tc candidateResultControl) compiler.Result {
	t.Helper()
	checked := candidateResultGoCheck(t, tc.raw)
	if tc.code != "" {
		if checked.Verified || len(checked.Semantic.Diagnostics) != 1 || checked.Semantic.Diagnostics[0].Code != tc.code || len(checked.Semantic.Unsupported) != 0 {
			t.Fatalf("%s: expected %s; verified=%v diagnostics=%+v unsupported=%+v", tc.name, tc.code, checked.Verified, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		d := checked.Semantic.Diagnostics[0]
		if tc.raw[d.Primary.StartByte:d.Primary.EndByte] != tc.token {
			t.Fatalf("%s wrong source origin: %+v", tc.name, d)
		}
		if tc.code == "E4101" {
			start := strings.Index(tc.raw, "=original") + 1
			if strings.HasSuffix(tc.name, "/pending-original") {
				start = strings.LastIndex(tc.raw, "original;")
			}
			if d.Primary.StartByte != start {
				t.Fatalf("%s wrong evaluation origin: %+v", tc.name, d)
			}
		}
		if tc.code == "E4103" {
			related := map[string]string{}
			for _, row := range d.Related {
				related[row.Role] = tc.raw[row.Span.StartByte:row.Span.EndByte]
			}
			if related["actual-error"] != "u32" || related["expected-error"] != "i32" {
				t.Fatalf("%s error origins: %+v", tc.name, d.Related)
			}
		}
	} else if checked.Verified != (tc.unsupported == 0) || len(checked.Semantic.Unsupported) != tc.unsupported || len(checked.Semantic.Diagnostics) != 0 {
		t.Fatalf("%s: expected %d unsupported signatures, got %+v", tc.name, tc.unsupported, checked.Semantic)
	}
	return checked
}

// No native compiler, image or cache producer is launched by these controls.
func TestCandidateCompositeResultGoControls(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "composite_result_transport.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/composite_result_transport.fy", fixture), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("new Fairy graph fixture syntax: %v diagnostics=%+v", err, parsed.Tree.Diagnostics)
	}
	for _, tc := range candidateResultControls() {
		t.Run(tc.name, func(t *testing.T) { candidateResultControlCheck(t, tc) })
	}
	for _, scalar := range candidateResultScalars {
		raw := fmt.Sprintf("module root;fn T id<T>(T value){return value;}fn void use(Result<%[1]s,u32> value){discard id<Result<%[1]s,u32>>(value);}", scalar)
		if candidateResultGoCheck(t, raw).Verified {
			t.Fatalf("composite generic argument admitted: %s", scalar)
		}
	}
	raw, _ := candidateCompositeResultSource()
	if checked := candidateResultGoCheck(t, raw); !checked.Verified {
		t.Fatalf("exact restored pipeline input Go admission: %+v", checked.Semantic)
	}
	t.Logf("105 source diagnostic controls, 7 composite argument refusals, exact native-pipeline input; source ASTs unchanged")
}

func candidateResultNativeGate(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Result native replay requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 after the Result Go/source readiness gate")
	}
}

func TestCandidateCompositeResultGraph(t *testing.T) {
	candidateResultNativeGate(t)
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "composite_result_transport.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::composite_result_transport;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\nfn i32 main(){result_descriptor_interned_and_written_payloads_are_canonical();signed_result_keeps_error_type_and_rolls_back_new_records();result_capacity_keeps_known_error_and_rolls_back_every_record();result_shallow_boundaries_rollback_materialized_leaves();result_batch_failure_restores_the_successful_prefix();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "generic_scalar_arguments", "generic_transport_scalars", "module_types", "module_catalog", "syntax", "scanner")
	if code, out, diagnostics := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostics != "" {
		t.Fatalf("Result graph cheap source check: %d %q %s", code, out, diagnostics)
	}
	for _, args := range [][]string{{"run"}, {"run", "--release", "--locked"}} {
		if code, out, diagnostics := runForTest(args, root); code != ExitSuccess || out != "" || diagnostics != "" {
			t.Fatalf("Result graph %v: %d %q %s", args, code, out, diagnostics)
		}
	}
}

func TestCandidateCompositeResultSourceDiagnostics(t *testing.T) {
	candidateResultNativeGate(t)
	var wire bytes.Buffer
	wire.WriteString("FYUSIG01")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(len(s)); wire.WriteString(s) }
	controls := candidateResultControls()
	count := 0
	for _, tc := range controls {
		if tc.code == "" {
			count++
		}
	}
	number(count)
	var bodyChecks strings.Builder
	for _, tc := range candidateResultControls() {
		checked := candidateResultControlCheck(t, tc)
		if tc.code != "" {
			d := checked.Semantic.Diagnostics[0]
			fmt.Fprintf(&bodyChecks, "{let p=result_body_problem(b%s);assert(p.code==%s && p.module_index==0 && p.start==%d && p.end==%d);}\n", strconv.Quote(tc.raw), strings.TrimPrefix(tc.code, "E"), d.Primary.StartByte, d.Primary.EndByte)
			continue
		}
		number(1)
		field(tc.name)
		field("root")
		field(tc.raw)
		number(0)
		number(len(checked.Semantic.Unsupported))
		for _, row := range checked.Semantic.Unsupported {
			number(0)
			number(row.Span.StartByte)
			number(row.Span.EndByte)
			field(strings.TrimPrefix(row.Feature, "type "))
		}
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	input := filepath.Join(root, "result-signature-oracle.bin")
	if err := os.WriteFile(input, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_signature_diagnostics.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::generic_signature_diagnostics;", "module main;\nimport result_body_diagnostics as result_body;", 1)
	if strings.Count(program, "assert(cases==24)") != 1 {
		t.Fatal("signature reader case-count anchor changed")
	}
	program = strings.Replace(program, "assert(cases==24)", fmt.Sprintf("assert(cases==%d)", count), 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(filesystem) fn i32 main(){signature_reports_match_go_original_spans_and_features();result_body::verify();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	parserAt := strings.Index(parserProbePrelude, "fn Result")
	support := strings.Replace(parserProbePrelude[:parserAt], "module main;", "module result_body_diagnostics;", 1) + candidateResultDiagnosticPrelude + parserProbePrelude[parserAt:]
	support += "\npub fn void verify(){" + bodyChecks.String() + "}\n"
	writeTestSource(t, root, "src/result_body_diagnostics.fy", support)
	copyCandidateParityModules(t, root, "generic_signature_diagnostics", "project_source_plan", "module_public_type_text", "module_project_compile", "module_signatures", "module_catalog", "syntax", "scanner", "integrity", "file_syntax")
	if code, out, diagnostics := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostics != "" {
		t.Fatalf("Result diagnostics cheap source check: %d %q %s", code, out, diagnostics)
	}
	// The source/origin driver now exceeds the optimized monolithic C command's
	// one-minute bound. Use the established partitions, not a larger product limit.
	build, nativeSupport := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, nativeSupport, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "source-diagnostics-"+profile.Name+".exe")
		if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		out, err := exec.CommandContext(ctx, path, input).CombinedOutput()
		cancel()
		if err != nil || len(out) != 0 {
			t.Fatalf("Result source diagnostics %s: %v %q", profile.Name, err, out)
		}
	}
	t.Logf("%d original signature cases plus 21 body/evaluation cases in both profiles", count)
}

const candidateResultDiagnosticPrelude = `
import module_project_compile as bodycheck;
import module_signatures as signatures;
import std::vec as v;
fn v::Vec<syntax::Parsed> result_sources(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();let p=switch(make(raw)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);return result_sources(raw);}};assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);return sources;}
fn bodycheck::Problem result_body_problem(ref u8[] raw){let sources=result_sources(raw);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return result_body_problem(raw);}};return switch(bodycheck::check_range(ref sources,ref metadata,0,1)){case .ok(_):{assert(false);result_body_problem(raw)}case .err(p):{p}};}
`
