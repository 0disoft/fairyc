module module_project_artifact_bodies;
import module_project_artifact_frames as frames;
import module_project_artifact_types as types;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_project_body_equal as equal;
import module_identity_artifact_model as identities;
import generic_source_plan as source;
import module_signatures as signatures;
import module_program_artifact_wire as budget;
import literal_types as literals;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Compare source-checked values directly, without building replacement body IR.
// Calls, arguments, cleanup control graphs and whole-project coverage remain
// separate gates. This map alone never authorizes Complete/cache publication.
fn bool field_matches(ref ir::Program program,ref interfaces::Interfaces imported,m::Field fresh,m::Field saved,ref types::Mapping map){
    if(fresh.id==0){return saved.id==0;}if(fresh.id>interfaces::field_count(imported) || saved.id==0 || saved.id>ir::field_count(program)){return false;}
    let a=interfaces::field_at(imported,fresh.id-1);let b=ir::field_at(program,saved.id-1);
    return fresh.module_index==saved.module_index && fresh.declaration==saved.declaration && fresh.source_index==saved.source_index && types::mapped(map,fresh.owner)==saved.owner && a.kind==b.kind && types::mapped(map,a.type)==b.type && a.module_index==b.module_index && a.declaration==b.declaration && a.source_index==b.source_index && a.start==b.start && a.end==b.end && a.payload_count==b.payload_count && bytes::equal(interfaces::field_name(imported,fresh.id-1),ir::field_name(program,saved.id-1));
}
fn u32 node(ref ir::Program program,ref ir::Body saved,usize at,ref syntax::Parsed parsed,ref m::Values fresh,ref values::Plan typed,ref interfaces::Interfaces imported,ref types::Mapping map,ref mut usize text_at,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}var e=m::expression_at(fresh,at+1);if(e.source_index==0 || e.source_index>syntax::expression_count(parsed)){return 1001;}var tree=syntax::expression_at(parsed,e.source_index);let old=ir::node_at(saved,at);let folded=e.constant!=0;let bits=e.constant_bits;let u32 scalar=if(folded){values::type_at(typed,e.type).scalar}else{0};
    if(folded){e.kind=1;e.constant=0;e.constant_bits=0;e.flexible=false;tree.kind=1;}
    if(e.field.id>0 && e.field.id<=interfaces::field_count(imported) && !budget::charge(work,interfaces::field_name(imported,e.field.id-1).len+1)){return 1002;}if(!field_matches(program,imported,e.field,old.value.field,map)){return 1001;}e.field.id=old.value.field.id;e.field.owner=types::mapped(map,e.field.owner);e.type=types::mapped(map,e.type);e.expected=types::mapped(map,e.expected);e.target=types::mapped(map,e.target);e.builtin.element=types::mapped(map,e.builtin.element);
    if(e.constant!=old.value.constant || e.constant_bits!=old.value.constant_bits || !equal::expression_same(e,old.value) || !equal::syntax_same(tree,old.syntax)){return 1001;}
    // Keep the lowerer's exact constant and decoded literal representation.
    let raw=if(folded && scalar==6){if(bits==1){b"true"}else{b"false"}}else if(folded){b""}else{syntax::expression_text(parsed,e.source_index)};
    if(!budget::charge(work,raw.len+1)){return 1002;}if(old.text_start!=*text_at || old.text_end-old.text_start!=raw.len || !bytes::equal(raw,ir::node_text(saved,at))){return 1001;};*text_at=old.text_end;if(old.cooked_start!=*text_at){return 1001;}
    var u32 literal_kind=0;var bytes::Buffer cooked=bytes::empty();
    if(folded){literal_kind=if(scalar==6){4}else{5};if(!budget::charge(work,20)){return 1002;}if(!bytes::append_usize(ref mut cooked,bits).is_ok){return 1003;}}
    else if(e.kind==1){literal_kind=syntax::literal_kind(parsed,e.source_index);let encoded=syntax::literal_cooked(parsed,e.source_index);if(!budget::charge(work,encoded.len)){return 1002;}if(literal_kind>=7 && literal_kind<=9){let code=literals::decode(ref mut cooked,encoded);if(code!=0){return if(code==3){1003}else{1001};}}else if(!bytes::append(ref mut cooked,encoded).is_ok){return 1003;}}
    if(old.literal_kind!=literal_kind || old.cooked_end-old.cooked_start!=bytes::len(ref cooked) || !bytes::equal(ir::node_cooked(saved,at),bytes::view(ref cooked))){return 1001;};*text_at=old.cooked_end;return 0;
}
fn u32 body(ref ir::Program program,ref ir::Body saved,ref syntax::Parsed parsed,ref m::Values fresh,ref values::Plan typed,ref interfaces::Interfaces imported,ref types::Mapping map,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let frame=m::frame(fresh);let declaration=syntax::declaration_at(parsed,frame.declaration-1);let root=if(frame.kind==2){declaration.cleanup_body}else{declaration.body};let start=if(frame.kind==2){declaration.cleanup_start}else{declaration.start};let end=if(frame.kind==2){declaration.cleanup_end}else{declaration.end};
    if(ir::body(saved)!=root || ir::source_bytes(saved)!=syntax::source_view(parsed).len || ir::start(saved)!=start || ir::end(saved)!=end || ir::node_count(saved)!=m::expression_count(fresh) || ir::local_count(saved)!=m::local_count(fresh) || ir::call_count(saved)!=m::call_count(fresh) || ir::argument_count(saved)!=m::argument_count(fresh) || ir::region_count(saved)!=m::region_count(fresh) || ir::placement_count(saved)!=m::placement_count(fresh) || ir::reset_count(saved)!=m::region_exit_count(fresh)){return 1001;}
    var usize text_at=0;var usize i=0;while(i<ir::node_count(saved)){let code=node(program,saved,i,parsed,fresh,typed,imported,map,ref mut text_at,work);if(code!=0){return code;}i+=1;}
    var usize at=0;i=1;while(i<=syntax::statement_count(parsed)){if(!budget::charge(work,1)){return 1002;}let s=syntax::statement_at(parsed,i);if(s.start>=start && s.end<=end){if(at>=ir::statement_count(saved)){return 1001;}let old=ir::statement_at(saved,at);let op=if(s.kind==4){syntax::assignment_operator(parsed,i)}else{b""};if(!budget::charge(work,op.len)){return 1002;}if(old.source_index!=i || !equal::statement_value_same(s,old.value) || old.operator_start!=text_at || old.operator_end-old.operator_start!=op.len || !bytes::equal(op,ir::assignment_operator(saved,at))){return 1001;}text_at=old.operator_end;at+=1;}i+=1;}if(at!=ir::statement_count(saved) || text_at!=ir::text_bytes(saved)){return 1001;}
    i=1;while(i<=ir::local_count(saved)){if(!budget::charge(work,1)){return 1002;}var a=m::local_at(fresh,i);a.type=types::mapped(map,a.type);a.expected=types::mapped(map,a.expected);if(!equal::local_same(a,ir::local_at(saved,i))){return 1001;}i+=1;}
    i=0;while(i<ir::region_count(saved)){if(!budget::charge(work,1)){return 1002;}var a=m::region_at(fresh,i);a.arena_type=types::mapped(map,a.arena_type);a.source_type=types::mapped(map,a.source_type);if(!equal::scope_same(a,ir::region_at(saved,i))){return 1001;}i+=1;}
    i=0;while(i<ir::placement_count(saved)){if(!budget::charge(work,1)){return 1002;}var a=m::placement_at(fresh,i);a.owner_type=types::mapped(map,a.owner_type);a.borrow_type=types::mapped(map,a.borrow_type);a.result_type=types::mapped(map,a.result_type);a.error_type=types::mapped(map,a.error_type);if(!equal::placement_same(a,ir::placement_at(saved,i))){return 1001;}i+=1;}
    i=0;while(i<ir::reset_count(saved)){if(!budget::charge(work,1)){return 1002;}if(!equal::exit_same(m::region_exit_at(fresh,i),ir::reset_at(saved,i))){return 1001;}i+=1;}return 0;
}
pub fn Result<frames::Mapping,u32> match(ref ir::Program program,ref identities::Snapshot ids,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let index=switch(frames::build(program,ids,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let map=switch(frames::match(ref index,checked,plan,sources,metadata,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let typed=checking::checked_values(checked);let imported=checking::checked_interfaces(checked);var usize i=0;
    while(i<values::count(typed)){let fresh=values::at(typed,i);let frame=m::frame(fresh);let target=frames::mapped(ref map,i+1);if(target==0 || target>ir::count(program)){return .err(1001);}let code=body(program,ir::at(program,target),v::at<syntax::Parsed>(sources,frame.module_index),fresh,typed,imported,frames::type_mapping(ref map),work);if(code!=0){return .err(code);}i+=1;}return .ok(map);
}
