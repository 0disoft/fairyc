package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type candidateAggregateExecution struct {
	fixture candidateAggregateLayoutFixture
	code    int
	output  string
}

const aggregateTraceMaker = aggregateLayoutMaker + `
@link_name("observe_cleanup")@effects(console)unsafe extern C fn void trace(i32 n);
@effects(console)pub fn void signal(i32 n){unsafe{trace(n);}}
@effects(console)pub resource Token<T>{T value;cleanup{unsafe{trace(this.value);}}}
@effects(console)pub resource Envelope<T>{Token<T> first;Token<T> second;cleanup{unsafe{trace(9);}}}
@effects(console)pub fn Envelope<i32> envelope(){return Envelope<i32>{.first=Token<i32>{.value=1},.second=Token<i32>{.value=2}};}
`

func candidateAggregateExecutions() []candidateAggregateExecution {
	return []candidateAggregateExecution{
		{candidateAggregateLayoutFixture{name: "constructed copies", root: `module root;import maker as m;fn i32 main(){let a=m::Cell<i32>{.value=40};var b=a;b.value+=2;assert(a.value==40);return b.value;}`}, 42, ""},
		{candidateAggregateLayoutFixture{name: "distinct scalar instances", root: `module root;import maker as m;fn i32 main(){let a=m::Cell<i32>{.value=42};let b=m::Cell<bool>{.value=true};assert(b.value);return a.value;}`}, 42, ""},
		{candidateAggregateLayoutFixture{name: "aggregate record arrays", maker: aggregateLayoutMaker + `pub struct Rows<T>{Cell<T>[2] values;}pub fn Rows<i32> rows(){return Rows<i32>{.values=[Cell<i32>{.value=21};2]};}`, root: `module root;import maker as m;fn i32 main(){let row=m::rows();return row.values[0].value+row.values[1].value;}`}, 42, ""},
		{candidateAggregateLayoutFixture{name: "nested cleanup and defer", maker: aggregateTraceMaker, root: `module root;import maker as m;@effects(console)fn i32 main(){let owner=m::envelope();defer m::signal(8);return 42;}`}, 42, "8,9,2,1,"},
		{candidateAggregateLayoutFixture{name: "partial construction propagation", maker: aggregateTraceMaker + `@effects(console)pub fn Result<i32,i32> missing(){return .err(7);}@effects(console)pub fn Result<Envelope<i32>,i32> partial(){return .ok(Envelope<i32>{.first=Token<i32>{.value=3},.second=Token<i32>{.value=missing()?}});}`, root: `module root;import maker as m;@effects(console)fn i32 main(){return switch(m::partial()){case .ok(value):{discard value;0}case .err(code):{code+35}};}`}, 42, "3,"},
		{candidateAggregateLayoutFixture{name: "loop cleanup exits", maker: aggregateTraceMaker + `@effects(console)pub fn void loop(){var i32 i=0;while(i<3){let token=Token<i32>{.value=i+1};i+=1;if(i==1){continue;}if(i==2){break;}}}`, root: `module root;import maker as m;@effects(console)fn i32 main(){m::loop();return 42;}`}, 42, "1,2,"},
	}
}

func TestCandidateGenericAggregateC(t *testing.T) {
	candidateAggregateCReplay(t, false)
}

func TestCandidateGenericAggregatePartitionC(t *testing.T) {
	candidateAggregateCReplay(t, true)
}

