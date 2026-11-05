package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func candidateAtlasSourceVersionMain() string {
	main := strings.SplitN(candidateAtlasTypesMain, " if(process::arg_count()==2)", 2)[0]
	main = strings.Replace(main, "@effects(filesystem,console)", "@effects(filesystem,console,environment)", 1)
	main = strings.Replace(main, "import project_atlas_types as types;", `import project_atlas_types as types;
import project_atlas_produce as produce;import project_atlas_schema as schema;import project_atlas_instances as instances;
import project_atlas_facts as facts;import project_lock_document as locks;import project_workspace_graph as graph;
import project_cache_environment as environment;import project_compiler_image_hash as image;
import project_atlas_current as current;import project_atlas_join as join;
import project_catalog as catalog;import project_catalog_options as options;import project_compiler_source_mode as mode;import project_atlas_source_version as source_version;`, 1)
	main = strings.Replace(main, " var sources::Budget", `
 let request=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};let version=options::version(ref request);
 var u8[1] control=[0];let n=switch(io::read_stdin(ref mut control)){case .ok(p):{p}case .err(_):{return 7;}};
 var sources::Budget`, 1)
	main = strings.Replace(main, " var usize source_work=0;", ` if(!source_version::select(ref source,version,ref mut meter).is_ok){return 7;}
 var usize source_work=0;`, 1)
	main = strings.Replace(main, " let complete=switch(compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work))", ` let aggregates=switch(mode::aggregates(ref source,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let complete=switch(if(aggregates){compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)})`, 1)
	return main + `
 let lock=switch(locks::project(resolved,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let metadata=switch(if(aggregates){schema::prepare_aggregates(ref source,ref table,ref complete,ref mut meter)}else{schema::prepare(ref source,ref table,ref complete,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 var rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 let observed=switch(if(aggregates){instances::collect_aggregates(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}else{instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 var usize image_work=0;let compiler_image=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return 7;}};
 let absent=table::assemble(bytes::empty(),v::empty<table::Coordinate>());
 var packages=v::empty<produce::Package>();var usize owner=0;var usize total=0;var usize external=0;
 while(owner<graph::node_count(graph)){
  let u32 selected=if(n==1 && control[0]==102){if(version==3){1}else{3}}else{version};
  let f=switch(facts::collect_version(graph,ref lock,ref table,compiler::public_interfaces(ref complete),p,image::value(ref compiler_image),owner,selected,instances::template_count(ref observed)>0,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
  if(n==1 && control[0]==98){meter=16777216;}
  let part=switch(if(n==1 && control[0]==99){produce::package_version(f,ref source,ref absent,ref complete,ref metadata,ref mut rows,ref observed,owner,version,ref mut meter)}else{produce::package_version(f,ref source,ref table,ref complete,ref metadata,ref mut rows,ref observed,owner,version,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
  total+=produce::call_count(ref part);external+=produce::external_count(ref part);if(!v::push<produce::Package>(ref mut packages,part).is_ok){return 7;}owner+=1;
 }
 var usize export_work=0;var usize query_work=0;var admitted=v::empty<current::Ready>();var out=bytes::empty();var page=bytes::empty();var usize matches=0;
 if(!w::raw(ref mut out,b"{\"maps\":[",ref mut export_work).is_ok){return 7;}owner=0;
 while(owner<v::len<produce::Package>(ref packages)){
  let part=v::at<produce::Package>(ref packages,owner);
  let encoded=switch(produce::encode(part,total-external,external,ref mut export_work)){case .ok(p):{p}case .err(_):{return 7;}};
  let ready=switch(produce::ready(part,total-external,external,ref mut query_work)){case .ok(p):{p}case .err(_):{return 7;}};
  if(bytes::equal(produce::uid(part),options::uid(ref request))){let selected=switch(catalog::encode(ref ready,ref request,ref mut query_work)){case .ok(p):{p}case .err(_):{return 7;}};if(!w::raw(ref mut page,bytes::view(ref selected),ref mut export_work).is_ok){return 7;}matches+=1;}
  if(!v::push<current::Ready>(ref mut admitted,ready).is_ok){return 7;}
  if(owner>0 && !w::raw(ref mut out,b",",ref mut export_work).is_ok || !w::raw(ref mut out,bytes::view(ref encoded),ref mut export_work).is_ok){return 7;}owner+=1;
 }
 if(matches!=1){return 7;}let joined=switch(join::join(admitted,8,8192,ref mut query_work)){case .ok(p):{p}case .err(_):{return 7;}};
 if(join::package_count(ref joined)!=graph::node_count(graph) || !w::raw(ref mut out,b"],\"catalog\":",ref mut export_work).is_ok || !w::raw(ref mut out,bytes::view(ref page),ref mut export_work).is_ok || !w::raw(ref mut out,b"}",ref mut export_work).is_ok){return 7;}
 return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`
}

