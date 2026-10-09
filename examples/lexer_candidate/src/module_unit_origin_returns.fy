module module_unit_origin_returns;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_origin_flow_model as g;
import module_unit_origin_flow_control as control;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;
import module_signatures as signatures;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import module_unit_origin_provenance as p;

pub struct ReturnRange {usize start;usize count;}
pub resource Plan {inputs::Inputs input;v::Vec<p::Row> allocations;v::Vec<sets::Entry> entries;v::Vec<usize> members;v::Vec<g::Value> summaries;v::Vec<g::Returned> returns;v::Vec<g::Unknown> unknowns;v::Vec<ReturnRange> return_ranges;usize rounds;usize work;cleanup{}}
pub fn usize count(ref Plan p){return v::len<g::Value>(ref p.summaries);}
pub fn usize summary(ref Plan p,usize frame){return v::at<g::Value>(ref p.summaries,frame-1).origin;}
pub fn usize source_count(ref Plan p,usize frame){return sets::count(ref p.entries,ref p.members,summary(p,frame));}
// Real source leaves name input owner leaf-2; kind 11 is a located Static.
// Leaf 2 remains the abstract static handle in the set algebra.
pub fn usize source(ref Plan p,usize frame,usize ordinal){return sets::member(ref p.entries,ref p.members,summary(p,frame),ordinal);}
pub fn ref inputs::Inputs owned_inputs(ref Plan p){return ref p.input;}
pub fn usize rounds(ref Plan p){return p.rounds;}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize unknown_count(ref Plan p){return v::len<g::Unknown>(ref p.unknowns);}
pub fn g::Unknown unknown_at(ref Plan p,usize index){return *v::at<g::Unknown>(ref p.unknowns,index);}
pub fn usize return_count(ref Plan p,usize frame){return v::at<ReturnRange>(ref p.return_ranges,frame-1).count;}
pub fn g::Returned return_at(ref Plan p,usize frame,usize ordinal){let range=*v::at<ReturnRange>(ref p.return_ranges,frame-1);assert(ordinal<range.count);return *v::at<g::Returned>(ref p.returns,range.start+ordinal);}
pub fn usize return_source_count(ref Plan p,g::Returned row){return sets::count(ref p.entries,ref p.members,row.origin);}
pub fn usize return_source(ref Plan p,g::Returned row,usize ordinal){return sets::member(ref p.entries,ref p.members,row.origin,ordinal);}
fn m::Problem validate(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref inputs::Inputs input,ref values::Plan primary,ref mut usize work){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return m::Problem{.code=12,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};}var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!m::charge(work,1)){return m::Problem{.code=10,.module_index=i,.start=0,.end=0,.expected=0,.actual=0};}if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return m::Problem{.code=12,.module_index=i,.start=0,.end=0,.expected=0,.actual=0};}i+=1;}
    var usize id=1;while(id<=inputs::frame_count(input)){let f=inputs::frame(input,id);let b=inputs::body(input,primary,id);let source=v::at<syntax::Parsed>(sources,f.value.module_index);i=1;while(i<=m::expression_count(b)){let e=m::expression_at(b,i);if(!m::charge(work,1)){return m::fail(f.value,10,e.start,e.end,0,0);}if(e.source_index==0 || e.source_index>syntax::expression_count(source)){return m::fail(f.value,12,e.start,e.end,0,0);}let n=syntax::expression_at(source,e.source_index);if(e.kind!=n.kind || e.start!=n.start || e.end!=n.end || e.left!=n.left || e.right!=n.right){return m::fail(f.value,12,e.start,e.end,0,0);}if(e.binding!=0){if(e.binding>m::local_count(b)){return m::fail(f.value,12,e.start,e.end,0,0);}let l=m::local_at(b,e.binding);if(l.kind!=10 && !bytes::equal(syntax::expression_text(source,e.source_index),syntax::source_view(source)[l.start..l.end])){return m::fail(f.value,12,e.start,e.end,0,0);}}i+=1;}id+=1;}return m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary){return build_budgeted(sources,metadata,primary,0);}
