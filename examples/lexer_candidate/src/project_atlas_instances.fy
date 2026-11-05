module project_atlas_instances;
import project_compiler_input as input;
import module_project_identified_ir as named;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import project_source_plan as planner;
import generic_source_plan as plan;
import generic_source_plan_input as planned;
import generic_source_catalog as catalog;
import generic_concrete_types as concrete;
import generic_transport_scalars as scalars;
import project_atlas_declarations as declarations;
import project_atlas_types as types;
import project_atlas_write as w;
import project_atlas_model as model;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Row{usize frame;usize node;usize id_start;usize id_end;declarations::Row declaration;}
pub resource Instances{bytes::Buffer names;v::Vec<plan::Plan> plans;v::Vec<Row> rows;v::Vec<usize> by_frame;v::Vec<usize> call_starts;v::Vec<usize> call_frames;usize templates;cleanup{}}
pub fn Instances empty(){return Instances{.names=bytes::empty(),.plans=v::empty<plan::Plan>(),.rows=v::empty<Row>(),.by_frame=v::empty<usize>(),.call_starts=v::empty<usize>(),.call_frames=v::empty<usize>(),.templates=0};}
fn ref plan::Plan original(ref Instances p){return v::at<plan::Plan>(ref p.plans,0);}
pub fn ref u8[] frame_identity(ref Instances p,usize frame){if(frame>=v::len<usize>(ref p.by_frame)){return bytes::view(ref p.names)[0..0];}let index=*v::at<usize>(ref p.by_frame,frame);if(index==0){return bytes::view(ref p.names)[0..0];}return identity(p,at(p,index-1));}
pub fn usize count(ref Instances p){return v::len<Row>(ref p.rows);}
pub fn Row at(ref Instances p,usize index){return *v::at<Row>(ref p.rows,index);}
pub fn usize template_count(ref Instances p){return p.templates;}
// Invocation.instance is a specialization context, not a linked frame ordinal.
// Retain the checked owner's resolved targets instead of assuming equal numbers.
pub fn usize call_frame(ref Instances p,usize frame,usize call){if(frame==0 || frame>=v::len<usize>(ref p.call_starts)){return 0;}let start=*v::at<usize>(ref p.call_starts,frame-1);let end=*v::at<usize>(ref p.call_starts,frame);if(start>end || call>=end-start){return 0;}return *v::at<usize>(ref p.call_frames,start+call);}
pub fn ref u8[] identity(ref Instances p,Row r){return bytes::view(ref p.names)[r.id_start..r.id_end];}
pub fn Result<bool,u32> arguments(ref mut bytes::Buffer out,ref Instances p,Row row,ref mut usize meter){
 let node=plan::at(original(p),row.node);let types=planned::types(plan::input(original(p)));discard w::raw(out,b"[",meter)?;var usize i=0;
 while(i<node.count){if(!work::charge(meter,1)){return .err(3);}if(node.arguments[i]==0 || node.arguments[i]>concrete::count(types)){return .err(7);}let t=concrete::at(types,node.arguments[i]-1);if(t.kind!=1 || !scalars::valid(t.scalar)){return .err(101);}if(i>0){discard w::raw(out,b",",meter)?;}discard types::builtin(out,concrete::text(types,t.name),meter)?;i+=1;}return w::raw(out,b"]",meter);
}
// Rebuild the source plan from exact Complete-bound input, and join by g0 bytes.
// Producer-local node/frame ordinals are never treated as persistent identity.
pub fn Result<Instances,u32> collect(ref input::Input source,ref coordinates::Coordinates table,ref named::Program checked,ref signatures::Program metadata,ref mut usize meter){
 let current=named::current(checked,input::sources(source),metadata,table,meter);if(current!=0){return .err(if(current==10){3}else{7});}
 let original=switch(planner::build(source,table,meter)){case .ok(p):{p}case .err(e):{return .err(if(e.code==10){3}else if(e.code==11){2}else{7});}};
 let program=named::ir_program(checked);if(named::frame_count(checked)!=ir::count(program)){return .err(7);}let templates=planned::templates(plan::input(ref original));var names=bytes::empty();var rows=v::empty<Row>();var usize frame=1;
 while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);let id=named::identity(checked,named::frame_at(checked,frame));
  if(id.len!=0){if(f.kind!=1 || f.module_index>=input::count(source) || f.declaration==0 || f.declaration>syntax::declaration_count(input::source(source,f.module_index))){return .err(101);}var usize found=plan::count(ref original);var usize i=0;
   while(i<plan::count(ref original)){let n=plan::at(ref original,i);let t=catalog::template_at(templates,n.template_index);if(!work::charge(meter,id.len+68)){return .err(3);}if(bytes::equal(id,plan::text(ref original,n.id))){if(found!=plan::count(ref original) || n.state!=2 || t.kind!=3 || t.module_index!=f.module_index || t.declaration+1!=f.declaration){return .err(7);}found=i;}i+=1;}
   if(found==plan::count(ref original) || id.len!=67 || !bytes::equal(id[..3],b"g0_")){return .err(7);}if(bytes::len(ref names)>1048576-id.len || !work::charge(meter,id.len+1)){return .err(3);}let id_start=bytes::len(ref names);if(!bytes::append(ref mut names,id).is_ok){return .err(2);}if(!v::push<Row>(ref mut rows,Row{.frame,.node=found,.id_start,.id_end=bytes::len(ref names),.declaration=declarations::Row{.module_index=f.module_index,.declaration=f.declaration-1}}).is_ok){return .err(2);}
  }frame+=1;
 }
 var usize i=1;while(i<v::len<Row>(ref rows)){let value=*v::at<Row>(ref rows,i);let id=plan::text(ref original,plan::at(ref original,value.node).id);var usize at=i;
  while(at>0){let prior=*v::at<Row>(ref rows,at-1);let previous=plan::text(ref original,plan::at(ref original,prior.node).id);if(!work::charge(meter,id.len+previous.len+1)){return .err(3);}if(bytes::equal(id,previous)){return .err(7);}if(!model::before(id,previous)){break;}{let slot=v::at_mut<Row>(ref mut rows,at);*slot=prior;}{let slot=v::at_mut<Row>(ref mut rows,at-1);*slot=value;}at-=1;}i+=1;
 }
 var by_frame=v::empty<usize>();i=0;while(i<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut by_frame,0).is_ok){return .err(2);}i+=1;}i=0;
 while(i<v::len<Row>(ref rows)){let r=*v::at<Row>(ref rows,i);let slot=v::at_mut<usize>(ref mut by_frame,r.frame);*slot=i+1;i+=1;}
 var call_starts=v::empty<usize>();var call_frames=v::empty<usize>();if(!v::push<usize>(ref mut call_starts,0).is_ok){return .err(2);}frame=1;
 while(frame<=ir::count(program)){let body=ir::at(program,frame);i=0;while(i<ir::call_count(body)){if(!work::charge(meter,1)){return .err(3);}let target=switch(named::call_frame(checked,frame,i,meter)){case .ok(p):{p}case .err(code):{return .err(if(code==1002){3}else{7});}};if(!v::push<usize>(ref mut call_frames,target).is_ok){return .err(2);}i+=1;}if(!v::push<usize>(ref mut call_starts,v::len<usize>(ref call_frames)).is_ok){return .err(2);}frame+=1;}
 let template_count=catalog::count(templates);var plans=v::empty<plan::Plan>();if(!v::push<plan::Plan>(ref mut plans,original).is_ok){return .err(2);}return .ok(Instances{.names,.plans,.rows,.by_frame,.call_starts,.call_frames,.templates=template_count});
}
