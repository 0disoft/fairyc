module tests::generic_aggregate_public;
import project_compiler_core as core;
import module_project_interfaces as interfaces;
import module_project_identified_ir as checked;
import module_public_interface as ordinary;
import module_public_interface_model as descriptor;
import module_public_type_text as coordinates;
import module_public_type_ids as names;
import module_catalog as catalog;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_project_public_type_order as order;
import module_signatures as signatures;
import module_types as types;
import module_state_layout as budget;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){assert(n<=72057594037927935);n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let first=*at;*at+=n;return wire[first..*at];}
fn coordinates::Coordinates coords(usize shift){var text=bytes::empty();var rows=v::empty<coordinates::Coordinate>();var usize i=1;while(i<=3){let first=bytes::len(ref text);assert(bytes::append(ref mut text,b"fypc1-sha256-").is_ok);var usize j=0;while(j<64){assert(bytes::append(ref mut text,if(i==1){b"1"}else if(i==2){b"2"}else{b"3"}).is_ok);j+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=i+shift,.start=first,.end=bytes::len(ref text)}).is_ok);i+=1;}return coordinates::assemble(text,rows);}

@effects(console) fn core::Complete build(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table){
 var usize source_work=0;var usize link_work=0;return switch(core::build_complete_aggregate_sources(sources,owners,imports,table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("aggregate Complete ");discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);assert(false);return build(sources,owners,imports,table);}};
}
@effects(console) pub fn void verify(ref u8[] wire){
 assert(wire.len>=16 && bytes::equal(wire[..8],b"FYGIP001"));var usize at=8;let count=number(wire,ref mut at);var usize k=0;
 while(k<count){let label=field(wire,ref mut at);let shift=number(wire,ref mut at);var sources=v::empty<syntax::Parsed>();var owners=v::empty<usize>();var usize[3] modules=[0;3];var usize i=0;
  while(i<3){let digit=number(wire,ref mut at);assert(digit>=1 && digit<=3);modules[digit-1]=i;assert(v::push<usize>(ref mut owners,digit+shift).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(field(wire,ref mut at))).is_ok);i+=1;}
  var imports=v::empty<catalog::Binding>();i=0;while(i<3){if(i==modules[0] || i==modules[1]){assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=i,.declaration=2,.target=if(i==modules[0]){modules[1]}else{modules[2]}}).is_ok);}i+=1;}let table=coords(shift);let reject=number(wire,ref mut at);
  if(reject!=0){var usize a=0;var usize b=0;assert(switch(core::build_complete_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut a,ref mut b)){case .ok(_):{false}case .err(e):{e.code!=0 && (reject!=6 || e.phase==6 && e.code==9)}});k+=1;continue;}
  let complete=build(ref sources,ref owners,ref imports,ref table);let records=core::public_interfaces(ref complete);assert(interfaces::count(records)==3);
  i=0;while(i<3){let row=interfaces::at(records,i);assert(interfaces::owner(row)==number(wire,ref mut at)+shift);var usize work=0;let state=budget::empty();let encoded=switch(ordinary::encode(ref state,interfaces::owner(row),interfaces::descriptor(row),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
   let expected=field(wire,ref mut at);if(!bytes::equal(bytes::view(ref encoded),expected)){discard io::write_stderr_bytes(label);discard io::write_stderr(" descriptor ");discard io::write_stderr_usize(i);}assert(bytes::equal(bytes::view(ref encoded),expected));same(interfaces::ordinary_hash(row),field(wire,ref mut at),b"ordinary hash");same(interfaces::source_hash(row),field(wire,ref mut at),b"source hash");let selected=interfaces::source_specializations(row);let total=number(wire,ref mut at);if(v::len<bytes::Buffer>(selected)!=total){discard io::write_stderr("specialization count ");discard io::write_stderr_usize(v::len<bytes::Buffer>(selected));discard io::write_stderr(" expected ");discard io::write_stderr_usize(total);}assert(v::len<bytes::Buffer>(selected)==total);var usize j=0;while(j<total){same(bytes::view(v::at<bytes::Buffer>(selected,j)),field(wire,ref mut at),b"specialization");j+=1;}i+=1;
  }
  if(k==0){guards(ref sources,ref owners,ref imports,ref table);}k+=1;
 }assert(at==wire.len);
}
fn void guards(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table){
 var usize a=limits::project_work();var usize b=0;assert(switch(core::build_complete_aggregate_sources(sources,owners,imports,table,ref mut a,ref mut b)){case .ok(_):{false}case .err(e):{e.code==10}});a=0;b=limits::project_work();assert(switch(core::build_complete_aggregate_sources(sources,owners,imports,table,ref mut a,ref mut b)){case .ok(_):{false}case .err(e):{e.code!=0}});
 a=0;b=0;assert(switch(core::build_complete_sources(sources,owners,imports,table,ref mut a,ref mut b)){case .ok(_):{false}case .err(e):{e.code!=0}});
 assert(!names::valid_aggregate(b"maker::_fy_generic_g0_short") && !names::valid_aggregate(b"::_fy_generic_g0_0000000000000000000000000000000000000000000000000000000000000000"));
 tuple_ordering(table);
}
// Insert an unused argument tuple before a source scalar. Restoring the current
// signature prefix moves the tuple, so the head's second edge must move with it.
fn void tuple_ordering(ref coordinates::Coordinates table){
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module order;pub fn i32 take(bool value){return 0;}")).is_ok);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var graph=v::empty<types::Type>();var usize integer=0;var usize boolean=0;var usize tail=0;var usize i=1;
 while(i<=signatures::type_count(ref metadata)){let t=signatures::type_at(ref metadata,i);assert(t.kind==1);if(t.scalar==6){assert(integer!=0);assert(v::push<types::Type>(ref mut graph,types::Type{.kind=22,.scalar=0,.module_index=0,.declaration=0,.first=integer,.second=0}).is_ok);tail=v::len<types::Type>(ref graph);}assert(v::push<types::Type>(ref mut graph,t).is_ok);if(t.scalar==1){integer=v::len<types::Type>(ref graph);}if(t.scalar==6){boolean=v::len<types::Type>(ref graph);}i+=1;}
 assert(tail!=0 && boolean!=0 && v::push<types::Type>(ref mut graph,types::Type{.kind=22,.scalar=0,.module_index=0,.declaration=0,.first=boolean,.second=tail}).is_ok);let head=v::len<types::Type>(ref graph);var glue=v::empty<clean::Glue>();i=0;while(i<head){assert(v::push<clean::Glue>(ref mut glue,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);i+=1;}
 let retained=switch(clean::assemble_aggregates(graph,v::empty<clean::Frame>(),glue,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let ordered=switch(order::build(ref retained,ref metadata,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(order::mapped(ref ordered,tail)!=tail && order::at(ref ordered,order::mapped(ref ordered,head)).second==order::mapped(ref ordered,tail));let empty=names::empty();work=0;assert(switch(coordinates::render_records_prepared(ref empty,ref sources,ref metadata,order::records(ref ordered),table,1,order::mapped(ref ordered,head),true,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});
}
@effects(console) pub fn i32 run(){var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{assert(false);return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;assert(bytes::len(ref input)<=1048576-n && bytes::append(ref mut input,raw[..n]).is_ok);}verify(bytes::view(ref input));return 0;}

@effects(console) fn void same(ref u8[] actual,ref u8[] expected,ref u8[] label){if(!bytes::equal(actual,expected)){discard io::write_stderr_bytes(label);discard io::write_stderr(" actual ");discard io::write_stderr_bytes(actual);discard io::write_stderr(" expected ");discard io::write_stderr_bytes(expected);}assert(bytes::equal(actual,expected));}
