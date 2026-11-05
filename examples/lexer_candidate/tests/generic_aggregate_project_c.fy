module tests::generic_aggregate_project_c;
import module_unit_body_ir_model as body_ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as emit;
import module_project_identified_compile as units;
import module_project_identified_ir as linked;
import module_project_types as project_types;
import module_signatures as signatures;
import generic_source_plan as plan;
import module_types as types;
import candidate_limits as limits;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) pub fn void verify(ref body_ir::Program raw,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref plan::Plan original){
 assert(clean::aggregates(body_ir::cleanup_ir(raw)) && !clean::project(body_ir::cleanup_ir(raw)));
 tuple_maps(metadata);
 var usize work=0;
 assert(switch(units::build_partitioned(sources,metadata,original,0,ref mut work)){case .ok(_):{false}case .err(e):{e.stage==1 && e.code==9}});
 assert(switch(units::build_partitioned(sources,metadata,original,limits::source_files()+1,ref mut work)){case .ok(_):{false}case .err(e):{e.stage==1 && e.code==10}});
 let input=switch(units::build_partitioned(sources,metadata,original,1,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate unit ");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);discard io::write_stderr(" module/range ");discard io::write_stderr_usize(e.module_index);discard io::write_stderr("/");discard io::write_stderr_usize(e.first);discard io::write_stderr("..");discard io::write_stderr_usize(e.stop);assert(false);return;}};
 assert(units::count(ref input)==v::len<syntax::Parsed>(sources) && units::file_count(ref input)==v::len<syntax::Parsed>(sources));
 var usize i=0;while(i<units::count(ref input)){let range=units::range_at(ref input,i);assert(range.first==i && range.stop==i+1);i+=1;}
 var u32 mode=0;
 while(mode<2){
  var usize linking=0;let p=switch(linked::build(metadata,ref input,ref mut linking)){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate linking ");discard io::write_stderr_u32(e.code);assert(false);return;}};
  assert(clean::project(body_ir::cleanup_ir(linked::ir_program(ref p))) && clean::aggregates(body_ir::cleanup_ir(linked::ir_program(ref p))));
  let entry=verify_linked(ref p,sources);assert(entry!=0);
  let parts=switch(if(mode==0){emit::partition_project(p,entry,1)}else{emit::partition_project_entry(p,entry,2)}){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate project C ");discard io::write_stderr_u32(e.code);assert(false);return;}};
  assert(emit::project_unit_count(ref parts)>1 && emit::project_work(ref parts)<=limits::project_work());
  if(mode==0){assert(emit::project_selected_frames(ref parts)==emit::project_frames(ref parts));}
  var output=bytes::empty();assert(bytes::append(ref mut output,b"FYCPARTS1\n").is_ok);member(ref mut output,emit::project_header(ref parts));member(ref mut output,emit::project_literals(ref parts));assert(bytes::append_usize(ref mut output,emit::project_unit_count(ref parts)).is_ok && bytes::append(ref mut output,b"\n").is_ok);
  i=0;var usize next=1;while(i<emit::project_unit_count(ref parts)){assert(emit::project_first(ref parts,i)==next && emit::project_stop(ref parts,i)>next && emit::project_stop(ref parts,i)-next<=if(mode==0){1}else{2});next=emit::project_stop(ref parts,i);member(ref mut output,emit::project_source(ref parts,i));i+=1;}assert(next==emit::project_frames(ref parts)+1);
  assert(io::write_usize(bytes::len(ref output)).is_ok && io::write("\n").is_ok && io::write_bytes(bytes::view(ref output)).is_ok);mode+=1;
 }
}
fn usize verify_linked(ref linked::Program named,ref v::Vec<syntax::Parsed> sources){
 let p=linked::ir_program(named);var usize i=1;var usize entry=0;var usize instances=0;var usize work=0;
 while(i<=clean::type_count(body_ir::cleanup_ir(p))){let t=clean::type_at(body_ir::cleanup_ir(p),i);let row=linked::type_at(named,i);let id=linked::identity(named,row);assert((row.node!=0)==(t.kind==20 || t.kind==21));if(row.node!=0){assert(id.len==67);instances+=1;}else{assert(id.len==0);}i+=1;}assert(instances!=0);
 i=1;while(i<=linked::frame_count(named)){let f=clean::frame_at(body_ir::cleanup_ir(p),i);let row=linked::frame_at(named,i);let id=linked::identity(named,row);let source=v::at<syntax::Parsed>(sources,f.module_index);let d=syntax::declaration_at(source,f.declaration-1);assert((row.node!=0)==(d.generic_parameters!=0));if(f.kind==2){assert(bytes::equal(id,linked::identity(named,linked::type_at(named,f.receiver_type))));}if(f.kind==3 && bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"main")){assert(entry==0);entry=i;}let body=body_ir::at(p,i);var usize j=0;while(j<body_ir::call_count(body)){let c=body_ir::call_at(body,j);let target=switch(linked::call_frame(named,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);return 0;}};if(c.kind==2){assert(target!=0 && linked::frame_at(named,target).node!=0);}j+=1;}i+=1;}
 return entry;
}
fn void member(ref mut bytes::Buffer output,ref u8[] data){assert(bytes::append_usize(output,data.len).is_ok && bytes::append(output,b"\n").is_ok && bytes::append(output,data).is_ok);}
// Two valid source-prefix graphs give the same unused tuple different local IDs.
// Its second link must map to the shared tail, never to the other unit's pad.
fn void tuple_maps(ref signatures::Program metadata){
 var inputs=v::empty<types::Graph>();var usize[2] heads=[0;2];var usize[2] tails=[0;2];var usize k=0;
 while(k<2){var graph=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(metadata)){assert(v::push<types::Type>(ref mut graph,signatures::type_at(metadata,i)).is_ok);i+=1;}let a=types::canonical(ref mut graph,types::Type{.kind=1,.scalar=36,.module_index=0,.declaration=0,.first=0,.second=0});let b=types::canonical(ref mut graph,types::Type{.kind=1,.scalar=35,.module_index=0,.declaration=0,.first=0,.second=0});assert(a!=0 && b!=0);if(k==1){assert(types::canonical(ref mut graph,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=a,.second=0})!=0);}tails[k]=types::canonical(ref mut graph,types::Type{.kind=22,.scalar=0,.module_index=0,.declaration=0,.first=a,.second=0});heads[k]=types::canonical(ref mut graph,types::Type{.kind=22,.scalar=0,.module_index=0,.declaration=0,.first=b,.second=tails[k]});assert(tails[k]!=0 && heads[k]!=0);let owned=switch(signatures::own_types(metadata,graph)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<types::Graph>(ref mut inputs,owned).is_ok);k+=1;}
 assert(heads[0]!=heads[1] && tails[0]!=tails[1]);let p=switch(project_types::build(metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(project_types::aggregates(ref p) && project_types::mapped(ref p,0,heads[0])==project_types::mapped(ref p,1,heads[1]) && project_types::mapped(ref p,0,tails[0])==project_types::mapped(ref p,1,tails[1]));assert(project_types::type_at(ref p,project_types::mapped(ref p,0,heads[0])).second==project_types::mapped(ref p,0,tails[0]));
}
