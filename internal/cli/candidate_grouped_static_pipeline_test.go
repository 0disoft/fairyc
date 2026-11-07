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
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCandidateGroupedStaticOperationModuleScope(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private native module-scope replay requires Windows AMD64; Linux deferred")
	}
	at := strings.Index(candidateGroupedMetadataFixture, "pub fn void joined_group_scope()")
	if at < 0 {
		t.Fatal("missing joined module scope fixture")
	}
	imports := `import static_operation_declarations as operations;
import static_operation_group_identity as group_identity;
import module_ast_join as joined;
import std::vec as v;
fn syntax::Parsed parsed(ref u8[] raw){return switch(make(raw)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
`
	candidateInlineProbe(t, []string{"static_operation_group_identity", "module_ast_join"}, imports+candidateGroupedMetadataFixture[at:], "joined_group_scope();")
}

// Bootstrap the actual authored compiler, then feed it original grouped Fairy
// sources. PreparePackages supplies only expected identities, never expanded ASTs
// or a substitute provider body for the compiler under test.
func TestCandidateGroupedStaticOperationPipeline(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("private compiler-host replay requires Windows AMD64; Linux deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the native compiler-host replay")
	}
	cache, err := filepath.Abs(filepath.Join("..", "..", ".cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(cache, "grouped-static-pipeline-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retained failed frozen source/image: %s", root)
		} else if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	copyLexerExample(t, root)
	oracle := candidateGroupedStaticOracle(t, root)
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	replace := func(raw, old, replacement string) string {
		t.Helper()
		if strings.Count(raw, old) != 1 {
			t.Fatalf("fixture anchor %q is absent or ambiguous", old)
		}
		return strings.Replace(raw, old, replacement, 1)
	}
	fixture := read("static_operation_plan_fixture")
	fixture = replace(fixture, "module tests::static_operation_plan_fixture;", "module static_operation_plan_fixture;\nimport generic_source_function_json as functions;\nimport static_operation_declarations as operations;\nimport static_operation_group_identity as group_identity;\nimport module_ast_join as joined;")
	fixture = replace(fixture, "assert(cases==4)", "assert(cases==7)")
	fixture = replace(fixture, "let input=prepared_input(", "grouped_metadata(ref sources,ref imports,ref digits,wire,ref mut at);let input=prepared_input(")
	fixture += candidateGroupedMetadataFixture
	writeTestSource(t, root, "src/static_operation_plan_fixture.fy", fixture)
	program := replace(read("static_operation_pipeline"), "module tests::static_operation_pipeline;", "module main;\nimport static_operation_plan_fixture as oracle;")
	raw, observer := candidateGroupedStaticSource()
	program += "\n@effects(console,filesystem) fn i32 main(){oracle::verify();oracle::joined_group_scope();emit_operations(b" + strconv.Quote(raw) + ");return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_compiler_core", "module_project_identified_ir", "module_public_type_text", "module_catalog", "module_unit_c_emit", "module_project_complete_c", "module_unit_body_ir_model", "module_unit_cleanup_mir_model", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "static_operation_declarations", "c_text", "syntax", "scanner", "project_source_plan", "project_diagnostic_codes", "generic_source_plan", "generic_source_plan_input", "generic_concrete_types", "generic_package_context", "generic_source_requests", "generic_source_dependencies", "generic_source_closure", "generic_source_dependency_model", "generic_source_dependency_json", "generic_source_catalog", "generic_package_bindings", "module_public_templates", "module_generic_interface", "generic_source_function_json", "static_operation_identity")
	// No source copy or fixture rewrite occurs after this point. Both producer
	// profiles use precisely the same frozen source closure and generated C parts.
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("frozen grouped compiler %s: %d parts", producer.Name, len(build.Sources))
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "grouped-static-"+producer.Name+".exe")
		if err := os.WriteFile(imagePath, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		command := exec.CommandContext(ctx, imagePath, oracle)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err = command.Run()
		cancel()
		if err != nil || stderr.Len() != 0 {
			t.Fatalf("grouped candidate %s: %v stderr=%s", producer.Name, err, &stderr)
		}
		packet, err := candidatebuild.Decode(stdout.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"int8_t", "int16_t", "intptr_t", "uint16_t", "uint64_t", "float", "double"} {
			if !strings.Contains(packet.Header, name+" fy_fn_") {
				t.Fatalf("missing grouped transport signature %s", name)
			}
		}
		for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, consumer, packet.Header, packet.Literals, packet.Units, runtimeSource+"\n"+observer)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "grouped-generated-"+producer.Name+"-"+consumer.Name+".exe")
			if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			output, err := exec.CommandContext(ctx, path).CombinedOutput()
			cancel()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 42 || len(output) != 0 {
				t.Fatalf("grouped generated %s/%s: %v %q", producer.Name, consumer.Name, err, output)
			}
			t.Logf("grouped generated %s/%s: exit 42, empty output", producer.Name, consumer.Name)
		}
	}
}

