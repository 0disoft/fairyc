module tests::generic_scalar_arguments;
import generic_scalar_arguments as arguments;
import generic_transport_scalars as transport;
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
fn void descriptors_preserve_names_and_materialize_direct_value_types(){var usize id=4097;while(id<=4103){let raw=source(arguments::name(id));let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));let root=argument_root(ref p,4);var v::Vec<types::Type> graph=v::empty<types::Type>();let empty=v::empty<usize>();let selected=types::resolve_argument(ref c,ref p,0,0,root,ref empty,ref mut graph);assert(selected.problem.code==0 && selected.type==id && types::count(ref graph)==0 && arguments::named(arguments::name(id))==id);
    let value=types::resolve(ref c,ref p,0,root,ref mut graph);assert(value.problem.code==0 && types::at(ref graph,value.type).scalar==transport::named(arguments::name(id)) && types::count(ref graph)==1);id+=1;}
    assert(!arguments::descriptor(0) && !arguments::descriptor(4096) && !arguments::descriptor(4104) && arguments::named(b"i32")==0 && arguments::named(b"Unknown")==0);
}
@test
fn void forwarded_arguments_keep_identity_and_value_substitution(){let raw=source(b"u16");let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> substitutions=v::empty<usize>();assert(v::push<usize>(ref mut substitutions,arguments::named(b"u16")).is_ok);let forwarded=types::resolve_argument(ref c,ref p,0,4,argument_root(ref p,3),ref substitutions,ref mut graph);assert(forwarded.problem.code==0 && forwarded.type==arguments::named(b"u16") && types::count(ref graph)==0);
    let constant=types::specialize(ref c,ref p,0,2,syntax::declaration_at(ref p,1).result_type,ref substitutions,ref mut graph);assert(constant.problem.code==0 && types::at(ref graph,constant.type).scalar==1 && types::count(ref graph)==1);
    let root=syntax::declaration_at(ref p,2).result_type;let location=syntax::type_at(ref p,root);let accepted=types::specialize(ref c,ref p,0,3,root,ref substitutions,ref mut graph);assert(accepted.problem.code==0 && types::at(ref graph,accepted.type).scalar==35 && types::count(ref graph)==2);
    let changed=source(b"u64");let stale=parsed(bytes::view(ref changed));assert(types::resolve_argument(ref c,ref stale,0,0,argument_root(ref stale,4),ref substitutions,ref mut graph).problem.code==12 && types::count(ref graph)==2);
}
@test
fn void invalid_argument_domains_stay_closed(){let raw=source(b"u16");let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> substitutions=v::empty<usize>();assert(v::push<usize>(ref mut substitutions,4104).is_ok);let root=syntax::declaration_at(ref p,1).result_type;let invalid=types::specialize(ref c,ref p,0,0,root,ref substitutions,ref mut graph);assert(invalid.problem.code==4501 && invalid.problem.start==0 && invalid.problem.end==0);assert(types::specialize(ref c,ref p,0,2,root,ref substitutions,ref mut graph).problem.code==4502 && types::count(ref graph)==0);
    assert(types::resolve_argument(ref c,ref p,0,0,argument_root(ref p,4),ref substitutions,ref mut graph).problem.code==4501 && types::count(ref graph)==0);
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=4100,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);assert(types::validate(ref c,ref graph)==9);
}

@test fn void composite_substitution_rejects_without_publishing_partial_types(){
    var usize shape=0;while(shape<6){var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module root;fn ").is_ok);let spelling=if(shape==0){b"Option<Option<T>>"}else if(shape==1){b"Result<T,i32>"}else if(shape==2){b"ref T"}else if(shape==3){b"ref T[]"}else if(shape==4){b"ptr T"}else{b"T[2]"};assert(bytes::append(ref mut raw,spelling).is_ok && bytes::append(ref mut raw,b" bad<T>(){}fn i32 main(){return 0;}").is_ok);
        let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var usize id=4097;
        while(id<=4103){var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,id).is_ok);
            let root=syntax::declaration_at(ref p,1).result_type;let rejected=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(rejected.problem.code!=0 && rejected.type==0 && types::count(ref graph)==0);id+=1;
        }shape+=1;
    }
    assert(!transport::valid(0) && !transport::valid(7) && !transport::valid(8) && !transport::valid(31) && !transport::valid(39) && !transport::valid(132));
}

@test fn void flat_option_substitution_materializes_only_its_direct_scalar_payload(){
    var usize id=4097;while(id<=4103){
        var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module root;fn Option<T> wrap<T>(Option<T> value){return value;}fn Option<").is_ok && bytes::append(ref mut raw,arguments::name(id)).is_ok && bytes::append(ref mut raw,b"> written(){}fn i32 main(){return 0;}").is_ok);
        let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,id).is_ok);
        let root=syntax::declaration_at(ref p,1).result_type;let result=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);
        assert(result.problem.code==0 && result.type==2 && types::count(ref graph)==2 && types::validate(ref c,ref graph)==0);
        let option=types::at(ref graph,result.type);let scalar=types::at(ref graph,option.first);assert(option.kind==5 && option.first==1 && option.second==0 && scalar.kind==1 && scalar.scalar==transport::named(arguments::name(id)));
        {let slot=v::at_mut<usize>(ref mut args,0);*slot=option.first;}
        assert(types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph).type==result.type && types::count(ref graph)==2);
        let written=types::resolve(ref c,ref p,0,syntax::declaration_at(ref p,2).result_type,ref mut graph);assert(written.problem.code==0 && written.type==result.type && types::count(ref graph)==2);id+=1;
    }
}

@test fn void flat_option_capacity_failure_rolls_back_the_materialized_payload(){
    let raw=b"module root;fn Option<T> wrap<T>(){}fn i32 main(){return 0;}";let p=parsed(raw);let c=built_catalog(raw);var v::Vec<types::Type> graph=v::empty<types::Type>();
    assert(types::canonical(ref mut graph,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0})==1);
    var usize length=1;while(length<=4093){assert(types::canonical(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=length})==length+1);length+=1;}
    var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,4100).is_ok);let root=syntax::declaration_at(ref p,1).result_type;
    let exact=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(exact.problem.code==0 && exact.type==4096 && types::count(ref graph)==4096);
    discard v::pop<types::Type>(ref mut graph);discard v::pop<types::Type>(ref mut graph);assert(types::canonical(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=4094})==4095);
    let failed=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(failed.problem.code==10 && failed.type==0 && types::count(ref graph)==4095);
    var usize i=1;while(i<=types::count(ref graph)){let t=types::at(ref graph,i);assert(t.module_index==0 && t.declaration==0 && t.kind==if(i==1){1}else{16} && t.scalar==if(i==1){1}else{0} && t.first==if(i==1){0}else{1} && t.second==if(i==1){0}else{i-1});i+=1;}
}
