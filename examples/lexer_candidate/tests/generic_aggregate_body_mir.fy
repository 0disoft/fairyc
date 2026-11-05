module tests::generic_aggregate_body_mir;
import module_unit_body_ir_model as body_ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_interfaces as interfaces;
import module_unit_region_model as regions;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) pub fn void verify(ref body_ir::Program p,ref checking::Plan owner){
 let checked=checking::checked_values(owner);let imported=checking::checked_interfaces(owner);expect(proof::check(p).code==0 && clean::aggregates(body_ir::cleanup_ir(p)),1);
 expect(body_ir::count(p)==values::count(checked) && body_ir::layout_count(p)==interfaces::shape_count(imported) && body_ir::field_count(p)==interfaces::field_count(imported),2);
 var usize i=0;while(i<body_ir::layout_count(p)){let a=body_ir::layout_at(p,i).value;let b=interfaces::shape_at(imported,i);expect(a.type==b.type && a.module_index==b.module_index && a.declaration==b.declaration && a.field_count==b.field_count && a.repr_c==b.repr_c,3);i+=1;}
 i=1;while(i<=body_ir::count(p)){let body=body_ir::at(p,i);let typed=values::at(checked,i-1);let frame=model::frame(typed);let f=clean::frame_at(body_ir::cleanup_ir(p),i);expect(f.kind==frame.kind && f.context==frame.context && f.module_index==frame.module_index && f.declaration==frame.declaration && f.result==frame.result,4);
  expect(body_ir::node_count(body)==model::expression_count(typed) && body_ir::local_count(body)==model::local_count(typed) && body_ir::call_count(body)==model::call_count(typed),5);var usize j=0;
  while(j<body_ir::node_count(body)){let a=body_ir::node_at(body,j).value;let b=model::expression_at(typed,j+1);expect(a.kind==b.kind && a.source_index==b.source_index && a.start==b.start && a.end==b.end && a.type==b.type && a.target==b.target && a.field.id==b.field.id && a.field.owner==b.field.owner,6);j+=1;}i+=1;
 }
 var usize tuple=1;while(tuple<=clean::type_count(body_ir::cleanup_ir(p)) && clean::type_at(body_ir::cleanup_ir(p),tuple).kind!=22){tuple+=1;}expect(tuple<=clean::type_count(body_ir::cleanup_ir(p)),7);
 var u32 mode=0;while(mode<=10){var usize frame=1;while(frame<=body_ir::count(p)){let b=body_ir::at(p,frame);if(body_ir::local_count(b)!=0 && has_value(b) && (mode<7 || mode>8 || body_ir::argument_count(b)!=0)){break;}frame+=1;}if(frame<=body_ir::count(p)){let actual=mutated(p,frame,tuple,mode);let u32 expected=if(mode==0){0}else if(mode==10){1002}else{1001};if(actual!=expected){discard io::write_stderr("body mutation ");discard io::write_stderr_u32(mode);discard io::write_stderr(" actual ");discard io::write_stderr_u32(actual);}expect(actual==expected,8);}else{expect(mode>=7,9);}mode+=1;}
}
fn u32 mutated(ref body_ir::Program p,usize frame,usize tuple,u32 mode){
 let b=body_ir::at(p,frame);var nodes=v::empty<body_ir::Node>();var statements=v::empty<body_ir::Statement>();var locals=v::empty<model::Local>();var calls=v::empty<model::Invocation>();var arguments=v::empty<model::Argument>();var scopes=v::empty<regions::Scope>();var placements=v::empty<regions::Placement>();var resets=v::empty<regions::Exit>();var bool changed=false;var usize i=0;
 while(i<body_ir::node_count(b)){var n=body_ir::node_at(b,i);if(!changed && n.value.state==1){changed=true;if(mode==1){n.value.type=tuple;}if(mode==2){n.value.expected=tuple;}if(mode==3){n.value.target=tuple;}if(mode==6){n.value.builtin.element=tuple;}}assert(v::push<body_ir::Node>(ref mut nodes,n).is_ok);i+=1;}if(!changed){return 1901;}
 i=0;while(i<body_ir::statement_count(b)){assert(v::push<body_ir::Statement>(ref mut statements,body_ir::statement_at(b,i)).is_ok);i+=1;}
 i=1;while(i<=body_ir::local_count(b)){var n=body_ir::local_at(b,i);if(i==1){if(mode==4){n.type=tuple;}if(mode==5){n.expected=tuple;}}assert(v::push<model::Local>(ref mut locals,n).is_ok);i+=1;}
 i=0;while(i<body_ir::call_count(b)){assert(v::push<model::Invocation>(ref mut calls,body_ir::call_at(b,i)).is_ok);i+=1;}
 i=0;while(i<body_ir::argument_count(b)){var a=body_ir::argument_at(b,i);if(i==0){if(mode==7){a.expected=tuple;}if(mode==8){a.actual=tuple;}}assert(v::push<model::Argument>(ref mut arguments,a).is_ok);i+=1;}
 i=0;while(i<body_ir::region_count(b)){assert(v::push<regions::Scope>(ref mut scopes,body_ir::region_at(b,i)).is_ok);i+=1;}
 i=0;while(i<body_ir::placement_count(b)){assert(v::push<regions::Placement>(ref mut placements,body_ir::placement_at(b,i)).is_ok);i+=1;}
 i=0;while(i<body_ir::reset_count(b)){assert(v::push<regions::Exit>(ref mut resets,body_ir::reset_at(b,i)).is_ok);i+=1;}
 let text=switch(bytes::copy(body_ir::text_view(b))){case .ok(p):{p}case .err(_):{return 1003;}};
 let rebuilt=body_ir::mark_inline(body_ir::assemble_body_regions(body_ir::body(b),body_ir::source_bytes(b),body_ir::start(b),body_ir::end(b),nodes,statements,locals,calls,arguments,scopes,placements,resets,text),body_ir::inline_hash(b));var usize work=if(mode==10){1048577}else{0};return proof::check_body(ref rebuilt,body_ir::cleanup_ir(p),if(mode==9){body_ir::count(p)+1}else{frame},ref mut work).code;
}

@effects(console) fn void expect(bool valid,usize stage){if(!valid){discard io::write_stderr("body proof stage ");discard io::write_stderr_usize(stage);}assert(valid);}

fn bool has_value(ref body_ir::Body b){var usize i=0;while(i<body_ir::node_count(b)){if(body_ir::node_at(b,i).value.state==1){return true;}i+=1;}return false;}
