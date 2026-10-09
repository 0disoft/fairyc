module tests::module_unit_origin_propagation;
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
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


@test @effects(console) fn void propagated_payloads_preserve_origins_and_implicit_error_sites(){
    let p0=checked(b"module main;fn Result<ref i32,i32> f(Result<ref i32,i32> x){return .ok(x?);}",b"");emit(0,ref p0);assert(diagnostics::exit_code(ref p0)==0);
    let p1=checked(b"module main;fn Result<ref i32,i32> f(ref i32 a){let Result<ref i32,i32> x=.ok(a);return .ok(x?);}",b"");emit(1,ref p1);assert(diagnostics::exit_code(ref p1)==0);
    let p2=checked(b"module main;fn Result<ref i32,i32> f(){let i32 n=1;let Result<ref i32,i32> x=.ok(ref n);return .ok(x?);}",b"");emit(2,ref p2);assert(diagnostics::exit_code(ref p2)==3);
    let p3=checked(b"module main;fn Result<i32,ref i32> f(Result<i32,ref i32> x){let i32 n=x?;return .ok(n);}",b"");emit(3,ref p3);assert(diagnostics::exit_code(ref p3)==0);
    let p4=checked(b"module main;fn Result<i32,ref i32> f(ref i32 a){let Result<i32,ref i32> x=.err(a);let i32 n=x?;return .ok(n);}",b"");emit(4,ref p4);assert(diagnostics::exit_code(ref p4)==0);
    let p5=checked(b"module main;fn Result<i32,ref i32> f(){let i32 n=1;let Result<i32,ref i32> x=.err(ref n);discard x?;return .ok(3);}",b"");emit(5,ref p5);assert(diagnostics::exit_code(ref p5)==3);
    let p6=checked(b"module main;fn Result<i32,ref str> f(){let Result<i32,ref str> x=.err(\"error\");discard x?;return .ok(3);}",b"");emit(6,ref p6);assert(diagnostics::exit_code(ref p6)==0);
    let p7=checked(b"module main;fn Result<i32,ref str> f(bool c){let Result<i32,ref str> x=if(c){.err(\"a\")}else{.err(\"b\")};discard x?;return .ok(3);}",b"");emit(7,ref p7);assert(diagnostics::exit_code(ref p7)==3);
    let p8=checked(b"module main;fn Result<i32,ref i32> get(ref i32 a){return .err(a);}fn Result<i32,ref i32> f(ref i32 a,ref i32 b){discard get(a)?;discard get(b)?;return .ok(3);}",b"");emit(8,ref p8);assert(diagnostics::exit_code(ref p8)==0);
    let p9=checked(b"module main;fn Result<Option<ref i32>,i32> f(Result<Option<ref i32>,i32> x){return .ok(x?);}",b"");emit(9,ref p9);assert(diagnostics::exit_code(ref p9)==0);
    let p10=checked(b"module main;fn Result<i32,Option<ref i32>> f(Result<i32,Option<ref i32>> x){discard x?;return .ok(3);}",b"");emit(10,ref p10);assert(diagnostics::exit_code(ref p10)==0);
    let p11=checked(b"module main;fn Result<ref i32,i32> f(){let Result<ref i32,i32> x=.err(3);return .ok(x?);}",b"");emit(11,ref p11);assert(diagnostics::exit_code(ref p11)==0);
    let p12=checked(b"module main;fn Result<i32,ref i32> f(){let Result<i32,ref i32> x=.ok(3);discard x?;return .ok(4);}",b"");emit(12,ref p12);assert(diagnostics::exit_code(ref p12)==0);
    let p13=checked(b"module main;fn Result<ref i32,i32> recurse(ref i32 a){return recurse(a);}fn Result<ref i32,i32> f(ref i32 a){return .ok(recurse(a)?);}",b"");emit(13,ref p13);assert(diagnostics::exit_code(ref p13)==2);
    let p14=checked(b"module main;fn Result<i32,ref i32> recurse(ref i32 a){return recurse(a);}fn Result<i32,ref i32> f(ref i32 a){discard recurse(a)?;return .ok(3);}",b"");emit(14,ref p14);assert(diagnostics::exit_code(ref p14)==2);
    let p15=checked(b"module main;import provider as p;fn Result<i32,ref i32> f(ref i32 a){discard p::get(a)?;return .ok(3);}",b"module provider;pub fn Result<i32,ref i32> get(ref i32 a){return .err(a);}");emit(15,ref p15);assert(diagnostics::exit_code(ref p15)==0);
    let p16=checked(b"module main;fn Result<ref T,i32> get<T>(ref T a){return .ok(a);}fn Result<ref i32,i32> f(ref i32 a){return .ok(get<i32>(a)?);}",b"");emit(16,ref p16);assert(diagnostics::exit_code(ref p16)==0);
    let p17=checked(b"module main;fn Result<i32,ref i32> f(Result<i32,ref i32> x,bool c){while(c){discard x?;break;}discard x;return .ok(3);}",b"");emit(17,ref p17);assert(diagnostics::exit_code(ref p17)==0);
    let p18=checked(b"module main;fn Result<ref i32,i32> f(Result<ref i32,i32> x,ref i32 a,bool c){var ref i32 r=a;while(c){r=x?;break;}discard x;return .ok(r);}",b"");emit(18,ref p18);assert(diagnostics::exit_code(ref p18)==3);
    let p19=checked(b"module main;fn Result<i32,ref i32> f(ref i32 a,ref i32 b,bool c){let Result<i32,ref i32> x=if(c){.err(a)}else{.err(b)};discard x?;return .ok(3);}",b"");emit(19,ref p19);assert(diagnostics::exit_code(ref p19)==3);
}
