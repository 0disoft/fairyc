module tests::native_packet;
import native_packet as packet;
import module_project_cli as cli;
import module_unit_c_emit as emit;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::fs as fs;
import std::text as text;

fn syntax::Parsed parsed(ref u8[] text){let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let result=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref result)==0);return result;}
fn emit::Bundle bundle(){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import math as math;fn i32 main(){return math::add(40,2);}fn i32 unused(){return 99;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module math;pub fn i32 add(i32 a,i32 b){return a+b;}")).is_ok);return switch(cli::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return bundle();}};}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let count=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..count])){case .ok(p):{p}case .err(_):{assert(false);return root();}};}
fn bytes::Buffer joined(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,leaf).is_ok);return out;}
@test @effects(filesystem,console) fn void streams_the_same_complete_packet_as_stdout_transport(){
    let result=bundle();let directory=root();let path=joined(bytes::view(ref directory),b"fresh.bin");let name=switch(text::validate_utf8(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};var fs::File file=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let expected=switch(packet::size(ref result)){case .ok(n):{n}case .err(_):{assert(false);return;}};let written=switch(packet::write(ref result,ref mut file)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(written==expected);assert(fs::sync(ref mut file).is_ok);assert(fs::close(file).is_ok);
    let data=switch(files::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::len(ref data)==expected && bytes::equal(bytes::view(ref data)[..10],b"FYCPARTS1\n"));assert(cli::packet(ref result)==0);
}
@test @effects(filesystem) fn void reports_write_failure_without_changing_the_prior_packet(){
    let result=bundle();let directory=root();let path=joined(bytes::view(ref directory),b"prior.bin");let name=switch(text::validate_utf8(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};var fs::File previous=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::write_all(ref mut previous,b"previous packet").is_ok);assert(fs::close(previous).is_ok);let before=switch(files::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};var fs::File file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(packet::write(ref result,ref mut file)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==3 && e.status!=0 && e.completed==0);}};assert(fs::close(file).is_ok);let after=switch(files::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));
}