func candidateAggregateCReplay(t *testing.T, partitioned bool) {
	t.Helper()
	cases := candidateAggregateExecutions()
	extra := make([]candidateAggregateLayoutFixture, len(cases))
	for i, c := range cases {
		extra[i] = c.fixture
	}
	wire := candidateAggregateLayoutOracle(t, extra...)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.ScalarSource(80)
	if err != nil {
		t.Fatal(err)
	}
	support += "\n#include <stdio.h>\nvoid observe_cleanup(int32_t value){printf(\"%d,\",(int)value);}\n"
	// The same native expectations independently constrain both compilers.
	if !partitioned {
		for _, c := range cases {
			unit := candidateAggregateReferenceC(t, c.fixture)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit, support)
				if err != nil {
					t.Fatalf("Go %s: %v", c.fixture.name, err)
				}
				candidateAggregateExecuteImage(t, image.Bytes, c.code, c.output)
			}
		}
	}
	builder := `let owner=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let lowered=switch(lower::lower_checked(sources,ref owner)){case .ok(p):{p}case .err(p):{return p.code;}};verify_values(ref lowered,sources);
 let checked=checking::checked_values(ref owner);let p=checking::checked_interfaces(ref owner);var usize i=types::count(records)+1;while(i<=values::type_count(checked)){assert(v::push<types::Type>(records,values::type_at(checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;}`
	imports := "import module_unit_checked_bodies as checking;import module_unit_body_ir_lower as lower;import module_unit_body_ir_model as body_ir;import module_unit_cleanup_mir_model as clean;import module_unit_c_emit as emit;import module_unit_c_aggregate as aggregate;import c_text as w;"
	checksFile := "generic_aggregate_c.fy"
	if partitioned {
		checksFile = "generic_aggregate_partition_c.fy"
		imports += "import module_unit_c_names as c_names;import candidate_limits as limits;import module_project_c_owner as c_owner;"
		builder = strings.Replace(builder, "verify_values(ref lowered,sources);", "verify_values(ref lowered,sources,metadata);", 1)
	}
	candidateAggregateBodyProbe(t, "module_unit_c_emit", wire, builder, checksFile, imports, func(t *testing.T, out string) {
		remaining := []byte(out)
		var chunks []string
		for len(remaining) != 0 {
			end := bytes.IndexByte(remaining, '\n')
			if end < 0 {
				t.Fatal("missing C length")
			}
			n, err := strconv.Atoi(string(remaining[:end]))
			if err != nil || n <= 0 || n > 2<<20 || n > len(remaining)-end-1 {
				t.Fatalf("invalid C length: %d %v", n, err)
			}
			remaining = remaining[end+1:]
			chunks = append(chunks, string(remaining[:n]))
			remaining = remaining[n:]
		}
		// Nine accepted baseline variants include module reversal and zero fields.
		modes := 1
		if partitioned {
			modes = 2
		}
		if len(chunks) != modes*(9+len(cases)) {
			t.Fatalf("C programs = %d", len(chunks))
		}
		for i, unit := range chunks {
			code, output := 0, ""
			program := i / modes
			if program >= 9 {
				code, output = cases[program-9].code, cases[program-9].output
			}
			packet := candidatebuild.Packet{Header: "/* single-buffer transport */\n", Literals: "typedef int fy_empty_literals;\n", Units: []string{unit}}
			if partitioned {
				var err error
				packet, err = candidatebuild.Decode([]byte(unit))
				if err != nil {
					t.Fatalf("partition packet %d: %v", i, err)
				}
				if len(packet.Units) < 2 {
					t.Fatalf("packet %d did not split bodies", i)
				}
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				// Keep the existing private candidate C consumer policy.
				image, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, packet.Header, packet.Literals, packet.Units, support)
				if err != nil {
					t.Fatalf("Fairy C %d/%s: %v", i, profile.Name, err)
				}
				candidateAggregateExecuteImage(t, image.Bytes, code, output)
			}
		}
	})
}

func candidateAggregateExecuteImage(t *testing.T, data []byte, code int, out string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "aggregate.exe")
	if err := os.WriteFile(path, data, 0700); err != nil {
		t.Fatal(err)
	}
	if actual, output := candidateAtlasProcess(t, path, nil); actual != code || output != out {
		t.Fatalf("aggregate execution: %d %q, want %d %q", actual, output, code, out)
	}
}

func candidateAggregateReferenceC(t *testing.T, fixture candidateAggregateLayoutFixture) string {
	t.Helper()
	maker := fixture.maker
	if maker == "" {
		maker = aggregateLayoutMaker
	}
	var input semantic.PackageInput
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
	for i, text := range []string{fixture.root, maker, aggregatePlanHelper} {
		name := []string{"root.fy", "maker.fy", "helper.fy"}[i]
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(name, []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Go source %s: %v", fixture.name, err)
		}
		id := hir.PackageID(i + 1)
		input.Files = append(input.Files, semantic.File{Package: id, Module: parsed.AST, Path: name})
		coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(fmt.Sprint(i+1), 64))
		if err != nil {
			t.Fatal(err)
		}
		coordinates[id] = coordinate
	}
	input.Imports = []semantic.PackageImport{{FromPackage: 1, ImportPath: "maker", ToPackage: 2}, {FromPackage: 2, ImportPath: "helper", ToPackage: 3}}
	checked, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !checked.Verified {
		t.Fatalf("Go aggregate %s: %v %+v %+v", fixture.name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	unit, err := cbackend.Emit(checked.MIR, hir.QualifiedModulePath(1, "root")+"::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	return unit.Source
}
