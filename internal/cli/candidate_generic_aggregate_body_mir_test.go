package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGenericAggregateBodyMIR(t *testing.T) {
	wire := candidateAggregateLayoutOracle(t, candidateAggregateMIRFixtures()...)
	builder := `let owner=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let lowered=switch(lower::lower_checked(sources,ref owner)){case .ok(p):{p}case .err(p):{return p.code;}};verify_values(ref lowered,ref owner);
 let checked=checking::checked_values(ref owner);let p=checking::checked_interfaces(ref owner);var usize i=types::count(records)+1;while(i<=values::type_count(checked)){assert(v::push<types::Type>(records,values::type_at(checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;}`
	candidateAggregateBodyProbe(t, "module_unit_body_ir_lower", wire, builder, "generic_aggregate_body_mir.fy", "import module_unit_checked_bodies as checking;import module_unit_body_ir_lower as lower;import module_unit_body_ir_model as body_ir;import module_unit_body_ir_verify as proof;import module_unit_cleanup_mir_model as clean;import module_unit_region_model as regions;")
}

func TestCandidateOrdinaryBodyMIR(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_unit_body_ir.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_unit_body_ir;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "") + `
@effects(console) fn i32 main(){
 body_ir_retains_original_blocks_calls_and_cleanup_frames();
 discard io::write("1\n");
 body_ir_preserves_literal_cooking_assignment_and_for_links();
 discard io::write("2\n");
 body_ir_retains_specialized_cleanup_and_partial_values();
 discard io::write("3\n");
 body_ir_rejects_malformed_owned_body_tables();
 discard io::write("4\n");
 body_ir_revalidates_utf8_cstr_termination_and_literal_types();
 discard io::write("5\n");
 return 0;
}`
	candidateAtlasProbeSize(t, "module_unit_body_ir_lower", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "1\n2\n3\n4\n5\n" {
			t.Fatalf("ordinary body MIR: %d %q", code, out)
		}
	}, "module_unit_body_ir_lookup")
}
