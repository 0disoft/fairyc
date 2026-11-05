module tests::generic_aggregate_layouts;
import project_source_plan as project;
import generic_aggregate_types as aggregates;
import generic_source_plan as plan;
import module_signatures as signatures;
import module_unit_interface_declarations as declarations;
import module_unit_interfaces as interfaces;
import module_unit_interface_work as meter;
import module_unit_shape_contracts as contracts;
import repr_c_declarations as repr;
import generic_source_plan_input as input;
import generic_concrete_types as concrete;
import module_types as types;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let start=*at;*at+=n;return wire[start..*at];}
fn coordinates::Coordinates coords(ref v::Vec<usize> digits,usize shift){var names=bytes::empty();var rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<v::len<usize>(digits)){let digit=*v::at<usize>(digits,i);let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize n=0;while(n<64){assert(bytes::append(ref mut names,if(digit==1){b"1"}else if(digit==2){b"2"}else{b"3"}).is_ok);n+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=digit-1+shift,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}return coordinates::assemble(names,rows);}
fn ref u8[] scalar(u32 id){if(id==1){return b"i32";}if(id==2){return b"i64";}if(id==3){return b"u8";}if(id==4){return b"u32";}if(id==5){return b"usize";}if(id==6){return b"bool";}if(id==32){return b"i8";}if(id==33){return b"i16";}if(id==34){return b"isize";}if(id==35){return b"u16";}if(id==36){return b"u64";}if(id==37){return b"f32";}if(id==38){return b"f64";}return b"void";}
fn void render(ref signatures::Program metadata,ref v::Vec<types::Type> records,usize id,ref mut bytes::Buffer out,usize depth){
 assert(depth<64);let t=types::at(records,id);if(t.kind==1){assert(bytes::append(out,scalar(t.scalar)).is_ok);return;}
 if(t.kind==2 || t.kind==3){let p=signatures::aggregate_source(metadata);let records=input::types(plan::input(aggregates::source_plan(p)));var usize i=0;while(i<concrete::count(records)){let row=concrete::at(records,i);if(row.kind==t.kind && row.module_index==t.module_index && row.declaration==t.declaration){assert(bytes::append(out,concrete::text(records,row.name)).is_ok);return;}i+=1;}assert(false);return;}
 if(t.kind==16){render(metadata,records,t.first,out,depth+1);assert(bytes::append(out,b"[").is_ok);decimal(out,t.second);assert(bytes::append(out,b"]").is_ok);return;}
 if(t.kind==12 && t.first!=0){assert(bytes::append(out,if(t.scalar==0){b"ptr "}else{b"ptr mut "}).is_ok);render(metadata,records,t.first,out,depth+1);return;}
 if(t.kind==20 || t.kind==21){let p=signatures::aggregate_source(metadata);let plan=aggregates::source_plan(p);var usize i=0;while(i<aggregates::count(p)){let a=aggregates::at(p,i);if(a.type==id){let n=plan::at(plan,a.node-1);assert(bytes::append(out,plan::text(plan,n.id)).is_ok);return;}i+=1;}assert(false);return;}
 assert(t.kind==5 || t.kind==6 || t.kind==7 || t.kind==8);assert(bytes::append(out,if(t.kind==5){b"Option<"}else if(t.kind==6){b"Result<"}else if(t.kind==7){b"ref "}else{b"ref mut "}).is_ok);render(metadata,records,t.first,out,depth+1);if(t.kind==6){assert(bytes::append(out,b",").is_ok);render(metadata,records,t.second,out,depth+1);}if(t.kind==5 || t.kind==6){assert(bytes::append(out,b">").is_ok);}
}
@effects(console) fn void matches(ref signatures::Program metadata,ref v::Vec<types::Type> records,usize id,ref u8[] expected){var out=bytes::empty();render(metadata,records,id,ref mut out,0);if(!bytes::equal(bytes::view(ref out),expected)){discard io::write_stderr_bytes(bytes::view(ref out));discard io::write_stderr_bytes(expected);}assert(bytes::equal(bytes::view(ref out),expected));}
fn v::Vec<types::Type> graph(ref signatures::Program p){var out=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(p)){assert(v::push<types::Type>(ref mut out,signatures::type_at(p,i)).is_ok);i+=1;}return out;}

