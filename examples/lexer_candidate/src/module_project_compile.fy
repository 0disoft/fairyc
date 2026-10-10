module module_project_compile;
import module_units as grouping;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import syntax as syntax;
import std::vec as v;

// Stages: 1 grouping, 2 binding, 3 closure, 4 values, 5 moves, 6 owned IR.
pub struct Problem {u32 stage;u32 code;usize module_index;usize start;usize end;usize first;usize stop;usize expected;usize actual;}
pub struct Range {usize first;usize stop;}
pub resource Plan {v::Vec<ir::Program> units;v::Vec<Range> ranges;usize files;usize retries;cleanup{}}
fn Problem failed(u32 stage,u32 code,usize module_index,usize start,usize end,usize first,usize stop,usize expected,usize actual){return Problem{.stage,.code,.module_index,.start,.end,.first,.stop,.expected,.actual};}
pub fn Result<ir::Program,Problem> range(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    let bound=switch(binding::bind(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(failed(2,e.code,e.module_index,e.start,e.end,first,stop,0,0));}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(e):{return .err(failed(3,e.code,e.module_index,e.start,e.end,first,stop,0,0));}};
    let typed=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(e):{return .err(failed(4,e.code,e.module_index,e.start,e.end,first,stop,e.expected,e.actual));}};
    let moved=switch(moves::build(sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{return .err(failed(5,e.code,e.module_index,e.start,e.end,first,stop,e.expected,e.actual));}};
    if(moves::count(ref moved)!=0){let e=moves::at(ref moved,0);return .err(failed(5,e.code,e.module_index,e.start,e.end,first,stop,0,0));}
    return switch(lower::lower(sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(6,e.code,e.module_index,e.start,e.end,first,stop,0,0))}};
}
fn bool budget(Problem p){return (p.stage>=2 && p.stage<=5 && p.code==10) || p.stage==3 && p.code==4503 || p.stage==6 && p.code==1002;}
fn Problem append_range(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop,usize depth,ref mut v::Vec<ir::Program> programs,ref mut v::Vec<Range> ranges,ref mut usize retries){
    switch(range(sources,metadata,first,stop)){
        case .ok(p):{
            if(!v::push<ir::Program>(programs,p).is_ok || !v::push<Range>(ranges,Range{.first,.stop}).is_ok){return failed(1,11,first,0,0,first,stop,0,0);}
            return failed(0,0,0,0,0,first,stop,0,0);
        }
        case .err(e):{
            // Only capacity failures can be retried. Single-source failures retain
            // the exact failing stage and location; no successful prefix escapes.
            if(!budget(e) || stop-first<=1 || depth>=8){return e;};
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
