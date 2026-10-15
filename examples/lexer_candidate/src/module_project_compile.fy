module module_project_compile;
import module_units as grouping;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_cleanup_policy as policy;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import module_identified_ir as identified;
import generic_source_plan as source_plan;
import syntax as syntax;
import std::vec as v;

// Stages: 1 grouping, 2 binding, 3 closure, 4 values, 5 moves, 6 owned IR,
// 7 borrows, 8 source identity attachment, 9 Result obligations.
pub struct Problem {u32 stage;u32 code;usize module_index;usize start;usize end;usize first;usize stop;usize expected;usize actual;usize declaration_start;usize declaration_end;usize move_start;usize move_end;usize header_start;usize header_end;usize field_module;usize field_start;usize field_end;}
pub struct Range {usize first;usize stop;}
// Source-bound body and cleanup checks precede lowering and identity attachment.
// Restored-IR comparison and Complete publication remain later gates.
pub resource CheckedBodies {checking::Plan checked;Range range;cleanup{}}
pub fn ref values::Plan checked_values(ref CheckedBodies p){return checking::checked_values(ref p.checked);}
pub fn ref moves::Plan checked_moves(ref CheckedBodies p){return checking::checked_moves(ref p.checked);}
pub fn ref interfaces::Interfaces checked_interfaces(ref CheckedBodies p){return checking::checked_interfaces(ref p.checked);}
pub fn ref policy::Plan checked_cleanup(ref CheckedBodies p){return checking::cleanup_policy(ref p.checked);}
pub fn ref checking::Plan checked_state(ref CheckedBodies p){return ref p.checked;}
pub fn Range checked_range(ref CheckedBodies p){return p.range;}
pub resource Plan {v::Vec<ir::Program> units;v::Vec<Range> ranges;usize files;usize retries;cleanup{}}
fn Problem failed(u32 stage,u32 code,usize module_index,usize start,usize end,usize first,usize stop,usize expected,usize actual){return Problem{.stage,.code,.module_index,.start,.end,.first,.stop,.expected,.actual,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0};}
fn Problem checked_failure(checking::Problem e,usize first,usize stop){var Problem p=failed(e.stage,e.code,e.module_index,e.start,e.end,first,stop,e.expected,e.actual);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return p;}
fn Result<ir::Program,Problem> lowered(ref v::Vec<syntax::Parsed> sources,ref CheckedBodies p){return switch(lower::lower_checked(sources,ref p.checked)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(6,e.code,e.module_index,e.start,e.end,p.range.first,p.range.stop,0,0))}};}
pub fn Result<CheckedBodies,Problem> check_range(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    return switch(checking::build(sources,metadata,first,stop)){case .ok(p):{.ok(CheckedBodies{.checked=p,.range=Range{.first,.stop}})}case .err(e):{.err(checked_failure(e,first,stop))}};
}
pub fn Result<ir::Program,Problem> range(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    let p=switch(check_range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(e);}};return lowered(sources,ref p);
}
// Stage 8 attaches source identities only after the same body/Flow/borrow gates.
// Neither failed checking nor failed attachment exposes a successful IR prefix.
pub fn Result<identified::Program,Problem> range_identified(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref source_plan::Plan plan,usize first,usize stop,ref mut usize work){
    let p=switch(check_range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let program=switch(lowered(sources,ref p)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return switch(identified::build(program,plan,sources,metadata,checked_interfaces(ref p),work)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(8,e.code,e.module_index,e.start,e.end,first,stop,e.kind as usize,e.id))}};
}
// Cleanup policy retains raw code 10 through the owned-IR lowerer; later
// IR construction and verification use 1002 for the same capacity class.
pub fn bool capacity_failure(Problem p){return (p.stage>=2 && p.stage<=7 || p.stage==9) && p.code==10 || (p.stage==3 || p.stage==7) && p.code==4503 || p.stage==6 && p.code==1002;}
fn Problem append_range(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,usize depth,ref mut v::Vec<ir::Program> programs,ref mut v::Vec<Range> ranges,ref mut usize retries){
    switch(range(sources,metadata,first,stop)){
        case .ok(p):{
            if(!v::push<ir::Program>(programs,p).is_ok || !v::push<Range>(ranges,Range{.first,.stop}).is_ok){return failed(1,11,first,0,0,first,stop,0,0);}
            return failed(0,0,0,0,0,first,stop,0,0);
        }
        case .err(e):{
            // Only capacity failures can be retried. Single-source failures retain
            // the exact failing stage and location; no successful prefix escapes.
            if(!capacity_failure(e) || stop-first<=1 || depth>=8){return e;};
            *retries=*retries+1;let middle=first+(stop-first)/2;
            let left=append_range(sources,metadata,first,middle,depth+1,programs,ranges,retries);if(left.code!=0){return left;}
            return append_range(sources,metadata,middle,stop,depth+1,programs,ranges,retries);
        }
    };
    return failed(1,12,first,0,0,first,stop,0,0);
}
pub fn Result<Plan,Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){
    let grouped=switch(grouping::build(sources)){case .ok(p):{p}case .err(e):{return .err(failed(1,e.code,e.module_index,e.start,e.end,0,v::len<syntax::Parsed>(sources),0,0));}};
    var v::Vec<ir::Program> programs=v::empty<ir::Program>();var v::Vec<Range> ranges=v::empty<Range>();var usize retries=0;var usize i=0;var usize next=0;
    while(i<grouping::count(ref grouped)){let unit=grouping::at(ref grouped,i);assert(unit.first==next && unit.first<unit.stop);let e=append_range(sources,metadata,unit.first,unit.stop,0,ref mut programs,ref mut ranges,ref mut retries);if(e.code!=0){return .err(e);}next=unit.stop;i+=1;}
    assert(next==grouping::file_count(ref grouped) && v::len<ir::Program>(ref programs)==v::len<Range>(ref ranges));return .ok(Plan{.units=programs,.ranges,.files=next,.retries});
}
pub fn ref v::Vec<ir::Program> programs(ref Plan p){return ref p.units;}
pub fn usize count(ref Plan p){return v::len<ir::Program>(ref p.units);}
pub fn Range at(ref Plan p,usize index){return *v::at<Range>(ref p.ranges,index);}
pub fn usize retries(ref Plan p){return p.retries;}
