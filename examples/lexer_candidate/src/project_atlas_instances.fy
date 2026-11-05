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

pub struct Row{usize frame;usize type;usize node;usize id_start;usize id_end;declarations::Row declaration;}
pub resource Instances{bytes::Buffer names;v::Vec<plan::Plan> plans;v::Vec<Row> rows;v::Vec<usize> by_frame;v::Vec<usize> call_starts;v::Vec<usize> call_frames;usize templates;bool aggregates;cleanup{}}
pub fn Instances empty(){return Instances{.names=bytes::empty(),.plans=v::empty<plan::Plan>(),.rows=v::empty<Row>(),.by_frame=v::empty<usize>(),.call_starts=v::empty<usize>(),.call_frames=v::empty<usize>(),.templates=0,.aggregates=false};}
fn ref plan::Plan original(ref Instances p){return v::at<plan::Plan>(ref p.plans,0);}
pub fn ref u8[] frame_identity(ref Instances p,usize frame){if(frame>=v::len<usize>(ref p.by_frame)){return bytes::view(ref p.names)[0..0];}let index=*v::at<usize>(ref p.by_frame,frame);if(index==0){return bytes::view(ref p.names)[0..0];}return identity(p,at(p,index-1));}
pub fn usize count(ref Instances p){return v::len<Row>(ref p.rows);}
pub fn Row at(ref Instances p,usize index){return *v::at<Row>(ref p.rows,index);}
pub fn usize template_count(ref Instances p){return p.templates;}
pub fn bool aggregate_types(ref Instances p){return p.aggregates;}
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
fn Result<bool,u32> retain(ref input::Input source,ref plan::Plan original,ref u8[] id,usize module_index,usize declaration,u32 kind,usize frame,usize type,ref mut bytes::Buffer names,ref mut v::Vec<Row> rows,ref mut usize meter){
 if(module_index>=input::count(source) || declaration==0 || declaration>syntax::declaration_count(input::source(source,module_index)) || id.len!=67 || !bytes::equal(id[..3],b"g0_")){return .err(7);}
 let templates=planned::templates(plan::input(original));var usize found=plan::count(original);var usize i=0;
 while(i<plan::count(original)){let n=plan::at(original,i);let t=catalog::template_at(templates,n.template_index);if(!work::charge(meter,id.len+68)){return .err(3);}if(bytes::equal(id,plan::text(original,n.id))){if(found!=plan::count(original) || n.state!=2 || t.kind!=kind || t.module_index!=module_index || t.declaration+1!=declaration){return .err(7);}found=i;}i+=1;}
 if(found==plan::count(original)){return .err(7);}if(bytes::len(names)>1048576-id.len || !work::charge(meter,id.len+1)){return .err(3);}let id_start=bytes::len(names);if(!bytes::append(names,id).is_ok){return .err(2);}return if(v::push<Row>(rows,Row{.frame,.type,.node=found,.id_start,.id_end=bytes::len(names),.declaration=declarations::Row{.module_index,.declaration=declaration-1}}).is_ok){.ok(true)}else{.err(2)};
}
pub fn Result<Instances,u32> collect(ref input::Input source,ref coordinates::Coordinates table,ref named::Program checked,ref signatures::Program metadata,ref mut usize meter){return collect_mode(source,table,checked,metadata,false,meter);}
pub fn Result<Instances,u32> collect_aggregates(ref input::Input source,ref coordinates::Coordinates table,ref named::Program checked,ref signatures::Program metadata,ref mut usize meter){return collect_mode(source,table,checked,metadata,true,meter);}
fn Result<Instances,u32> collect_mode(ref input::Input source,ref coordinates::Coordinates table,ref named::Program checked,ref signatures::Program metadata,bool aggregates,ref mut usize meter){
 let program=named::ir_program(checked);let context=ir::cleanup_ir(program);if(clean::aggregates(context)!=aggregates || signatures::aggregate_types(metadata)!=aggregates){return .err(7);}
 let current=named::current(checked,input::sources(source),metadata,table,meter);if(current!=0){return .err(if(current==10){3}else{7});}
 let original=switch(if(aggregates){planner::build_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,meter)}else{planner::build(source,table,meter)}){case .ok(p):{p}case .err(e):{return .err(if(e.code==10){3}else if(e.code==11){2}else{7});}};
 if(named::frame_count(checked)!=ir::count(program)){return .err(7);}let templates=planned::templates(plan::input(ref original));var names=bytes::empty();var rows=v::empty<Row>();var usize frame=1;
 while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(context,frame);let id=named::identity(checked,named::frame_at(checked,frame));
  if(id.len!=0){if(f.kind==1){discard retain(source,ref original,id,f.module_index,f.declaration,3,frame,0,ref mut names,ref mut rows,meter)?;}else if(f.kind!=2 || !aggregates){return .err(101);}}frame+=1;
 }
 if(aggregates){var usize type=1;while(type<=clean::type_count(context)){if(!work::charge(meter,1)){return .err(3);}let t=clean::type_at(context,type);if(t.kind==20 || t.kind==21){let id=named::identity(checked,named::type_at(checked,type));var usize cleanup_frame=0;
   if(t.kind==21){let glue=clean::glue_at(context,type);if(glue.kind!=1){return .err(7);}cleanup_frame=glue.frame;if(cleanup_frame!=0){if(cleanup_frame>ir::count(program)){return .err(7);}let f=clean::frame_at(context,cleanup_frame);if(f.kind!=2 || f.receiver_type!=type || f.module_index!=t.module_index || f.declaration!=t.declaration || !bytes::equal(id,named::identity(checked,named::frame_at(checked,cleanup_frame)))){return .err(7);}}}
   discard retain(source,ref original,id,t.module_index,t.declaration,if(t.kind==20){4}else{6},cleanup_frame,type,ref mut names,ref mut rows,meter)?;
  }type+=1;}if(v::len<Row>(ref rows)!=plan::count(ref original)){return .err(7);}
 }
 var usize i=1;while(i<v::len<Row>(ref rows)){let value=*v::at<Row>(ref rows,i);let id=plan::text(ref original,plan::at(ref original,value.node).id);var usize at=i;
  while(at>0){let prior=*v::at<Row>(ref rows,at-1);let previous=plan::text(ref original,plan::at(ref original,prior.node).id);if(!work::charge(meter,id.len+previous.len+1)){return .err(3);}if(bytes::equal(id,previous)){return .err(7);}if(!model::before(id,previous)){break;}{let slot=v::at_mut<Row>(ref mut rows,at);*slot=prior;}{let slot=v::at_mut<Row>(ref mut rows,at-1);*slot=value;}at-=1;}i+=1;
 }
 var by_frame=v::empty<usize>();i=0;while(i<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut by_frame,0).is_ok){return .err(2);}i+=1;}i=0;
 while(i<v::len<Row>(ref rows)){let r=*v::at<Row>(ref rows,i);if(r.frame!=0){let slot=v::at_mut<usize>(ref mut by_frame,r.frame);if(*slot!=0){return .err(7);};*slot=i+1;}i+=1;}
 if(aggregates){frame=1;while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}if(clean::frame_at(context,frame).kind==2 && named::identity(checked,named::frame_at(checked,frame)).len!=0 && *v::at<usize>(ref by_frame,frame)==0){return .err(7);}frame+=1;}}
 var call_starts=v::empty<usize>();var call_frames=v::empty<usize>();if(!v::push<usize>(ref mut call_starts,0).is_ok){return .err(2);}frame=1;
 while(frame<=ir::count(program)){let body=ir::at(program,frame);i=0;while(i<ir::call_count(body)){if(!work::charge(meter,1)){return .err(3);}let target=switch(named::call_frame(checked,frame,i,meter)){case .ok(p):{p}case .err(code):{return .err(if(code==1002){3}else{7});}};if(!v::push<usize>(ref mut call_frames,target).is_ok){return .err(2);}i+=1;}if(!v::push<usize>(ref mut call_starts,v::len<usize>(ref call_frames)).is_ok){return .err(2);}frame+=1;}
 let template_count=catalog::count(templates);var plans=v::empty<plan::Plan>();if(!v::push<plan::Plan>(ref mut plans,original).is_ok){return .err(2);}return .ok(Instances{.names,.plans,.rows,.by_frame,.call_starts,.call_frames,.templates=template_count,.aggregates});
}