fn void decimal(ref mut bytes::Buffer out,usize value){var u8[20] digits=[0;20];var usize n=0;var usize number=value;while(true){let alphabet=b"0123456789";digits[n]=alphabet[number%10];n+=1;number=number/10;if(number==0){break;}}while(n>0){n-=1;let ref u8[] raw=ref digits;assert(bytes::append(out,raw[n..n+1]).is_ok);}return;}
fn u32 build_fields(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources,ref mut v::Vec<types::Type> records,ref mut v::Vec<interfaces::Shape> shapes,ref mut v::Vec<interfaces::Field> fields,ref mut bytes::Buffer names,ref mut meter::Budget work){
 var pending=v::empty<usize>();var seen=bytes::empty();var usize m=0;
 while(m<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,m);var usize d=1;while(d<syntax::declaration_count(source)){let declaration=syntax::declaration_at(source,d);if((declaration.kind==4 || declaration.kind==6 || declaration.kind==11) && declaration.generic_parameters==0){let checked=contracts::declaration(source,m,d,false,ref mut work.work);if(checked.code!=0){return checked.code;}let resolved=signatures::declaration_type(metadata,m,d+1,records);if(resolved.problem.code!=0){return resolved.problem.code;}}d+=1;}m+=1;}
 var usize i=1;while(i<=types::count(records)){let t=types::at(records,i);if(t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13 || t.kind==20 || t.kind==21){let checked=declarations::shape(sources,metadata,i,records,shapes,fields,names,ref mut pending,ref mut seen,0,v::len<syntax::Parsed>(sources),work);if(checked.code!=0){return checked.code;}}i+=1;}
 return contracts::aggregate_fields(sources,records,shapes,fields,ref mut work.work).code;
}
@effects(console) fn void verify_fields(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources,ref u8[] label,usize expected,ref u8[] wire,ref mut usize at){
 var records=graph(metadata);var shapes=v::empty<interfaces::Shape>();var fields=v::empty<interfaces::Field>();var names=bytes::empty();var meter::Budget work=meter::Budget{.work=0};let code=build_fields(metadata,sources,ref mut records,ref mut shapes,ref mut fields,ref mut names,ref mut work);let actual=code as usize;if(actual!=expected){discard io::write_stderr_bytes(label);discard io::write_stderr_u32(code);}assert(actual==expected);if(expected!=0){return;}
 let count=number(wire,at);if(v::len<interfaces::Shape>(ref shapes)!=count){discard io::write_stderr_bytes(b"shape count ");discard io::write_stderr_usize(v::len<interfaces::Shape>(ref shapes));discard io::write_stderr_usize(count);}assert(v::len<interfaces::Shape>(ref shapes)==count);var usize i=0;
 while(i<count){let expected_name=field(wire,at);let module_index=number(wire,at);var usize selected=v::len<interfaces::Shape>(ref shapes);var usize j=0;while(j<v::len<interfaces::Shape>(ref shapes)){let s=*v::at<interfaces::Shape>(ref shapes,j);var text=bytes::empty();if(s.module_index==module_index){render(metadata,ref records,s.type,ref mut text,0);if(bytes::equal(bytes::view(ref text),expected_name)){selected=j;break;}}j+=1;}if(selected==v::len<interfaces::Shape>(ref shapes)){discard io::write_stderr_bytes(b"not found\n");}assert(selected<v::len<interfaces::Shape>(ref shapes));let s=*v::at<interfaces::Shape>(ref shapes,selected);let begin=number(wire,at);let end=number(wire,at);if(s.start!=begin || s.end!=end){discard io::write_stderr_bytes(b"span ");discard io::write_stderr_usize(s.start);discard io::write_stderr_usize(begin);discard io::write_stderr_usize(s.end);discard io::write_stderr_usize(end);}assert(s.start==begin && s.end==end);let t=types::at(ref records,s.type);assert((t.kind==3 || t.kind==21)==(number(wire,at)==1) && s.repr_c==(number(wire,at)==1) && s.cleanup_nonempty==(number(wire,at)==1));let count=number(wire,at);assert(s.field_count==count);j=0;
  while(j<count){let f=*v::at<interfaces::Field>(ref fields,s.field_start+j);assert(f.kind==0 && f.module_index==s.module_index && f.declaration==s.declaration && f.payload_count==0);assert(bytes::equal(bytes::view(ref names)[f.name_start..f.name_end],field(wire,at)) && f.start==number(wire,at) && f.end==number(wire,at));matches(metadata,ref records,f.type,field(wire,at));j+=1;}i+=1;
 }
}
@effects(console) fn void verify(ref u8[] wire){
 assert(wire.len>=16 && bytes::equal(wire[..8],b"FYGFLD01"));var usize at=8;let cases=number(wire,ref mut at);var usize k=0;
 while(k<cases){let label=field(wire,ref mut at);assert(number(wire,ref mut at)==3);var sources=v::empty<syntax::Parsed>();var digits=v::empty<usize>();var owners=v::empty<usize>();var usize[3] modules=[0;3];let usize shift=if(k==1){19}else{0};var usize i=0;
  while(i<3){let digit=number(wire,ref mut at);assert(digit>=1 && digit<=3);modules[digit-1]=i;assert(v::push<usize>(ref mut digits,digit).is_ok && v::push<usize>(ref mut owners,digit-1+shift).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(field(wire,ref mut at))).is_ok);i+=1;}
  var imports=v::empty<catalog::Binding>();i=0;while(i<3){let digit=*v::at<usize>(ref digits,i);if(digit!=3){assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=i,.declaration=2,.target=modules[digit]}).is_ok);}i+=1;}
  let table=coords(ref digits,shift);var usize work=0;let original=switch(project::build_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(label);discard io::write_stderr_u32(e.code);assert(false);return;}};
  let p=switch(aggregates::build(original,ref sources,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(label);discard io::write_stderr_u32(e.code);assert(false);return;}};let expected=number(wire,ref mut at);
  let metadata=switch(signatures::build_aggregates(ref sources,p,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(label);discard io::write_stderr_u32(e.code);assert(false);return;}};verify_fields(ref metadata,ref sources,label,expected,wire,ref mut at);if(k==0){guards(ref metadata,ref sources);}k+=1;
 }assert(at==wire.len);
}
@effects(console) pub fn i32 run(){var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{assert(false);return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;assert(bytes::len(ref input)<=1048576-n && bytes::append(ref mut input,raw[..n]).is_ok);}verify(bytes::view(ref input));return 0;}

