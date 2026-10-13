module module_project_identified_compile;
import module_project_compile as compile;
import module_units as grouping;
import module_identified_ir as identified;
import module_signatures as signatures;
import generic_source_plan as source;
import syntax as syntax;
import std::vec as v;

// Every retained unit owns its checked IR and its original-source identities.
// No resource field is moved out and no borrowed source/planner owner is stored.
pub resource Plan{v::Vec<identified::Program> units;v::Vec<compile::Range> ranges;usize files;usize retries;cleanup{}}
pub fn Plan empty(){return Plan{.units=v::empty<identified::Program>(),.ranges=v::empty<compile::Range>(),.files=0,.retries=0};}
pub fn usize count(ref Plan p){return v::len<identified::Program>(ref p.units);}
pub fn usize file_count(ref Plan p){return p.files;}
pub fn usize retries(ref Plan p){return p.retries;}
pub fn ref identified::Program at(ref Plan p,usize index){assert(index<count(p));return v::at<identified::Program>(ref p.units,index);}
pub fn compile::Range range_at(ref Plan p,usize index){assert(index<count(p));return *v::at<compile::Range>(ref p.ranges,index);}
fn compile::Problem failed(u32 code,usize first,usize stop){return compile::Problem{.stage=1,.code,.module_index=first,.start=0,.end=0,.first,.stop,.expected=0,.actual=0,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0};}
fn compile::Problem append(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref source::Plan original,usize first,usize stop,usize depth,ref mut v::Vec<identified::Program> units,ref mut v::Vec<compile::Range> ranges,ref mut usize retries,ref mut usize work){
    switch(compile::range_identified(sources,metadata,original,first,stop,work)){
        case .ok(p):{
            if(!v::push<identified::Program>(units,p).is_ok || !v::push<compile::Range>(ranges,compile::Range{.first,.stop}).is_ok){return failed(11,first,stop);}
            var compile::Problem result=failed(0,first,stop);result.stage=0;return result;
        }
        case .err(e):{
            // Checking capacity can split. Attachment failures (stage 8) never
            // retry with a reset meter or expose previously collected units.
            if(!compile::capacity_failure(e) || stop-first<=1 || depth>=8){return e;};
            *retries=*retries+1;let middle=first+(stop-first)/2;
            let left=append(sources,metadata,original,first,middle,depth+1,units,ranges,retries,work);if(left.code!=0){return left;}
            return append(sources,metadata,original,middle,stop,depth+1,units,ranges,retries,work);
        }
    };return failed(12,first,stop);
}
pub fn Result<Plan,compile::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref source::Plan original,ref mut usize work){
    let grouped=switch(grouping::build(sources)){case .ok(p):{p}case .err(e):{var compile::Problem result=failed(e.code,0,v::len<syntax::Parsed>(sources));result.module_index=e.module_index;result.start=e.start;result.end=e.end;return .err(result);}};
    var v::Vec<identified::Program> units=v::empty<identified::Program>();var v::Vec<compile::Range> ranges=v::empty<compile::Range>();var usize retries=0;var usize next=0;var usize i=0;
    while(i<grouping::count(ref grouped)){let unit=grouping::at(ref grouped,i);assert(unit.first==next && unit.first<unit.stop);let e=append(sources,metadata,original,unit.first,unit.stop,0,ref mut units,ref mut ranges,ref mut retries,work);if(e.code!=0){return .err(e);}next=unit.stop;i+=1;}
    assert(next==grouping::file_count(ref grouped) && v::len<identified::Program>(ref units)==v::len<compile::Range>(ref ranges));return .ok(Plan{.units,.ranges,.files=next,.retries});
}
