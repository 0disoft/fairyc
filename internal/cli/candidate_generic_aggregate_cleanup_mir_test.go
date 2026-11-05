package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGenericAggregateCleanupMIR(t *testing.T) {
	cases := append(candidateAggregateFlowFixtures(), candidateAggregateLayoutFixture{
		name:  "eight scalar arguments",
		maker: aggregateLayoutMaker + `pub struct Octet<A,B,C,D,E,F,G,H>{A value;}pub fn Octet<i32,i32,i32,i32,i32,i32,i32,i32> octet(Octet<i32,i32,i32,i32,i32,i32,i32,i32> value){return value;}`,
	})
	wire := candidateAggregateLayoutOracle(t, cases...)
	builder := `let owner=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let lowered=switch(lower::lower_checked(sources,ref owner)){case .ok(p):{p}case .err(p):{return p.code;}};verify_values(ref lowered,ref owner);
 let checked=checking::checked_values(ref owner);let p=checking::checked_interfaces(ref owner);var usize i=types::count(records)+1;while(i<=values::type_count(checked)){assert(v::push<types::Type>(records,values::type_at(checked,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::shape_count(p)){assert(v::push<interfaces::Shape>(shapes,interfaces::shape_at(p,i)).is_ok);i+=1;}
 i=0;while(i<interfaces::field_count(p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(p))).is_ok);return 0;}`
	candidateAggregateBodyProbe(t, "module_unit_cleanup_mir_lower", wire, builder, "generic_aggregate_cleanup_mir.fy", "import module_unit_checked_bodies as checking;import module_unit_moves as moves;import module_unit_cleanup_mir_lower as lower;import module_unit_cleanup_mir_model as clean;import module_unit_cleanup_mir_verify as proof;")
}

func TestCandidateOrdinaryCleanupMIR(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_unit_cleanup_mir.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_unit_cleanup_mir;", "module main;", 1)
	main = strings.Replace(main, "import std::vec as v;", "import std::vec as v;import std::io as io;", 1)
	main = strings.ReplaceAll(main, "@test", "") + `
@effects(console) fn i32 main(){
 cleanup_mir_owns_provider_identity_and_ordered_fields();
 discard io::write("1\n");
 cleanup_mir_owns_distinct_partial_operand_values();
 discard io::write("2\n");
 cleanup_receiver_exemption_does_not_exempt_same_type_locals();
 discard io::write("3\n");
 cleanup_mir_rejects_invalid_handoffs_and_missing_authority();
 discard io::write("4\n");
 return 0;
}`
	candidateAtlasProbeSize(t, "module_unit_cleanup_mir_lower", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "1\n2\n3\n4\n" {
			t.Fatalf("ordinary cleanup MIR: %d %q", code, out)
		}
	})
}
