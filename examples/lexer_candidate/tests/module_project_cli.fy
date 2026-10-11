module tests::module_project_cli;
import module_project_cli as cli;
import module_unit_c_emit as emit;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let result=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    assert(syntax::problem_count(ref result)==0);return result;
}
@test @effects(console)
fn void root_main_selection_and_cross_module_packet(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import math as math;fn i32 main(){return math::add(40,2)-42;}fn i32 unused(){return 99;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module math;pub fn i32 add(i32 a,i32 b){return a+b;}")).is_ok);
    let bundle=switch(cli::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(emit::project_frames(ref bundle)==3 && emit::project_selected_frames(ref bundle)==2 && emit::project_unit_count(ref bundle)==1);
    assert(cli::packet(ref bundle)==0);
}
@test
fn void dependency_main_and_invalid_entry_do_not_publish_a_bundle(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module root;fn void helper(){}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module other;pub fn i32 main(){return 0;}")).is_ok);
    switch(cli::build(ref sources)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==4 && e.code==1001 && e.module_index==0);}};
    var v::Vec<syntax::Parsed> invalid=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut invalid,parsed(b"module main;fn void main(){}")).is_ok);
    switch(cli::build(ref invalid)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 && e.code==1001 && e.module_index==0);}};
}

@test @effects(console)
fn void highest_source_module_keeps_nominal_and_callable_identity(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import last as last;fn i32 main(){let last::S value=last::make();return value.n-7;}")).is_ok);
    var usize i=1;while(i<511){
        var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module filler").is_ok && bytes::append_usize(ref mut text,i).is_ok && bytes::append(ref mut text,b";").is_ok);
        assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref text))).is_ok);i+=1;
    }
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module last;pub struct S{i32 n;}pub fn S make(){return S{.n=7};}")).is_ok);
    let bundle=switch(cli::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(emit::project_frames(ref bundle)==2 && emit::project_selected_frames(ref bundle)==2);
    assert(cli::packet(ref bundle)==0);
}
