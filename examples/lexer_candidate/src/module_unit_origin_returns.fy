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

pub resource Plan {inputs::Inputs input;v::Vec<sets::Entry> entries;v::Vec<usize> members;v::Vec<usize> summaries;usize rounds;usize work;cleanup{}}
pub fn usize count(ref Plan p){return v::len<usize>(ref p.summaries);}
pub fn usize summary(ref Plan p,usize frame){return *v::at<usize>(ref p.summaries,frame-1);}
pub fn usize source_count(ref Plan p,usize frame){return sets::count(ref p.entries,ref p.members,summary(p,frame));}
// Source leaf 2 is Static; other known leaves name input owner leaf-2.
pub fn usize source(ref Plan p,usize frame,usize ordinal){return sets::member(ref p.entries,ref p.members,summary(p,frame),ordinal);}
pub fn ref inputs::Inputs owned_inputs(ref Plan p){return ref p.input;}
pub fn usize rounds(ref Plan p){return p.rounds;}
pub fn usize work(ref Plan p){return p.work;}
fn m::Problem validate(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref inputs::Inputs input,ref values::Plan primary,ref mut usize work){
    if(v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return m::Problem{.code=12,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};}var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!m::charge(work,1)){return m::Problem{.code=10,.module_index=i,.start=0,.end=0,.expected=0,.actual=0};}if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return m::Problem{.code=12,.module_index=i,.start=0,.end=0,.expected=0,.actual=0};}i+=1;}
    var usize id=1;while(id<=inputs::frame_count(input)){let f=inputs::frame(input,id);let b=inputs::body(input,primary,id);let source=v::at<syntax::Parsed>(sources,f.value.module_index);i=1;while(i<=m::expression_count(b)){let e=m::expression_at(b,i);if(!m::charge(work,1)){return m::fail(f.value,10,e.start,e.end,0,0);}if(e.source_index==0 || e.source_index>syntax::expression_count(source)){return m::fail(f.value,12,e.start,e.end,0,0);}let n=syntax::expression_at(source,e.source_index);if(e.kind!=n.kind || e.start!=n.start || e.end!=n.end || e.left!=n.left || e.right!=n.right){return m::fail(f.value,12,e.start,e.end,0,0);}if(e.binding!=0){if(e.binding>m::local_count(b)){return m::fail(f.value,12,e.start,e.end,0,0);}let l=m::local_at(b,e.binding);if(l.kind!=10 && !bytes::equal(syntax::expression_text(source,e.source_index),syntax::source_view(source)[l.start..l.end])){return m::fail(f.value,12,e.start,e.end,0,0);}}i+=1;}id+=1;}return m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary){return build_budgeted(sources,metadata,primary,0);}
pub fn Result<Plan,m::Problem> build_budgeted(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary,usize prior_work){
    if(prior_work>=1048576){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}let input=switch(inputs::build(sources,metadata,primary)){case .ok(p):{p}case .err(p):{return .err(p);}};var usize work=prior_work;if(!m::charge(ref mut work,inputs::work(ref input))){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}let valid=validate(sources,metadata,ref input,primary,ref mut work);if(valid.code!=0){return .err(valid);}var v::Vec<sets::Entry> entries=v::empty<sets::Entry>();var v::Vec<usize> members=v::empty<usize>();var v::Vec<usize> summaries=v::empty<usize>();var usize id=1;
    while(id<=inputs::frame_count(ref input)){let f=inputs::frame(ref input,id);let bool borrower=f.value.result!=0 && borrows::borrowed(inputs::type_at(ref input,primary,id,f.value.result));if(!v::push<usize>(ref mut summaries,if(borrower){1}else{0}).is_ok){return .err(m::fail(f.value,11,0,0,0,0));}id+=1;}
    var usize rounds=0;var bool changed=true;while(changed && rounds<64){changed=false;rounds+=1;id=1;while(id<=inputs::frame_count(ref input)){let f=inputs::frame(ref input,id);let body=inputs::body(ref input,primary,id);let bool borrower=f.value.result!=0 && borrows::borrowed(inputs::type_at(ref input,primary,id,f.value.result));if(borrower){var g::Context c=g::Context{.frame=id,.value=f.value,.borrower,.returned=0,.seen=false,.work,.loop_depth=0,.probes=0};var v::Vec<usize> env=v::empty<usize>();var usize local=1;
                while(local<=f.local_count){if(!m::charge(ref mut c.work,1)){return .err(g::fail(ref c,10,0,0));}let owner=inputs::owner(ref input,f.local_start+local);let usize origin=if(owner.kind==1 && owner.borrowed){switch(sets::owner(f.local_start+local)){case .ok(n):{n}case .err(code):{return .err(g::fail(ref c,code,owner.start,owner.end));}}}else{0};if(!v::push<usize>(ref mut env,origin).is_ok){return .err(g::fail(ref c,11,owner.start,owner.end));}local+=1;}
                var v::Vec<g::Edge> edges=v::empty<g::Edge>();let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,f.value.module_index),f.value.declaration-1);let evaluated=switch(control::block(sources,ref input,primary,ref summaries,ref mut entries,ref mut members,ref mut env,ref mut edges,ref mut c,if(f.value.kind==2){d.cleanup_body}else{d.body},0)){case .ok(r):{r}case .err(p):{return .err(p);}};if(v::len<g::Edge>(ref edges)!=0 || (evaluated.outcomes & 12)!=0){return .err(g::fail(ref c,12,d.start,d.end));}let usize result=if(c.seen){c.returned}else{1};if(*v::at<usize>(ref summaries,id-1)!=result){changed=true;let slot=v::at_mut<usize>(ref mut summaries,id-1);*slot=result;}work=c.work;
            }id+=1;
        }}if(changed){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}return .ok(Plan{.input,.entries,.members,.summaries,.rounds,.work});
}
