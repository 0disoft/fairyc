module project_atlas_calls;
import project_atlas_write as w;
import project_atlas_declarations as declarations;
import project_atlas_instances as instances;
import project_atlas_aggregate as aggregate;
import module_public_type_text as coordinates;
import project_compiler_input as input;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_atlas_model as model;
import project_path as work;
import syntax as syntax;
import module_signatures as signatures;
import std::bytes as bytes;
import std::vec as v;

pub struct Call{declarations::Row target;usize frame;usize specialization;usize start;usize end;bool external;}
fn ref u8[] call_identity(ref instances::Instances context,Call call){if(call.specialization!=0){return instances::planned_identity(context,call.specialization-1);}return instances::frame_identity(context,call.frame);}
fn Result<bool,u32> before(ref input::Input p,ref instances::Instances context,Call a,Call b,ref mut usize meter){
 if(!work::charge(meter,1)){return .err(3);}if(a.start!=b.start){return .ok(a.start<b.start);}if(a.end!=b.end){return .ok(a.end<b.end);}
 var left=bytes::empty();var right=bytes::empty();discard declarations::key(ref mut left,p,a.target,meter)?;discard declarations::key(ref mut right,p,b.target,meter)?;
 if(!bytes::equal(bytes::view(ref left),bytes::view(ref right))){return .ok(model::before(bytes::view(ref left),bytes::view(ref right)));}return .ok(model::before(call_identity(context,a),call_identity(context,b)));
}
fn Result<bool,u32> add(ref input::Input p,ref instances::Instances context,ref mut v::Vec<Call> result,Call value,ref mut usize meter){
 if(v::len<Call>(result)>=65536){return .err(3);}if(!v::push<Call>(result,value).is_ok){return .err(2);}
 var usize j=v::len<Call>(result)-1;
 while(j>0){let prior=*v::at<Call>(result,j-1);if(!before(p,context,value,prior,meter)?){break;}{let slot=v::at_mut<Call>(result,j);*slot=prior;}{let slot=v::at_mut<Call>(result,j-1);*slot=value;}j-=1;}return .ok(true);
}
fn Result<usize,u32> frame(ref ir::Program program,declarations::Row row,ref mut usize meter){
 var usize frame=1;while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);if((f.kind==1 || f.kind==3) && f.module_index==row.module_index && f.declaration==row.declaration+1){return .ok(frame);}frame+=1;}return .err(9);
}
fn Result<v::Vec<Call>,u32> calls(ref input::Input p,ref ir::Program program,ref instances::Instances context,declarations::Row row,usize frame,ref mut usize meter){
 if(frame==0){return .ok(v::empty<Call>());}if(frame>ir::count(program)){return .err(9);}let body=ir::at(program,frame);var result=v::empty<Call>();var usize i=0;
 while(i<ir::call_count(body)){
  if(!work::charge(meter,1)){return .err(3);}let c=ir::call_at(body,i);i+=1;
  if(!c.checked || c.kind!=1 && c.kind!=2 || c.target_module>=input::count(p) || c.target_declaration==0 || c.target_declaration>syntax::declaration_count(input::source(p,c.target_module))){return .err(103);}
  let target=declarations::Row{.module_index=c.target_module,.declaration=c.target_declaration-1};let d=syntax::declaration_at(input::source(p,target.module_index),target.declaration);var usize selected=0;
  if(d.kind!=3 && d.kind!=9 || d.kind==9 && input::origin(p,target.module_index).owner!=input::origin(p,row.module_index).owner){return .err(103);}
  if(d.generic_parameters!=0){selected=instances::call_frame(context,frame,i-1);if(c.kind!=2 || selected==0 || selected>ir::count(program) || instances::frame_identity(context,selected).len==0){return .err(103);}let f=clean::frame_at(ir::cleanup_ir(program),selected);if(f.kind!=1 || f.module_index!=c.target_module || f.declaration!=c.target_declaration || f.context!=c.instance){return .err(103);}}
  discard add(p,context,ref mut result,Call{.target,.frame=selected,.specialization=0,.start=c.start,.end=c.end,.external=d.kind==9},meter)?;
 }
 i=0;while(i<instances::inline_count(context)){
  if(!work::charge(meter,1)){return .err(3);}let site=instances::inline_at(context,i);i+=1;if(site.frame!=frame){continue;}
  if(instances::planned_identity(context,site.node).len==0){return .err(103);}
  discard add(p,context,ref mut result,Call{.target=site.declaration,.frame=0,.specialization=site.node+1,.start=site.start,.end=site.end,.external=false},meter)?;
 }return .ok(result);
}
struct Counts{usize calls;usize external;}
fn Result<Counts,u32> body(ref mut bytes::Buffer out,ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref instances::Instances context,declarations::Row row,usize frame,ref mut usize meter){
 let calls=calls(p,program,context,row,frame,meter)?;var usize total=0;var usize external=0;
 discard w::raw(out,b"{\"function\":",meter)?;discard declarations::key(out,p,row,meter)?;
 discard w::raw(out,b",\"provenance\":\"compiler-observed\",\"calls\":[",meter)?;var usize j=0;
 while(j<v::len<Call>(ref calls)){
  let call=*v::at<Call>(ref calls,j);if(j>0){discard w::raw(out,b",",meter)?;}
  discard w::raw(out,b"{",meter)?;if(call.frame!=0 || call.specialization!=0){discard w::pair(out,b"specialization",call_identity(context,call),meter)?;discard w::raw(out,b",",meter)?;}
  discard w::raw(out,b"\"span\":",meter)?;discard declarations::span(out,p,row.module_index,call.start,call.end,meter)?;
  discard w::raw(out,if(call.external){b",\"target_kind\":\"external\",\"target\":"}else{b",\"target_kind\":\"resolved\",\"target\":"},meter)?;discard declarations::key(out,p,call.target,meter)?;
  if(call.external){let id=signatures::identity(metadata,call.target.module_index,call.target.declaration+1);if(id==0 || !signatures::at(metadata,id).foreign){return .err(103);}discard w::raw(out,b",\"certainty\":\"opaque\",\"abi\":\"C\",",meter)?;discard w::pair(out,b"link_name",signatures::linker_name(metadata,id),meter)?;discard w::raw(out,b"}",meter)?;external+=1;}
  else{discard w::raw(out,b",\"certainty\":\"exact\",\"abi\":\"\",\"link_name\":\"\"}",meter)?;}
  total+=1;j+=1;
 }
 discard w::raw(out,b"],\"trap_categories\":[],\"unsafe_sites\":[],\"move_sites\":[],\"drop_sites\":[],\"allocation_sites\":[],\"return_error_sites\":[],\"borrows\":[]}",meter)?;return .ok(Counts{.calls=total,.external});
}
pub resource Summaries{bytes::Buffer text;bytes::Buffer specializations;usize calls;usize external;usize functions;usize specialization_count;usize templates;usize implementations;cleanup{}}
pub fn ref u8[] text(ref Summaries p){return bytes::view(ref p.text);}
pub fn ref u8[] specializations(ref Summaries p){return bytes::view(ref p.specializations);}
pub fn usize count(ref Summaries p){return p.calls;}
pub fn usize external_count(ref Summaries p){return p.external;}
pub fn usize function_count(ref Summaries p){return p.functions;}
pub fn usize specialization_count(ref Summaries p){return p.specialization_count;}
pub fn usize template_count(ref Summaries p){return p.templates;}
pub fn usize implementation_count(ref Summaries p){return p.implementations;}
pub fn Result<Summaries,u32> summaries(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref v::Vec<declarations::Row> rows,ref mut usize meter){let context=instances::empty();return summaries_generics(p,program,metadata,ref context,rows,meter);}
pub fn Result<Summaries,u32> summaries_generics(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref instances::Instances context,ref v::Vec<declarations::Row> rows,ref mut usize meter){
 let table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());return summaries_aggregates(p,program,metadata,ref table,context,rows,meter);
}
pub fn Result<Summaries,u32> summaries_aggregates(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref coordinates::Coordinates table,ref instances::Instances context,ref v::Vec<declarations::Row> rows,ref mut usize meter){
 var out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;var usize total=0;var usize external=0;var usize functions=0;var usize i=0;
 while(i<v::len<declarations::Row>(rows)){
  if(!work::charge(meter,1)){return .err(3);}let row=*v::at<declarations::Row>(rows,i);i+=1;let d=syntax::declaration_at(input::source(p,row.module_index),row.declaration);if(d.kind!=3 || d.generic_parameters!=0){continue;}
  if(functions>0){discard w::raw(ref mut out,b",",meter)?;}let c=body(ref mut out,p,program,metadata,context,row,frame(program,row,meter)?,meter)?;total+=c.calls;external+=c.external;functions+=1;
 }
 discard w::raw(ref mut out,b"]",meter)?;var specs=bytes::empty();discard w::raw(ref mut specs,b"[",meter)?;var usize specialized=0;i=0;
 while(i<instances::count(context)){if(!work::charge(meter,1)){return .err(3);}let row=instances::at(context,i);i+=1;var usize j=0;var bool local=false;
  while(j<v::len<declarations::Row>(rows)){if(!work::charge(meter,1)){return .err(3);}let d=*v::at<declarations::Row>(rows,j);if(d.module_index==row.declaration.module_index && d.declaration==row.declaration.declaration){local=true;break;}j+=1;}if(!local){continue;}
  if(specialized>0){discard w::raw(ref mut specs,b",",meter)?;}discard w::raw(ref mut specs,b"{",meter)?;discard w::pair(ref mut specs,b"id",instances::identity(context,row),meter)?;discard w::raw(ref mut specs,b",\"declaration\":",meter)?;discard declarations::key(ref mut specs,p,row.declaration,meter)?;
  discard w::raw(ref mut specs,b",\"type_arguments\":",meter)?;discard instances::arguments(ref mut specs,context,row,meter)?;
  if(row.type!=0){discard w::raw(ref mut specs,b",\"aggregate\":",meter)?;discard aggregate::declaration(ref mut specs,p,table,program,row,meter)?;}
  if(row.type==0 || row.frame!=0){if(row.type==0 && row.frame==0 && !instances::inline_leaf(context,row)){return .err(103);}discard w::raw(ref mut specs,if(row.type==0){b",\"function\":"}else{b",\"cleanup\":"},meter)?;let c=body(ref mut specs,p,program,metadata,context,row.declaration,row.frame,meter)?;total+=c.calls;external+=c.external;}
  discard w::raw(ref mut specs,b"}",meter)?;specialized+=1;
 }
 discard w::raw(ref mut specs,b"]",meter)?;return .ok(Summaries{.text=out,.specializations=specs,.calls=total,.external,.functions,.specialization_count=specialized,.templates=instances::template_count(context),.implementations=instances::count(context)});
}
