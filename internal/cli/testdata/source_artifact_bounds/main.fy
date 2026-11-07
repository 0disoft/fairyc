module main;
import artifact_regression as regression;
import candidate_limits as limits;
import generic_package_bindings as bindings;
import module_source_artifact_model as model;
import module_source_artifact_encode as encoder;
import module_source_artifact_decode as decoder;
import module_source_artifact_rebind as rebinding;
import module_source_provenance as provenance;
import module_project_artifact_current as mapping;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import candidate_artifact_wire as wire;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import std::process as process;
fn syntax::Parsed source(usize index){
 var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module m").is_ok && bytes::append_usize(ref mut raw,index).is_ok && bytes::append_byte(ref mut raw,59).is_ok);
 let s=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return source(index);}};
 let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return source(index);}};assert(syntax::problem_count(ref p)==0);return p;
}
fn coordinates::Coordinates coordinate(){
 var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize i=0;while(i<64){assert(bytes::append_byte(ref mut names,49).is_ok);i+=1;}
 var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=77}).is_ok);return coordinates::assemble(names,rows);
}
@effects(console) fn void boundary(usize count){
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;
 while(i<count){assert(v::push<syntax::Parsed>(ref mut sources,source(i)).is_ok && v::push<usize>(ref mut owners,0).is_ok);i+=1;}
 let imports=v::empty<catalog::Binding>();let table=coordinate();var usize work=0;
 let result=bindings::build(ref sources,ref owners,ref imports,ref table,ref mut work);
 if(count>limits::source_files()){
  assert(switch(result){case .ok(_):{false}case .err(e):{e.code==9 && work==0}});
  var v::Vec<bindings::Module> invalid=v::empty<bindings::Module>();i=0;while(i<count){assert(v::push<bindings::Module>(ref mut invalid,bindings::Module{.owner=0,.namespace=bindings::Text{.start=0,.end=0},.path=bindings::Text{.start=0,.end=0},.first=0,.count=0,.source_size=0,.digest=[0;8]}).is_ok);i+=1;}
  work=0;assert(switch(model::assemble(bytes::empty(),invalid,v::empty<bindings::Import>(),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}});return;
 }
 let original=switch(result){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;
 let saved=switch(model::extract(ref original,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(model::count(ref saved)==count);
 work=0;let data=switch(encoder::encode(ref saved,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;
 let restored=switch(decoder::decode(wire::content(ref data),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(model::count(ref restored)==count);
 if(count==1024){
  let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;assert(provenance::current(ref original,ref sources,ref metadata,ref table,ref mut work)==0);
  var v::Vec<syntax::Parsed> reverse=v::empty<syntax::Parsed>();i=0;while(i<count){assert(v::push<syntax::Parsed>(ref mut reverse,source(count-1-i)).is_ok);i+=1;}
  work=0;let current=switch(rebinding::rebind(ref restored,ref reverse,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};
  assert(bindings::count(ref current)==count && bytes::equal(bindings::text(ref current,bindings::module_at(ref current,0).path),b"m1023"));
  mapping::artifact_mapping_probe(ref restored,ref current);
 }
 assert(io::write_usize(count).is_ok && io::write(" sources ok\n").is_ok);
}
@effects(console) fn i32 decode_input(){
 var bytes::Buffer data=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 8;}};if(n==0){break;}let ref u8[] raw=ref chunk;assert(bytes::append(ref mut data,raw[..n]).is_ok);}
 var usize work=0;return switch(decoder::decode(bytes::view(ref data),ref mut work)){case .ok(p):{discard io::write_usize(model::count(ref p));discard io::write("/");discard io::write_usize(model::import_count(ref p));0}case .err(code):{discard io::write_u32(code);7}};
}
@effects(filesystem,console) fn i32 main(){
 if(process::arg_count()>2){return decode_input();}
 assert(limits::source_files()==1024 && limits::project_imports()==8192);
 regression::source_provenance_matches_independent_golden_and_detaches();
 regression::source_provenance_codec_preserves_exact_work_and_rejects_exhaustion();
 regression::source_rebinding_accepts_source_order_and_owner_renumbering();
 regression::source_rebinding_rejects_stale_bytes_provider_and_coordinate_drift();
 regression::source_provenance_rejects_truncation_invalid_spans_and_plumbing();
 assert(io::write("legacy golden/work/reorder/stale/forged ok\n").is_ok);
 boundary(512);boundary(513);boundary(1024);boundary(1025);assert(io::write("1025 refused\n").is_ok);return 0;
}
