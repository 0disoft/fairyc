module tests::module_generic_transport;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import module_unit_c_emit as c;
import module_project_complete_c as complete_c;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interpreter as interpreter;
import module_project_artifact_encode as encode;
import module_project_artifact_decode as decode;
import candidate_artifact_wire as wire;
import generic_transport_scalars as transport;
import c_text as w;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn coordinates::Coordinates table(){let raw=b"fypc1-sha256-1111111111111111111111111111111111111111111111111111111111111111";let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return table();}};var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=raw.len}).is_ok);return coordinates::assemble(names,rows);}
@effects(console) fn core::Complete compiled_mode(ref u8[] raw,bool aggregates){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();let table=table();var usize work=0;var usize linking=0;
    let result=switch(if(aggregates){core::build_complete_aggregate_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking)}else{core::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking)}){
        case .ok(p):{p}
        case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);discard io::write_stderr("/");discard io::write_stderr_usize(e.start);assert(false);return compiled_mode(raw,aggregates);}
    };
    work=0;let saved=switch(encode::encode(core::checked_program(ref result),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return compiled_mode(raw,aggregates);}};work=0;
    let artifact=wire::content(ref saved);
    let detached=switch(if(aggregates){decode::decode_aggregates(artifact,ref mut work)}else{decode::decode(artifact,ref mut work)}){case .ok(p):{p}case .err(_):{assert(false);return compiled_mode(raw,aggregates);}};
    var usize source_work=0;var usize artifact_work=0;var usize link_work=0;
    return switch(if(aggregates){core::restore_complete_aggregate_sources(detached,ref sources,ref owners,ref imports,ref table,ref mut source_work,ref mut artifact_work,ref mut link_work)}else{core::restore_complete_sources(detached,ref sources,ref owners,ref imports,ref table,ref mut source_work,ref mut artifact_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);compiled_mode(raw,aggregates)}};
}
@effects(console) fn core::Complete compiled(ref u8[] raw){return compiled_mode(raw,false);}
fn void rejected(ref u8[] raw){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();let table=table();var usize work=0;var usize linking=0;
    assert(switch(core::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking)){case .ok(_):{false}case .err(e):{e.code!=0 && e.end>e.start && e.end<=raw.len}});
}
fn void append(ref mut bytes::Buffer raw,ref u8[] part){assert(bytes::append(raw,part).is_ok);}
@test
@effects(console)
fn void inferred_scalar_calls_restore_and_execute(){
    let raw=b"module root;fn T id<T>(T value){return value;}fn T answer<T>(){return id(42);}fn T relay<T>(){return answer();}fn T forward<T>(T value){let local=id(value);return local;}fn i32 main(){let u32 value={relay()};assert(value==42);assert(id(true));discard id<i32>(42);discard id(42);let i32 number=42;if(true){let bool number=true;assert(id(number));}assert(id(number)==42);let u32 branch=if (true) {let u32 inner=42;assert(id(inner)==42);relay()} else if (false) {relay()} else {let Option<bool> choice=.some(true);switch (choice) {case .some(_):{relay()} case .none:{relay()}}};assert(branch==42);return forward<i32>(number);}";
    let result=compiled(raw);
    let owner=core::checked_program(ref result);
    let program=checked::ir_program(owner);
    var usize entry=0;var usize i=1;
    while(i<=ir::count(program)){if(clean::frame_at(ir::cleanup_ir(program),i).declaration==6){entry=i;}i+=1;}
    assert(entry!=0);
    let plan=switch(interpreter::prepare(program)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let args=v::empty<usize>();var heap=v::empty<usize>();var usize steps=65536;
    let outcome=interpreter::execute(owner,ref plan,entry,ref args,ref mut heap,ref mut steps,0);
    assert(outcome.error==0 && outcome.trap==0 && outcome.bits==42);
}
@test
@effects(console)
fn void inferred_constructor_calls_restore_and_execute(){
    let raw=b"module root;fn bool nested<T>(Option<Option<T>> value){discard value;return true;}fn bool success<T>(Result<T,u32> value){discard value;return true;}fn bool failure<T>(Result<i32,T> value){discard value;return true;}fn i32 main(){let u8 number=42;assert(nested((.some(.some(42)))));assert(nested<i32>(.some(.some(42))));assert(success(.ok(number as u32)));assert(failure(.err(true)));let Option<Option<i32>> absent=.none;assert(nested(absent));let Result<u32,u32> accepted=.ok(42);assert(success(accepted));let Result<i32,bool> failed=.err(true);assert(failure(failed));return 42;}";
    let result=compiled(raw);
    let owner=core::checked_program(ref result);
    let program=checked::ir_program(owner);
    var usize entry=0;var usize i=1;
    while(i<=ir::count(program)){if(clean::frame_at(ir::cleanup_ir(program),i).declaration==5){entry=i;}i+=1;}
    assert(entry!=0);
    let body=ir::at(program,entry);
    assert(ir::call_count(body)==7);
    let inferred=ir::call_at(body,0);
    assert(inferred.instance!=0 && inferred.instance==ir::call_at(body,1).instance);
    assert(inferred.instance==ir::call_at(body,4).instance);
    assert(ir::call_at(body,2).instance==ir::call_at(body,5).instance);
    assert(ir::call_at(body,3).instance==ir::call_at(body,6).instance);
    let plan=switch(interpreter::prepare(program)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let args=v::empty<usize>();var heap=v::empty<usize>();var usize steps=65536;
    let outcome=interpreter::execute(owner,ref plan,entry,ref args,ref mut heap,ref mut steps,0);
    assert(outcome.error==0 && outcome.trap==0 && outcome.bits==42);
    rejected(b"module root;fn bool accept<T>(Option<T> value){discard value;return true;}fn i32 main(){discard accept(.none);return 0;}");
    rejected(b"module root;fn bool accept<T>(Result<T,u32> value){discard value;return true;}fn i32 main(){discard accept(.err(42));return 0;}");
}
@test
@effects(console)
fn void inferred_tagged_results_restore_and_execute(){
    let raw=b"module root;fn T answer<T>(){return 42;}fn Option<T> option<T>(){return .some(answer());}fn Result<A,B> result<A,B>(){return .ok(answer());}fn i32 main(){let Option<i32> left=option();let Result<i32,u32> right=result();let Option<Result<i32,u32>> nested=.some(.err(answer()));assert(switch(nested){case .some(payload):{!payload.is_ok}case .none:{false}});assert(right.is_ok);assert(present(wrap(42)));assert(present(wrap<u32>(42)));assert(check(ready(42)));assert(check(relay(true)));return switch(left){case .some(value):{value}case .none:{0}};}fn Option<T> wrap<T>(T value){return .some(value);}fn bool present<T>(Option<T> value){return switch(value){case .some(_):{true}case .none:{false}};}fn Result<T,u32> ready<T>(T value){return .ok(value);}fn bool check<T,E>(Result<T,E> value){return value.is_ok;}fn Result<T,u32> relay<T>(T value){let ok=check(ready(value));assert(ok);return ready(value);}";
    let result=compiled(raw);
    let owner=core::checked_program(ref result);
    let program=checked::ir_program(owner);
    var usize entry=0;var usize i=1;
    while(i<=ir::count(program)){if(clean::frame_at(ir::cleanup_ir(program),i).declaration==5){entry=i;}i+=1;}
    assert(entry!=0);
    let plan=switch(interpreter::prepare(program)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let args=v::empty<usize>();var heap=v::empty<usize>();var usize steps=65536;
    let outcome=interpreter::execute(owner,ref plan,entry,ref args,ref mut heap,ref mut steps,0);
    assert(outcome.error==0 && outcome.trap==0 && outcome.bits==42);
}
@test
@effects(console)
fn void inferred_cleanup_calls_share_explicit_instances(){
    let raw=b"module root;fn T id<T>(T value){return value;}resource Hold<T>{T value;cleanup{let T inferred=id(42);let T written=id<T>(42);assert(inferred==written);}}fn i32 main(){let value=Hold<u32>{.value=42};discard value;return 42;}";
    let result=compiled_mode(raw,true);
    let owner=core::checked_program(ref result);
    let program=checked::ir_program(owner);
    var usize entry=0;var usize cleanup_frame=0;var usize i=1;
    while(i<=ir::count(program)){
        let frame=clean::frame_at(ir::cleanup_ir(program),i);
        if(frame.kind==2){cleanup_frame=i;}
        if(frame.declaration==4){entry=i;}
        i+=1;
    }
    assert(entry!=0 && cleanup_frame!=0);
    let body=ir::at(program,cleanup_frame);
    assert(ir::call_count(body)==2);
    let inferred=ir::call_at(body,0);
    assert(inferred.instance!=0 && inferred.instance==ir::call_at(body,1).instance);
    let plan=switch(interpreter::prepare(program)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let args=v::empty<usize>();var heap=v::empty<usize>();var usize steps=65536;
    let outcome=interpreter::execute(owner,ref plan,entry,ref args,ref mut heap,ref mut steps,0);
    assert(outcome.error==0 && outcome.trap==0 && outcome.bits==42);
}
@test fn void extended_transport_keeps_operations_and_composite_signatures_closed(){
    var u32 scalar=32;while(scalar<=38){let name=transport::name(scalar);var usize shape=0;while(shape<8){
        var bytes::Buffer raw=bytes::empty();append(ref mut raw,b"module root;");
        if(shape==0){append(ref mut raw,b"fn T bad<T>(T a,T b){return a+b;}fn ");append(ref mut raw,name);append(ref mut raw,b" probe(");append(ref mut raw,name);append(ref mut raw,b" a,");append(ref mut raw,name);append(ref mut raw,b" b){return bad<");append(ref mut raw,name);append(ref mut raw,b">(a,b);}");}
        else if(shape==1){append(ref mut raw,b"fn Option<Option<Option<T>>> bad<T>(){return .none;}");}
        else if(shape==2){append(ref mut raw,b"fn Result<T,i64> bad<T>(){return .err(1);}");}
        else if(shape==3){append(ref mut raw,b"fn ref T bad<T>(ref T p){return p;}");}
        else if(shape==4){append(ref mut raw,b"fn ref T[] bad<T>(ref T[] p){return p;}");}
        else if(shape==5){append(ref mut raw,b"fn ptr T bad<T>(ptr T p){return p;}");}
        else if(shape==6){append(ref mut raw,b"fn T[2] bad<T>(T x){return [x;2];}");}
        else{append(ref mut raw,b"unsafe extern C fn ");append(ref mut raw,name);append(ref mut raw,b" direct();");}
        append(ref mut raw,b"fn i32 main(){");if(shape>=1 && shape<=6){append(ref mut raw,b"discard bad<");append(ref mut raw,name);append(ref mut raw,b">();");}append(ref mut raw,b"return 0;}");rejected(bytes::view(ref raw));shape+=1;
    }scalar+=1;}
}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok);}
// Private C observers seed the same boundaries as the owning Go native fixture.
@effects(console) pub fn void emit_transport(ref u8[] raw){
    let result=compiled(raw);let program=checked::ir_program(core::checked_program(ref result));var usize entry=0;var usize[7] probes=[0;7];var usize i=1;
    while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);if(frame.module_index==0 && frame.declaration==13){entry=i;}if(frame.module_index==0 && frame.declaration>=5 && frame.declaration<=11){probes[frame.declaration-5]=i;}i+=1;}assert(entry!=0);
    var bytes::Buffer bridges=bytes::empty();var w::State status=w::State{.code=0};assert(w::text(ref mut bridges,ref mut status,b"#include \"fy_project.h\"\n"));i=0;var u32 code=32;while(i<7){assert(probes[i]!=0);assert(w::type(ref mut bridges,ref mut status,code) && w::text(ref mut bridges,ref mut status,b" fairyc_probe_") && w::text(ref mut bridges,ref mut status,transport::name(code)) && w::text(ref mut bridges,ref mut status,b"(") && w::type(ref mut bridges,ref mut status,code) && w::text(ref mut bridges,ref mut status,b" value){return fy_fn_") && w::number(ref mut bridges,ref mut status,probes[i]) && w::text(ref mut bridges,ref mut status,b"(value); }\n"));i+=1;code+=1;}
    let bundle=switch(complete_c::partition(result,entry,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return;}};
    assert(io::write("FYCPARTS1\n").is_ok);member(c::project_header(ref bundle));member(c::project_literals(ref bundle));assert(io::write_usize(c::project_unit_count(ref bundle)).is_ok && io::write("\n").is_ok);i=0;while(i<c::project_unit_count(ref bundle)){if(i+1==c::project_unit_count(ref bundle)){var bytes::Buffer tail=bytes::empty();append(ref mut tail,c::project_source(ref bundle,i));append(ref mut tail,bytes::view(ref bridges));member(bytes::view(ref tail));}else{member(c::project_source(ref bundle,i));}i+=1;}
}