fn u32 shape_failure(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources,ref mut v::Vec<types::Type> records,usize id,usize initial){
 var shapes=v::empty<interfaces::Shape>();var fields=v::empty<interfaces::Field>();var names=bytes::empty();var pending=v::empty<usize>();var seen=bytes::empty();var meter::Budget work=meter::Budget{.work=initial};let code=declarations::shape(sources,metadata,id,records,ref mut shapes,ref mut fields,ref mut names,ref mut pending,ref mut seen,0,v::len<syntax::Parsed>(sources),ref mut work).code;assert(code!=0 && v::len<interfaces::Shape>(ref shapes)==0 && v::len<interfaces::Field>(ref fields)==0 && bytes::len(ref names)==0);return code;
}
fn void guards(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources){
 var records=graph(metadata);var usize id=0;var usize i=1;while(i<=types::count(ref records)){if(types::at(ref records,i).kind==20){id=i;break;}i+=1;}assert(id!=0);let t=types::at(ref records,id);var usize work=0;assert(contracts::declaration(v::at<syntax::Parsed>(sources,t.module_index),t.module_index,t.declaration-1,false,ref mut work).code==2);
 let invalid=types::count(ref records)+1;assert(shape_failure(metadata,sources,ref mut records,0,0)==12 && shape_failure(metadata,sources,ref mut records,invalid,0)==12);
 assert(shape_failure(metadata,sources,ref mut records,id,1048576)==10);
 {let slot=v::at_mut<types::Type>(ref mut records,id-1);slot.kind=21;}assert(shape_failure(metadata,sources,ref mut records,id,0)==12);{let slot=v::at_mut<types::Type>(ref mut records,id-1);*slot=t;}
 var changed=v::empty<syntax::Parsed>();i=0;while(i<v::len<syntax::Parsed>(sources)){var raw=bytes::empty();assert(bytes::append(ref mut raw,syntax::source_view(v::at<syntax::Parsed>(sources,i))).is_ok);if(i==t.module_index){assert(bytes::append(ref mut raw,b"\n").is_ok);}assert(v::push<syntax::Parsed>(ref mut changed,parsed(bytes::view(ref raw))).is_ok);i+=1;}assert(shape_failure(metadata,ref changed,ref mut records,id,0)==12);
 let valid=parsed(b"module probe;@repr(C) struct Cell<T>{T value;}");assert(repr::read(ref valid,1).code==5106 && repr::read_template(ref valid,1).repr_c);
 let malformed=parsed(b"module probe;@repr(C,C) struct Cell<T>{T value;}");assert(repr::read_template(ref malformed,1).code==5106);
 let reserved=parsed(b"module probe;@repr(C) struct Cell<T>{T INT32_MAX;}");assert(repr::read_template(ref reserved,1).code==5106);
}
