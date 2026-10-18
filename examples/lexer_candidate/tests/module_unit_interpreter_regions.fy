module tests::module_unit_interpreter_regions;
import module_unit_interpreter as interpreter;
import module_unit_interpreter_values as runtime;
import project_compiler_core as core;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(text);}};let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(text);}};return switch(syntax::parse(tokens,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(text)}};}
@effects(filesystem) fn bytes::Buffer allocator(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(files::load(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn identified::Program checked(ref u8[] text,ref u8[] allocator){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(allocator)).is_ok);
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok && v::push<usize>(ref mut owners,1).is_ok);
    var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok);
    var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<2){let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize j=0;while(j<64){assert(bytes::append(ref mut names,if(i==0){b"1"}else{b"2"}).is_ok);j+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=i,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}
    let table=coordinates::assemble(names,rows);var usize source_work=0;var usize link_work=0;
    return switch(core::build_identified_sources(ref sources,ref owners,ref imports,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.phase);discard io::write("/");discard io::write_u32(e.stage);discard io::write("/");discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write("\n");discard io::write_bytes(text);discard io::write("\n");assert(false);checked(text,allocator)}};
}
@effects(console) fn void run(ref u8[] text,ref u8[] allocator,u32 trap){
    let owned=checked(text,allocator);let p=identified::ir_program(ref owned);let plan=switch(interpreter::prepare(p)){case .ok(p):{p}case .err(e):{discard io::write_u32(e);assert(false);return;}};
    var usize frame=1;while(frame<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),frame);if(f.module_index==0 && f.kind==3 && f.declaration==3){break;}frame+=1;}
    var v::Vec<usize> heap=v::empty<usize>();var usize steps=1000000;var v::Vec<usize> args=v::empty<usize>();var usize i=0;
    // Supply two abstract borrowed Arenas. Neither constructor nor destructor
    // is executed; the source is still the exact authenticated SDK module.
    while(i<2){let arena=runtime::allocate(ref mut heap,3,ref mut steps);assert(arena.error==0);let root=runtime::rx_cell(ref mut heap,arena.bits,ref mut steps);assert(root.error==0);let borrow=runtime::rx_descriptor(ref mut heap,root.bits,0,0,0,ref mut steps);assert(borrow.error==0 && v::push<usize>(ref mut args,borrow.bits).is_ok);i+=1;}
    let done=interpreter::execute(ref owned,ref plan,frame,ref args,ref mut heap,ref mut steps,0);
    if(done.error!=0 || done.trap!=trap){discard io::write_u32(done.error);discard io::write("/");discard io::write_u32(done.trap);discard io::write("\n");assert(false);}assert(steps<1000000);
    // All activations have reset on successful completion. Traps deliberately
    // leave the activation live, as they never unwind cleanup/reset blocks.
    var usize active=0;i=0;while(i<v::len<usize>(ref heap)){let size=*v::at<usize>(ref heap,i);if(size==2 && i+2<v::len<usize>(ref heap) && *v::at<usize>(ref heap,i+1)==1){let owner=*v::at<usize>(ref heap,i+2);if(owner==*v::at<usize>(ref args,0) || owner==*v::at<usize>(ref args,1)){active+=1;}}i+=size+1;}
    assert(active==if(trap==0){0}else{1});
}
@test @effects(filesystem,console) fn void lexical_region_execution_and_exit_paths(){
    let sdk=allocator();let source=bytes::view(ref sdk);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{let ref i32 x=switch(r.place(40)){case .ok(n):{n}case .err(_):{assert(false);return;}};let ref i32 y=switch(r.place(2)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(*x+*y==42);}}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{assert(read(r)==42);}}fn i32 read(Region r){return switch(r.place(42)){case .ok(n):{*n}case .err(_):{assert(false);0}};}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{let ref Pair n=switch(r.place(Pair{.value=42})){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(n.value==42);let i32[2] items=[20,22];assert(r.place(items).is_ok);}}struct Pair{i32 value;}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{region s in b{discard r.place(1);discard s.place(2);return;}}}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){var i32 i=0;while(i<3){region r in a{discard r.place(i);i+=1;continue;}}while(true){region r in a{discard r.place(4);break;}}assert(i==3);}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{let Guard g=Guard{.value=42};discard r.place(3);return;}}resource Guard{i32 value;cleanup{assert(this.value==42);}}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){discard route(a);}fn Result<i32,i32> route(ref mut alloc::Arena a){region r in a{discard r.place(3);let i32 n=fail()?;return .ok(n);}}fn Result<i32,i32> fail(){return .err(7);}",source,0);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{discard r.place(3);assert(false);}}",source,6);
    run(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b){region r in a{let Guard g=Guard{};discard r.place(3);return;}}resource Guard{cleanup{assert(false);}}",source,6);
}
@test fn void reset_rejects_double_reset_and_preserves_activation_identity(){
    var v::Vec<usize> heap=v::empty<usize>();var usize steps=100;let root=runtime::rx_cell(ref mut heap,0,ref mut steps);let owner=runtime::rx_descriptor(ref mut heap,root.bits,0,0,0,ref mut steps);let first=runtime::region_begin(owner.bits,ref mut heap,ref mut steps);assert(first.error==0 && runtime::region_reset(first.bits,ref mut heap).error==0 && runtime::region_reset(first.bits,ref mut heap).error==4);let second=runtime::region_begin(owner.bits,ref mut heap,ref mut steps);assert(second.error==0 && first.bits!=second.bits && runtime::item(ref heap,first.bits,0).bits==0 && runtime::item(ref heap,second.bits,0).bits==1);
}
