module project_atlas_calls;
import project_atlas_write as w;
import project_atlas_declarations as declarations;
import project_compiler_input as input;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_atlas_model as model;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
// The compiler may materialize an ordinary source function in a concrete frame.
// Every source declaration is separately admitted as nongeneric; frame/instance
// handles never become Atlas identities or imply generic-map support.
pub struct Call{declarations::Row target;usize start;usize end;}
fn Result<bool,u32> before(ref input::Input p,Call a,Call b,ref mut usize meter){if(!work::charge(meter,1)){return .err(3);}if(a.start!=b.start){return .ok(a.start<b.start);}if(a.end!=b.end){return .ok(a.end<b.end);}var bytes::Buffer left=bytes::empty();var bytes::Buffer right=bytes::empty();discard declarations::key(ref mut left,p,a.target,meter)?;discard declarations::key(ref mut right,p,b.target,meter)?;return .ok(model::before(bytes::view(ref left),bytes::view(ref right)));}
fn Result<v::Vec<Call>,u32> calls(ref input::Input p,ref ir::Program program,declarations::Row row,ref mut usize meter){var usize frame=1;while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);if((f.kind==1 || f.kind==3) && f.module_index==row.module_index && f.declaration==row.declaration+1){break;}frame+=1;}if(frame>ir::count(program)){return .err(9);}let body=ir::at(program,frame);var v::Vec<Call> result=v::empty<Call>();var usize i=0;while(i<ir::call_count(body)){if(!work::charge(meter,1)){return .err(3);}let c=ir::call_at(body,i);if(!c.checked || c.kind!=1 && c.kind!=2 || c.target_module>=input::count(p) || c.target_declaration==0 || c.target_declaration>syntax::declaration_count(input::source(p,c.target_module))){return .err(103);}let target=declarations::Row{.module_index=c.target_module,.declaration=c.target_declaration-1};if(syntax::declaration_at(input::source(p,target.module_index),target.declaration).kind!=3){return .err(103);}if(!v::push<Call>(ref mut result,Call{.target,.start=c.start,.end=c.end}).is_ok){return .err(2);}var usize j=i;let value=*v::at<Call>(ref result,i);while(j>0){let prior=*v::at<Call>(ref result,j-1);if(!before(p,value,prior,meter)?){break;}{let slot=v::at_mut<Call>(ref mut result,j);*slot=prior;}{let slot=v::at_mut<Call>(ref mut result,j-1);*slot=value;}j-=1;}i+=1;}return .ok(result);}
pub resource Summaries{bytes::Buffer text;usize calls;cleanup{}}
pub fn ref u8[] text(ref Summaries p){return bytes::view(ref p.text);}
pub fn usize count(ref Summaries p){return p.calls;}
pub fn Result<Summaries,u32> summaries(ref input::Input p,ref ir::Program program,ref v::Vec<declarations::Row> rows,ref mut usize meter){var bytes::Buffer out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;var usize total=0;var usize i=0;while(i<v::len<declarations::Row>(rows)){let row=*v::at<declarations::Row>(rows,i);let calls=calls(p,program,row,meter)?;if(i>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{\"function\":",meter)?;discard declarations::key(ref mut out,p,row,meter)?;discard w::raw(ref mut out,b",\"provenance\":\"compiler-observed\",\"calls\":[",meter)?;var usize j=0;while(j<v::len<Call>(ref calls)){let call=*v::at<Call>(ref calls,j);if(j>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{\"span\":",meter)?;discard declarations::span(ref mut out,p,row.module_index,call.start,call.end,meter)?;discard w::raw(ref mut out,b",\"target_kind\":\"resolved\",\"target\":",meter)?;discard declarations::key(ref mut out,p,call.target,meter)?;discard w::raw(ref mut out,b",\"certainty\":\"exact\",\"abi\":\"\",\"link_name\":\"\"}",meter)?;total+=1;j+=1;}discard w::raw(ref mut out,b"],\"trap_categories\":[],\"unsafe_sites\":[],\"move_sites\":[],\"drop_sites\":[],\"allocation_sites\":[],\"return_error_sites\":[],\"borrows\":[]}",meter)?;i+=1;}discard w::raw(ref mut out,b"]",meter)?;return .ok(Summaries{.text=out,.calls=total});}
