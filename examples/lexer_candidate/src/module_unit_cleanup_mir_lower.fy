module module_unit_cleanup_mir_lower;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_policy as policy;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_move_model as move;
import module_unit_moves as moves;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import effect_declarations as effects;
import std::vec as v;

fn ir::Problem fail(u32 code,usize module_index,usize start,usize end){return ir::Problem{.code,.module_index,.start,.end};}
// Ordinary SDK signatures also describe Vec instantiations whose bodies were
// never selected. They have no storage or destructor to materialize. Reject a
// missing shape whenever a retained body actually carries that owner.
fn bool carries(ref values::Plan p,usize actual,usize wanted,usize depth,ref mut usize work){if(depth>=64 || !m::charge(work,1)){return true;}if(actual==wanted){return true;}if(actual==0){return false;}let t=values::type_at(p,actual);return (t.kind>=5 && t.kind<=8 || t.kind==12 && t.first!=0 || t.kind==16) && (carries(p,t.first,wanted,depth+1,work) || t.kind==6 && carries(p,t.second,wanted,depth+1,work));}
fn bool used(ref values::Plan p,usize type,ref mut usize work){var usize i=0;while(i<values::count(p)){let body=values::at(p,i);if(carries(p,m::frame(body).result,type,0,work)){return true;}var usize j=1;while(j<=m::local_count(body)){if(carries(p,m::local_at(body,j).type,type,0,work)){return true;}j+=1;}j=1;while(j<=m::expression_count(body)){let e=m::expression_at(body,j);if(e.state==1 && carries(p,e.type,type,0,work)){return true;}j+=1;}i+=1;}return false;}
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
    if(policy::count(ref checked)!=0){let e=policy::at(ref checked,0);return .err(fail(e.code,e.module_index,e.start,e.end));}var usize work=policy::work(ref checked);
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();var v::Vec<ir::Glue> glues=v::empty<ir::Glue>();var v::Vec<ir::Field> fields=v::empty<ir::Field>();var v::Vec<ir::Operand> operands=v::empty<ir::Operand>();var v::Vec<ir::Drop> drops=v::empty<ir::Drop>();var usize i=1;
    while(i<=values::type_count(typed)){if(!m::charge(ref mut work,1)){return .err(fail(1002,0,0,0));}let t=values::type_at(typed,i);if(!v::push<types::Type>(ref mut graph,t).is_ok){return .err(fail(1003,0,0,0));}i+=1;}
    i=0;while(i<values::count(typed)){let body=values::at(typed,i);let f=m::frame(body);if(f.module_index>=v::len<syntax::Parsed>(sources)){return .err(fail(1001,0,0,0));}let source=v::at<syntax::Parsed>(sources,f.module_index);if(f.declaration==0 || f.declaration>syntax::declaration_count(source)){return .err(fail(1001,f.module_index,0,0));}let authority=effects::read(source,f.declaration-1);if(authority.code!=0){return .err(fail(authority.code,f.module_index,authority.start,authority.end));}
        var usize receiver=0;var usize receiver_type=0;if(f.kind==2){var usize j=1;while(j<=m::local_count(body)){if(!m::charge(ref mut work,1)){return .err(fail(1002,f.module_index,0,0));}let n=m::local_at(body,j);if(n.kind==10){if(receiver!=0){return .err(fail(1001,f.module_index,n.start,n.end));}receiver=j;receiver_type=n.type;}j+=1;}}
        if(!v::push<ir::Frame>(ref mut frames,ir::Frame{.kind=f.kind,.context=f.context,.module_index=f.module_index,.declaration=f.declaration,.result=f.result,.effects=authority.bits,.receiver,.receiver_type}).is_ok){return .err(fail(1003,f.module_index,0,0));}i+=1;
    }
    i=1;while(i<=values::type_count(typed)){let t=values::type_at(typed,i);var ir::Glue g=ir::Glue{.kind=if(t.kind==3 || t.kind==13){1}else if(t.kind==5){2}else if(t.kind==6){3}else{0},.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0};
        if(g.kind==1){var usize found=0;var usize j=0;while(j<interfaces::shape_count(imported)){if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,0,0));}let s=interfaces::shape_at(imported,j);if(s.type==i){found=j+1;}j+=1;}if(found==0){if(t.kind!=13 || used(typed,i,ref mut work)){return .err(fail(if(work>=1048576){1002}else{1001},t.module_index,0,0));}if(!v::push<ir::Glue>(ref mut glues,g).is_ok){return .err(fail(1003,t.module_index,0,0));}i+=1;continue;}let s=interfaces::shape_at(imported,found-1);g.effects=s.cleanup_effects;g.abilities=s.abilities;g.nonempty=s.cleanup_nonempty;g.field_start=v::len<ir::Field>(ref fields);g.field_count=s.field_count;
            if(s.module_index!=t.module_index || s.declaration!=t.declaration){return .err(fail(1001,t.module_index,s.start,s.end));}j=0;while(j<s.field_count){let f=interfaces::field_at(imported,s.field_start+j);if(!m::charge(ref mut work,1) || v::len<ir::Field>(ref fields)>=1024){return .err(fail(1002,t.module_index,s.start,s.end));}if(!v::push<ir::Field>(ref mut fields,ir::Field{.owner=i,.ordinal=j,.type=f.type}).is_ok){return .err(fail(1003,t.module_index,s.start,s.end));}j+=1;}
            if(g.nonempty){j=0;while(j<v::len<ir::Frame>(ref frames)){if(!m::charge(ref mut work,1)){return .err(fail(1002,t.module_index,s.start,s.end));}let f=*v::at<ir::Frame>(ref frames,j);if(f.kind==2 && f.receiver_type==i){if(g.frame!=0){return .err(fail(1001,t.module_index,s.start,s.end));}g.frame=j+1;}j+=1;}if(g.frame==0){return .err(fail(1001,t.module_index,s.start,s.end));}}
        }if(!v::push<ir::Glue>(ref mut glues,g).is_ok){return .err(fail(1003,t.module_index,0,0));}i+=1;
    }
    i=0;while(i<moves::cleanup_count(moved)){let request=moves::cleanup_at(moved,i);if(request.kind!=5 || request.frame==0 || request.frame>v::len<ir::Frame>(ref frames)){return .err(fail(1001,0,request.start,request.end));}let value=switch(operand(typed,request,ref mut operands,ref mut work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(!m::charge(ref mut work,1)){return .err(fail(1002,0,request.start,request.end));}if(!v::push<ir::Drop>(ref mut drops,ir::Drop{.operand=value,.exit=request.exit,.start=request.start,.end=request.end}).is_ok){return .err(fail(1003,0,request.start,request.end));}i+=1;}
    return ir::assemble(graph,frames,glues,fields,operands,drops,work);
}
