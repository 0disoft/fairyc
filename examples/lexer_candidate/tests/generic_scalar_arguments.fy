module tests::generic_scalar_arguments;
import generic_scalar_arguments as arguments;
import module_types as types;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(value):{assert(syntax::problem_count(ref value)==0);value}case .err(_):{assert(false);parsed(raw)}};}
fn catalog::Catalog built_catalog(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);return switch(catalog::build(ref sources)){case .ok(value):{value}case .err(_):{assert(false);built_catalog(raw)}};}
fn usize argument_root(ref syntax::Parsed p,usize declaration){let d=syntax::declaration_at(p,declaration);var usize i=1;while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==20 && e.start>=d.start && e.end<=d.end){return syntax::specialization_argument(p,i,0);}i+=1;}assert(false);return 0;}
fn bytes::Buffer source(ref u8[] name){var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module root;fn i32 constant<T>(){return 42;}fn T used<T>(T x){return x;}fn i32 forward<T>(){return constant<T>();}fn i32 main(){return constant<").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b">();}").is_ok);return raw;}
@test
fn void descriptors_preserve_names_without_allocating_value_types(){var usize id=4097;while(id<=4103){let raw=source(arguments::name(id));let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));let root=argument_root(ref p,4);var v::Vec<types::Type> graph=v::empty<types::Type>();let empty=v::empty<usize>();let selected=types::resolve_argument(ref c,ref p,0,0,root,ref empty,ref mut graph);assert(selected.problem.code==0 && selected.type==id && types::count(ref graph)==0 && arguments::named(arguments::name(id))==id);
    let value=types::resolve(ref c,ref p,0,root,ref mut graph);assert(value.problem.code!=0 && value.type==0 && types::count(ref graph)==0);id+=1;}
    assert(!arguments::descriptor(0) && !arguments::descriptor(4096) && !arguments::descriptor(4104) && arguments::named(b"i32")==0 && arguments::named(b"Unknown")==0);
}
@test
fn void forwarded_arguments_keep_identity_but_value_substitution_rejects(){let raw=source(b"u16");let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> substitutions=v::empty<usize>();assert(v::push<usize>(ref mut substitutions,arguments::named(b"u16")).is_ok);let forwarded=types::resolve_argument(ref c,ref p,0,4,argument_root(ref p,3),ref substitutions,ref mut graph);assert(forwarded.problem.code==0 && forwarded.type==arguments::named(b"u16") && types::count(ref graph)==0);
    let constant=types::specialize(ref c,ref p,0,2,syntax::declaration_at(ref p,1).result_type,ref substitutions,ref mut graph);assert(constant.problem.code==0 && types::at(ref graph,constant.type).scalar==1 && types::count(ref graph)==1);
    let root=syntax::declaration_at(ref p,2).result_type;let location=syntax::type_at(ref p,root);let rejected=types::specialize(ref c,ref p,0,3,root,ref substitutions,ref mut graph);assert(rejected.problem.code==5 && rejected.problem.start==location.start && rejected.problem.end==location.end && rejected.type==0 && types::count(ref graph)==1);
    let changed=source(b"u64");let stale=parsed(bytes::view(ref changed));assert(types::resolve_argument(ref c,ref stale,0,0,argument_root(ref stale,4),ref substitutions,ref mut graph).problem.code==12 && types::count(ref graph)==1);
}
@test
fn void invalid_argument_domains_and_ordinary_values_stay_closed(){let raw=source(b"u16");let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> substitutions=v::empty<usize>();assert(v::push<usize>(ref mut substitutions,4104).is_ok);let root=syntax::declaration_at(ref p,1).result_type;let invalid=types::specialize(ref c,ref p,0,0,root,ref substitutions,ref mut graph);assert(invalid.problem.code==4501 && invalid.problem.start==0 && invalid.problem.end==0);assert(types::specialize(ref c,ref p,0,2,root,ref substitutions,ref mut graph).problem.code==4502 && types::count(ref graph)==0);
    assert(types::resolve_argument(ref c,ref p,0,0,argument_root(ref p,4),ref substitutions,ref mut graph).problem.code==4501 && types::count(ref graph)==0);
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=4100,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);assert(types::validate(ref c,ref graph)==9);
}
