module tests::module_project_compile;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_signatures as signatures;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
fn syntax::Parsed text(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(p):{p}case .err(_):{assert(false);return text(b"module empty;");}};return parsed(b);}
@effects(console) fn void failed(compile::Problem e){discard io::write("stage ");discard io::write_u32(e.stage);discard io::write(" code ");discard io::write_u32(e.code);discard io::write(" module ");discard io::write_usize(e.module_index);discard io::write(" at ");discard io::write_usize(e.start);discard io::write("..");discard io::write_usize(e.end);discard io::write(" range ");discard io::write_usize(e.first);discard io::write("..");discard io::write_usize(e.stop);discard io::write("\n");}
@test fn void coordinated_units_cover_all_sources_and_link(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(b"module main;import provider as p;fn i32 main(){return p::run(40);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,text(b"module provider;import helper as h;pub fn i32 run(i32 n){return h::add(n,2);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,text(b"module helper;pub fn i32 add(i32 a,i32 b){return a+b;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let units=switch(compile::build(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(compile::count(ref units)==1 && compile::retries(ref units)==0);let range=compile::at(ref units,0);assert(range.first==0 && range.stop==3);let p=switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::count(ref p)==3);
}
@test fn void coordinated_units_preserve_semantic_failures(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(b"module main;fn i32 main(){return true;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};switch(compile::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==4 && e.code!=10 && e.module_index==0 && e.start<e.end && e.first==0 && e.stop==1);}};
}
@test @effects(filesystem,console) fn void original_main_all_units_compile(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){var u8[4096] path=[0;4096];let n=switch(process::arg(i+1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let b=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write("signatures ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    let units=switch(compile::build(ref sources,ref metadata)){case .ok(p):{p}case .err(e):{failed(e);assert(false);return;}};discard io::write("compiled units ");discard io::write_usize(compile::count(ref units));discard io::write(" retries ");discard io::write_usize(compile::retries(ref units));discard io::write("\n");var usize next=0;i=0;while(i<compile::count(ref units)){let range=compile::at(ref units,i);assert(range.first==next && range.first<range.stop);next=range.stop;i+=1;}assert(next==count);
}

@test @effects(filesystem,console) fn void original_origin_flow_unit_compile(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){var u8[4096] path=[0;4096];let n=switch(process::arg(i+1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let b=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write("signatures ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    let declaration=syntax::declaration_at(v::at<syntax::Parsed>(ref sources,138),0);assert(bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,138))[declaration.name_start..declaration.name_end],b"module_unit_origin_flow"));
    let unit=switch(compile::range(ref sources,ref metadata,138,139)){case .ok(p):{p}case .err(e):{failed(e);assert(false);return;}};assert(proof::check(ref unit).code==0);var usize maximum=0;i=1;while(i<=ir::count(ref unit)){let count=ir::node_count(ir::at(ref unit,i));if(count>maximum){maximum=count;}i+=1;}assert(maximum>4160 && maximum<=limits::program_expressions());discard io::write("maximum nodes ");discard io::write_usize(maximum);discard io::write(" owned frames ");discard io::write_usize(ir::count(ref unit));discard io::write(" work ");discard io::write_usize(ir::work(ref unit));discard io::write("\n");
}
