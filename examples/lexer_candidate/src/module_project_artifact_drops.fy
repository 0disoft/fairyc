module module_project_artifact_drops;
import module_project_artifact_calls as calls;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_moves as moves;
import module_unit_move_model as move;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_identity_artifact_model as identities;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::vec as v;

// Source-derived cleanup requests are compared without constructing new MIR.
// Per-frame ordering matters; the interleaving of independent function frames
// does not. Recipes, full metadata and whole-project coverage remain separate.
resource Index {v::Vec<usize> heads;v::Vec<usize> next;cleanup{}}
fn Result<Index,u32> index(ref clean::Program p,ref mut usize work){var v::Vec<usize> heads=v::empty<usize>();var v::Vec<usize> tails=v::empty<usize>();var v::Vec<usize> next=v::empty<usize>();var usize i=0;
    while(i<clean::frame_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut heads,0).is_ok || !v::push<usize>(ref mut tails,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<clean::drop_count(p)){if(!budget::charge(work,1)){return .err(1002);}let d=clean::drop_at(p,i);if(d.operand==0 || d.operand>clean::operand_count(p)){return .err(1001);}let o=clean::operand_at(p,d.operand);if(o.frame==0 || o.frame>clean::frame_count(p)){return .err(1001);}if(!v::push<usize>(ref mut next,0).is_ok){return .err(1003);}let tail=*v::at<usize>(ref tails,o.frame-1);if(tail==0){let cell=v::at_mut<usize>(ref mut heads,o.frame-1);*cell=i+1;}else{let cell=v::at_mut<usize>(ref mut next,tail-1);*cell=i+1;}let cell=v::at_mut<usize>(ref mut tails,o.frame-1);*cell=i+1;i+=1;}return .ok(Index{.heads,.next});
}
fn Result<m::Expression,u32> expression(ref m::Values p,usize source_index,ref mut usize work){var usize low=1;var usize high=m::expression_count(p)+1;while(low<high){if(!budget::charge(work,1)){return .err(1002);}let mid=low+(high-low)/2;if(m::expression_at(p,mid).source_index<source_index){low=mid+1;}else{high=mid;}}if(!budget::charge(work,1)){return .err(1002);}if(low>m::expression_count(p) || m::expression_at(p,low).source_index!=source_index){return .err(1001);}return .ok(m::expression_at(p,low));}
fn Result<clean::Operand,u32> operand(ref values::Plan typed,ref frames::Mapping map,move::Fact request,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(request.kind!=5 || request.frame==0 || request.frame>values::count(typed) || (request.binding==0)==(request.expression==0) || request.type==0){return .err(1001);}let body=values::at(typed,request.frame-1);let u32 kind=if(request.binding!=0){1}else{2};let id=if(request.binding!=0){request.binding}else{request.expression};var usize start=0;var usize end=0;
    if(kind==1){if(id>m::local_count(body)){return .err(1001);}let local=m::local_at(body,id);if(local.kind==10 || local.type!=request.type){return .err(1001);}start=local.start;end=local.end;}
    else{let node=switch(expression(body,id,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(node.type!=request.type || node.state!=1){return .err(1001);}start=node.start;end=node.end;}
    let frame=frames::mapped(map,request.frame);let type=types::mapped(frames::type_mapping(map),request.type);if(frame==0 || type==0){return .err(1001);}return .ok(clean::Operand{.frame,.kind,.id,.type,.start,.end});
}
fn bool same(clean::Operand a,clean::Operand b){return a.frame==b.frame && a.kind==b.kind && a.id==b.id && a.type==b.type && a.start==b.start && a.end==b.end;}
pub fn Result<frames::Mapping,u32> match(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let map=switch(calls::match(program,ids,checked,plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let saved=ir::cleanup_ir(program);let indexed=switch(index(saved,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var v::Vec<usize> cursors=v::empty<usize>();var usize i=0;while(i<clean::frame_count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut cursors,*v::at<usize>(ref indexed.heads,i)).is_ok){return .err(1003);}i+=1;}var v::Vec<u8> seen=v::empty<u8>();i=0;while(i<clean::operand_count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(1003);}i+=1;}let typed=checking::checked_values(checked);let moved=checking::checked_moves(checked);i=0;
    while(i<moves::cleanup_count(moved)){let request=moves::cleanup_at(moved,i);let expected=switch(operand(typed,ref map,request,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!budget::charge(work,1)){return .err(1002);}let at=*v::at<usize>(ref cursors,expected.frame-1);if(at==0){return .err(1001);}let drop=clean::drop_at(saved,at-1);let actual=clean::operand_at(saved,drop.operand);if(!same(expected,actual) || drop.exit!=request.exit || drop.start!=request.start || drop.end!=request.end){return .err(1001);}{let cell=v::at_mut<u8>(ref mut seen,drop.operand-1);*cell=1;}{let cell=v::at_mut<usize>(ref mut cursors,expected.frame-1);*cell=*v::at<usize>(ref indexed.next,at-1);}i+=1;}
    i=1;while(i<=clean::frame_count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(frames::used(ref map,i) && *v::at<usize>(ref cursors,i-1)!=0){return .err(1001);}i+=1;}
    i=1;while(i<=clean::operand_count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(frames::used(ref map,clean::operand_at(saved,i).frame) && *v::at<u8>(ref seen,i-1)==0){return .err(1001);}i+=1;}return .ok(map);
}
