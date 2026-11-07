module main;
import module_project_metadata as project;
import module_project_types as graph;
import module_types as types;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_interfaces as interfaces;
import module_unit_cleanup_mir_model as clean;
import module_metadata_artifact_model as model;
import module_metadata_artifact_encode as encoder;
import module_metadata_artifact_decode as decoder;
import module_state_layout as state;
import module_unit_sdk_sources as sdk;
import candidate_artifact_wire as wire;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

// Unique canonical nominal coordinates, with an i32 field in every layout.
// These are detached component fixtures, not source-authenticated executables.
fn v::Vec<types::Type> descriptors(usize count){
 var v::Vec<types::Type> result=v::empty<types::Type>();
 assert(v::push<types::Type>(ref mut result,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
 var usize i=0;while(i<count){assert(v::push<types::Type>(ref mut result,types::Type{.kind=2,.scalar=0,.module_index=0,.declaration=i+1,.first=0,.second=0}).is_ok);i+=1;}return result;
}
fn clean::Program cleaned(usize count){
 var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();var usize i=0;
 while(i<=count){assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);i+=1;}
 return switch(clean::assemble(descriptors(count),v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(_):{assert(false);return cleaned(count);}};
}
fn v::Vec<ir::Layout> layouts(usize start,usize count){
 var v::Vec<ir::Layout> result=v::empty<ir::Layout>();var usize i=0;
 while(i<count){assert(v::push<ir::Layout>(ref mut result,ir::Layout{.value=interfaces::Shape{.type=start+i+2,.module_index=0,.declaration=start+i+1,.field_start=i,.field_count=1,.abilities=7,.repr_c=false,.start=0,.end=1,.cleanup_nonempty=false,.cleanup_effects=0},.name_start=0,.name_end=1}).is_ok);i+=1;}return result;
}
fn v::Vec<interfaces::Field> fields(usize start,usize count){
 var v::Vec<interfaces::Field> result=v::empty<interfaces::Field>();var usize i=0;
 while(i<count){assert(v::push<interfaces::Field>(ref mut result,interfaces::Field{.kind=1,.type=1,.module_index=0,.declaration=start+i+1,.source_index=1,.start=0,.end=1,.name_start=1,.name_end=2,.payload_start=0,.payload_count=0}).is_ok);i+=1;}return result;
}
fn bytes::Buffer names(){return switch(bytes::copy(b"Sv")){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn Result<ir::Program,clean::Problem> program(usize total,usize start,usize count){return ir::assemble(cleaned(total),v::empty<ir::Body>(),layouts(start,count),fields(start,count),v::empty<interfaces::Callable>(),v::empty<usize>(),names(),0);}
fn ir::Program unit(usize total,usize start,usize count){return switch(program(total,start,count)){case .ok(p):{p}case .err(_):{assert(false);return unit(total,start,count);}};}
fn model::Metadata metadata(usize count){return model::project_layout_limit_fixture(layouts(0,count),fields(0,count),names());}
fn u32 rejected(ref u8[] raw,ref clean::Program c){var usize work=0;return switch(decoder::decode(raw,c,0,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
fn void intact(ref model::Metadata p,usize count){
 assert(model::layout_count(p)==count && model::field_count(p)==count && model::callable_count(p)==0 && model::parameter_count(p)==0 && bytes::equal(model::names(p),b"Sv"));var usize i=0;
 while(i<count){let s=model::layout_at(p,i);let f=model::field_at(p,i);assert(s.value.type==i+2 && s.value.declaration==i+1 && s.value.field_start==i && s.value.field_count==1 && s.name_start==0 && s.name_end==1);assert(f.type==1 && f.declaration==i+1 && f.name_start==1 && f.name_end==2);i+=1;}
}
fn void merged(usize count,bool accept){
 var v::Vec<ir::Program> inputs=v::empty<ir::Program>();var v::Vec<graph::Unit> units=v::empty<graph::Unit>();var usize start=0;
 while(start<count){let usize n=if(count-start>512){512}else{count-start};assert(v::push<ir::Program>(ref mut inputs,unit(count,start,n)).is_ok);assert(v::push<graph::Unit>(ref mut units,graph::Unit{.start=0,.count=count+1}).is_ok);start+=n;}
 // Replay a duplicate recipe after the last unique layout, including at 2048.
 assert(v::push<ir::Program>(ref mut inputs,unit(count,0,1)).is_ok);assert(v::push<graph::Unit>(ref mut units,graph::Unit{.start=0,.count=count+1}).is_ok);
 // All unit descriptors were independently verified above. Identity mapping
 // needs no retained suffix, matching the producer's canonical-prefix plan.
 let plan=graph::project_layout_limit_fixture(types::project_layout_limit_fixture(descriptors(count)),units,count+1);
 switch(project::build(ref inputs,ref plan)){case .ok(p):{
  assert(accept && project::layout_count(ref p)==count && project::field_count(ref p)==count);var usize i=0;
  while(i<count){let s=project::layout_at(ref p,i);let f=project::field_at(ref p,i);assert(s.value.type==i+2 && s.value.declaration==i+1 && s.value.field_start==i && f.declaration==i+1);i+=1;}
  var usize unit_index=0;var usize base=0;while(base<count){let usize n=if(count-base>512){512}else{count-base};i=0;while(i<n){assert(project::mapped_field(ref p,unit_index,i+1)==base+i+1);i+=1;}base+=n;unit_index+=1;}assert(project::mapped_field(ref p,unit_index,1)==1);
 }case .err(e):{assert(!accept && e.code==1002);}};
 // The merge borrows inputs: even rejection cannot truncate the final unit.
 assert(ir::layout_count(v::at<ir::Program>(ref inputs,0))==512 && ir::layout_count(v::at<ir::Program>(ref inputs,v::len<ir::Program>(ref inputs)-1))==1);
}
@effects(console) fn void check_case(usize count,ref u8[] golden){
 let accept=count<=2048;let c=cleaned(count);let p=metadata(count);var usize work=0;
 assert(model::check(ref p,ref c,0,ref mut work)==if(accept){0}else{1002});work=0;
 switch(model::assemble(layouts(0,count),fields(0,count),v::empty<interfaces::Callable>(),v::empty<usize>(),names(),ref c,0,ref mut work)){case .ok(q):{assert(accept);intact(ref q,count);}case .err(code):{assert(!accept && code==1002);}};
 work=0;switch(encoder::encode(ref p,ref c,0,ref mut work)){case .ok(encoded):{
  assert(accept && bytes::equal(wire::content(ref encoded),golden));var usize decode_work=0;
  let q=switch(decoder::decode(golden,ref c,0,ref mut decode_work)){case .ok(q):{q}case .err(_):{assert(false);return;}};intact(ref q,count);
  // The last row and every header/table boundary must not become a shorter success.
  assert(rejected(golden[..31],ref c)==1001 && rejected(golden[..32],ref c)==1001 && rejected(golden[..32+count*104-1],ref c)==1001 && rejected(golden[..golden.len-1],ref c)==1001);
  var bytes::Buffer extra=bytes::empty();assert(bytes::append(ref mut extra,golden).is_ok && bytes::append_byte(ref mut extra,0).is_ok && rejected(bytes::view(ref extra),ref c)==1001);
  var usize again_work=0;let again=switch(encoder::encode(ref q,ref c,0,ref mut again_work)){case .ok(q):{q}case .err(_):{assert(false);return;}};assert(bytes::equal(wire::content(ref again),golden));
  // Retain exact independent codec-work refusal semantics.
  var usize exhausted=16777216-decode_work+1;assert(switch(decoder::decode(golden,ref c,0,ref mut exhausted)){case .ok(_):{false}case .err(code):{code==1002}} && exhausted<=16777216);
 }case .err(code):{assert(!accept && code==1002 && rejected(golden,ref c)==1002);}};
 switch(program(count,0,count)){case .ok(q):{
  assert(accept && ir::layout_count(ref q)==count && proof::check(ref q).code==0);work=0;
  let extracted=switch(model::extract(ref q,ref mut work)){case .ok(q):{q}case .err(_):{assert(false);return;}};intact(ref extracted,count);work=0;
  let restored=switch(model::restore(ref extracted,state::empty(),sdk::empty(),cleaned(count),v::empty<ir::Body>(),ref mut work)){case .ok(q):{q}case .err(_):{assert(false);return;}};assert(ir::layout_count(ref restored)==count && proof::check(ref restored).code==0);
 }case .err(e):{assert(!accept && e.code==1002);}};
 merged(count,accept);discard io::write("layouts=");discard io::write_usize(count);discard io::write(if(accept){" admitted exact-roundtrip merge=exact\n"}else{" refused=1002 merge=refused\n"});
}
@effects(console) fn i32 main(){
 var bytes::Buffer input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>4194304-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize work=0;var cursor=switch(wire::cursor(raw,b"FYLIM001",ref mut work)){case .ok(c):{c}case .err(_):{return 2;}};let cases=wire::number_from(raw,ref mut cursor,ref mut work);assert(cases==5);var usize i=0;
 while(i<cases){let count=wire::number_from(raw,ref mut cursor,ref mut work);let n=wire::count_from(raw,ref mut cursor,1048576,1,ref mut work);assert(wire::status(ref cursor)==0);check_case(count,raw[cursor.offset..cursor.offset+n]);cursor.offset+=n;i+=1;}assert(wire::end(raw,ref mut cursor,ref mut work)==0);return 0;
}
