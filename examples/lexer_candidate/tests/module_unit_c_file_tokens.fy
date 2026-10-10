module tests::module_unit_c_file_tokens;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_bodies as bodies;
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
import std::text as text;
import std::fs as fs;
import sha256 as hash;

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(filesystem) fn bytes::Buffer input(usize index){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(index,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void error(u32 code,usize module_index,usize start,usize end){discard io::write_u32(code);discard io::write(" module ");discard io::write_usize(module_index);discard io::write(" at ");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);}
@effects(console,filesystem) fn ir::Program retained(ref u8[] root){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);var usize i=1;while(i<=16){let source=input(i);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    discard io::write("retained instances ");discard io::write_usize(bodies::count(closure::owned(ref ready)));discard io::write("\n");
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write("values ");error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write("lower ");error(e.code,e.module_index,e.start,e.end);retained(root)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);let result=clean::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && result.kind==1 && result.scalar==1){let b=ir::at(p,i);var usize j=1;var bool parameter=false;while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){parameter=true;}j+=1;}if(!parameter){assert(found==0);found=i;}}i+=1;}assert(found!=0);return found;}
@effects(console,filesystem) fn void output(ref ir::Program p){var bytes::Buffer c=bytes::empty();let result=emit::render(p,entry(p),ref mut c);if(result.code!=0){discard io::write("C ");error(result.code,result.module_index,result.start,result.end);}var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(17,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let name=switch(text::validate_utf8(bytes::view(ref path)[..n])){case .ok(n):{n}case .err(_):{assert(false);return;}};var fs::File file=switch(fs::create_truncate_utf8(name)){case .ok(f):{f}case .err(_):{assert(false);return;}};let written=fs::write_all(ref mut file,bytes::view(ref c));let closed=switch(fs::close(file)){case .ok(_):{true}case .err(_):{false}};assert(switch(written){case .ok(n):{n==bytes::len(ref c)}case .err(_):{false}} && closed);discard io::write("C bytes ");discard io::write_usize(bytes::len(ref c));discard io::write(" SHA256 words");let digest=hash::digest(bytes::view(ref c));var usize i=0;while(i<8){discard io::write(" ");discard io::write_u32(digest[i]);i+=1;}discard io::write("\n");}
// Args: ten private compiler sources, six authentic SDK sources, then the C artifact path.
@test @effects(console,filesystem) fn void sdk_c_executes_original_file_tokens(){let p=retained(b"module main;import file_tokens as t;import std::process as p;@effects(filesystem,console) fn i32 main(){if(p::arg_count()!=2){return 8;}var u8[4096] path=[0;4096];let n=switch(p::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{return 8;}};return t::run(ref path[..n]);}");output(ref p);}
