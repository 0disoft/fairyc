module main;
import syntax as syntax;
import scanner as scanner;
import module_unit_body_ast as ast;
import bodies as entries;
import body_state as state;
import types as types;
import records as records;
import attributes as attrs;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn void check(usize id,bool ok){
 if(!ok){discard io::write_stderr("entry lookup check ");discard io::write_stderr_usize(id);discard io::write_stderr("\n");assert(false);}
}
@effects(console) fn void append(ref mut bytes::Buffer raw,ref u8[] text){check(1,bytes::append(raw,text).is_ok);}
@effects(console) fn syntax::Parsed parsed(bytes::Buffer raw){
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(2,false);return parsed(bytes::empty());}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{check(3,false);return parsed(bytes::empty());}};
 check(4,syntax::problem_count(ref p)==0 && !syntax::closure_prepared(ref p));return p;
}
@effects(console) fn syntax::Parsed text(ref u8[] raw){
 let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{check(5,false);return text(raw);}};return parsed(data);
}
@effects(console) fn ast::Ast built(ref syntax::Parsed file,usize declaration,ref mut usize work){
 return switch(ast::build(file,declaration,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);check(6,false);return built(file,declaration,work);}};
}
// Expectations are source records, not another implementation of the ID search.
@effects(console) fn void entry(syntax::Entry actual,syntax::Entry want){
 check(10,actual.kind==want.kind && actual.start==want.start && actual.end==want.end);
 check(11,actual.name_start==want.name_start && actual.name_end==want.name_end);
 check(12,actual.type_start==want.type_start && actual.type_end==want.type_end);
 check(13,actual.type_root==want.type_root && actual.expression==want.expression);
}
@effects(console) fn void projected_entry(ref ast::Ast body,usize id,syntax::Entry want,usize type_offset,usize expression_offset){
 let actual=ast::entry_at(body,id-1);
 check(14,actual.id==id && actual.kind==want.kind && actual.start==want.start && actual.end==want.end);
 check(15,actual.name_start==want.name_start && actual.name_end==want.name_end && actual.type_start==want.type_start && actual.type_end==want.type_end);
 check(16,if(want.type_root==0){actual.type_root==0}else{actual.type_root+type_offset==want.type_root});
 check(17,if(want.expression==0){actual.expression==0}else{actual.expression+expression_offset==want.expression});
}
@effects(console) fn void origins(ref syntax::Parsed file,ref ast::Ast body,usize entries,usize expressions,usize statements,usize types,usize entry_offset,usize expression_offset,usize type_offset){
 check(18,ast::entry_count(body)==entries && ast::expression_count(body)==expressions && ast::statement_count(body)==statements && ast::type_count(body)==types);
 var usize i=1;while(i<=entries){check(19,ast::entry_origin(body,i)==entry_offset+i);i+=1;}
 i=1;while(i<=expressions){check(20,ast::expression_origin(body,i)==expression_offset+i);let old=syntax::expression_at(file,expression_offset+i);let actual=ast::expression_at(body,i);check(21,actual.kind==old.kind && actual.start==old.start && actual.end==old.end && actual.token==old.token);i+=1;}
 i=1;while(i<=types){check(22,ast::type_origin(body,i)==type_offset+i);let old=syntax::type_at(file,type_offset+i);let actual=ast::type_at(body,i);check(23,actual.kind==old.kind && actual.start==old.start && actual.end==old.end);i+=1;}
}
@effects(console) fn void limits(ref syntax::Parsed file,usize cost){
 var usize work=1048576-cost;let exact=built(file,2,ref mut work);check(24,work==1048576 && ast::entry_count(ref exact)==4);
 work=1048576-cost+1;check(25,switch(ast::build(file,2,ref mut work)){case .ok(_):{false}case .err(code):{code==10}} && work==1048576);
 // Structural errors still precede charging, even with an exhausted budget.
 work=1048576;check(26,switch(ast::build(file,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}} && work==1048576);
}
@effects(console) fn void ordered(){
 let file=text(b"module ordered;fn i32 run(i32 value){let i32 local=value;var i32 other=7;return local;}");
 // Absolute byte spans and arena roots from the authored text above.
 let syntax::Entry[4] want=[
  syntax::Entry{.kind=1,.start=26,.end=35,.name_start=30,.name_end=35,.type_start=26,.type_end=29,.type_root=2,.expression=0},
  syntax::Entry{.kind=2,.start=37,.end=57,.name_start=45,.name_end=50,.type_start=41,.type_end=44,.type_root=3,.expression=1},
  syntax::Entry{.kind=3,.start=57,.end=73,.name_start=65,.name_end=70,.type_start=61,.type_end=64,.type_root=4,.expression=2},
  syntax::Entry{.kind=4,.start=73,.end=86,.name_start=0,.name_end=0,.type_start=0,.type_end=0,.type_root=0,.expression=3}
 ];
 check(30,syntax::entry_count(ref file)==4);var usize work=0;let body=built(ref file,2,ref mut work);
 origins(ref file,ref body,4,3,7,4,0,0,0);var usize i=0;while(i<4){entry(syntax::entry_at(ref file,i),want[i]);projected_entry(ref body,i+1,want[i],0,0);i+=1;}
 // 18 source arena rows + 18 retained rows; no attributes or reachability work.
 check(31,work==36);limits(ref file,36);
}
@effects(console) fn void nested(){
 let file=text(b"module nested;fn i32 run(i32 value){let i32 outer={let i32 inner=value;inner};return outer;}");
 // outer reserves ID 2 before its initializer appends inner (ID 3).
 // All four logical IDs are checked: physical order is value, inner, outer, return.
 let syntax::Entry[4] want=[
  syntax::Entry{.kind=1,.start=25,.end=34,.name_start=29,.name_end=34,.type_start=25,.type_end=28,.type_root=2,.expression=0},
  syntax::Entry{.kind=2,.start=36,.end=78,.name_start=44,.name_end=49,.type_start=40,.type_end=43,.type_root=3,.expression=3},
  syntax::Entry{.kind=2,.start=51,.end=71,.name_start=59,.name_end=64,.type_start=55,.type_end=58,.type_root=4,.expression=1},
  syntax::Entry{.kind=4,.start=78,.end=91,.name_start=0,.name_end=0,.type_start=0,.type_end=0,.type_root=0,.expression=4}
 ];
 check(32,syntax::entry_count(ref file)==4);var usize work=0;let body=built(ref file,2,ref mut work);
 origins(ref file,ref body,4,4,8,4,0,0,0);var usize i=0;while(i<4){entry(syntax::entry_at(ref file,i),want[i]);projected_entry(ref body,i+1,want[i],0,0);i+=1;}
 check(33,work==40);limits(ref file,40);
}
@effects(console) fn void many(){
 var bytes::Buffer raw=bytes::empty();var v::Vec<syntax::Entry> want=v::empty<syntax::Entry>();append(ref mut raw,b"module many;");var usize f=0;
 // Fixed workload: 24 functions, each with eight parameters, eight locals and a return.
 // Names have fixed one-digit suffixes; source records are captured as written.
 while(f<24){append(ref mut raw,b"fn i32 f");check(40,bytes::append_usize(ref mut raw,f).is_ok);append(ref mut raw,b"(");var usize j=0;
  while(j<8){if(j!=0){append(ref mut raw,b",");}let start=bytes::len(ref raw);append(ref mut raw,b"i32 p");check(41,bytes::append_usize(ref mut raw,j).is_ok);
   check(42,v::push<syntax::Entry>(ref mut want,syntax::Entry{.kind=1,.start,.end=start+6,.name_start=start+4,.name_end=start+6,.type_start=start,.type_end=start+3,.type_root=17*f+j+2,.expression=0}).is_ok);j+=1;
  }append(ref mut raw,b"){");j=0;
  while(j<8){let start=bytes::len(ref raw);append(ref mut raw,b"let i32 l");check(43,bytes::append_usize(ref mut raw,j).is_ok);append(ref mut raw,b"=p0;");
   check(44,v::push<syntax::Entry>(ref mut want,syntax::Entry{.kind=2,.start,.end=start+14,.name_start=start+8,.name_end=start+10,.type_start=start+4,.type_end=start+7,.type_root=17*f+j+10,.expression=9*f+j+1}).is_ok);j+=1;
  }let start=bytes::len(ref raw);append(ref mut raw,b"return l0;}");
  check(44,v::push<syntax::Entry>(ref mut want,syntax::Entry{.kind=4,.start,.end=start+10,.name_start=0,.name_end=0,.type_start=0,.type_end=0,.type_root=0,.expression=9*f+9}).is_ok);f+=1;
 }
 let file=parsed(raw);check(45,syntax::declaration_count(ref file)==25 && syntax::entry_count(ref file)==408 && v::len<syntax::Entry>(ref want)==408);
 var usize i=0;while(i<408){entry(syntax::entry_at(ref file,i),*v::at<syntax::Entry>(ref want,i));i+=1;}
 var usize work=0;f=0;while(f<24){let before=work;let body=built(ref file,f+2,ref mut work);origins(ref file,ref body,17,9,19,17,17*f,9*f,17*f);
  i=0;while(i<17){projected_entry(ref body,i+1,*v::at<syntax::Entry>(ref want,17*f+i),17*f,9*f);i+=1;}
  // 24*62 whole-file rows scanned + 62 rows retained per projection.
  check(46,work-before==1550);f+=1;
 }check(47,work==37200);discard io::write("functions=24 entries=408 work=");discard io::write_usize(work);discard io::write("\n");
}
@effects(console) fn void first_duplicate(ref syntax::Parsed file){
 check(50,syntax::entry_count(file)==2 && bytes::equal(syntax::source_view(file),b"A B"));
 // Index 1 requests ID 2. Both physical rows have that ID: the first wins,
 // even though the second row's ID equals its physical slot plus one.
 entry(syntax::entry_at(file,1),syntax::Entry{.kind=2,.start=0,.end=1,.name_start=0,.name_end=1,.type_start=0,.type_end=0,.type_root=0,.expression=0});
}
@effects(console) fn void duplicate(){
 // Only this fallback-contract case assembles arbitrary arena IDs. The ordered,
 // nested and workload cases above all use the actual production parser.
 let raw=switch(bytes::copy(b"A B")){case .ok(p):{p}case .err(_):{check(51,false);return;}};
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(52,false);return;}};
 var v::Vec<entries::Entry> rows=v::empty<entries::Entry>();
 check(53,v::push<entries::Entry>(ref mut rows,entries::Entry{.kind=2,.start=0,.end=1,.name_start=0,.name_end=1,.type_start=0,.type_end=0,.type_root=0,.expression=0,.id=2}).is_ok);
 check(54,v::push<entries::Entry>(ref mut rows,entries::Entry{.kind=3,.start=2,.end=3,.name_start=2,.name_end=3,.type_start=0,.type_end=0,.type_root=0,.expression=0,.id=2}).is_ok);
 let file=syntax::assemble(stream,v::empty<syntax::Declaration>(),v::empty<syntax::Problem>(),v::empty<state::Node>(),rows,v::empty<state::Statement>(),v::empty<types::Node>(),v::empty<records::Field>(),v::empty<attrs::Attribute>(),v::empty<attrs::Argument>());
 first_duplicate(ref file);var usize work=0;
 let owned=switch(syntax::copy(ref file,ref mut work)){case .ok(p):{p}case .err(_):{check(55,false);return;}};
 first_duplicate(ref owned);check(56,work>0);discard io::write("duplicate=first copy=first\n");
}
@effects(console) fn void assembled_order(ref syntax::Parsed file){
 check(60,syntax::entry_count(file)==2 && bytes::equal(syntax::source_view(file),b"A B"));
 entry(syntax::entry_at(file,0),syntax::Entry{.kind=2,.start=0,.end=1,.name_start=0,.name_end=1,.type_start=0,.type_end=0,.type_root=0,.expression=0});
 entry(syntax::entry_at(file,1),syntax::Entry{.kind=3,.start=2,.end=3,.name_start=2,.name_end=3,.type_start=0,.type_end=0,.type_root=0,.expression=0});
}
@effects(console) fn void ordered_assembly(){
 // Public assembly of ordinary ordered IDs must retain both records on copy.
 let raw=switch(bytes::copy(b"A B")){case .ok(p):{p}case .err(_):{check(61,false);return;}};
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(62,false);return;}};
 var v::Vec<entries::Entry> rows=v::empty<entries::Entry>();
 check(63,v::push<entries::Entry>(ref mut rows,entries::Entry{.kind=2,.start=0,.end=1,.name_start=0,.name_end=1,.type_start=0,.type_end=0,.type_root=0,.expression=0,.id=1}).is_ok);
 check(64,v::push<entries::Entry>(ref mut rows,entries::Entry{.kind=3,.start=2,.end=3,.name_start=2,.name_end=3,.type_start=0,.type_end=0,.type_root=0,.expression=0,.id=2}).is_ok);
 let file=syntax::assemble(stream,v::empty<syntax::Declaration>(),v::empty<syntax::Problem>(),v::empty<state::Node>(),rows,v::empty<state::Statement>(),v::empty<types::Node>(),v::empty<records::Field>(),v::empty<attrs::Attribute>(),v::empty<attrs::Argument>());
 assembled_order(ref file);var usize work=0;
 let owned=switch(syntax::copy(ref file,ref mut work)){case .ok(p):{p}case .err(_):{check(65,false);return;}};
 assembled_order(ref owned);check(66,work>0);discard io::write("assembled=ordered copy=ordered\n");
}
@effects(console) fn i32 main(){
 ordered();nested();discard io::write("ordered=4 nested=4 origins=ok work=36,40 limits=ok\n");many();duplicate();ordered_assembly();return 0;
}
