module tests::generic_aggregate_plan;
import project_source_plan as project;
import generic_aggregate_types as aggregates;
import module_types as types;
import pointer_types as pointers;
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
pub fn void verify(ref u8[] wire){verify_mode(wire,false);}
@effects(console) pub fn void verify_types(ref u8[] wire){verify_mode(wire,true);}
@effects(console) fn void verify_mode(ref u8[] wire,bool materialized){
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
  work=16777216;assert(switch(project::build_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10}});
  if(!materialized){c+=1;continue;}if(c==0){verify_rejections(ref sources,ref owners,ref imports,ref table);}work=0;let concrete=switch(aggregates::build(actual,ref sources,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(label);discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);assert(false);return;}};verify_concrete(ref concrete,ref sources,wire,ref mut at);if(c==0){verify_bounds(ref concrete,ref sources);}
  c+=1;
 }assert(at==wire.len);
}

@effects(console)
fn bytes::Buffer input(){var out=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{assert(false);return out;}};if(n==0){return out;}let ref u8[] raw=ref chunk;assert(bytes::len(ref out)<=1048576-n && bytes::append(ref mut out,raw[..n]).is_ok);}return out;}
@effects(console) pub fn i32 run(){let packet=input();verify(bytes::view(ref packet));return 0;}

