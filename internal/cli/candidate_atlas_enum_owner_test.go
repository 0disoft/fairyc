package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type atlasEnumOwnerMethod struct {
	name, receiver, text string
	parameters, calls    int
}

type atlasEnumOwnerFixture struct {
	atlasOwnerQualifiedFixture
	methods []atlasEnumOwnerMethod
}

func atlasEnumOwnerFixtures() []atlasEnumOwnerFixture {
	fixtures := []atlasEnumOwnerFixture{
		{atlasOwnerQualifiedFixture{name: "old-unused-static", source: `module main;enum State{ready;}fn i32 State::answer(){return 42;}fn i32 main(){return 42;}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{{"State::answer", "", "fn i32 State::answer(){return 42;}", 0, 0}}},
		{atlasOwnerQualifiedFixture{name: "static-qualified-call", source: `module main;enum State{ready;}fn i32 State::answer(){return 42;}fn i32 main(){return State::answer();}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{{"State::answer", "", "fn i32 State::answer(){return 42;}", 0, 1}}},
		{atlasOwnerQualifiedFixture{name: "value-receiver", source: `module main;enum State{ready;}fn i32 State::answer(State this){return 42;}fn i32 main(){let State state=.ready;return state.answer();}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{{"State::answer", "value", "fn i32 State::answer(State this){return 42;}", 1, 1}}},
		{atlasOwnerQualifiedFixture{name: "shared-receiver", source: `module main;enum State{ready;}fn i32 State::answer(ref State this){return 42;}fn i32 main(){let State state=.ready;let view=ref state;return view.answer();}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{{"State::answer", "shared", "fn i32 State::answer(ref State this){return 42;}", 1, 1}}},
		{atlasOwnerQualifiedFixture{name: "exclusive-receiver", source: `module main;enum State{ready;}fn i32 State::answer(ref mut State this){return 42;}fn i32 main(){var State state=.ready;let view=ref mut state;return view.answer();}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{{"State::answer", "exclusive", "fn i32 State::answer(ref mut State this){return 42;}", 1, 1}}},
		{atlasOwnerQualifiedFixture{name: "two-owners-and-ffi", source: `module main;
enum State{ready;}enum Other{ready;}
unsafe extern C fn i32 native_value(i32 value);
fn i32 helper(i32 value){return value;}
fn i32 State::answer(i32 value){return helper(value)+unsafe{native_value(0)};}
fn i32 Other::answer(){return 2;}
fn i32 answer(){return 0;}
fn i32 main(){return State::answer(40)+Other::answer()+answer();}`, uid: "example.session.app"}, []atlasEnumOwnerMethod{
			{"State::answer", "", "fn i32 State::answer(i32 value){return helper(value)+unsafe{native_value(0)};}", 1, 1},
			{"Other::answer", "", "fn i32 Other::answer(){return 2;}", 0, 1},
		}},
		{atlasOwnerQualifiedFixture{name: "imported-static", source: `module main;import answer as a;fn i32 main(){return a::State::answer(42);}`, member: "module answer;\npub enum State{ready;}\n/// 원본 enum owner.\npub fn i32 State::answer(i32 value){return value;}", uid: "example.session.answer"}, []atlasEnumOwnerMethod{{"State::answer", "", "pub fn i32 State::answer(i32 value){return value;}", 1, 1}}},
	}
	for _, index := range []int{2, 3, 4} {
		f := fixtures[index]
		f.name += "-inferred-factory"
		f.source = strings.Replace(f.source, "fn i32 main()", "fn State make(){return .ready;}fn i32 main()", 1)
		f.source = strings.Replace(f.source, "let State state=.ready;", "let state=make();", 1)
		f.source = strings.Replace(f.source, "var State state=.ready;", "var state=make();", 1)
		fixtures = append(fixtures, f)
	}
	return fixtures
}

func atlasEnumOwnerNegativeFixtures() []atlasOwnerQualifiedFixture {
	return []atlasOwnerQualifiedFixture{
		{name: "missing-owner", source: `module main;enum State{ready;}fn i32 Missing::answer(){return 42;}fn i32 main(){return 42;}`},
		{name: "wrong-qualified-owner", source: `module main;enum State{ready;}enum Other{ready;}fn i32 State::answer(){return 42;}fn i32 main(){return Other::answer();}`},
		{name: "unqualified-fallback", source: `module main;enum State{ready;}fn i32 State::answer(){return 42;}fn i32 main(){return answer();}`},
		{name: "wrong-receiver-owner", source: `module main;enum State{ready;}enum Other{ready;}fn i32 State::answer(State this){return 42;}fn i32 main(){let Other other=.ready;return other.answer();}`},
	}
}

type atlasEnumOwnerOracle struct {
	fixture  atlasEnumOwnerFixture
	root     string
	prepared preparedAtlasWorkspace
	parts    map[string]atlasmap.Partition
	global   candidateAtlasSDKVectorGlobal
}

func prepareAtlasEnumOwnerOracle(t *testing.T, fixture atlasEnumOwnerFixture) atlasEnumOwnerOracle {
	t.Helper()
	root := atlasOwnerQualifiedWorkspace(t, fixture.atlasOwnerQualifiedFixture)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		t.Fatalf("CURRENT Go Atlas v3 positive oracle %s: %d %s", fixture.name, code, &diagnostic)
	}
	f := atlasEnumOwnerOracle{fixture: fixture, root: root, prepared: prepared, parts: map[string]atlasmap.Partition{}}
	for _, p := range prepared.partitions {
		encoded, err := atlasmap.EncodeV3(p, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := atlasmap.DecodeV3(encoded, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := atlasmap.EncodeV3(decoded, 4<<20)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatalf("Go v3 map copy/codec changed facts: %v", err)
		}
		f.parts[decoded.Header().PackageInstance] = decoded
	}
	for _, p := range f.parts {
		for _, fn := range p.Functions() {
			for _, call := range fn.Calls {
				if call.TargetKind == semanticsnapshot.CallTargetResolved {
					f.global.calls++
				} else {
					f.global.external++
				}
			}
		}
	}
	verifyAtlasEnumOwnerFacts(t, f)
	if _, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}); err != nil {
		t.Fatalf("Go enum owner complete workspace join: %v", err)
	}
	return f
}

func verifyAtlasEnumOwnerFacts(t *testing.T, f atlasEnumOwnerOracle) {
	t.Helper()
	methods, calls := map[string]int{}, map[string]int{}
	for _, p := range f.parts {
		for _, fn := range p.Functions() {
			for _, call := range fn.Calls {
				for _, method := range f.fixture.methods {
					if call.Target.Name == method.name {
						if call.TargetKind != semanticsnapshot.CallTargetResolved || call.Certainty != semanticsnapshot.CertaintyExact || call.Target.Kind != semanticsnapshot.DeclarationFunction {
							t.Fatal("enum associated call lost exact resolution", call)
						}
						calls[method.name]++
					}
				}
			}
		}
		if p.Header().PackageUID != f.fixture.uid {
			continue
		}
		index, err := atlasmap.NewQueryIndexV3(p, f.prepared.expectations[p.Header().PackageInstance])
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range p.Declarations() {
			if d.Owner == nil {
				continue
			}
			var expected *atlasEnumOwnerMethod
			for i := range f.fixture.methods {
				if f.fixture.methods[i].name == d.Key.Name {
					expected = &f.fixture.methods[i]
				}
			}
			if expected == nil {
				t.Fatal("unexpected enum owner declaration", d)
			}
			owner, _, _ := strings.Cut(expected.name, "::")
			if d.Owner.Kind != semanticsnapshot.TypeNominal || d.Owner.Name != d.Key.Module+"::"+owner || d.Owner.PackageCoordinate != p.Header().PackageCoordinate || d.Receiver != expected.receiver || len(d.Parameters) != expected.parameters {
				t.Fatalf("enum nominal owner/receiver/parameter identity differs: %+v", d)
			}
			foundEnum := false
			for _, nominal := range p.Declarations() {
				if nominal.Key.Module == d.Key.Module && nominal.Key.Name == owner && nominal.Key.Kind == semanticsnapshot.DeclarationEnum {
					foundEnum = true
				}
			}
			if !foundEnum {
				t.Fatal("method owner is not the original enum declaration")
			}
			raw, file := f.fixture.source, "main.fy"
			if f.fixture.member != "" {
				raw, file = f.fixture.member, "answer.fy"
			}
			start := strings.Index(raw, expected.text)
			if start < 0 || d.Span.Source != file || d.Span.StartByte != start || d.Span.EndByte != start+len(expected.text) || d.Span.PackageInstance != d.Key.PackageInstance {
				t.Fatalf("original enum method/body byte range differs: got %+v want %s:%d..%d", d.Span, file, start, start+len(expected.text))
			}
			body := strings.Index(expected.text, "{")
			if body < 0 || raw[d.Span.StartByte+body:d.Span.EndByte] != expected.text[body:] {
				t.Fatal("declaration span does not retain the exact source body")
			}
			if expected.receiver != "" && (d.Parameters[0].Name != "this" || d.Parameters[0].Span.Source != file) {
				t.Fatal("enum receiver parameter lost original name/source")
			}
			catalog, err := index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Name: expected.name, Kinds: []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationFunction}, Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 4 << 20}})
			if err != nil || !catalog.Complete || catalog.Total != 1 || len(catalog.Symbols) != 1 || catalog.Symbols[0].Name != expected.name {
				t.Fatalf("enum qualified exact-name catalog differs: %+v %v", catalog, err)
			}
			methods[expected.name]++
		}
	}
	for _, method := range f.fixture.methods {
		if methods[method.name] != 1 || calls[method.name] != method.calls {
			t.Fatalf("enum method inventory %s: methods=%d calls=%d want 1/%d", method.name, methods[method.name], calls[method.name], method.calls)
		}
	}
	if f.fixture.name == "two-owners-and-ffi" && f.global.external != 1 {
		t.Fatalf("Go FFI oracle must retain one external call: %+v", f.global)
	}
}

func TestAtlasEnumOwnerSourceOracle(t *testing.T) {
	for _, fixture := range atlasEnumOwnerFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			f := prepareAtlasEnumOwnerOracle(t, fixture)
			t.Logf("CURRENT Go Atlas v3: %d partitions, %d enum methods, %d resolved + %d external calls", len(f.parts), len(fixture.methods), f.global.calls, f.global.external)
		})
	}
	for _, fixture := range atlasEnumOwnerNegativeFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := atlasOwnerQualifiedWorkspace(t, fixture)
			var diagnostic bytes.Buffer
			if _, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3); code == ExitSuccess || diagnostic.Len() == 0 {
				t.Fatalf("CURRENT Go Atlas must refuse wrong owner: %d %s", code, &diagnostic)
			}
		})
	}
}

// Reuse the leaf producer's source-owned facts, current expectations and join.
// Both modes copy the Parsed source arenas. Restore additionally transports the
// checked program through its real artifact codec and source admission boundary.
func candidateAtlasEnumOwnerProduceMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasGenericProduceMain()
	replace := func(old, next string) {
		t.Helper()
		if strings.Count(main, old) != 1 {
			t.Fatalf("enum owner probe requires one exact port: %q", old)
		}
		main = strings.Replace(main, old, next, 1)
	}
	replace("import project_atlas_produce as produce;", `import project_atlas_produce as produce;
import module_project_artifact_encode as saved_encoder;import module_project_artifact_decode as saved_decoder;
import candidate_artifact_wire as saved_wire;import scanner as scanner;`)
	replace(` let source=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};`, `
 let original_source=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let source=switch(input::with_source(ref original_source,0,input::source(ref original_source,0),ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 var u8[64] replay_argument=[0;64];let usize replay_size=if(process::arg_count()==2){switch(process::arg(1,ref mut replay_argument)){case .ok(n):{n}case .err(_):{return 7;}}}else{0};let ref u8[] replay_raw=ref replay_argument;let replay_mode=replay_raw[..replay_size];`)
	replace(` let complete=switch(compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(p):{discard io::write_usize(p.stage as usize);discard io::write("/");discard io::write_usize(p.code as usize);return 7;}};`, `
 let complete=switch(enum_complete(ref source,ref table,bytes::equal(replay_mode,b"restore"))){case .ok(p):{p}case .err(code):{discard io::write("complete/");discard io::write_usize(code as usize);return 7;}};`)
	replace(` let metadata=switch(schema::prepare(ref source,ref table,ref complete,ref mut meter))`, `
 if(bytes::equal(replay_mode,b"stale-source")){
  var changed=bytes::empty();if(!bytes::append(ref mut changed,syntax::source_view(input::source(ref source,0))).is_ok || !bytes::append(ref mut changed,b"\n").is_ok){return 9;}
  let tokens=switch(scanner::collect(changed,false,8388608,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return 9;}};
  let parsed=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{return 9;}};
  let stale=switch(input::with_source(ref source,0,ref parsed,ref mut meter)){case .ok(p):{p}case .err(_):{return 9;}};
  return switch(schema::prepare(ref stale,ref table,ref complete,ref mut meter)){case .ok(_):{0}case .err(code):{discard io::write_usize(code as usize);7}};
 }
 let metadata=switch(schema::prepare(ref source,ref table,ref complete,ref mut meter))`)
	return main + `
fn Result<compiler::Complete,u32> enum_complete(ref input::Input source,ref table::Coordinates table,bool replay){
 var usize source_work=0;var usize link_work=0;
 let fresh=switch(compiler::build_complete(source,table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
 if(!replay){return .ok(fresh);}
 var usize encode_work=0;let encoded=saved_encoder::encode(compiler::checked_program(ref fresh),ref mut encode_work)?;
 var usize decode_work=0;let detached=saved_decoder::decode(saved_wire::content(ref encoded),ref mut decode_work)?;
 source_work=0;var usize artifact_work=0;link_work=0;
 return switch(compiler::restore_complete_sources(detached,input::sources(source),input::owners(source),input::bindings(source),table,ref mut source_work,ref mut artifact_work,ref mut link_work)){case .ok(p):{.ok(p)}case .err(e):{.err(e.code)}};
}
`
}

func TestCandidateAtlasEnumOwnerProduction(t *testing.T) {
	var fixtures []atlasEnumOwnerOracle
	for _, fixture := range atlasEnumOwnerFixtures() {
		fixtures = append(fixtures, prepareAtlasEnumOwnerOracle(t, fixture))
	}
	type negative struct{ name, root string }
	var negatives []negative
	for _, fixture := range atlasEnumOwnerNegativeFixtures() {
		negatives = append(negatives, negative{fixture.name, atlasOwnerQualifiedWorkspace(t, fixture)})
	}
	main := candidateAtlasEnumOwnerProduceMain(t)
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		imageBytes, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		compiler := buildidentity.Identity(sha256.Sum256(imageBytes))
		t.Setenv("PATH", "")
		maps := 0
		for _, f := range fixtures {
			t.Run(f.fixture.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				var freshOutput string
				for _, mode := range []string{"fresh", "restore"} {
					var args []string
					if mode == "restore" {
						args = []string{mode}
					}
					code, output, diagnostic := fetchCLIProcess(t, image, f.root, args, "")
					if code != ExitSuccess || diagnostic != "" {
						t.Fatalf("Fairy enum owner %s: %d %s %s", mode, code, output, diagnostic)
					}
					var documents []json.RawMessage
					if err := json.Unmarshal([]byte(output), &documents); err != nil || len(documents) != len(f.parts) {
						t.Fatalf("enum owner map inventory: %v got %d want %d", err, len(documents), len(f.parts))
					}
					var actual []atlasmap.Partition
					current := map[string]atlasmap.LoadExpectation{}
					for _, raw := range documents {
						got, err := atlasmap.DecodeV3(raw, 4<<20)
						if err != nil {
							t.Fatalf("Go rejected actual Fairy enum owner map: %v", err)
						}
						h := got.Header()
						want, ok := f.parts[h.PackageInstance]
						if !ok || current[h.PackageInstance].CompilerBuild != (buildidentity.Identity{}) {
							t.Fatal("unexpected or duplicate enum owner partition", h.PackageInstance)
						}
						verifyCandidateAtlasSDKVectorPartition(t, got, want)
						verifyCandidateAtlasSDKVectorCompleteness(t, got, want, f.global)
						if h.CompilerBuild != compiler.String() || !reflect.DeepEqual(got.Files(), want.Files()) || !reflect.DeepEqual(got.Modules(), want.Modules()) {
							t.Fatal("enum owner source/image identity differs", h.PackageUID)
						}
						// Current source/interface expectations come from Go, while the
						// native executable identity is independently hashed above.
						expected := f.prepared.expectations[h.PackageInstance]
						expected.CompilerBuild = compiler
						if _, err := atlasmap.NewQueryIndexV3(got, expected); err != nil {
							t.Fatalf("independent current-source map admission: %v", err)
						}
						current[h.PackageInstance] = expected
						actual = append(actual, got)
						maps++
					}
					if _, err := atlasmap.NewWorkspaceCallIndexV3(actual, current, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}); err != nil {
						t.Fatalf("independent enum owner map join: %v", err)
					}
					if mode == "fresh" {
						freshOutput = output
					} else if output != freshOutput {
						t.Fatal("artifact restore changed canonical enum owner maps")
					}
				}
				if f.fixture.name == "old-unused-static" {
					code, out, diagnostic := fetchCLIProcess(t, image, f.root, []string{"stale-source"}, "")
					if code != 7 || out != "7" || diagnostic != "" {
						t.Fatalf("stale enum owner source must refuse: %d %q %q", code, out, diagnostic)
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("enum owner map producer changed workspace bytes")
				}
			})
		}
		for _, bad := range negatives {
			t.Run(bad.name, func(t *testing.T) {
				before := remoteTree(t, bad.root)
				code, out, diagnostic := fetchCLIProcess(t, image, bad.root, nil, "")
				if code != 7 || diagnostic != "" || strings.HasPrefix(out, "[") || !reflect.DeepEqual(before, remoteTree(t, bad.root)) {
					t.Fatalf("wrong enum owner must refuse before map publication: %d %q %q", code, out, diagnostic)
				}
			})
		}
		t.Logf("enum owner maps=%d: %d sources x copied-fresh/restored; independent current Go v3 identities/facts/completeness", maps, len(fixtures))
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph", "module_project_artifact_encode", "module_project_artifact_decode", "candidate_artifact_wire", "scanner")
}
