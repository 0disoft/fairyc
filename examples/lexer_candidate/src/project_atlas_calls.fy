module project_atlas_calls;
import project_atlas_write as w;
import project_atlas_declarations as declarations;
import project_compiler_input as input;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_atlas_model as model;
import project_path as work;
import syntax as syntax;
import module_signatures as signatures;
import std::bytes as bytes;
import std::vec as v;
// The compiler may materialize an ordinary source function in a concrete frame.
// Every source declaration is separately admitted as nongeneric; frame/instance
// handles never become Atlas identities or imply generic-map support.
pub struct Call{declarations::Row target;usize start;usize end;bool external;}
fn Result<bool,u32> before(ref input::Input p,Call a,Call b,ref mut usize meter){
 if(!work::charge(meter,1)){return .err(3);}if(a.start!=b.start){return .ok(a.start<b.start);}if(a.end!=b.end){return .ok(a.end<b.end);}
 var left=bytes::empty();var right=bytes::empty();discard declarations::key(ref mut left,p,a.target,meter)?;discard declarations::key(ref mut right,p,b.target,meter)?;return .ok(model::before(bytes::view(ref left),bytes::view(ref right)));
}
fn Result<v::Vec<Call>,u32> calls(ref input::Input p,ref ir::Program program,declarations::Row row,ref mut usize meter){
 var usize frame=1;while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);if((f.kind==1 || f.kind==3) && f.module_index==row.module_index && f.declaration==row.declaration+1){break;}frame+=1;}
 if(frame>ir::count(program)){return .err(9);}let body=ir::at(program,frame);var result=v::empty<Call>();var usize i=0;
 while(i<ir::call_count(body)){
  if(!work::charge(meter,1)){return .err(3);}let c=ir::call_at(body,i);i+=1;
  if(!c.checked || c.kind!=1 && c.kind!=2 || c.target_module>=input::count(p) || c.target_declaration==0 || c.target_declaration>syntax::declaration_count(input::source(p,c.target_module))){return .err(103);}
  let target=declarations::Row{.module_index=c.target_module,.declaration=c.target_declaration-1};let d=syntax::declaration_at(input::source(p,target.module_index),target.declaration);
  if(d.kind!=3 && d.kind!=9 || d.generic_parameters!=0 || d.kind==9 && input::origin(p,target.module_index).owner!=input::origin(p,row.module_index).owner){return .err(103);}
  if(v::len<Call>(ref result)>=65536){return .err(3);}if(!v::push<Call>(ref mut result,Call{.target,.start=c.start,.end=c.end,.external=d.kind==9}).is_ok){return .err(2);}
  var usize j=v::len<Call>(ref result)-1;let value=*v::at<Call>(ref result,j);
  while(j>0){let prior=*v::at<Call>(ref result,j-1);if(!before(p,value,prior,meter)?){break;}{let slot=v::at_mut<Call>(ref mut result,j);*slot=prior;}{let slot=v::at_mut<Call>(ref mut result,j-1);*slot=value;}j-=1;}
 }return .ok(result);
}
pub resource Summaries{bytes::Buffer text;usize calls;usize external;usize functions;cleanup{}}
pub fn ref u8[] text(ref Summaries p){return bytes::view(ref p.text);}
pub fn usize count(ref Summaries p){return p.calls;}
pub fn usize external_count(ref Summaries p){return p.external;}
pub fn usize function_count(ref Summaries p){return p.functions;}
pub fn Result<Summaries,u32> summaries(ref input::Input p,ref ir::Program program,ref signatures::Program metadata,ref v::Vec<declarations::Row> rows,ref mut usize meter){
 var out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;var usize total=0;var usize external=0;var usize functions=0;var usize i=0;
 while(i<v::len<declarations::Row>(rows)){
  if(!work::charge(meter,1)){return .err(3);}let row=*v::at<declarations::Row>(rows,i);i+=1;
  if(syntax::declaration_at(input::source(p,row.module_index),row.declaration).kind!=3){continue;}
  let calls=calls(p,program,row,meter)?;if(functions>0){discard w::raw(ref mut out,b",",meter)?;}functions+=1;
  discard w::raw(ref mut out,b"{\"function\":",meter)?;discard declarations::key(ref mut out,p,row,meter)?;
  discard w::raw(ref mut out,b",\"provenance\":\"compiler-observed\",\"calls\":[",meter)?;var usize j=0;
  while(j<v::len<Call>(ref calls)){
   let call=*v::at<Call>(ref calls,j);if(j>0){discard w::raw(ref mut out,b",",meter)?;}
   discard w::raw(ref mut out,b"{\"span\":",meter)?;discard declarations::span(ref mut out,p,row.module_index,call.start,call.end,meter)?;
   discard w::raw(ref mut out,if(call.external){b",\"target_kind\":\"external\",\"target\":"}else{b",\"target_kind\":\"resolved\",\"target\":"},meter)?;
   discard declarations::key(ref mut out,p,call.target,meter)?;
   if(call.external){
    let id=signatures::identity(metadata,call.target.module_index,call.target.declaration+1);if(id==0 || !signatures::at(metadata,id).foreign){return .err(103);}
    discard w::raw(ref mut out,b",\"certainty\":\"opaque\",\"abi\":\"C\",",meter)?;
    discard w::pair(ref mut out,b"link_name",signatures::linker_name(metadata,id),meter)?;discard w::raw(ref mut out,b"}",meter)?;external+=1;
   }else{discard w::raw(ref mut out,b",\"certainty\":\"exact\",\"abi\":\"\",\"link_name\":\"\"}",meter)?;}
   total+=1;j+=1;
  }
  discard w::raw(ref mut out,b"],\"trap_categories\":[],\"unsafe_sites\":[],\"move_sites\":[],\"drop_sites\":[],\"allocation_sites\":[],\"return_error_sites\":[],\"borrows\":[]}",meter)?;
 }discard w::raw(ref mut out,b"]",meter)?;return .ok(Summaries{.text=out,.calls=total,.external,.functions});
}