fn ref u8[] scalar(u32 id){if(id==1){return b"i32";}if(id==2){return b"i64";}if(id==3){return b"u8";}if(id==4){return b"u32";}if(id==5){return b"usize";}if(id==6){return b"bool";}if(id==32){return b"i8";}if(id==33){return b"i16";}if(id==34){return b"isize";}if(id==35){return b"u16";}if(id==36){return b"u64";}if(id==37){return b"f32";}if(id==38){return b"f64";}return b"void";}
fn void render(ref aggregates::Prepared p,usize id,ref mut bytes::Buffer out,usize depth){
 assert(depth<64);let t=types::at(aggregates::type_records(p),id);
 if(t.kind==1){assert(bytes::append(out,scalar(t.scalar)).is_ok);return;}
 if(t.kind==20 || t.kind==21){var usize i=0;while(i<aggregates::count(p)){let a=aggregates::at(p,i);if(a.type==id){let plan=aggregates::source_plan(p);let n=plan::at(plan,a.node-1);assert(bytes::append(out,plan::text(plan,n.id)).is_ok);return;}i+=1;}assert(false);return;}
 if(t.kind==12){assert(bytes::append(out,if(t.first!=0 && t.scalar==1 || t.first==0 && pointers::mutable(t.scalar)){b"ptr mut "}else{b"ptr "}).is_ok);if(t.first!=0){render(p,t.first,out,depth+1);}else{assert(bytes::append(out,scalar(pointers::inner(t.scalar))).is_ok);}return;}
 assert(t.kind==5 || t.kind==6 || t.kind==7 || t.kind==8 || t.kind==16);if(t.kind!=16){assert(bytes::append(out,if(t.kind==5){b"Option<"}else if(t.kind==6){b"Result<"}else if(t.kind==7){b"ref "}else{b"ref mut "}).is_ok);}render(p,t.first,out,depth+1);
 if(t.kind==6){assert(bytes::append(out,b",").is_ok);render(p,t.second,out,depth+1);}if(t.kind==5 || t.kind==6){assert(bytes::append(out,b">").is_ok);}if(t.kind==16){assert(bytes::append(out,b"[").is_ok && bytes::append_usize(out,t.second).is_ok && bytes::append(out,b"]").is_ok);}
}
@effects(console) fn void type_matches(ref aggregates::Prepared p,usize id,ref u8[] expected){var out=bytes::empty();render(p,id,ref mut out,0);if(!bytes::equal(bytes::view(ref out),expected)){discard io::write_stderr(" type mismatch actual ");discard io::write_stderr_bytes(bytes::view(ref out));discard io::write_stderr(" expected ");discard io::write_stderr_bytes(expected);}assert(bytes::equal(bytes::view(ref out),expected));}
fn usize find_instance(ref aggregates::Prepared p,ref u8[] id){let plan=aggregates::source_plan(p);var usize i=0;while(i<aggregates::count(p)){let a=aggregates::at(p,i);let n=plan::at(plan,a.node-1);if(bytes::equal(plan::text(plan,n.id),id)){return i;}i+=1;}assert(false);return 0;}
@effects(console) fn void verify_concrete(ref aggregates::Prepared p,ref v::Vec<syntax::Parsed> sources,ref u8[] wire,ref mut usize at){
 let count=number(wire,at);assert(aggregates::count(p)==count);var usize i=0;while(i<count){let id=field(wire,at);let a=aggregates::at(p,find_instance(p,id));assert(a.module_index==number(wire,at));let fields=number(wire,at);assert(a.count==fields);let source=v::at<syntax::Parsed>(sources,a.module_index);var usize j=0;
  while(j<fields){let f=aggregates::field_at(p,a.first+j);let original=syntax::field_at(source,f.source_index);let name=field(wire,at);let start=number(wire,at);let end=number(wire,at);if(f.start!=start || f.end!=end){discard io::write_stderr_usize(f.start);discard io::write_stderr_usize(start);discard io::write_stderr_usize(f.end);discard io::write_stderr_usize(end);}assert(bytes::equal(syntax::source_view(source)[original.name_start..original.name_end],name) && f.start==start && f.end==end);type_matches(p,f.type,field(wire,at));j+=1;}i+=1;
 }
 let functions=number(wire,at);assert(functions==4);i=0;let plan=aggregates::source_plan(p);let c=prepared::templates(plan::input(plan));
 while(i<functions){let id=field(wire,at);var usize found=0;var usize j=0;while(j<plan::count(plan)){let n=plan::at(plan,j);if(bytes::equal(plan::text(plan,n.id),id)){found=j+1;break;}j+=1;}assert(found!=0);let n=plan::at(plan,found-1);let t=templates::template_at(c,n.template_index);let source=v::at<syntax::Parsed>(sources,t.module_index);let d=syntax::declaration_at(source,t.declaration);var usize result=0;j=0;
  while(j<aggregates::use_count(p)){let u=aggregates::use_at(p,j);if(u.node==found && u.kind==1){assert(result==0);result=u.type;}j+=1;}assert(result!=0);type_matches(p,result,field(wire,at));let count=number(wire,at);var usize parameter_count=0;var usize entry=d.entry_start;while(entry<d.entry_start+d.entry_count){if(syntax::entry_at(source,entry).kind==1){parameter_count+=1;}entry+=1;}assert(count==parameter_count);var usize k=0;
  while(k<count){var usize ordinal=0;entry=d.entry_start;while(entry<d.entry_start+d.entry_count){if(syntax::entry_at(source,entry).kind==1){if(ordinal==k){break;}ordinal+=1;}entry+=1;}var usize type=0;j=0;while(j<aggregates::use_count(p)){let u=aggregates::use_at(p,j);if(u.node==found && u.kind==2 && u.source_index==entry){assert(type==0);type=u.type;}j+=1;}assert(type!=0);type_matches(p,type,field(wire,at));k+=1;}i+=1;
 }
 var usize constructors=0;i=0;while(i<aggregates::use_count(p)){let u=aggregates::use_at(p,i);if(u.kind==4){var bool found=false;var usize j=0;while(j<aggregates::count(p)){if(aggregates::at(p,j).type==u.type){found=true;}j+=1;}assert(found);constructors+=1;}i+=1;}assert(constructors==5);
 assert(types::validate(aggregates::declarations(p),aggregates::type_records(p))!=0);
}

