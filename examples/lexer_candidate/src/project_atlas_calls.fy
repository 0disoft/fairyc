module project_atlas_calls;
import project_atlas_write as w;
import project_atlas_declarations as declarations;
import project_atlas_instances as instances;
import project_atlas_closures as closures;
import project_atlas_aggregate as aggregate;
import project_atlas_drop_sites as drops;
import project_atlas_trap_kinds as traps;
import project_atlas_unsafe_sites as unsafe_sites;
import project_atlas_move_sites as moves;
import project_atlas_allocation_sites as allocations;
import project_atlas_return_error_sites as error_sites;
import module_borrow_observations as observations;
import project_atlas_reachability as reach;
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
fn Result<bool,u32> before(ref input::Input p,ref instances::Instances context,ref closures::Projection projection,usize caller,Call a,Call b,ref mut usize meter){
 if(!work::charge(meter,1)){return .err(3);}if(a.start!=b.start){return .ok(a.start<b.start);}if(a.end!=b.end){return .ok(a.end<b.end);}
 var left=bytes::empty();var right=bytes::empty();discard closures::target(ref mut left,p,projection,a.target,caller,meter)?;discard closures::target(ref mut right,p,projection,b.target,caller,meter)?;
 if(!bytes::equal(bytes::view(ref left),bytes::view(ref right))){return .ok(model::before(bytes::view(ref left),bytes::view(ref right)));}return .ok(model::before(call_identity(context,a),call_identity(context,b)));
}
fn Result<bool,u32> add(ref input::Input p,ref instances::Instances context,ref closures::Projection projection,usize caller,ref mut v::Vec<Call> result,Call value,ref mut usize meter){
 if(v::len<Call>(result)>=65536){return .err(3);}if(!v::push<Call>(result,value).is_ok){return .err(2);}
 var usize j=v::len<Call>(result)-1;
 while(j>0){let prior=*v::at<Call>(result,j-1);if(!before(p,context,projection,caller,value,prior,meter)?){break;}{let slot=v::at_mut<Call>(result,j);*slot=prior;}{let slot=v::at_mut<Call>(result,j-1);*slot=value;}j-=1;}return .ok(true);
}
fn Result<usize,u32> frame(ref ir::Program program,declarations::Row row,ref mut usize meter){
 var usize frame=1;while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);if((f.kind==1 || f.kind==3) && f.module_index==row.module_index && f.declaration==row.declaration+1){return .ok(frame);}frame+=1;}return .err(9);
}
fn Result<v::Vec<Call>,u32> calls(ref input::Input p,ref ir::Program program,ref instances::Instances context,ref closures::Projection projection,declarations::Row row,usize frame,ref reach::Observations reached,ref mut usize meter){
 if(frame==0){return .ok(v::empty<Call>());}if(frame>ir::count(program)){return .err(9);}let body=ir::at(program,frame);var result=v::empty<Call>();var usize i=0;
 while(i<ir::call_count(body)){
  if(!work::charge(meter,1)){return .err(3);}let c=ir::call_at(body,i);i+=1;
  if(!c.checked || c.kind!=1 && c.kind!=2 || c.target_module>=input::count(p) || c.target_declaration==0 || c.target_declaration>syntax::declaration_count(input::source(p,c.target_module))){return .err(103);}
  if(!reach::completed(reached,frame,c.expression,meter)?){continue;}
  let target=declarations::Row{.module_index=c.target_module,.declaration=c.target_declaration-1};let d=syntax::declaration_at(input::source(p,target.module_index),target.declaration);var usize selected=0;
  if(d.kind!=3 && d.kind!=9 || d.kind==9 && input::origin(p,target.module_index).owner!=input::origin(p,row.module_index).owner){return .err(103);}
  if(d.generic_parameters!=0){selected=instances::call_frame(context,frame,i-1);if(c.kind!=2 || selected==0 || selected>ir::count(program) || instances::frame_identity(context,selected).len==0){return .err(103);}let f=clean::frame_at(ir::cleanup_ir(program),selected);if(f.kind!=1 || f.module_index!=c.target_module || f.declaration!=c.target_declaration || f.context!=c.instance){return .err(103);}}
  discard add(p,context,projection,frame,ref mut result,Call{.target,.frame=selected,.specialization=0,.start=c.start,.end=c.end,.external=d.kind==9},meter)?;
 }
 i=0;while(i<instances::inline_count(context)){
  if(!work::charge(meter,1)){return .err(3);}let site=instances::inline_at(context,i);i+=1;if(site.frame!=frame){continue;}
  if(instances::planned_identity(context,site.node).len==0){return .err(103);}
  if(!reach::completed(reached,frame,ir::node(body,site.expression),meter)?){continue;}
  discard add(p,context,projection,frame,ref mut result,Call{.target=site.declaration,.frame=0,.specialization=site.node+1,.start=site.start,.end=site.end,.external=false},meter)?;
 }return .ok(result);
}
struct Counts{usize calls;usize external;usize drops;usize traps;usize unsafe_sites;usize moves;usize allocations;usize errors;usize loans;}
fn Result<Counts,u32> body(ref mut bytes::Buffer out,ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref instances::Instances context,ref closures::Projection projection,closures::Row projected,usize frame,ref drops::Collection observed,ref unsafe_sites::Collection unsafe_observed,ref moves::Collection moved,ref allocations::Collection allocated,ref error_sites::Collection propagated,ref observations::Table borrowed,ref reach::Observations reached,ref mut usize meter){
 let row=projected.declaration;let calls=calls(p,program,context,projection,row,frame,reached,meter)?;var usize total=0;var usize external=0;
 discard w::raw(out,b"{\"function\":",meter)?;discard closures::key(out,p,projection,projected,meter)?;
 discard w::raw(out,b",\"provenance\":\"compiler-observed\",\"calls\":[",meter)?;var usize j=0;
 while(j<v::len<Call>(ref calls)){
  let call=*v::at<Call>(ref calls,j);if(j>0){discard w::raw(out,b",",meter)?;}
  discard w::raw(out,b"{",meter)?;if(call.frame!=0 || call.specialization!=0){discard w::pair(out,b"specialization",call_identity(context,call),meter)?;discard w::raw(out,b",",meter)?;}
  discard w::raw(out,b"\"span\":",meter)?;discard declarations::span(out,p,row.module_index,call.start,call.end,meter)?;
  discard w::raw(out,if(call.external){b",\"target_kind\":\"external\",\"target\":"}else{b",\"target_kind\":\"resolved\",\"target\":"},meter)?;discard closures::target(out,p,projection,call.target,frame,meter)?;
  if(call.external){let id=signatures::identity(metadata,call.target.module_index,call.target.declaration+1);if(id==0 || !signatures::at(metadata,id).foreign){return .err(103);}discard w::raw(out,b",\"certainty\":\"opaque\",\"abi\":\"C\",",meter)?;discard w::pair(out,b"link_name",signatures::linker_name(metadata,id),meter)?;discard w::raw(out,b"}",meter)?;external+=1;}
  else{discard w::raw(out,b",\"certainty\":\"exact\",\"abi\":\"\",\"link_name\":\"\"}",meter)?;}
  total+=1;j+=1;
 }
 if(frame>drops::frame_count(observed) || frame>unsafe_sites::frame_count(unsafe_observed) || frame>moves::frame_count(moved) || frame>allocations::frame_count(allocated) || frame>error_sites::frame_count(propagated) || observations::ready(borrowed) && frame>observations::count(borrowed)){return .err(9);}let sites=if(frame==0){drops::Range{.start=0,.end=0}}else{drops::range(observed,frame-1)};
 let u32 mask=if(frame==0){0}else{drops::trap_at(observed,frame-1)};var usize trap_count=0;j=0;
 discard w::raw(out,b"],\"trap_categories\":[",meter)?;
 while(j<6){if(!work::charge(meter,1)){return .err(3);}if(traps::contains(mask,j)){if(trap_count>0){discard w::raw(out,b",",meter)?;}discard w::string(out,traps::category(j),meter)?;trap_count+=1;}j+=1;}
 discard w::raw(out,b"],\"unsafe_sites\":[",meter)?;var usize unsafe_count=0;
 if(frame==0){let site=unsafe_sites::leaf(unsafe_observed,row.module_index,meter)?;discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;unsafe_count=1;}
 else{let range=unsafe_sites::range(unsafe_observed,frame-1);j=0;while(j<range.end-range.start){let site=unsafe_sites::at(unsafe_observed,range.start+j);if(j>0){discard w::raw(out,b",",meter)?;}discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;j+=1;}unsafe_count=range.end-range.start;}
 discard w::raw(out,b"],\"move_sites\":[",meter)?;let moved_range=if(frame==0){moves::Range{.start=0,.end=0}}else{moves::range(moved,frame-1)};j=0;
 while(j<moved_range.end-moved_range.start){let site=moves::at(moved,moved_range.start+j);if(j>0){discard w::raw(out,b",",meter)?;}discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;j+=1;}
 discard w::raw(out,b"],\"drop_sites\":[",meter)?;j=0;
 while(j<sites.end-sites.start){let site=drops::at(observed,sites.start+j);if(j>0){discard w::raw(out,b",",meter)?;}discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;j+=1;}
 discard w::raw(out,b"],\"allocation_sites\":[",meter)?;let allocation_range=if(frame==0){allocations::Range{.start=0,.end=0}}else{allocations::range(allocated,frame-1)};j=0;
 while(j<allocation_range.end-allocation_range.start){let site=allocations::at(allocated,allocation_range.start+j);if(j>0){discard w::raw(out,b",",meter)?;}discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;j+=1;}
 discard w::raw(out,b"],\"return_error_sites\":[",meter)?;let error_range=if(frame==0){error_sites::Range{.start=0,.end=0}}else{error_sites::range(propagated,frame-1)};j=0;
 while(j<error_range.end-error_range.start){let site=error_sites::at(propagated,error_range.start+j);if(j>0){discard w::raw(out,b",",meter)?;}discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;j+=1;}
 discard w::raw(out,b"],\"borrows\":[",meter)?;let usize loan_count=if(frame==0 || !observations::ready(borrowed)){0}else{observations::site_count(borrowed,frame)};j=0;
 while(j<loan_count){if(!work::charge(meter,1)){return .err(3);}let site=observations::at(borrowed,frame,j);if(j>0){discard w::raw(out,b",",meter)?;}
  discard w::raw(out,b"{\"span\":",meter)?;discard declarations::span(out,p,row.module_index,site.start,site.end,meter)?;discard w::raw(out,b",",meter)?;
  discard w::pair(out,b"kind",if(site.mutable){b"mutable"}else{b"shared"},meter)?;discard w::raw(out,if(site.escapes){b",\"escapes\":true}"}else{b",\"escapes\":false}"},meter)?;j+=1;
 }discard w::raw(out,b"]}",meter)?;return .ok(Counts{.calls=total,.external,.drops=sites.end-sites.start,.traps=trap_count,.unsafe_sites=unsafe_count,.moves=moved_range.end-moved_range.start,.allocations=allocation_range.end-allocation_range.start,.errors=error_range.end-error_range.start,.loans=loan_count});
}
pub resource Summaries{bytes::Buffer text;bytes::Buffer specializations;usize calls;usize drops;usize traps;usize unsafe_sites;usize moves;usize allocations;usize errors;usize loans;usize external;usize functions;usize specialization_count;usize templates;usize implementations;bool observed_borrows;cleanup{}}
pub fn ref u8[] text(ref Summaries p){return bytes::view(ref p.text);}
pub fn ref u8[] specializations(ref Summaries p){return bytes::view(ref p.specializations);}
pub fn usize count(ref Summaries p){return p.calls;}
pub fn usize drop_count(ref Summaries p){return p.drops;}
pub fn usize trap_count(ref Summaries p){return p.traps;}
pub fn usize unsafe_count(ref Summaries p){return p.unsafe_sites;}
pub fn usize move_count(ref Summaries p){return p.moves;}
pub fn usize allocation_count(ref Summaries p){return p.allocations;}
pub fn usize return_error_count(ref Summaries p){return p.errors;}
pub fn usize borrow_count(ref Summaries p){return p.loans;}
pub fn bool borrows_observed(ref Summaries p){return p.observed_borrows;}
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
 let projection=closures::build(p,program,context,rows,meter)?;return summaries_projected(p,program,metadata,table,context,rows,ref projection,meter);
}
pub fn Result<Summaries,u32> summaries_projected(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref coordinates::Coordinates table,ref instances::Instances context,ref v::Vec<declarations::Row> rows,ref closures::Projection projection,ref mut usize meter){
 let absent=observations::absent();return summaries_mode(p,program,metadata,table,context,rows,projection,ref absent,meter);
}
pub fn Result<Summaries,u32> summaries_observed(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref coordinates::Coordinates table,ref instances::Instances context,ref v::Vec<declarations::Row> rows,ref closures::Projection projection,ref observations::Table borrowed,ref mut usize meter){
 if(!observations::ready(borrowed)){return .err(9);}return summaries_mode(p,program,metadata,table,context,rows,projection,borrowed,meter);
}
fn Result<Summaries,u32> summaries_mode(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref coordinates::Coordinates table,ref instances::Instances context,ref v::Vec<declarations::Row> rows,ref closures::Projection projection,ref observations::Table borrowed,ref mut usize meter){
 if(observations::ready(borrowed) && observations::count(borrowed)!=ir::count(program)){return .err(9);}
 let observed=drops::collect_observed_all(program,65536,6,meter)?;
 let unsafe_observed=unsafe_sites::collect_all(program,65536,meter)?;
 let moved=moves::collect_all(program,65536,meter)?;
 let allocated=allocations::collect_all(program,65536,meter)?;
 let propagated=error_sites::collect_all(program,65536,meter)?;
 let reached=reach::collect_all(program,meter)?;
 var out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;var usize total=0;var usize drop_total=0;var usize trap_total=0;var usize unsafe_total=0;var usize move_total=0;var usize allocation_total=0;var usize error_total=0;var usize loan_total=0;var usize external=0;var usize functions=0;var usize i=0;
 while(i<closures::count(projection)){
  if(!work::charge(meter,1)){return .err(3);}let projected=closures::at(projection,i);let row=projected.declaration;i+=1;let d=syntax::declaration_at(input::source(p,row.module_index),row.declaration);if(d.kind!=3 || d.generic_parameters!=0){continue;}
  if(functions>0){discard w::raw(ref mut out,b",",meter)?;}let c=body(ref mut out,p,program,metadata,context,projection,projected,if(projected.frame!=0){projected.frame}else{frame(program,row,meter)?},ref observed,ref unsafe_observed,ref moved,ref allocated,ref propagated,borrowed,ref reached,meter)?;total+=c.calls;drop_total+=c.drops;trap_total+=c.traps;unsafe_total+=c.unsafe_sites;move_total+=c.moves;allocation_total+=c.allocations;error_total+=c.errors;loan_total+=c.loans;external+=c.external;functions+=1;
 }
 discard w::raw(ref mut out,b"]",meter)?;var specs=bytes::empty();discard w::raw(ref mut specs,b"[",meter)?;var usize specialized=0;i=0;
 while(i<instances::count(context)){if(!work::charge(meter,1)){return .err(3);}let row=instances::at(context,i);i+=1;var usize j=0;var bool local=false;
  while(j<v::len<declarations::Row>(rows)){if(!work::charge(meter,1)){return .err(3);}let d=*v::at<declarations::Row>(rows,j);if(d.module_index==row.declaration.module_index && d.declaration==row.declaration.declaration){local=true;break;}j+=1;}if(!local){continue;}
  if(specialized>0){discard w::raw(ref mut specs,b",",meter)?;}discard w::raw(ref mut specs,b"{",meter)?;discard w::pair(ref mut specs,b"id",instances::identity(context,row),meter)?;discard w::raw(ref mut specs,b",\"declaration\":",meter)?;discard declarations::key(ref mut specs,p,row.declaration,meter)?;
  discard w::raw(ref mut specs,b",\"type_arguments\":",meter)?;discard instances::arguments(ref mut specs,context,row,meter)?;
  if(row.type!=0){discard w::raw(ref mut specs,b",\"aggregate\":",meter)?;discard aggregate::declaration(ref mut specs,p,table,program,row,meter)?;}
  if(row.type==0 || row.frame!=0){if(row.type==0 && row.frame==0 && !instances::inline_leaf(context,row)){return .err(103);}discard w::raw(ref mut specs,if(row.type==0){b",\"function\":"}else{b",\"cleanup\":"},meter)?;let c=body(ref mut specs,p,program,metadata,context,projection,closures::plain(row.declaration),row.frame,ref observed,ref unsafe_observed,ref moved,ref allocated,ref propagated,borrowed,ref reached,meter)?;total+=c.calls;drop_total+=c.drops;trap_total+=c.traps;unsafe_total+=c.unsafe_sites;move_total+=c.moves;allocation_total+=c.allocations;error_total+=c.errors;loan_total+=c.loans;external+=c.external;}
  discard w::raw(ref mut specs,b"}",meter)?;specialized+=1;
 }
 discard w::raw(ref mut specs,b"]",meter)?;return .ok(Summaries{.text=out,.specializations=specs,.calls=total,.drops=drop_total,.traps=trap_total,.unsafe_sites=unsafe_total,.moves=move_total,.allocations=allocation_total,.errors=error_total,.loans=loan_total,.observed_borrows=observations::ready(borrowed),.external,.functions,.specialization_count=specialized,.templates=instances::template_count(context),.implementations=instances::count(context)});
}
