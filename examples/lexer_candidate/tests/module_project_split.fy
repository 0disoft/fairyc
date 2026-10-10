module tests::module_project_split;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
fn syntax::Parsed text(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(p):{p}case .err(_):{assert(false);return text(b"module empty;");}};return parsed(b);}
fn syntax::Parsed provider(ref u8[] name){
    var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module ").is_ok);assert(bytes::append(ref mut source,name).is_ok);assert(bytes::append(ref mut source,b";").is_ok);var u8 i=0;
    while(i<8){let u8[1] letter=[97+i];assert(bytes::append(ref mut source,b"fn T ").is_ok);assert(bytes::append(ref mut source,ref letter[..]).is_ok);assert(bytes::append(ref mut source,b"<T>(T n){return n;}").is_ok);i+=1;}
    assert(bytes::append(ref mut source,b"pub fn i32 run(){").is_ok);i=0;
    while(i<8){let u8[1] letter=[97+i];var usize type=0;while(type<5){assert(bytes::append(ref mut source,b"discard ").is_ok);assert(bytes::append(ref mut source,ref letter[..]).is_ok);let call=if(type==0){b"<bool>(false);"}else if(type==1){b"<u8>(1);"}else if(type==2){b"<u32>(1);"}else if(type==3){b"<i32>(1);"}else{b"<usize>(1);"};assert(bytes::append(ref mut source,call).is_ok);type+=1;}i+=1;}
    assert(bytes::append(ref mut source,b"return 0;}").is_ok);return parsed(source);
}
@test fn void generic_capacity_splits_without_losing_source_coverage(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(b"module main;import left as a;import right as b;fn i32 main(){return a::run()+b::run();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,provider(b"left")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,provider(b"right")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(compile::range(ref sources,ref metadata,0,3)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==3 && e.code==4503);}};
    let units=switch(compile::build(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(compile::count(ref units)==3 && compile::retries(ref units)==2);var usize i=0;while(i<3){let range=compile::at(ref units,i);assert(range.first==i && range.stop==i+1);i+=1;}let p=switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::count(ref p)==83);
}
@test fn void move_diagnostics_cannot_be_returned_as_successful_owned_ir(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(b"module main;resource Item{i32 n;cleanup{}}fn void take(Item n){discard n;}fn i32 main(){let value=Item{.n=7};take(value);take(value);return 0;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(compile::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==5 && e.code>=1100 && e.start<e.end && e.first==0 && e.stop==1);}};
}