fn void verify_bounds(ref aggregates::Prepared p,ref v::Vec<syntax::Parsed> sources){
 let plan=aggregates::source_plan(p);var usize selected=0;while(selected<aggregates::count(p) && plan::at(plan,aggregates::at(p,selected).node-1).count!=1){selected+=1;}assert(selected<aggregates::count(p));let a=aggregates::at(p,selected);let c=aggregates::declarations(p);
 var graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);var args=v::empty<usize>();assert(v::push<usize>(ref mut args,1).is_ok);
 var usize i=1;while(i<4094){assert(v::push<types::Type>(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=i}).is_ok);i+=1;}
 let success=types::aggregate(c,a.module_index,a.declaration,ref args,ref mut graph);assert(success.problem.code==0 && types::count(ref graph)==4096);let repeated=types::aggregate(c,a.module_index,a.declaration,ref args,ref mut graph);assert(repeated.problem.code==0 && repeated.type==success.type && types::count(ref graph)==4096);
 discard v::pop<types::Type>(ref mut graph);discard v::pop<types::Type>(ref mut graph);assert(types::count(ref graph)==4094);assert(v::push<types::Type>(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=4094}).is_ok);
 let failed=types::aggregate(c,a.module_index,a.declaration,ref args,ref mut graph);assert(failed.problem.code==10 && types::count(ref graph)==4095 && types::at(ref graph,4095).kind==16);
 assert(v::push<types::Type>(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=4095}).is_ok);let full=types::aggregate(c,a.module_index,a.declaration,ref args,ref mut graph);assert(full.problem.code==10 && types::count(ref graph)==4096);
 let empty=v::empty<usize>();assert(types::aggregate(c,a.module_index,a.declaration,ref empty,ref mut graph).problem.code==4502 && types::count(ref graph)==4096);
 let source=v::at<syntax::Parsed>(sources,a.module_index);let foreign=v::at<syntax::Parsed>(sources,(a.module_index+1)%v::len<syntax::Parsed>(sources));assert(types::aggregate_resolve(c,foreign,a.module_index,0,1,ref empty,ref mut graph).problem.code==12 && types::count(ref graph)==4096);
 assert(types::aggregate_resolve(c,source,a.module_index,0,0,ref empty,ref mut graph).problem.code==9 && types::count(ref graph)==4096);
}

fn void verify_rejections(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table){
 var usize work=0;let stale=switch(project::build_aggregate_sources(sources,owners,imports,table,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var changed=v::empty<syntax::Parsed>();var usize i=0;
 while(i<v::len<syntax::Parsed>(sources)){var raw=bytes::empty();assert(bytes::append(ref mut raw,syntax::source_view(v::at<syntax::Parsed>(sources,i))).is_ok);if(i==0){assert(bytes::append(ref mut raw,b"\n").is_ok);}assert(v::push<syntax::Parsed>(ref mut changed,parsed(bytes::view(ref raw))).is_ok);i+=1;}
 work=0;assert(switch(aggregates::build(stale,ref changed,ref mut work)){case .ok(_):{false}case .err(p):{p.code==12}});
 work=0;let bounded=switch(project::build_aggregate_sources(sources,owners,imports,table,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=16777216;assert(switch(aggregates::build(bounded,sources,ref mut work)){case .ok(_):{false}case .err(p):{p.code==10}});
 var one=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut one,parsed(b"module root;struct Box<T>{Missing value;}fn void main(){let box=Box<i32>{.value=1};}")).is_ok);var digits=v::empty<usize>();assert(v::push<usize>(ref mut digits,1).is_ok);let coordinate=coords(ref digits,0);var owner=v::empty<usize>();assert(v::push<usize>(ref mut owner,0).is_ok);let no_imports=v::empty<catalog::Binding>();work=0;
 let unresolved=switch(project::build_aggregate_sources(ref one,ref owner,ref no_imports,ref coordinate,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;assert(switch(aggregates::build(unresolved,ref one,ref mut work)){case .ok(_):{false}case .err(p):{p.code==5}});
}