pub fn Result<Plan,m::Problem> build_budgeted(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary,usize prior_work){
    if(prior_work>=1048576){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}let input=switch(inputs::build(sources,metadata,primary)){case .ok(p):{p}case .err(p):{return .err(p);}};var usize work=prior_work;if(!m::charge(ref mut work,inputs::work(ref input))){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}let valid=validate(sources,metadata,ref input,primary,ref mut work);if(valid.code!=0){return .err(valid);}var v::Vec<sets::Entry> entries=v::empty<sets::Entry>();var v::Vec<usize> members=v::empty<usize>();var v::Vec<p::Row> allocations=p::empty();var v::Vec<g::Value> summaries=v::empty<g::Value>();var usize id=1;
    while(id<=inputs::frame_count(ref input)){let f=inputs::frame(ref input,id);let leaves=switch(inputs::borrow_leaves(ref input,primary,id,f.value.result,ref mut work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(leaves>1){return .err(m::fail(f.value,2,0,0,0,0));}let bool borrower=leaves==1;if(!v::push<g::Value>(ref mut summaries,g::Value{.origin=if(borrower){1}else{0},.allocation=0}).is_ok){return .err(m::fail(f.value,11,0,0,0,0));}id+=1;}
    var v::Vec<g::Returned> returns=v::empty<g::Returned>();var v::Vec<g::Unknown> unknowns=v::empty<g::Unknown>();var v::Vec<ReturnRange> return_ranges=v::empty<ReturnRange>();var usize rounds=0;var bool changed=true;while(changed && rounds<64){changed=false;rounds+=1;if(!m::charge(ref mut work,v::len<g::Unknown>(ref unknowns)+v::len<g::Returned>(ref returns)+v::len<ReturnRange>(ref return_ranges)+1)){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}while(v::len<g::Unknown>(ref unknowns)>0){discard v::pop<g::Unknown>(ref mut unknowns);}while(v::len<g::Returned>(ref returns)>0){discard v::pop<g::Returned>(ref mut returns);}while(v::len<ReturnRange>(ref return_ranges)>0){discard v::pop<ReturnRange>(ref mut return_ranges);}id=1;while(id<=inputs::frame_count(ref input)){let usize return_start=v::len<g::Returned>(ref returns);let f=inputs::frame(ref input,id);let body=inputs::body(ref input,primary,id);let leaves=switch(inputs::borrow_leaves(ref input,primary,id,f.value.result,ref mut work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(leaves>1){return .err(m::fail(f.value,2,0,0,0,0));}let bool borrower=leaves==1;if(borrower || m::region_count(body)!=0 || m::placement_count(body)!=0){var g::Context c=g::Context{.frame=id,.value=f.value,.borrower,.tagged_return=f.tagged_return,.sites=0,.returned=0,.allocation=0,.seen=false,.work,.loop_depth=0,.probes=0};var v::Vec<g::Value> env=v::empty<g::Value>();var usize local=1;
                while(local<=f.local_count){if(!m::charge(ref mut c.work,1)){return .err(g::fail(ref c,10,0,0));}let owner=inputs::owner(ref input,f.local_start+local);let usize origin=if((owner.kind==1 || owner.kind==13) && owner.borrowed){switch(sets::owner(f.local_start+local)){case .ok(n):{n}case .err(code):{return .err(g::fail(ref c,code,owner.start,owner.end));}}}else{0};if(!v::push<g::Value>(ref mut env,g::Value{.origin,.allocation=0}).is_ok){return .err(g::fail(ref c,11,owner.start,owner.end));}local+=1;}
                var v::Vec<g::Edge> edges=v::empty<g::Edge>();let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,f.value.module_index),f.value.declaration-1);let evaluated=switch(control::block(sources,ref input,primary,ref summaries,ref mut entries,ref mut members,ref mut allocations,ref mut returns,ref mut unknowns,ref mut env,ref mut edges,ref mut c,if(f.value.kind==2){d.cleanup_body}else{d.body},0)){case .ok(r):{r}case .err(p):{return .err(p);}};if(v::len<g::Edge>(ref edges)!=0 || (evaluated.outcomes & 12)!=0){return .err(g::fail(ref c,12,d.start,d.end));}let result=g::Value{.origin=if(!borrower){0}else if(c.seen){c.returned}else{1},.allocation=if(borrower && c.seen){c.allocation}else{0}};let prior=*v::at<g::Value>(ref summaries,id-1);if(prior.origin!=result.origin || prior.allocation!=result.allocation){changed=true;let slot=v::at_mut<g::Value>(ref mut summaries,id-1);*slot=result;}work=c.work;
            }if(!m::charge(ref mut work,1)){return .err(m::fail(f.value,10,0,0,0,0));}if(!v::push<ReturnRange>(ref mut return_ranges,ReturnRange{.start=return_start,.count=v::len<g::Returned>(ref returns)-return_start}).is_ok){return .err(m::fail(f.value,11,0,0,0,0));}id+=1;
        }}if(changed){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}return .ok(Plan{.input,.allocations,.entries,.members,.summaries,.returns,.unknowns,.return_ranges,.rounds,.work});
}

pub fn Result<usize,u32> return_allocation(ref Plan p,g::Returned row,usize owner,ref mut usize work){return p::lookup(ref p.allocations,row.allocation,owner,work);}
