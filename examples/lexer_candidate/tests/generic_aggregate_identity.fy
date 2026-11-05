module tests::generic_aggregate_identity;
import module_unit_body_ir_model as body_ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_identified_ir as identified;
import module_source_specializations as selected;
import module_signatures as signatures;
import generic_source_plan as plan;
import generic_aggregate_types as aggregates;
import generic_source_plan_input as input;
import generic_source_catalog as templates;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}

pub fn void verify(ref identified::Program named,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref interfaces::Interfaces graph,ref plan::Plan original){
 let p=identified::ir_program(named);assert(clean::aggregates(body_ir::cleanup_ir(p)) && !clean::project(body_ir::cleanup_ir(p)));
 let prepared=signatures::aggregate_source(metadata);var usize instances=0;var usize i=1;
 while(i<=clean::type_count(body_ir::cleanup_ir(p))){
  let t=clean::type_at(body_ir::cleanup_ir(p),i);let row=identified::type_at(named,i);let id=identified::identity(named,row);
  if(t.kind==20 || t.kind==21){var usize expected=0;var usize j=0;while(j<aggregates::count(prepared)){let a=aggregates::at(prepared,j);if(a.type==i){assert(expected==0 && a.module_index==t.module_index && a.declaration==t.declaration);expected=a.node;}j+=1;}assert(expected!=0 && row.node==expected && bytes::equal(id,plan::text(original,plan::at(original,expected-1).id)));instances+=1;}
  else{assert(row.node==0 && id.len==0);}i+=1;
 }
 assert(instances==aggregates::count(prepared));
 var usize work=0;i=1;
 while(i<=identified::frame_count(named)){
  let f=clean::frame_at(body_ir::cleanup_ir(p),i);let row=identified::frame_at(named,i);let id=identified::identity(named,row);let source=v::at<syntax::Parsed>(sources,f.module_index);let d=syntax::declaration_at(source,f.declaration-1);
  assert((row.node!=0)==(d.generic_parameters!=0));
  if(row.node!=0){assert(id.len==67 && bytes::equal(id,plan::text(original,plan::at(original,row.node-1).id)));let n=plan::at(original,row.node-1);let t=templates::template_at(input::templates(plan::input(original)),n.template_index);assert(t.module_index==f.module_index && t.declaration+1==f.declaration);if(f.kind==2){assert(clean::type_at(body_ir::cleanup_ir(p),f.receiver_type).kind==21 && bytes::equal(id,identified::identity(named,identified::type_at(named,f.receiver_type))));}}
  else{assert(f.kind==3 && id.len==0);}
  let body=body_ir::at(p,i);var usize j=0;while(j<body_ir::call_count(body)){let c=body_ir::call_at(body,j);let target=switch(identified::call_frame(named,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);return;}};if(c.kind==2){assert(target!=0 && identified::frame_at(named,target).node!=0);}else if(c.kind==1 && target!=0){assert(identified::frame_at(named,target).node==0);}else{assert(target==0);}j+=1;}i+=1;
 }
 var usize measured=0;assert(switch(selected::build(original,sources,metadata,graph,ref mut measured)){case .ok(_):{true}case .err(_):{false}});assert(measured>0 && measured<16777216);
 work=16777216-measured;assert(switch(selected::build(original,sources,metadata,graph,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==16777216);
 work=16777216-measured+1;assert(switch(selected::build(original,sources,metadata,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10}});
 work=16777217;assert(switch(selected::build(original,sources,metadata,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10}});
 var changed=v::empty<syntax::Parsed>();i=0;while(i<v::len<syntax::Parsed>(sources)){var raw=switch(bytes::copy(syntax::source_view(v::at<syntax::Parsed>(sources,i)))){case .ok(b):{b}case .err(_):{assert(false);return;}};if(i==0){assert(bytes::append(ref mut raw,b"\n").is_ok);}assert(v::push<syntax::Parsed>(ref mut changed,parsed(bytes::view(ref raw))).is_ok);i+=1;}
 work=0;assert(switch(selected::build(original,ref changed,metadata,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==12}});
}
