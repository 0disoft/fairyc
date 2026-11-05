package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGenericAggregateIdentities(t *testing.T) {
	cases := candidateAggregateExecutions()
	extra := make([]candidateAggregateLayoutFixture, len(cases))
	for i, c := range cases {
		extra[i] = c.fixture
	}
	wire := candidateAggregateLayoutOracle(t, extra...)
	builder := `let owner=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let lowered=switch(lower::lower_checked(sources,ref owner)){case .ok(p):{p}case .err(p):{return p.code;}};
 let checked=checking::checked_values(ref owner);let p=checking::checked_interfaces(ref owner);let original=aggregates::source_plan(signatures::aggregate_source(metadata));var usize identity_work=0;
 let named=switch(identified::build(lowered,original,sources,metadata,p,ref mut identity_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate identity ");discard io::write_stderr_u32(e.code);discard io::write_stderr(" kind/id ");discard io::write_stderr_u32(e.kind);discard io::write_stderr_usize(e.id);return e.code;}};verify_values(ref named,sources,metadata,p,original);
 var usize i=types::count(records)+1;while(i<=values::type_count(checked)){assert(v::push<types::Type>(records,values::type_at(checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;}`
	imports := "import module_unit_checked_bodies as checking;import module_unit_body_ir_lower as lower;import module_unit_body_ir_model as body_ir;import module_unit_cleanup_mir_model as clean;import module_identified_ir as identified;import module_source_specializations as selected;import module_unit_generics as generics;import generic_source_catalog as templates;"
	candidateAggregateBodyProbe(t, "module_unit_c_emit", wire, builder, "generic_aggregate_identity.fy", imports)
}

func TestCandidateOrdinarySourceIdentities(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_source_specializations.fy"))
	if err != nil {
		t.Fatal(err)
	}
	sdk, err := filepath.Abs(filepath.Join("..", "sdkstdlib", "sources", "optional", "std", "vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_source_specializations;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + `
@effects(filesystem,console) fn i32 main(){
 typed_functions_vectors_cleanup_and_calls_select_source_ids();
 stale_sources_owners_and_exact_work_limits_fail_closed();
 selected_id_bytes_survive_all_original_owners();
 return 0;
}`
	candidateAtlasProbeSize(t, "module_source_specializations", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, nil, sdk); code != 0 || out != "" {
			t.Fatalf("ordinary source identities: %d %q", code, out)
		}
	}, "project_source_plan", "generic_source_closure", "generic_source_dependency_json")
}