func candidateGroupedStaticSource() (string, string) {
	// Reuse the existing all-thirteen-scalar observer and provider bodies intact.
	raw, observer := candidateStaticOperationSource()
	raw = strings.Replace(raw, "module root;", "module root;@interface(Transport) pub fn X sibling<X>(X value){}", 1)
	raw = strings.Replace(raw, "@interface fn T transport<T>(T x){}", "@interface(Transport) pub fn T transport<T>(T x){}", 1)
	for _, typ := range []string{"i32", "i64", "u8", "u32", "usize", "bool", "i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		raw += fmt.Sprintf("@implements(sibling,%s) fn %s sibling_%s(%s x){return x;}", typ, typ, typ, typ)
	}
	return raw, observer
}

// The six grouped equal-width variants isolate closure changes from incidental
// spans. A seventh legacy case proves the old ungrouped identity recipe too.
func candidateGroupedStaticOracle(t *testing.T, dir string) string {
	t.Helper()
	root := "module root;import helper as h;fn i32 main(){assert(h::algorithm<i64>(14)==42);return h::algorithm<i32>(21);}"
	provider := "module helper;@interface(Arithmetic) pub fn T transform<T>(T value){}@interface(Arithmetic) pub fn X seed<X>(){}@implements(transform,i64) fn i64 triple(i64 value){return value*3;}@implements(seed,bool) fn bool seed_bool(){return false;}@implements(transform,bool) fn bool logical(bool value){return value;}@implements(seed,i64) fn i64 seed64(){return 41;}@implements(transform,i32) fn i32 twice(i32 value){return value*2;}@implements(seed,i32) fn i32 seed32(){return 42;}pub fn T algorithm<T>(T value){return transform<T>(value);}fn i32 unused(){return 1;}"
	change := func(old, replacement string) string {
		t.Helper()
		if len(old) != len(replacement) || strings.Count(provider, old) != 1 {
			t.Fatalf("invalid equal-width mutation %q", old)
		}
		return strings.Replace(provider, old, replacement, 1)
	}
	cases := []struct{ name, provider string }{
		{"base", provider},
		{"selected provider body", change("value*2", "value*3")},
		{"unused sibling body", change("return 42;", "return 43;")},
		{"unused sibling unused bool body", change("return false;", "return true ;")},
		{"unused sibling declaration", change("X seed<X>", "Y seed<Y>")},
		{"private body", change("return 1;", "return 2;")},
		{"legacy ungrouped", strings.ReplaceAll(provider, "@interface(Arithmetic)", "@interface")},
	}
	var wire bytes.Buffer
	wire.WriteString("FYGPLAN1")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(len(s)); wire.WriteString(s) }
	marshal := func(v any) string {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	number(len(cases))
	var base genericsource.PackageExpansion
	for c, tc := range cases {
		inputs := make([]genericsource.PackageModule, 2)
		for i, raw := range []string{root, tc.provider} {
			parsed, err := frontend.AnalyzeSyntax(source.NewFile([]string{"root.fy", "helper.fy"}[i], []byte(raw)), source.DefaultLimits())
			if err != nil {
				t.Fatalf("parse %s/%d: %v", tc.name, i, err)
			}
			if parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("parse %s/%d: %+v", tc.name, i, parsed.Tree.Diagnostics)
			}
			inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(i+1), 64)), Imports: map[string]genericplan.PackageID{}}
		}
		inputs[0].Imports["helper"] = inputs[1].Owner
		expanded, err := genericsource.PreparePackages(inputs, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("real Go PreparePackages oracle %s: %v", tc.name, err)
		}
		if c == 0 {
			base = expanded
		} else if c < 6 {
			stable := slices.Equal(base.Plan.Roots, expanded.Plan.Roots) && slices.Equal(base.Templates, expanded.Templates)
			if c == 5 {
				if !stable || !slices.EqualFunc(base.Plan.Specializations, expanded.Plan.Specializations, func(a, b genericplan.Specialization) bool { return a.ID == b.ID }) {
					t.Fatal("equal-width private edit changed dependent identities")
				}
			} else if slices.Equal(base.Plan.Roots, expanded.Plan.Roots) || slices.Equal(base.Templates, expanded.Templates) {
				t.Fatalf("equal-width %s did not change roots and public closures", tc.name)
			}
		}
		field(tc.name)
		number(2)
		for i, raw := range []string{root, tc.provider} {
			number(i + 1)
			field(raw)
			if i == 0 {
				number(1)
				field("h")
				number(1)
			} else {
				number(0)
			}
		}
		// Independent raw-AST oracle: no private Go operation map is consulted.
		operations := map[string]ast.Function{}
		providers := map[string][]ast.Function{}
		for _, fn := range inputs[1].Module.Functions {
			for _, attr := range fn.Attributes {
				if attr.Name == "interface" {
					operations[fn.Name] = fn
				}
				if attr.Name == "implements" {
					name := attr.Arguments[0].Raw
					providers[name] = append(providers[name], fn)
				}
			}
		}
		var names []string
		for name := range operations {
			names = append(names, name)
		}
		slices.Sort(names)
		var canonical []ast.Function
		for _, name := range names {
			slices.SortFunc(providers[name], func(a, b ast.Function) int { return strings.Compare(a.ResultType.Path, b.ResultType.Path) })
			canonical = append(canonical, operations[name])
			canonical = append(canonical, providers[name]...)
		}
		definition := genericplan.DigestText("source-static-operation-group-v1", marshal(canonical))
		number(len(names))
		for _, name := range names {
			fn := operations[name]
			memberDefinition := definition
			if c == 6 {
				memberDefinition = genericplan.DigestText("source-static-operation-v1", marshal(append([]ast.Function{fn}, providers[name]...)))
			}
			id := string(inputs[1].Owner) + "::helper::" + name
			field(name)
			field(id + "::operation")
			field(memberDefinition.String())
			base := genericplan.DigestText("source-package-function-v0", marshal(fn)+"\nnull")
			field(genericplan.DigestText("source-static-bound-operation-v1", base.String()+memberDefinition.String()).String())
			field(marshal(fn))
			number(len(providers[name]))
			for _, p := range providers[name] {
				field(p.ResultType.Path)
				field(string(inputs[1].Owner) + "::helper::" + p.Name)
				field(genericplan.DigestText("source-static-provider-v1", marshal(p)).String())
				field(marshal(p))
			}
		}
		number(len(expanded.Plan.Roots))
		for _, id := range expanded.Plan.Roots {
			field(string(id))
		}
		number(len(expanded.Plan.Specializations))
		for _, spec := range expanded.Plan.Specializations {
			field(string(spec.ID))
			field(string(spec.Declaration))
			var args []string
			for _, b := range spec.Bindings {
				args = append(args, string(b.Type))
			}
			field(string(spec.Declaration) + "<" + strings.Join(args, ",") + ">")
			number(len(spec.Bindings))
			for _, b := range spec.Bindings {
				field(string(b.Parameter))
				field(string(b.Type))
			}
			number(len(spec.Dependencies))
			for _, id := range spec.Dependencies {
				field(string(id))
			}
		}
		templates := slices.Clone(expanded.Templates)
		slices.SortFunc(templates, func(a, b genericsource.PublicTemplate) int { return strings.Compare(a.Name, b.Name) })
		number(len(templates))
		for _, v := range templates {
			field(v.Name)
			field(v.Closure.String())
		}
		t.Logf("Go PreparePackages %s: %d roots, %d specializations, %d public templates", tc.name, len(expanded.Plan.Roots), len(expanded.Plan.Specializations), len(templates))
	}
	name := filepath.Join(dir, "grouped-static-oracle.bin")
	if err := os.WriteFile(name, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return name
}

