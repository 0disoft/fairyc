module module_unit_cleanup_mir_lower;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_policy as policy;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_cleanup_liveness as liveness;
import module_unit_value_model as m;
import module_unit_move_model as move;
import module_unit_moves as moves;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import module_signatures as signatures;
import effect_declarations as effects;
import std::vec as v;

fn ir::Problem fail(u32 code,usize module_index,usize start,usize end){return ir::Problem{.code,.module_index,.start,.end};}
// These are generated cleanup recipe fields, not source AST fields. They are
// a subset of the imported unit interface's 2048 fields; project IR has its
// separate 8192-field bound after units are linked.
fn usize recipe_field_limit(){return 2048;}
// Private per-lowering index: Interfaces keeps the first shape, but cleanup
// recipes have always selected the last. Duplicate cleanup frames are retained
// as a sentinel, not rejected until that type's ordered fields were processed.
struct Selection {usize shape;usize frame;}
resource Lookup {v::Vec<Selection> rows;cleanup{}}
fn Result<Lookup,ir::Problem> index_recipes(usize count,ref v::Vec<interfaces::Shape> shapes,ref v::Vec<ir::Frame> frames,usize module_index,ref mut usize work){
    if(count>4096){return .err(fail(1002,module_index,0,0));}
    var v::Vec<Selection> rows=v::empty<Selection>();if(!v::reserve<Selection>(ref mut rows,count+1).is_ok){return .err(fail(1003,module_index,0,0));}var usize i=0;
    while(i<=count){if(!m::charge(work,1)){return .err(fail(1002,module_index,0,0));}if(!v::push<Selection>(ref mut rows,Selection{.shape=0,.frame=0}).is_ok){return .err(fail(1003,module_index,0,0));}i+=1;}
    i=0;while(i<v::len<interfaces::Shape>(shapes)){if(!m::charge(work,1)){return .err(fail(1002,module_index,0,0));}let s=*v::at<interfaces::Shape>(shapes,i);if(s.type!=0 && s.type<=count){let slot=v::at_mut<Selection>(ref mut rows,s.type);slot.shape=i+1;}i+=1;}
    i=0;while(i<v::len<ir::Frame>(frames)){if(!m::charge(work,1)){return .err(fail(1002,module_index,0,0));}let f=*v::at<ir::Frame>(frames,i);if(f.kind==2 && f.receiver_type!=0 && f.receiver_type<=count){let slot=v::at_mut<Selection>(ref mut rows,f.receiver_type);slot.frame=if(slot.frame==0){i+1}else{v::len<ir::Frame>(frames)+1};}i+=1;}
    return .ok(Lookup{.rows});
}
fn Result<usize,ir::Problem> recipe_frame(ref Lookup index,usize type,usize frame_count,usize module_index,usize start,usize end,ref mut usize work){
    if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let frame=v::at<Selection>(ref index.rows,type).frame;
    return if(frame==0 || frame>frame_count){.err(fail(1001,module_index,start,end))}else{.ok(frame)};
}
// Signatures also describe resource types whose bodies were never selected.
// They have no storage or destructor to materialize. Reject a
// missing shape whenever a retained body actually carries that owner.
fn Result<m::Expression,ir::Problem> expression(ref m::Values body,usize id,ref mut usize work){var usize low=1;var usize high=m::expression_count(body)+1;while(low<high){if(!m::charge(work,1)){return .err(fail(1002,m::frame(body).module_index,0,0));}let mid=low+(high-low)/2;if(m::expression_at(body,mid).source_index<id){low=mid+1;}else{high=mid;}}if(low>m::expression_count(body) || m::expression_at(body,low).source_index!=id){return .err(fail(1001,m::frame(body).module_index,0,0));}return .ok(m::expression_at(body,low));}
fn Result<usize,ir::Problem> operand(ref values::Plan typed,move::Fact request,ref mut v::Vec<ir::Operand> operands,ref mut usize work){
    let body=values::at(typed,request.frame-1);let frame=m::frame(body);var usize start=0;var usize end=0;let u32 kind=if(request.binding!=0){1}else{2};let id=if(request.binding!=0){request.binding}else{request.expression};
    if((request.binding==0)==(request.expression==0) || id==0){return .err(fail(1001,frame.module_index,request.start,request.end));}
    if(kind==1){if(id>m::local_count(body)){return .err(fail(1001,frame.module_index,request.start,request.end));}let n=m::local_at(body,id);if(n.type!=request.type || n.kind==10){return .err(fail(1001,frame.module_index,request.start,request.end));}start=n.start;end=n.end;}
    else{let e=switch(expression(body,id,work)){case .ok(e):{e}case .err(p):{return .err(p);}};if(e.type!=request.type || e.state!=1){return .err(fail(1001,frame.module_index,request.start,request.end));}start=e.start;end=e.end;}
    var usize i=0;while(i<v::len<ir::Operand>(operands)){if(!m::charge(work,1)){return .err(fail(1002,frame.module_index,start,end));}let previous=*v::at<ir::Operand>(operands,i);if(previous.frame==request.frame && previous.kind==kind && previous.id==id){if(previous.type!=request.type || previous.start!=start || previous.end!=end){return .err(fail(1001,frame.module_index,start,end));}return .ok(i+1);}i+=1;}
    if(v::len<ir::Operand>(operands)>=16384 || !m::charge(work,1)){return .err(fail(1002,frame.module_index,start,end));}return if(v::push<ir::Operand>(operands,ir::Operand{.frame=request.frame,.kind,.id,.type=request.type,.start,.end}).is_ok){.ok(v::len<ir::Operand>(operands))}else{.err(fail(1003,frame.module_index,start,end))};
}
pub fn Result<ir::Program,ir::Problem> lower(ref v::Vec<syntax::Parsed> sources,ref values::Plan typed,ref interfaces::Interfaces imported,ref moves::Plan moved){
    if(moves::count(moved)!=0){let e=moves::at(moved,0);return .err(fail(e.code,e.module_index,e.start,e.end));}
    let checked=switch(policy::build(sources,typed,imported,moved)){case .ok(p):{p}case .err(p):{return .err(fail(p.code,p.module_index,p.start,p.end));}};
    if(policy::count(ref checked)!=0){let e=policy::at(ref checked,0);return .err(fail(e.code,e.module_index,e.start,e.end));}return lower_policy(sources,typed,imported,moved,policy::work(ref checked));
}
// Only this source-bound owner can skip policy construction. Raw callers keep
// the original checked entry above; the implementation entry stays private.
pub fn Result<ir::Program,ir::Problem> lower_checked(ref v::Vec<syntax::Parsed> sources,ref checking::Plan checked){var usize validation=0;let valid=checking::validate_sources(checked,sources,ref mut validation);if(valid.code!=0){return .err(fail(valid.code,valid.module_index,valid.start,valid.end));}return lower_policy(sources,checking::checked_values(checked),checking::checked_interfaces(checked),checking::checked_moves(checked),policy::work(checking::cleanup_policy(checked)));}
fn Result<ir::Program,ir::Problem> lower_policy(ref v::Vec<syntax::Parsed> sources,ref values::Plan typed,ref interfaces::Interfaces imported,ref moves::Plan moved,usize prior_work){var usize work=prior_work;var bool aggregate_types=false;
    let used_types=switch(liveness::build(typed,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();var v::Vec<ir::Glue> glues=v::empty<ir::Glue>();var v::Vec<ir::Field> fields=v::empty<ir::Field>();var v::Vec<ir::Operand> operands=v::empty<ir::Operand>();var v::Vec<ir::Drop> drops=v::empty<ir::Drop>();var usize i=1;
    while(i<=values::type_count(typed)){if(!m::charge(ref mut work,1)){return .err(fail(1002,0,0,0));}let t=values::type_at(typed,i);aggregate_types=aggregate_types || t.kind==20 || t.kind==21 || t.kind==22;if(!v::push<types::Type>(ref mut graph,t).is_ok){return .err(fail(1003,0,0,0));}i+=1;}
    i=0;while(i<values::count(typed)){let body=values::at(typed,i);let f=m::frame(body);if(f.module_index>=v::len<syntax::Parsed>(sources)){return .err(fail(1001,0,0,0));}let source=v::at<syntax::Parsed>(sources,f.module_index);if(f.declaration==0 || f.declaration>syntax::declaration_count(source)){return .err(fail(1001,f.module_index,0,0));}let authority=effects::read(source,f.declaration-1);if(authority.code!=0){return .err(fail(authority.code,f.module_index,authority.start,authority.end));}
        var usize receiver=0;var usize receiver_type=0;if(f.kind==2){var usize j=1;while(j<=m::local_count(body)){if(!m::charge(ref mut work,1)){return .err(fail(1002,f.module_index,0,0));}let n=m::local_at(body,j);if(n.kind==10){if(receiver!=0){return .err(fail(1001,f.module_index,n.start,n.end));}receiver=j;receiver_type=n.type;}j+=1;}}
        if(!v::push<ir::Frame>(ref mut frames,ir::Frame{.kind=f.kind,.context=f.context,.module_index=f.module_index,.declaration=f.declaration,.result=f.result,.effects=authority.bits,.receiver,.receiver_type}).is_ok){return .err(fail(1003,f.module_index,0,0));}i+=1;
    }
    var Lookup index=Lookup{.rows=v::empty<Selection>()};
    i=1;while(i<=values::type_count(typed)){let t=values::type_at(typed,i);var ir::Glue g=ir::Glue{.kind=if(t.kind==3 || t.kind==13 || t.kind==21){1}else if(t.kind==5){2}else if(t.kind==6){3}else{0},.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0};
        if(g.kind==1){if(v::len<Selection>(ref index.rows)==0){index=switch(index_recipes(values::type_count(typed),interfaces::shape_records(imported),ref frames,t.module_index,ref mut work)){case .ok(p):{p}case .err(p):{return .err(p);}};}if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,0,0));}let found=v::at<Selection>(ref index.rows,i).shape;var usize j=0;if(found==0){if((t.kind!=13 && t.kind!=3 && t.kind!=21) || *v::at<u8>(ref used_types,i)!=0){return .err(fail(if(work>=1048576){1002}else{1001},t.module_index,0,0));}if(!v::push<ir::Glue>(ref mut glues,g).is_ok){return .err(fail(1003,t.module_index,0,0));}i+=1;continue;}let s=interfaces::shape_at(imported,found-1);g.effects=s.cleanup_effects;g.abilities=s.abilities;g.nonempty=s.cleanup_nonempty;g.field_start=v::len<ir::Field>(ref fields);g.field_count=s.field_count;
            if(s.module_index!=t.module_index || s.declaration!=t.declaration){return .err(fail(1001,t.module_index,s.start,s.end));}j=0;while(j<s.field_count){let f=interfaces::field_at(imported,s.field_start+j);if(!m::charge(ref mut work,1) || v::len<ir::Field>(ref fields)>=recipe_field_limit()){return .err(fail(1002,t.module_index,s.start,s.end));}if(!v::push<ir::Field>(ref mut fields,ir::Field{.owner=i,.ordinal=j,.type=f.type}).is_ok){return .err(fail(1003,t.module_index,s.start,s.end));}j+=1;}
            if(g.nonempty){g.frame=switch(recipe_frame(ref index,i,v::len<ir::Frame>(ref frames),t.module_index,s.start,s.end,ref mut work)){case .ok(frame):{frame}case .err(p):{return .err(p);}};}
        }if(!v::push<ir::Glue>(ref mut glues,g).is_ok){return .err(fail(1003,t.module_index,0,0));}i+=1;
    }
    i=0;while(i<moves::cleanup_count(moved)){let request=moves::cleanup_at(moved,i);if(request.kind!=5 || request.frame==0 || request.frame>v::len<ir::Frame>(ref frames)){return .err(fail(1001,0,request.start,request.end));}let value=switch(operand(typed,request,ref mut operands,ref mut work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(!m::charge(ref mut work,1)){return .err(fail(1002,0,request.start,request.end));}if(!v::push<ir::Drop>(ref mut drops,ir::Drop{.operand=value,.exit=request.exit,.start=request.start,.end=request.end}).is_ok){return .err(fail(1003,0,request.start,request.end));}i+=1;}
    return if(aggregate_types){ir::assemble_aggregates(graph,frames,glues,fields,operands,drops,work)}else{ir::assemble(graph,frames,glues,fields,operands,drops,work)};
}

pub fn Result<ir::Program,ir::Problem> lower_checked_catalog(ref v::Vec<syntax::Parsed> sources,ref checking::Plan checked,ref signatures::Program metadata){var usize validation=0;let valid=checking::validate_sources_catalog(checked,sources,metadata,ref mut validation);if(valid.code!=0){return .err(fail(valid.code,valid.module_index,valid.start,valid.end));}return lower_policy(sources,checking::checked_values(checked),checking::checked_interfaces(checked),checking::checked_moves(checked),policy::work(checking::cleanup_policy(checked)));}