func TestCandidateAtlasSourceVersion(t *testing.T) {
	type fixture struct {
		name, root string
		minimum    int
		versions   []preparedAtlasWorkspace
	}
	ordinary := candidateAtlasFamilyWorkspace(t)
	owner := writeWorkspaceSessionFixture(t, atlasOwnerQualifiedNamespaceSource(true))
	used := candidateAtlasGenericProductionWorkspace(t, atlasGenericProductionUsedSource)
	unused := writeWorkspaceSessionFixture(t, `module main;pub fn T unused<T>(T value){return value;}fn i32 main(){return 0;}`)
	aggregate := writeWorkspaceSessionFixture(t, `module main;import answer as a;pub struct Cell<T>{T value;}pub resource Token<T>{Cell<T> payload;cleanup{}}pub struct Unused<T>{T value;}fn Token<T> make<T>(T value){return Token<T>{.payload=Cell<T>{.value=value}};}fn i32 main(){let t=make<i32>(a::value());discard make<u32>(7);return t.payload.value;}`)
	unusedAggregate := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	writeWorkspaceFile(t, unusedAggregate, "libs/answer/src/answer.fy", `module answer;pub struct Unused<T>{T value;}pub resource UnusedToken<T>{T value;cleanup{}}pub fn i32 value(){return 42;}`)
	sdkUnused := writeWorkspaceSessionFixture(t, candidateAtlasSDKVectorUnusedSource)
	sdkUsed := writeWorkspaceSessionFixture(t, candidateAtlasSDKVectorUsedSource)
	sdkScalars := writeWorkspaceSessionFixture(t, atlasSDKCommandSource)
	fixtures := []fixture{{"ordinary-families", ordinary, 1, nil}, {"owner-namespaces", owner, 1, nil}, {"used-functions", used, 2, nil}, {"unused-functions", unused, 2, nil}, {"used-aggregates", aggregate, 3, nil}, {"unused-dependency-aggregates", unusedAggregate, 3, nil}, {"sdk-unused", sdkUnused, 3, nil}, {"sdk-used", sdkUsed, 3, nil}, {"sdk-scalars", sdkScalars, 1, nil}}
	for i := range fixtures {
		f := &fixtures[i]
		for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(f.root, false, &diagnostic, schema)
			if (code == 0) != (v+1 >= f.minimum) {
				t.Fatalf("%s/v%d independent source admission: %d %s", f.name, v+1, code, &diagnostic)
			}
			f.versions = append(f.versions, prepared)
		}
	}
	candidateAtlasProbeSize(t, "project_atlas_source", candidateAtlasSourceVersionMain(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		imageBytes, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		compiler := buildidentity.Identity(sha256.Sum256(imageBytes))
		calls, maps := 0, 0
		for _, f := range fixtures {
			before := remoteTree(t, f.root)
			for v, prepared := range f.versions {
				t.Run(fmt.Sprintf("%s/v%d", f.name, v+1), func(t *testing.T) {
					r := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 1 << 20}}
					args := candidateCatalogArgs(candidateCatalogFixture{uid: "example.session.app", version: v + 1}, r, 1<<20)
					calls++
					var code int
					var out, stderr string
					if strings.HasPrefix(f.name, "sdk-") {
						code, out, stderr = candidateAtlasSDKCLIProcess(t, image, f.root, args)
					} else {
						code, out, stderr = fetchCLIProcess(t, image, f.root, args, "")
					}
					if v+1 < f.minimum {
						if code != 7 || out != "" || stderr != "" {
							t.Fatalf("source family must reject atomically: %d %q %q", code, out, stderr)
						}
						return
					}
					if code != 0 || stderr != "" {
						t.Fatalf("source producer: %d %s %s", code, out, stderr)
					}
					var result struct {
						Maps    []json.RawMessage `json:"maps"`
						Catalog json.RawMessage   `json:"catalog"`
					}
					if err := json.Unmarshal([]byte(out), &result); err != nil || len(result.Maps) != len(prepared.partitions) {
						t.Fatalf("source partitions: %v %d want %d", err, len(result.Maps), len(prepared.partitions))
					}
					seen := map[string]bool{}
					for _, raw := range result.Maps {
						decode := []func([]byte, uint64) (atlasmap.Partition, error){atlasmap.Decode, atlasmap.DecodeV2, atlasmap.DecodeV3}[v]
						got, err := decode(raw, 4<<20)
						if err != nil {
							t.Fatalf("Go rejected actual source map: %v", err)
						}
						h := got.Header()
						want, found := selectAtlasPartition(prepared.partitions, h.PackageUID)
						if !found || seen[h.PackageInstance] || h.CompilerBuild != compiler.String() || h.PackageInstance != want.Header().PackageInstance || h.InterfaceHash != want.Header().InterfaceHash || h.GenericInterfaceHash != want.Header().GenericInterfaceHash {
							t.Fatalf("independent source/header identities differ: %+v", h)
						}
						seen[h.PackageInstance] = true
						encode := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[v]
						canonical, err := encode(want, 4<<20)
						if err != nil {
							t.Fatal(err)
						}
						want, err = decode(canonical, 4<<20)
						if err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(got.Declarations(), want.Declarations()) || !reflect.DeepEqual(got.Symbols(), want.Symbols()) {
							a, b := got.Declarations(), want.Declarations()
							for i := 0; i < len(a) && i < len(b); i++ {
								if !reflect.DeepEqual(a[i], b[i]) {
									t.Fatalf("original declaration %s differs\ngot %+v\nwant %+v", h.PackageUID, a[i], b[i])
								}
							}
							t.Fatalf("original source declaration inventory/symbols differ for %s", h.PackageUID)
						}
						compareAtlasProducedFunctions(t, got.Functions(), want.Functions())
						a, b := got.Specializations(), want.Specializations()
						if len(a) != len(b) {
							t.Fatalf("specialization count: %d != %d", len(a), len(b))
						}
						for i, instance := range a {
							w := b[i]
							if instance.ID != w.ID || instance.Declaration != w.Declaration || !reflect.DeepEqual(instance.TypeArguments, w.TypeArguments) || !reflect.DeepEqual(instance.Aggregate, w.Aggregate) || (instance.Cleanup == nil) != (w.Cleanup == nil) {
								t.Fatal("independent source specialization differs")
							}
							if instance.Function.Function.Name != "" {
								compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{instance.Function}, []semanticsnapshot.FunctionSummary{w.Function})
							}
							if instance.Cleanup != nil {
								compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{*instance.Cleanup}, []semanticsnapshot.FunctionSummary{*w.Cleanup})
							}
						}
						e := prepared.expectations[h.PackageInstance]
						e.CompilerBuild = compiler
						query := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[v]
						index, err := query(got, e)
						if err != nil {
							t.Fatalf("independent current source rejected Fairy map: %v", err)
						}
						if h.PackageUID == "example.session.app" {
							golden, err := candidateCatalogOracle(t, candidateCatalogFixture{version: v + 1, index: index}, r, 1<<20)
							if err != nil || !bytes.Equal(result.Catalog, golden) {
								t.Fatalf("actual source/current catalog differs: %v\ngot %s\nwant %s", err, result.Catalog, golden)
							}
						}
						maps++
					}
					if f.name == "ordinary-families" {
						controls := []string{"f"}
						if v == 0 {
							controls = append(controls, "b", "c")
						}
						for _, control := range controls {
							calls++
							code, out, stderr := candidateMCPProcess(t, image, f.root, args, control)
							if code != 7 || out != "" || stderr != "" {
								t.Fatalf("work/coordinate/facts control %s: %d %.200q %.200q", control, code, out, stderr)
							}
						}
					}
				})
			}
			if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
				t.Fatal("source generation changed workspace bytes")
			}
		}
		t.Logf("versioned source native processes: %d, independent current maps: %d", calls, maps)
	}, "project_catalog")
}
