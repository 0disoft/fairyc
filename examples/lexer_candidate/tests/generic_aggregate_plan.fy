module tests::generic_aggregate_plan;
import project_source_plan as project;
import generic_source_plan as plan;
import generic_source_plan_input as prepared;
import generic_source_catalog as templates;
import generic_concrete_types as concrete;
import module_public_templates as public_templates;
import module_generic_interface as public_model;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize out=0;var usize i=0;while(i<8){out=out*256+(wire[*at+i] as usize);i+=1;}*at+=8;return out;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let start=*at;*at+=n;return wire[start..*at];}
fn coordinates::Coordinates coords(ref v::Vec<usize> digits,usize shift){var names=bytes::empty();var rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<v::len<usize>(digits)){let digit=*v::at<usize>(digits,i);assert(digit>=1 && digit<=3);let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize n=0;while(n<64){assert(bytes::append(ref mut names,if(digit==1){b"1"}else if(digit==2){b"2"}else{b"3"}).is_ok);n+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=digit-1+shift,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}return coordinates::assemble(names,rows);}

@effects(console)
pub fn void verify(ref u8[] wire){
 assert(wire.len>=16 && bytes::equal(wire[..8],b"FYGAGG01"));var usize at=8;let cases=number(wire,ref mut at);var usize c=0;
 while(c<cases){let label=field(wire,ref mut at);let total=number(wire,ref mut at);assert(total>=1 && total<=3);var sources=v::empty<syntax::Parsed>();var owners=v::empty<usize>();var digits=v::empty<usize>();var imports=v::empty<catalog::Binding>();let usize shift=if(c==2){19}else{0};var usize i=0;
  while(i<total){let digit=number(wire,ref mut at);assert(v::push<usize>(ref mut digits,digit).is_ok && v::push<usize>(ref mut owners,digit-1+shift).is_ok);let raw=field(wire,ref mut at);assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);let count=number(wire,ref mut at);var usize j=0;
   while(j<count){let alias=field(wire,ref mut at);let p=v::at<syntax::Parsed>(ref sources,i);let d=syntax::declaration_at(p,j+1);assert(d.kind==2 && bytes::equal(alias,syntax::source_view(p)[d.name_start..d.name_end]));assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=i,.declaration=j+2,.target=number(wire,ref mut at)}).is_ok);j+=1;}i+=1;
  }
  let expected=number(wire,ref mut at);let table=coords(ref digits,shift);var usize work=0;
  if(expected==1){assert(switch(project::build_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code!=0}});c+=1;continue;}
  assert(expected==0);assert(switch(project::build_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.stage==2 && e.code==4501}});work=0;
  let actual=switch(project::build_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(label);discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);return;}};
  let roots=number(wire,ref mut at);assert(plan::root_count(ref actual)==roots);i=0;while(i<roots){let n=plan::at(ref actual,plan::root_at(ref actual,i)-1);assert(bytes::equal(plan::text(ref actual,n.id),field(wire,ref mut at)));i+=1;}
  let nodes=number(wire,ref mut at);assert(plan::count(ref actual)==nodes);i=0;
  while(i<nodes){let n=plan::at(ref actual,plan::ordered_at(ref actual,i)-1);let expected_id=field(wire,ref mut at);if(!bytes::equal(plan::text(ref actual,n.id),expected_id)){discard io::write_stderr_bytes(label);discard io::write_stderr_bytes(plan::text(ref actual,n.key));discard io::write_stderr(" expected ");discard io::write_stderr_bytes(expected_id);discard io::write_stderr(" actual ");discard io::write_stderr_bytes(plan::text(ref actual,n.id));assert(false);}
   let catalog=prepared::templates(plan::input(ref actual));let row=templates::template_at(catalog,n.template_index);assert(bytes::equal(templates::text(catalog,row.id),field(wire,ref mut at)) && bytes::equal(plan::text(ref actual,n.key),field(wire,ref mut at)));let count=number(wire,ref mut at);assert(n.count==count && n.state==2);var usize j=0;
   while(j<count){let parameter=templates::parameter_at(catalog,row.first+j);let types=prepared::types(plan::input(ref actual));let type=concrete::at(types,n.arguments[j]-1);assert(bytes::equal(templates::text(catalog,parameter.name),field(wire,ref mut at)) && bytes::equal(concrete::text(types,type.id),field(wire,ref mut at)));j+=1;}
   let edges=number(wire,ref mut at);assert(n.dependencies==edges);j=0;while(j<edges){let target=plan::dependency_at(ref actual,n.first+j);let child=plan::at(ref actual,target-1);assert(bytes::equal(plan::text(ref actual,child.id),field(wire,ref mut at)));var bool precedes=false;var usize k=0;while(k<i){if(plan::ordered_at(ref actual,k)==target){precedes=true;}k+=1;}assert(precedes);j+=1;}i+=1;
  }
  let catalog=prepared::templates(plan::input(ref actual));let dependencies=prepared::dependencies(plan::input(ref actual));work=0;let public=switch(public_templates::collect_records(catalog,dependencies,1+shift,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let count=number(wire,ref mut at);assert(public_model::count(ref public)==count);i=0;
  while(i<count){let p=public_model::at(ref public,i);assert(bytes::equal(public_model::text(ref public,p.name),field(wire,ref mut at)) && bytes::equal(public_model::text(ref public,p.closure),field(wire,ref mut at)));i+=1;}
  work=16777216;assert(switch(project::build_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10}});c+=1;
 }assert(at==wire.len);
}

@effects(console)
fn bytes::Buffer input(){var out=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{assert(false);return out;}};if(n==0){return out;}let ref u8[] raw=ref chunk;assert(bytes::len(ref out)<=1048576-n && bytes::append(ref mut out,raw[..n]).is_ok);}return out;}
@effects(console) pub fn i32 run(){let packet=input();verify(bytes::view(ref packet));return 0;}
