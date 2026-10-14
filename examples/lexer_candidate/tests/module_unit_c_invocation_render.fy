module tests::module_unit_c_invocation_render;
import module_project_cli as cli;
import module_unit_c_emit as emit;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@test @effects(console)
fn void many_original_calls_emit_one_checked_cross_module_packet(){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module main;import helper as h;fn i32 main(){var i32 sum=0;").is_ok);
    var usize i=0;while(i<128){assert(bytes::append(ref mut text,b"sum+=h::add(1,2);").is_ok);i+=1;}assert(bytes::append(ref mut text,b"return sum-384;}").is_ok);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref text))).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module helper;pub fn i32 add(i32 a,i32 b){return a+b;}")).is_ok);
    let bundle=switch(cli::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(emit::project_frames(ref bundle)==2 && emit::project_selected_frames(ref bundle)==2 && emit::project_unit_count(ref bundle)==1);
    assert(cli::packet(ref bundle)==0);
}
