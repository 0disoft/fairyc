module tests::module_unit_origin_diagnostics;
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
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);primary(sources,metadata)}};}
fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return checked(root,provider);}};let primary=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(_):{assert(false);return checked(root,provider);}};return switch(diagnostics::collect(ref p)){case .ok(p):{p}case .err(_):{assert(false);checked(root,provider)}};}
fn void failure(ref u8[] text,u32 code,ref u8[] token,ref u8[] a,ref u8[] b){let p=checked(text,b"");assert(diagnostics::count(ref p)==1 && diagnostics::exit_code(ref p)==if(code==2){2}else{3});let d=diagnostics::at(ref p,0);assert(d.code==code && d.module_index==0 && bytes::equal(text[d.start..d.end],token));assert(d.related_count==if(code==2103){2}else if(code==2102){1}else{0});if(d.related_count!=0){let first=diagnostics::related_at(ref p,d,0);assert(first.role==if(code==2102){1}else{2} && first.module_index==0 && bytes::equal(text[first.start..first.end],a));if(d.related_count==2){let second=diagnostics::related_at(ref p,d,1);assert(second.role==2 && second.module_index==0 && bytes::equal(text[second.start..second.end],b));}}assert(diagnostics::work(ref p)<1048576);}
@test fn void direct_return_diagnostics_preserve_primary_and_related_spans(){
    failure(b"module main;fn ref i32 bad(){let i32 local=1;return ref local;}",2102,b"ref local",b"local",b"");
    failure(b"module main;fn ref i32 bad(i32 n){return ref n;}",2102,b"ref n",b"n",b"");
    failure(b"module main;fn ref i32 choose(ref i32 a,ref i32 b,bool flag){var ref i32 r=a;if(flag){r=b;};return r;}",2103,b"r",b"a",b"b");
    failure(b"module main;fn ref str choose(bool flag){return if(flag){\"a\"}else{\"b\"};}",2103,b"if(flag){\"a\"}else{\"b\"}",b"\"a\"",b"\"b\"");
    failure(b"module main;fn ref i32 recursive(ref i32 a){return recursive(a);}",2,b"recursive(a)",b"",b"");
    let p=checked(b"module main;fn ref i32 alternatives(ref i32 a,ref i32 b,bool flag){if(flag){return a;};return b;}fn ref i32 dead(ref i32 a,ref i32 b){return a;return b;}",b"");assert(diagnostics::count(ref p)==0 && diagnostics::exit_code(ref p)==0);
}
@test fn void provider_origins_and_generic_duplicates_keep_source_identity(){
    let root=b"module main;import provider as p;fn ref str forward(bool flag){return p::choose(flag);}";let provider=b"module provider;pub fn ref str choose(bool flag){if(flag){return \"a\";};return \"b\";}";let p=checked(root,provider);assert(diagnostics::count(ref p)==1);let d=diagnostics::at(ref p,0);assert(d.code==2103 && d.module_index==0 && bytes::equal(root[d.start..d.end],b"p::choose(flag)") && d.related_count==2);let a=diagnostics::related_at(ref p,d,0);let b=diagnostics::related_at(ref p,d,1);assert(a.module_index==1 && b.module_index==1 && bytes::equal(provider[a.start..a.end],b"\"a\"") && bytes::equal(provider[b.start..b.end],b"\"b\""));
    let generic=b"module main;fn ref T choose<T>(ref T a,ref T b,bool flag){return if(flag){a}else{b};}fn void main(){let i32 a=1;let i32 b=2;let i64 x=3;let i64 y=4;discard choose<i32>(ref a,ref b,true);discard choose<i64>(ref x,ref y,false);}";let q=checked(generic,b"");assert(diagnostics::count(ref q)==1);let e=diagnostics::at(ref q,0);assert(e.code==2103 && e.related_count==2 && bytes::equal(generic[e.start..e.end],b"if(flag){a}else{b}"));
}
@test fn void diagnostic_collection_exhaustion_is_a_failure(){
    let text=b"module main;fn ref i32 bad(){let i32 local=1;return ref local;}";var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let primary=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(diagnostics::collect_budgeted(ref p,1048576)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};let report=switch(diagnostics::collect_budgeted(ref p,0)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(diagnostics::work(ref report)>origins::work(ref p));
}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}
@test @effects(console) fn void diagnostic_equivalence_receipt(){
    let a=checked(b"module main;fn ref i32 bad(){let i32 local=1;return ref local;}",b"");emit(0,ref a);
    let b=checked(b"module main;fn ref i32 bad(i32 n){return ref n;}",b"");emit(1,ref b);
    let c=checked(b"module main;fn ref i32 choose(ref i32 a,ref i32 b,bool flag){var ref i32 r=a;if(flag){r=b;};return r;}",b"");emit(2,ref c);
    let d=checked(b"module main;fn ref str choose(bool flag){return if(flag){\"a\"}else{\"b\"};}",b"");emit(3,ref d);
    let e=checked(b"module main;fn ref i32 recursive(ref i32 a){return recursive(a);}",b"");emit(4,ref e);
    let f=checked(b"module main;fn ref i32 alternatives(ref i32 a,ref i32 b,bool flag){if(flag){return a;};return b;}fn ref i32 dead(ref i32 a,ref i32 b){return a;return b;}",b"");emit(5,ref f);
    let g=checked(b"module main;import provider as p;fn ref str forward(bool flag){return p::choose(flag);}",b"module provider;pub fn ref str choose(bool flag){if(flag){return \"a\";};return \"b\";}");emit(6,ref g);
    let h=checked(b"module main;fn ref T choose<T>(ref T a,ref T b,bool flag){return if(flag){a}else{b};}fn void main(){let i32 a=1;let i32 b=2;let i64 x=3;let i64 y=4;discard choose<i32>(ref a,ref b,true);discard choose<i64>(ref x,ref y,false);}",b"");emit(7,ref h);
}
