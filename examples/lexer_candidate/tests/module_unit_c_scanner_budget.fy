module tests::module_unit_c_scanner_budget;
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
@effects(console,filesystem) fn ir::Program retained(ref u8[] root){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);var usize i=1;while(i<=14){let source=input(i);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);i+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    discard io::write("retained instances ");discard io::write_usize(bodies::count(closure::owned(ref ready)));discard io::write("\n");
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write("values ");error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{error(e.code,e.module_index,e.start,e.end);return retained(root);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write("lower ");error(e.code,e.module_index,e.start,e.end);retained(root)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);let result=clean::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && result.kind==1 && result.scalar==1){let b=ir::at(p,i);var usize j=1;var bool parameter=false;while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){parameter=true;}j+=1;}if(!parameter){assert(found==0);found=i;}}i+=1;}assert(found!=0);return found;}
@effects(console,filesystem) fn void output(ref ir::Program p){var bytes::Buffer c=bytes::empty();let result=emit::render(p,entry(p),ref mut c);if(result.code!=0){discard io::write("C ");error(result.code,result.module_index,result.start,result.end);}var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(15,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let name=switch(text::validate_utf8(bytes::view(ref path)[..n])){case .ok(n):{n}case .err(_):{assert(false);return;}};var fs::File file=switch(fs::create_truncate_utf8(name)){case .ok(f):{f}case .err(_):{assert(false);return;}};let written=fs::write_all(ref mut file,bytes::view(ref c));let closed=switch(fs::close(file)){case .ok(_):{true}case .err(_):{false}};assert(switch(written){case .ok(n):{n==bytes::len(ref c)}case .err(_):{false}} && closed);discard io::write("C bytes ");discard io::write_usize(bytes::len(ref c));discard io::write(" SHA256 words");let digest=hash::digest(bytes::view(ref c));var usize i=0;while(i<8){discard io::write(" ");discard io::write_u32(digest[i]);i+=1;}discard io::write("\n");}
// Args: eight private sources, six authentic SDK sources, then the C artifact path.
@test @effects(console,filesystem) fn void sdk_c_executes_original_scanner_budgets(){let p=retained(b"module main;import scanner as s;import file_source as f;import std::bytes as b;import std::process as p;fn Result<usize,i32> number(usize index){var u8[32] raw=[0;32];let n=switch(p::arg(index,ref mut raw)){case .ok(n):{n}case .err(_):{return .err(8);}};if(n==0){return .err(8);}var usize value=0;var usize i=0;while(i<n){let u8 c=raw[i];if(c<48 || c>57){return .err(8);}let usize digit=(c-48) as usize;if(value>429496729 || (value==429496729 && digit>5)){return .err(8);}value=value*10+digit;i+=1;}return .ok(value);}@effects(filesystem,console) fn i32 main(){if(p::arg_count()!=7){return 8;}let previous=switch(number(2)){case .ok(n):{n}case .err(code):{return code;}};if(previous>1){return 8;}let max_bytes=switch(number(3)){case .ok(n):{n}case .err(code):{return code;}};let max_tokens=switch(number(4)){case .ok(n):{n}case .err(code):{return code;}};let max_diagnostics=switch(number(5)){case .ok(n):{n}case .err(code):{return code;}};let max_work=switch(number(6)){case .ok(n):{n}case .err(code):{return code;}};var u8[4096] path=[0;4096];let n=switch(p::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{return 8;}};let input=switch(f::load(ref path[..n])){case .ok(b):{b}case .err(code):{return if(code==4){4}else if(code==3){3}else{1};}};return s::report_work(b::view(ref input),previous==1,max_bytes,max_tokens,max_diagnostics,max_work);}");output(ref p);}
