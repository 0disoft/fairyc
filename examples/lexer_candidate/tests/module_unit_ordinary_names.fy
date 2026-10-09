module tests::module_unit_ordinary_names;
import module_unit_ordinary_names as ordinary;
import module_unit_names as names;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import module_unit_plan as units;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@test fn void selected_ordinary_bodies_preserve_provider_and_original_arena_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 helper(i32 n){return n;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn T ignored<T>(T n){discard missing;return n;}fn i32 helper(i32 n){return p::helper(n);}fn i32 main(){let i32 n=1;if(true){let i32 n=2;discard n;}for(var i32 k=0;k<1;k+=1){discard k;}for(let u8 ch in b\"x\"){discard ch;}let Option<i32> maybe=.some(n);let i32 result=switch(maybe){case .some(value):{helper(value)}case .none:{0}};return n;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,1,2)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let usize types=interfaces::type_count(ref imported);
    let p=switch(ordinary::build(ref sources,ref metadata,ref bound,ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ordinary::count(ref p)==2 && interfaces::type_count(ref imported)==types && ordinary::work(ref p)<1048576);var usize providers=0;var usize patterns=0;var usize bytes=0;var usize i=1;
    while(i<=ordinary::count(ref p)){let f=ordinary::at(ref p,i);assert(ordinary::module_index(f)==1 && ordinary::declaration(f)>=4);let r=ordinary::resolved(f);let source=v::at<syntax::Parsed>(ref sources,1);var usize j=1;while(j<=names::references(r)){let use=names::reference_at(r,j);let n=syntax::expression_at(source,use.source_index);assert(use.source_index==ordinary::expression_origin(f,use.expression) && n.kind==2 && use.start==n.start && use.end==n.end && names::use_at(r,use.expression)==j);if(use.kind==2 && use.module_index==0){providers+=1;assert(use.declaration==2 && use.declaration_kind==3);}j+=1;}j=1;while(j<=names::local_count(r)){let symbol=names::symbol_at(r,j);if(symbol.entry!=0){let e=syntax::entry_at(source,ordinary::entry_origin(f,symbol.entry)-1);assert(e.name_start==symbol.start && e.name_end==symbol.end);if(symbol.kind==1){assert(symbol.type!=0 && interfaces::type_at(ref imported,symbol.type).scalar==1);}}if(symbol.kind==4){patterns+=1;assert(symbol.type==0);}if(symbol.kind==5){bytes+=1;assert(interfaces::type_at(ref imported,symbol.type).scalar==3);}j+=1;}i+=1;}assert(providers==1 && patterns==1 && bytes==1);
}
fn void denied(ref u8[] text,u32 code,ref u8[] token){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(units::build(ref sources,ref metadata)){case .ok(p):{assert(code==0 && units::count(ref p)==1);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],token));}};}
@test fn void ordinary_scope_errors_reject_at_original_source_spans(){
    denied(b"module main;fn i32 main(){discard later;let i32 later=1;return later;}",1,b"later");
    denied(b"module main;fn i32 main(){let i32 n=1;let i32 n=2;return n;}",2,b"n");
    denied(b"module main;fn i32 one(i32 n){return n;}fn i32 main(){return n;}",1,b"n");
    denied(b"module main;fn i32 main(){for(var i32 k=0;k<1;k+=1){discard k;}return k;}",1,b"k");
    denied(b"module main;fn i32 main(){let Option<i32> v=.none;discard switch(v){case .some(n):{n}case .none:{0}};return n;}",1,b"n");
}
@test fn void stale_sources_and_interface_ranges_are_not_rebound(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module first;fn i32 first(){return 1;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module last;fn i32 last(){return 2;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let first=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let last=switch(binding::bind(ref sources,ref metadata,1,2)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref first)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(ordinary::build(ref sources,ref metadata,ref last,ref imported)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==1);}};
    discard v::pop<syntax::Parsed>(ref mut sources);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module last;fn i32 last(){return 3;}")).is_ok);let fresh=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(ordinary::build(ref sources,ref fresh,ref first,ref imported)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==1);}};
}
fn void suffix(ref mut bytes::Buffer out,usize n){let digits=b"0123456789";if(n>=100){assert(bytes::append_byte(out,digits[n/100]).is_ok);}if(n>=10){assert(bytes::append_byte(out,digits[(n/10)%10]).is_ok);}assert(bytes::append_byte(out,digits[n%10]).is_ok);}
@test fn void ordinary_local_admission_keeps_the_128_name_boundary(){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module main;fn i32 main(i32 n){").is_ok);var usize i=0;while(i<127){assert(bytes::append(ref mut text,b"let i32 v").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"=0;").is_ok);i+=1;}let usize prefix=bytes::len(ref text);assert(bytes::append(ref mut text,b"return n;}").is_ok);denied(bytes::view(ref text),0,b"");assert(bytes::resize_zeroed(ref mut text,prefix).is_ok);assert(bytes::append(ref mut text,b"let i32 v127=0;return n;}").is_ok);denied(bytes::view(ref text),10,b"v127");
}
