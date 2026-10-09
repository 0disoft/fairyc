module tests::module_unit_origin_tagged;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_origin_returns as origins;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import module_unit_origin_check as checker;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);primary(sources,metadata)}};}
@effects(console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return checked(root,provider);}};let primary=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return checked(root,provider);}};return switch(diagnostics::collect(ref p)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}

@test @effects(console) fn void tagged_return_origin_cases(){
    let p0=checked(b"module main;fn Option<ref i32> f(){return .none;}",b"");emit(0,ref p0);assert(diagnostics::exit_code(ref p0)==0);
    let p1=checked(b"module main;fn Option<ref i32> f(ref i32 a){return .some(a);}",b"");emit(1,ref p1);assert(diagnostics::exit_code(ref p1)==0);
    let p2=checked(b"module main;fn Option<ref i32> f(){let i32 a=1;return .some(ref a);}",b"");emit(2,ref p2);assert(diagnostics::exit_code(ref p2)==3);
    let p3=checked(b"module main;fn Option<ref i32> f(ref i32 a,ref i32 b,bool c){return if(c){.some(a)}else{.some(b)};}",b"");emit(3,ref p3);assert(diagnostics::exit_code(ref p3)==3);
    let p4=checked(b"module main;fn Option<ref i32> f(ref i32 a,bool c){return if(c){.some(a)}else{.none};}",b"");emit(4,ref p4);assert(diagnostics::exit_code(ref p4)==0);
    let p5=checked(b"module main;fn Option<ref i32> f(){return .none;}fn Option<ref i32> g(){return f();}",b"");emit(5,ref p5);assert(diagnostics::exit_code(ref p5)==0);
    let p6=checked(b"module main;fn Option<ref i32> f(Option<ref i32> a){return a;}",b"");emit(6,ref p6);assert(diagnostics::exit_code(ref p6)==0);
    let p7=checked(b"module main;fn Result<Option<ref i32>,i32> f(){let i32 a=1;return .ok(.some(ref a));}",b"");emit(7,ref p7);assert(diagnostics::exit_code(ref p7)==3);
    let p8=checked(b"module main;fn Result<Option<ref i32>,i32> f(){return .ok(.none);}",b"");emit(8,ref p8);assert(diagnostics::exit_code(ref p8)==0);
    let p9=checked(b"module main;fn Result<ref i32,i32> f(){return .err(3);}",b"");emit(9,ref p9);assert(diagnostics::exit_code(ref p9)==0);
    let p10=checked(b"module main;fn Result<i32,ref i32> f(){let i32 a=1;return .err(ref a);}",b"");emit(10,ref p10);assert(diagnostics::exit_code(ref p10)==3);
    let p11=checked(b"module main;fn Option<ref i32> f(){let Option<ref i32> a=.none;return a;}",b"");emit(11,ref p11);assert(diagnostics::exit_code(ref p11)==0);
    let p12=checked(b"module main;fn Option<ref i32> f(ref i32 a,bool c){var Option<ref i32> v=.none;if(c){v=.some(a);};return v;}",b"");emit(12,ref p12);assert(diagnostics::exit_code(ref p12)==0);
    let p13=checked(b"module main;fn Option<ref i32> f(bool c){let i32 a=1;var Option<ref i32> v=.none;if(c){v=.some(ref a);};return v;}",b"");emit(13,ref p13);assert(diagnostics::exit_code(ref p13)==3);
    let p14=checked(b"module main;fn Option<ref i32> id(Option<ref i32> a){return a;}fn Option<ref i32> f(){return id(.none);}",b"");emit(14,ref p14);assert(diagnostics::exit_code(ref p14)==0);
    let p15=checked(b"module main;fn Option<ref i32> id(Option<ref i32> a){return a;}fn Option<ref i32> f(){let i32 a=1;return id(.some(ref a));}",b"");emit(15,ref p15);assert(diagnostics::exit_code(ref p15)==3);
    let p16=checked(b"module main;fn Option<ref i32> f(ref i32 a,bool c){var Option<ref i32> v=.none;while(c){v=.some(a);break;}return v;}",b"");emit(16,ref p16);assert(diagnostics::exit_code(ref p16)==0);
    let p17=checked(b"module main;fn Option<ref str> f(bool c){return if(c){.some(\"a\")}else{.some(\"b\")};}",b"");emit(17,ref p17);assert(diagnostics::exit_code(ref p17)==3);
    let p18=checked(b"module main;fn Option<ref i32> f(ref i32 a,ref i32 b,bool c){if(c){return .some(a);};return .some(b);}",b"");emit(18,ref p18);assert(diagnostics::exit_code(ref p18)==0);
    let p19=checked(b"module main;import provider as p;fn Result<Option<ref i32>,i32> f(ref i32 a){return p::get(a);}",b"module provider;pub fn Result<Option<ref i32>,i32> get(ref i32 a){return .ok(.some(a));}");emit(19,ref p19);assert(diagnostics::exit_code(ref p19)==0);
    let p20=checked(b"module main;fn ref Option<ref i32> f(ref i32 a){let Option<ref i32> o=.some(a);return ref o;}",b"");emit(20,ref p20);assert(diagnostics::exit_code(ref p20)==3);
    let p21=checked(b"module main;fn ref Option<ref i32> f(Option<ref i32> o){return ref o;}",b"");emit(21,ref p21);assert(diagnostics::exit_code(ref p21)==3);
    let p22=checked(b"module main;fn ref Option<ref i32> f(ref Option<ref i32> o){return o;}",b"");emit(22,ref p22);assert(diagnostics::exit_code(ref p22)==0);
    let p23=checked(b"module main;fn Option<ref T> id<T>(ref T a){return .some(a);}fn Option<ref i32> f(ref i32 a){return id<i32>(a);}",b"");emit(23,ref p23);assert(diagnostics::exit_code(ref p23)==0);
}
fn void unsupported(ref u8[] source){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok);switch(checker::check(ref sources)){case .ok(_):{assert(false);}case .err(p):{assert(p.phase==5 && p.code==2);}};}
@test fn void multiple_borrow_leaves_and_pending_propagation_are_explicitly_unsupported(){
    unsupported(b"module main;fn Result<ref i32,ref i32> f(ref i32 a,ref i32 b){return .ok(a);}");
    unsupported(b"module main;fn Result<ref i32,i32> f(Result<ref i32,i32> a){return .ok(a?);}");
}