const candidateGroupedMetadataFixture = `
fn void grouped_metadata(ref v::Vec<syntax::Parsed> sources,ref v::Vec<catalog::Binding> imports,ref v::Vec<usize> digits,ref u8[] wire,ref mut usize at){
    let rows=collected(sources,imports,digits,0);let source=v::at<syntax::Parsed>(sources,1);let count=number(wire,at);assert(count==2);var usize i=0;
    while(i<count){let name=field(wire,at);var usize selected=0;var usize j=0;while(j<templates::count(ref rows)){let row=templates::template_at(ref rows,j);if(row.module_index==1 && bytes::equal(templates::text(ref rows,row.name),name)){selected=j+1;}j+=1;}assert(selected!=0);let row=templates::template_at(ref rows,selected-1);
        assert(bytes::equal(templates::text(ref rows,row.capability),field(wire,at)) && bytes::equal(templates::text(ref rows,row.operation_definition),field(wire,at)) && bytes::equal(templates::text(ref rows,row.definition),field(wire,at)));
        var usize work=0;let original=switch(functions::encode(source,row.declaration,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref original),field(wire,at)));
        let providers=number(wire,at);assert(row.provider_count==providers && providers==3);j=0;while(j<providers){let provider=templates::provider_at(ref rows,row.provider_first+j);let scalar=field(wire,at);assert(provider.scalar==operations::scalar(scalar) && bytes::equal(templates::text(ref rows,provider.id),field(wire,at)) && bytes::equal(templates::text(ref rows,provider.definition),field(wire,at)));let declaration=switch(operations::select(source,row.declaration+1,provider.scalar,ref mut work)){case .ok(d):{d}case .err(_):{assert(false);return;}};work=0;let original=switch(functions::encode(source,declaration-1,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref original),field(wire,at)));j+=1;}i+=1;
    }
}
pub fn void joined_group_scope(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module first;@interface(Same) pub fn T z<T>(){}@interface(Same) pub fn U a<U>(){}")).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(b"module second;@interface(Same) pub fn V y<V>(){}@interface(Same) pub fn W b<W>(){}")).is_ok);let p=switch(joined::join(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;assert(operations::validate(ref p,ref mut work)==0);
    // The actual identity module is imported explicitly by this generated fixture.
    work=0;let first=switch(group_identity::definition(ref p,2,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};work=0;let first_other=switch(group_identity::definition(ref p,3,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};work=0;let second=switch(group_identity::definition(ref p,5,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref first),bytes::view(ref first_other)) && !bytes::equal(bytes::view(ref first),bytes::view(ref second)));
    let isolated=parsed(b"module first;@interface(Same) pub fn T z<T>(){}@interface(Same) pub fn U a<U>(){}");work=0;let expected=switch(group_identity::definition(ref isolated,2,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref first),bytes::view(ref expected)));
}
`
