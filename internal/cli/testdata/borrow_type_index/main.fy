module main;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import module_unit_value_model as m;
import module_unit_value_place as places;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn void ensure(usize step,bool condition){if(!condition){discard io::write_stderr("borrow fixture step ");discard io::write_stderr_usize(step);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn void append(ref mut bytes::Buffer raw,ref u8[] text){ensure(1,bytes::append(raw,text).is_ok);}
// All records used by the index come from the real public builder. Forty-eight
// unrelated structs/functions stay below the per-source 128 limit, not Graph's cap.
@effects(console) fn interfaces::Interfaces built(usize count){
 var raw=bytes::empty();append(ref mut raw,b"module sample;");var usize i=0;
 while(i<count){append(ref mut raw,b"struct S");ensure(2,bytes::append_usize(ref mut raw,i).is_ok);append(ref mut raw,b"{i32 value;}fn void inspect");ensure(2,bytes::append_usize(ref mut raw,i).is_ok);append(ref mut raw,b"(ref S");ensure(2,bytes::append_usize(ref mut raw,i).is_ok);append(ref mut raw,b" a,ref mut S");ensure(2,bytes::append_usize(ref mut raw,i).is_ok);append(ref mut raw,b" b){}");i+=1;}
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{ensure(3,false);return built(1);}};
 let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{ensure(4,false);return built(1);}};ensure(5,syntax::problem_count(ref parsed)==0);
 var sources=v::empty<syntax::Parsed>();ensure(6,v::push<syntax::Parsed>(ref mut sources,parsed).is_ok);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{ensure(7,false);return built(1);}};
 let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{ensure(8,false);return built(1);}};
 return switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);ensure(9,false);return built(1);}};
}
fn types::Type reference(usize target,bool mutable){return types::Type{.kind=if(mutable){8}else{7},.scalar=0,.module_index=0,.declaration=0,.first=target,.second=0};}
// Ordered six-field equality matches canonical's public record semantics; return
// the first match, independently of the private index layout.
fn usize linear(ref v::Vec<types::Type> graph,usize target,bool mutable){var usize i=1;while(i<=types::count(graph)){let t=types::at(graph,i);if(t.kind==if(mutable){8}else{7} && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==target && t.second==0){return i;}i+=1;}return 0;}
@effects(console) fn usize oracle(ref interfaces::Interfaces p){
 let n=interfaces::type_count(p);ensure(10,n>0 && n<=4096);var usize i=0;var usize hits=0;
 while(i<=n+1){ensure(11,interfaces::borrow_type(p,i,false)==linear(interfaces::type_records(p),i,false));ensure(12,interfaces::borrow_type(p,i,true)==linear(interfaces::type_records(p),i,true));let a=interfaces::borrow_type(p,i,false);let b=interfaces::borrow_type(p,i,true);if(a!=0){ensure(13,b!=0 && a!=b);hits+=1;}i+=1;}
 ensure(14,hits==interfaces::shape_count(p));return 2*(n+2);
}
@effects(console) fn v::Vec<types::Type> clone(ref interfaces::Interfaces p){var graph=v::empty<types::Type>();var usize i=1;while(i<=interfaces::type_count(p)){ensure(15,v::push<types::Type>(ref mut graph,interfaces::type_at(p,i)).is_ok);i+=1;}return graph;}
fn m::Frame frame(){return m::Frame{.kind=3,.context=0,.module_index=0,.declaration=1,.result=0};}
fn m::Expression expression(usize type){return m::Expression{.constant=0,.constant_bits=0,.source_index=1,.binding=1,.kind=2,.start=17,.end=23,.left=0,.right=0,.target=0,.field=m::Field{.id=0,.owner=0,.module_index=0,.declaration=0,.source_index=0},.place=m::Place{.kind=1,.root=1,.scope=1,.projection=0,.mutable=true,.indirect=false,.resource_field=false,.borrowable=true},.builtin=m::Builtin{.kind=0,.element=0,.slice=false,.unsafe_context=false},.expected=0,.type,.state=1,.flexible=false,.minimum=false};}
@effects(console) fn void hit(ref interfaces::Interfaces p,ref mut v::Vec<types::Type> graph,usize target,bool mutable){
 let e=expression(target);let expected=linear(graph,target,mutable);ensure(16,expected!=0);var usize work=1048575;let old_count=types::count(graph);
 ensure(17,switch(places::borrow_cached(frame(),e,e,mutable,p,graph,ref mut work)){case .ok(id):{id==expected && work==1048576 && types::count(graph)==old_count}case .err(_):{false}});
 ensure(18,switch(places::borrow_cached(frame(),e,e,mutable,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10 && e.start==17 && e.end==23 && work==1048576}});
 work=1048575;ensure(19,switch(places::borrow(frame(),e,e,mutable,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10 && work==1048575}});
}
@effects(console) fn void permissions(ref interfaces::Interfaces p,ref mut v::Vec<types::Type> graph,usize target){
 let e=expression(target);var bad=e;bad.kind=0;bad.place=m::no_place();var usize work=1048576;
 ensure(20,switch(places::borrow_cached(frame(),e,bad,true,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1008 && e.start==17 && e.end==23 && work==1048576}});
 ensure(21,switch(places::borrow_cached(frame(),e,bad,false,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==2 && work==1048576}});
 ensure(22,switch(places::borrow(frame(),e,bad,true,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1008 && work==1048576}});
 ensure(23,switch(places::borrow(frame(),e,bad,false,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==2 && work==1048576}});
 bad=e;bad.place.mutable=false;ensure(24,switch(places::borrow_cached(frame(),e,bad,true,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1008 && work==1048576}});
 bad=e;bad.place.resource_field=true;ensure(25,switch(places::borrow_cached(frame(),e,bad,true,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1008 && work==1048576}});
 work=0;ensure(26,switch(places::borrow_cached(frame(),e,bad,false,p,graph,ref mut work)){case .ok(id):{id==interfaces::borrow_type(p,target,false) && work==1}case .err(_):{false}});
}
@effects(console) fn void fallback_and_cap(ref interfaces::Interfaces p,ref mut v::Vec<types::Type> graph,usize target){
 var usize scalar=1;while(scalar<=types::count(graph) && (types::at(graph,scalar).kind!=1 || types::at(graph,scalar).scalar!=1)){scalar+=1;}ensure(27,scalar<=types::count(graph));
 // A prefix target with no cached reference pays one lookup plus the ordinary
 // canonical scan, then repeats via exactly that miss path (no cache mutation).
 ensure(28,interfaces::borrow_type(p,scalar,false)==0);let e=expression(scalar);var usize work=0;let n=types::count(graph);
 let shared=switch(places::borrow_cached(frame(),e,e,false,p,graph,ref mut work)){case .ok(id):{id}case .err(_):{ensure(29,false);return;}};ensure(30,shared==n+1 && work==n+2 && shared==linear(graph,scalar,false));
 work=0;ensure(31,switch(places::borrow_cached(frame(),e,e,false,p,graph,ref mut work)){case .ok(id):{id==shared && work==types::count(graph)+2}case .err(_):{false}});
 let outside=types::canonical(graph,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=scalar,.second=0});ensure(32,outside>interfaces::type_count(p) && interfaces::borrow_type(p,outside,false)==0);let extra=expression(outside);work=0;let before=types::count(graph);
 let added=switch(places::borrow_cached(frame(),extra,extra,false,p,graph,ref mut work)){case .ok(id):{id}case .err(_):{ensure(33,false);return;}};ensure(34,added==before+1 && work==before+1 && added==linear(graph,outside,false));
 work=0;ensure(35,switch(places::borrow(frame(),extra,extra,false,graph,ref mut work)){case .ok(id):{id==added && work==types::count(graph)+1}case .err(_):{false}});
 // Fill with distinct legal scalar-array records via the public canonical API;
 // source-kind limits do not substitute for the independently bounded Graph.
 var usize length=1;while(types::count(graph)<4096){ensure(36,types::canonical(graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=scalar,.second=length})!=0);length+=1;}ensure(37,types::count(graph)==4096);
 hit(p,graph,target,false);hit(p,graph,target,true);
 let unseen=types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=scalar,.second=4096};ensure(38,types::canonical(graph,unseen)==0 && types::count(graph)==4096);
 // Prefix-external mutable reference was never interned: full graph must fail.
 work=0;ensure(39,switch(places::borrow_cached(frame(),extra,extra,true,p,graph,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10 && work==4097 && types::count(graph)==4096}});
}
@effects(console) fn i32 main(){
 let small=built(1);let large=built(48);var usize queries=oracle(ref small)+oracle(ref large);ensure(40,interfaces::type_count(ref large)>interfaces::type_count(ref small));
 // Setup has a separate bounded cost: zero-initialize 2*(G+1), then visit G.
 // The public total also includes signature/shape/generic setup, not only index.
 ensure(41,interfaces::work(ref small)>=3*interfaces::type_count(ref small)+2 && interfaces::work(ref large)>=3*interfaces::type_count(ref large)+2 && interfaces::work(ref large)<=1048576);
 var a=clone(ref small);var b=clone(ref large);let first=interfaces::shape_at(ref small,0).type;let last=interfaces::shape_at(ref large,interfaces::shape_count(ref large)-1).type;
 hit(ref small,ref mut a,first,false);hit(ref small,ref mut a,first,true);hit(ref large,ref mut b,last,false);hit(ref large,ref mut b,last,true);permissions(ref small,ref mut a,first);fallback_and_cap(ref small,ref mut a,first);
 discard io::write("small_types=");discard io::write_usize(interfaces::type_count(ref small));discard io::write(" large_types=");discard io::write_usize(interfaces::type_count(ref large));discard io::write(" small_setup=");discard io::write_usize(interfaces::work(ref small));discard io::write(" large_setup=");discard io::write_usize(interfaces::work(ref large));discard io::write(" queries=");discard io::write_usize(queries);discard io::write(" fast=1 graph_cap=4096\nok\n");return 0;
}
