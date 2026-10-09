module tests::module_unit_origin_region;
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
import std::process as process;
import file_source as files;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);primary(sources,metadata)}};}
@effects(filesystem) fn syntax::Parsed allocator(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return allocator();}};let data=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return allocator();}};return parsed(bytes::view(ref data));}
@effects(filesystem,console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,allocator()).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(p):{discard io::write("metadata problem ");discard io::write_u32(p.code);discard io::write(" module ");discard io::write_usize(p.module_index);discard io::write(" span ");discard io::write_usize(p.start);discard io::write("..");discard io::write_usize(p.end);discard io::write("\n");assert(false);return checked(root,provider);}};let primary=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(p):{discard io::write("origin problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return checked(root,provider);}};return switch(diagnostics::collect(ref p)){case .ok(p):{p}case .err(_):{assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


@test @effects(filesystem,console) fn void allocated_returns_follow_region_inputs(){
    let p0=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,i32 n){return r.place(n);}",b"");emit(0,ref p0);assert(diagnostics::exit_code(ref p0)==0);
    let p1=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,i32 n){return .ok(r.place(n)?);}",b"");emit(1,ref p1);assert(diagnostics::exit_code(ref p1)==0);
    let p2=checked(b"module main;fn ref i32 f(Region r,i32 n){return switch(r.place(n)){case .ok(v):{v}case .err(e):{assert(false);}};}",b"");emit(2,ref p2);assert(diagnostics::exit_code(ref p2)==0);
    let p3=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region x,Region y,bool c){return if(c){x.place(1)}else{y.place(2)};}",b"");emit(3,ref p3);assert(diagnostics::exit_code(ref p3)==3);
    let p4=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,ref i32 a,bool c){return .ok(if(c){r.place(1)?}else{a});}",b"");emit(4,ref p4);assert(diagnostics::exit_code(ref p4)==3);
    let p5=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> store(Region r,i32 n){return r.place(n);}fn Result<ref i32,a::AllocError> f(Region r){return store(r,3);}",b"");emit(5,ref p5);assert(diagnostics::exit_code(ref p5)==0);
    let p6=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> store(i32 n,Region r){return r.place(n);}fn Result<ref i32,a::AllocError> f(Region r){return store(3,r);}",b"");emit(6,ref p6);assert(diagnostics::exit_code(ref p6)==0);
    let p7=checked(b"module main;import std::alloc as a;import provider as p;fn Result<ref i32,a::AllocError> f(Region r){return p::store(r,3);}",b"module provider;import std::alloc as a;pub fn Result<ref i32,a::AllocError> store(Region r,i32 n){return r.place(n);}");emit(7,ref p7);assert(diagnostics::exit_code(ref p7)==0);
    let p8=checked(b"module main;import std::alloc as a;fn Result<ref T,a::AllocError> store<T>(Region r,T n){return r.place(n);}fn Result<ref i32,a::AllocError> f(Region r){return store<i32>(r,3);}",b"");emit(8,ref p8);assert(diagnostics::exit_code(ref p8)==0);
    let p9=checked(b"module main;import std::alloc as a;struct S{i32 n;}fn Result<ref S,a::AllocError> f(Region r){return r.place(S{.n=3});}",b"");emit(9,ref p9);assert(diagnostics::exit_code(ref p9)==0);
    let p10=checked(b"module main;import std::alloc as a;enum E{None;Value{i32 n;};}fn Result<ref E,a::AllocError> f(Region r){let E v=.Value{.n=3};return r.place(v);}",b"");emit(10,ref p10);assert(diagnostics::exit_code(ref p10)==0);
    let p11=checked(b"module main;import std::alloc as a;fn Result<ref Option<i32>,a::AllocError> f(Region r){let Option<i32> v=.some(3);return r.place(v);}",b"");emit(11,ref p11);assert(diagnostics::exit_code(ref p11)==0);
    let p12=checked(b"module main;fn i32 f(Region r){let i32[2] v=[1,2];discard r.place(v);return 7;}",b"");emit(12,ref p12);assert(diagnostics::exit_code(ref p12)==0);
    let p13=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,bool c){return if(c){r.place(1)}else{r.place(2)};}",b"");emit(13,ref p13);assert(diagnostics::exit_code(ref p13)==0);
    let p14=checked(b"module main;import std::alloc as a;fn Result<ref i32,a::AllocError> f(Region r,bool c){if(c){return r.place(3);};return f(r,true);}",b"");emit(14,ref p14);assert(diagnostics::exit_code(ref p14)==2);
    let p15=checked(b"module main;fn void f(Region r){let ptr void p=null;discard r.place(p);}",b"");emit(15,ref p15);assert(diagnostics::exit_code(ref p15)==0);
    let p16=checked(b"module main;import std::alloc as a;struct U{i32 n;}struct S{U first;Option<U> later;}fn Result<ref S,a::AllocError> f(Region r){return r.place(S{.first=U{.n=1},.later=.some(U{.n=2})});}",b"");emit(16,ref p16);assert(diagnostics::exit_code(ref p16)==0);
}
