module tests::generic_aggregate_partition_c;
import module_unit_body_ir_model as body_ir;
import module_unit_c_emit as emit;
import module_unit_c_names as c_names;
import module_unit_c_aggregate as aggregate;
import module_unit_cleanup_mir_model as clean;
import module_unit_checked_bodies as checking;
import module_unit_body_ir_lower as lower;
import module_project_c_owner as c_owner;
import module_signatures as signatures;
import candidate_limits as limits;
import c_text as w;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}

@effects(console) pub fn void verify(ref body_ir::Program p,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){
 var usize entry=0;var usize cleanup_frame=0;var usize i=1;
 while(i<=body_ir::count(p)){let f=clean::frame_at(body_ir::cleanup_ir(p),i);let source=v::at<syntax::Parsed>(sources,f.module_index);let d=syntax::declaration_at(source,f.declaration-1);if(f.kind==3 && bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"main")){assert(entry==0);entry=i;}if(f.kind==2){cleanup_frame=i;}i+=1;}
 assert(entry!=0 && cleanup_frame!=0);
 assert(clean::aggregates(body_ir::cleanup_ir(p)) && !clean::project(body_ir::cleanup_ir(p)));
 assert(ordinary(parsed(b"module plain;fn i32 main(){return 0;}"))==1001);
 assert(switch(emit::emit_test(p,c_names::empty(),entry,2)){case .ok(_):{false}case .err(e):{e.code==1001}});
 assert(switch(emit::emit_aggregates(p,entry,0,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
 assert(switch(emit::emit_aggregates(p,entry,limits::project_frames()+1,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
 assert(switch(emit::emit_aggregates(p,0,2,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
 assert(switch(emit::emit_aggregates(p,body_ir::count(p)+1,2,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
 assert(switch(emit::emit_aggregates(p,cleanup_frame,2,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
 // Neither tuple metadata nor this private route opens ordinary project emission.
 i=1;while(i<=clean::type_count(body_ir::cleanup_ir(p))){if(clean::type_at(body_ir::cleanup_ir(p),i).kind==22){var text=bytes::empty();var w::State state=w::State{.code=0};assert(!aggregate::type(p,i,ref mut text,ref mut state) && state.code==1004);}i+=1;}
 ordinary_partitions(sources,metadata,entry);
 var u32 mode=0;
 while(mode<2){
  let usize max_frames=if(mode==0){1}else{2};
  let parts=switch(emit::emit_aggregates(p,entry,max_frames,mode==1)){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate partitions ");discard io::write_stderr_u32(e.code);assert(false);return;}};
  let stats=emit::emission_stats(ref parts);assert(stats.frames==body_ir::count(p) && stats.selected!=0 && stats.selected<=stats.frames && stats.work<=limits::project_work());if(mode==0){assert(stats.selected==stats.frames);}
  var output=bytes::empty();assert(bytes::append(ref mut output,b"FYCPARTS1\n").is_ok);member(ref mut output,emit::emission_header(ref parts));member(ref mut output,emit::emission_literals(ref parts));assert(bytes::append_usize(ref mut output,emit::emission_unit_count(ref parts)).is_ok && bytes::append(ref mut output,b"\n").is_ok);
  i=0;var usize next=1;
  while(i<emit::emission_unit_count(ref parts)){let range=emit::emission_range(ref parts,i);assert(range.first==next && range.stop>range.first && range.stop-range.first<=max_frames);next=range.stop;member(ref mut output,emit::emission_source(ref parts,i));i+=1;}
  assert(next==body_ir::count(p)+1);
  assert(io::write_usize(bytes::len(ref output)).is_ok && io::write("\n").is_ok && io::write_bytes(bytes::view(ref output)).is_ok);mode+=1;
 }
 assert(clean::aggregates(body_ir::cleanup_ir(p)) && !clean::project(body_ir::cleanup_ir(p)));
}
fn void member(ref mut bytes::Buffer output,ref u8[] data){assert(bytes::append_usize(output,data.len).is_ok && bytes::append(output,b"\n").is_ok && bytes::append(output,data).is_ok);}
fn u32 ordinary(syntax::Parsed unit){
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,unit).is_ok);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{return e.code;}};
 let checked=switch(checking::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return e.code;}};
 let p=switch(lower::lower_checked(ref sources,ref checked)){case .ok(p):{p}case .err(e):{return e.code;}};
 return switch(emit::emit_aggregates(ref p,1,2,false)){case .ok(_):{0}case .err(e):{e.code}};
}
fn void ordinary_partitions(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize entry){
 let checked=switch(checking::build(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let p=switch(lower::lower_checked(sources,ref checked)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(switch(emit::partition(p,entry,2)){case .ok(_):{false}case .err(e):{e.code==1001}});
 let second=switch(lower::lower_checked(sources,ref checked)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let owned=switch(c_owner::plain(second)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(switch(emit::partition_scoped(owned,c_names::empty(),entry,2,false)){case .ok(_):{false}case .err(e):{e.code==1001}});
}
