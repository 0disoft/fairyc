module tests::composite_result_transport;
import generic_scalar_arguments as arguments;
import generic_transport_scalars as transport;
import module_types as types;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn catalog::Catalog built_catalog(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);return switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);built_catalog(raw)}};}
fn types::Type scalar(u32 code){return types::Type{.kind=1,.scalar=code,.module_index=0,.declaration=0,.first=0,.second=0};}
fn bool same(types::Type a,types::Type b){return a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && a.first==b.first && a.second==b.second;}
fn v::Vec<types::Type> copied(ref v::Vec<types::Type> graph){var v::Vec<types::Type> rows=v::empty<types::Type>();var usize i=1;while(i<=types::count(graph)){assert(v::push<types::Type>(ref mut rows,types::at(graph,i)).is_ok);i+=1;}return rows;}
fn void unchanged(ref v::Vec<types::Type> graph,ref v::Vec<types::Type> before){assert(types::count(graph)==types::count(before));var usize i=1;while(i<=types::count(before)){assert(same(types::at(graph,i),types::at(before,i)));i+=1;}}
fn bytes::Buffer wrapped_source(ref u8[] payload){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"module root;fn Result<T,u32> wrap<T>(Result<T,u32> value){return value;}fn Result<").is_ok && bytes::append(ref mut out,payload).is_ok && bytes::append(ref mut out,b",u32> written(){}fn Result<T,E> bound<T,E>(Result<T,E> value){return value;}").is_ok);return out;}

@test fn void result_descriptor_interned_and_written_payloads_are_canonical(){
    var usize descriptor=4097;while(descriptor<=4103){let raw=wrapped_source(arguments::name(descriptor));let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var v::Vec<types::Type> graph=v::empty<types::Type>();assert(types::canonical(ref mut graph,scalar(4))==1);var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,descriptor).is_ok);let root=syntax::declaration_at(ref p,1).result_type;
        let resolved=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(resolved.problem.code==0 && resolved.type==3 && types::count(ref graph)==3);let result=types::at(ref graph,resolved.type);assert(result.kind==6 && result.first==2 && result.second==1 && result.scalar==0 && same(types::at(ref graph,2),scalar(transport::named(arguments::name(descriptor)))));
        {let slot=v::at_mut<usize>(ref mut args,0);*slot=2;}let interned=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(interned.problem.code==0 && interned.type==resolved.type && types::count(ref graph)==3);
        let written=types::resolve(ref c,ref p,0,syntax::declaration_at(ref p,2).result_type,ref mut graph);assert(written.problem.code==0 && written.type==resolved.type && types::count(ref graph)==3);
        assert(v::push<usize>(ref mut args,1).is_ok);let bound=types::specialize(ref c,ref p,0,4,syntax::declaration_at(ref p,3).result_type,ref args,ref mut graph);assert(bound.problem.code==0 && bound.type==resolved.type && types::count(ref graph)==3 && types::validate(ref c,ref graph)==0);descriptor+=1;
    }
}

fn v::Vec<types::Type> filled(usize count){var v::Vec<types::Type> graph=v::empty<types::Type>();assert(types::canonical(ref mut graph,scalar(4))==1 && types::canonical(ref mut graph,scalar(1))==2);var usize i=3;while(i<=count){assert(types::canonical(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=2,.second=i-2})==i);i+=1;}return graph;}
@test fn void result_capacity_keeps_known_error_and_rolls_back_every_record(){
    let raw=wrapped_source(b"u16");let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));let root=syntax::declaration_at(ref p,1).result_type;var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,4100).is_ok);
    var graph=filled(4094);let exact=types::specialize(ref c,ref p,0,2,root,ref args,ref mut graph);assert(exact.problem.code==0 && exact.type==4096 && types::count(ref graph)==4096 && same(types::at(ref graph,4095),scalar(35)));let result=types::at(ref graph,4096);assert(result.kind==6 && result.first==4095 && result.second==1 && types::validate(ref c,ref graph)==0);
    var full=filled(4095);let before=copied(ref full);assert(types::validate(ref c,ref full)==0);let failed=types::specialize(ref c,ref p,0,2,root,ref args,ref mut full);assert(failed.type==0 && failed.problem.code==10);unchanged(ref full,ref before);
    let written=types::resolve(ref c,ref p,0,syntax::declaration_at(ref p,2).result_type,ref mut full);assert(written.type==0 && written.problem.code==10);unchanged(ref full,ref before);
}

@test fn void result_shallow_boundaries_rollback_materialized_leaves(){
    var usize shape=0;while(shape<10){let spelling=if(shape==0){b"Result<T,i32>"}else if(shape==1){b"Result<u32,T>"}else if(shape==2){b"Result<T,T>"}else if(shape==3){b"Result<Option<T>,u32>"}else if(shape==4){b"Result<Result<T,u32>,u32>"}else if(shape==5){b"Option<Result<T,u32>>"}else if(shape==6){b"ref Result<T,u32>"}else if(shape==7){b"ptr Result<T,u32>"}else if(shape==8){b"Result<T,u32>[2]"}else{b"Result<T,Unknown>"};var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module root;fn ").is_ok && bytes::append(ref mut raw,spelling).is_ok && bytes::append(ref mut raw,b" rejected<T>(){}").is_ok);let p=parsed(bytes::view(ref raw));let c=built_catalog(bytes::view(ref raw));var usize descriptor=4097;
        while(descriptor<=4103){var graph=filled(4);let before=copied(ref graph);var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,descriptor).is_ok);let r=types::specialize(ref c,ref p,0,2,syntax::declaration_at(ref p,1).result_type,ref args,ref mut graph);assert(r.type==0 && r.problem.code!=0);unchanged(ref graph,ref before);descriptor+=1;}shape+=1;
    }
    let raw=b"module root;fn Result<T,E> bound<T,E>(){}";let p=parsed(raw);let c=built_catalog(raw);var graph=filled(4);let before=copied(ref graph);var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,4100).is_ok && v::push<usize>(ref mut args,2).is_ok);let r=types::specialize(ref c,ref p,0,2,syntax::declaration_at(ref p,1).result_type,ref args,ref mut graph);assert(r.type==0 && r.problem.code!=0);unchanged(ref graph,ref before);
}

@test fn void result_batch_failure_restores_the_successful_prefix(){
    let raw=b"module root;fn Result<T,u32> mixed<T>(Result<T,i32> bad){}";let p=parsed(raw);let c=built_catalog(raw);var graph=filled(4);let before=copied(ref graph);let d=syntax::declaration_at(ref p,1);let entry=syntax::entry_at(ref p,d.entry_start);var v::Vec<usize> roots=v::empty<usize>();assert(v::push<usize>(ref mut roots,d.result_type).is_ok && v::push<usize>(ref mut roots,entry.type_root).is_ok);var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,4100).is_ok);
    switch(types::specialize_many(ref c,ref p,0,2,ref roots,ref args,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.request==2 && e.problem.code!=0);}};unchanged(ref graph,ref before);
}
