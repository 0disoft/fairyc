module tests::module_unit_c_sdk;
import module_unit_c_emit as emit;
import module_unit_c_sdk as sdk;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_cleanup_mir_model as clean;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(filesystem) fn bytes::Buffer input(usize index){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(index,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void error(u32 code,usize start,usize end){discard io::write_u32(code);discard io::write(" at ");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);}
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,2)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write("values ");error(e.code,e.start,e.end);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write("lower ");error(e.code,e.start,e.end);retained(root,provider)}};
}
fn usize entry(ref ir::Program p){var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.module_index==0 && f.kind==3){return i;}i+=1;}assert(false);return 0;}
@effects(console) fn void output(ref ir::Program p){var bytes::Buffer c=bytes::empty();let result=emit::render(p,entry(p),ref mut c);if(result.code!=0){discard io::write("C ");error(result.code,result.start,result.end);}assert(io::write_bytes(bytes::view(ref c)).is_ok);}
@test @effects(console,filesystem) fn void sdk_c_executes_process_arguments_through_exact_result_bridge(){let source=input(1);let p=retained(b"module main;import std::process as p;fn i32 main(){assert(p::arg_count()==3);var u8[4] data=[0;4];let usize n=switch(p::arg(1,ref mut data[..])){case .ok(n):{n}case .err(_):{assert(false);0}};assert(n==3 && data[0]==97 && data[1]==98 && data[2]==99);let usize second=switch(p::arg(2,ref mut data[..])){case .ok(n):{n}case .err(_):{assert(false);0}};assert(second==2 && data[0]==206 && data[1]==187);assert(!p::arg(3,ref mut data[..]).is_ok);var u8[1] small=[0];assert(!p::arg(1,ref mut small[..]).is_ok);return 0;}",bytes::view(ref source));output(ref p);}
@test @effects(console,filesystem) fn void sdk_c_executes_arena_factory_switch_and_owned_cleanup(){let source=input(2);let p=retained(b"module main;import std::alloc as alloc;fn i32 main(){let alloc::Arena arena=switch(alloc::create()){case .ok(value):{value}case .err(_):{assert(false);return 1;}};discard arena;let Result<alloc::Arena,alloc::AllocError> ignored=alloc::create();discard ignored;return 0;}",bytes::view(ref source));output(ref p);}
@test @effects(console,filesystem) fn void sdk_c_executes_console_numbers_bytes_and_record_result_bridges(){let source=input(3);let p=retained(b"module main;import std::io as io;@effects(console)fn i32 main(){assert(io::write_u32(42).is_ok);discard io::write(\"|\");assert(io::write_i32(-7).is_ok);discard io::write(\"|\");assert(io::write_usize(9).is_ok);discard io::write(\"|\");assert(io::write_i64(-9223372036854775808).is_ok);discard io::write(\"|λ\\n\");return 0;}",bytes::view(ref source));output(ref p);}
@test @effects(console,filesystem) fn void sdk_c_plan_respects_work_budget_before_authority_or_output(){let source=input(1);let p=retained(b"module main;import std::process as p;fn i32 main(){discard p::arg_count();return 0;}",bytes::view(ref source));var usize work=1048576;assert(switch(sdk::build(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e.code==1002}});}
@test @effects(console,filesystem) fn void sdk_c_executes_stdin_and_stderr_byte_results(){let source=input(3);let p=retained(b"module main;import std::io as io;@effects(console)fn i32 main(){var u8[5] data=[0;5];var usize position=0;while(position<5){let usize n=switch(io::read_stdin(ref mut data[position..])){case .ok(n):{n}case .err(_):{assert(false);0}};assert(n>0);position+=n;}assert(data[0]==65 && data[1]==0 && data[2]==206 && data[3]==187 && data[4]==90);let usize eof=switch(io::read_stdin(ref mut data[..])){case .ok(n):{n}case .err(_):{assert(false);1}};assert(eof==0);let ref u8[] got=ref data;assert(io::write_bytes(got).is_ok);assert(io::write_stderr_u32(42).is_ok);discard io::write_stderr(\"|\");assert(io::write_stderr_i32(-7).is_ok);discard io::write_stderr(\"|\");assert(io::write_stderr_usize(9).is_ok);discard io::write_stderr(\"|\");assert(io::write_stderr_i64(-9223372036854775808).is_ok);discard io::write_stderr(\"|\");assert(io::write_stderr_bool(true).is_ok);discard io::write_stderr(\"|λ\\n\");return 0;}",bytes::view(ref source));output(ref p);}
