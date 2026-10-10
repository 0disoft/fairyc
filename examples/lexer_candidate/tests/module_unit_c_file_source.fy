module tests::module_unit_c_file_source;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import module_unit_region_model as regions;
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
@effects(console,filesystem) fn ir::Program retained(ref u8[] root){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);var usize i=1;while(i<=5){let source=input(i);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write("values ");error(e.code,e.start,e.end);return retained(root);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write("lower ");error(e.code,e.start,e.end);retained(root)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);let result=clean::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && result.kind==1 && result.scalar==1){let b=ir::at(p,i);var usize j=1;var bool parameter=false;while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){parameter=true;}j+=1;}if(!parameter){assert(found==0);found=i;}}i+=1;}assert(found!=0);return found;}
@effects(console) fn void output(ref ir::Program p){var bytes::Buffer c=bytes::empty();let result=emit::render(p,entry(p),ref mut c);if(result.code!=0){discard io::write("C ");error(result.code,result.start,result.end);}assert(io::write_bytes(bytes::view(ref c)).is_ok);}
@test @effects(console,filesystem) fn void sdk_c_executes_original_file_loader_limits_and_errors(){let p=retained(b"module main;import file_source as f;import std::bytes as b;@effects(filesystem) fn i32 main(){let b::Buffer small=switch(f::load(b\"small-λ.bin\")){case .ok(v):{v}case .err(_):{assert(false);return 1;}};let ref u8[] view=b::view(ref small);assert(view.len==5 && view[0]==65 && view[1]==0 && view[2]==206 && view[3]==187 && view[4]==90);let b::Buffer boundary=switch(f::load(b\"exact.bin\")){case .ok(v):{v}case .err(_):{assert(false);return 1;}};let ref u8[] exact=b::view(ref boundary);assert(exact.len==65536 && exact[0]==0 && exact[65535]==24);let b::Buffer empty=switch(f::manifest(b\"empty.bin\")){case .ok(v):{v}case .err(_):{assert(false);return 1;}};assert(b::len(ref empty)==0);switch(f::load(b\"too-big.bin\")){case .ok(_):{assert(false);}case .err(code):{assert(code==4);}};switch(f::load(b\"missing.bin\")){case .ok(_):{assert(false);}case .err(code):{assert(code==1);}};switch(f::load(b\"\\xff\")){case .ok(_):{assert(false);}case .err(code):{assert(code==1);}};switch(f::manifest(b\"\")){case .ok(_):{assert(false);}case .err(code):{assert(code==1);}};switch(f::manifest(b\"\\0\")){case .ok(_):{assert(false);}case .err(code):{assert(code==1);}};return 0;}");output(ref p);}
@test @effects(console,filesystem) fn void sdk_c_executes_original_file_loader_allocation_failure(){let p=retained(b"module main;import file_source as f;@effects(filesystem) fn i32 main(){switch(f::load(b\"small-λ.bin\")){case .ok(_):{assert(false);}case .err(code):{assert(code==3);}};return 0;}");output(ref p);}
