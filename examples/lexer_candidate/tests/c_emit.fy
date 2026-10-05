module tests::c_emit;
import c_emit as emit;
import c_text as writer;
import mir_lower as mir;
import mir_model as m;
import hir_lower as hir;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn m::Program program(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(s): { s } case .err(_): { assert(false); return program(b"module main;"); } };
    let syntax::Parsed parsed = switch(syntax::parse(stream,512)) { case .ok(p): { p } case .err(_): { assert(false); return program(b"module main;"); } };
    let typing::Typed typed = typing::check(names::resolve(parsed)); assert(typing::problem(ref typed).code == 0);
    let h = switch(hir::lower(typed,0)) { case .ok(p): { p } case .err(_): { assert(false); return program(b"module main;"); } };
    return switch(mir::lower(h)) { case .ok(p): { p } case .err(_): { assert(false); return program(b"module main;"); } };
}
@test
fn void entry_rejections_buffer_preservation_determinism_and_writer_limit() {
    let m::Program empty = program(b"module main;"); let m::Program void_main = program(b"module main;fn void main(){}");
    let m::Program parameters = program(b"module main;fn i32 main(i32 x){return x;}");
    var bytes::Buffer out = bytes::empty(); assert(bytes::append(ref mut out,b"prefix").is_ok);
    assert(emit::render(ref empty,ref mut out) == 1 && bytes::equal(bytes::view(ref out),b"prefix"));
    assert(emit::render(ref void_main,ref mut out) == 1 && bytes::equal(bytes::view(ref out),b"prefix"));
    assert(emit::render(ref parameters,ref mut out) == 1 && bytes::equal(bytes::view(ref out),b"prefix"));
    let m::Program good = program(b"module main;fn i32 main(){return 7;}");
    var bytes::Buffer first = bytes::empty(); var bytes::Buffer second = bytes::empty();
    assert(emit::render(ref good,ref mut first) == 0 && emit::render(ref good,ref mut second) == 0 && bytes::equal(bytes::view(ref first),bytes::view(ref second)));
    var bytes::Buffer full = bytes::empty(); assert(bytes::resize_zeroed(ref mut full,2097152).is_ok);
    var writer::State s = writer::State { .code = 0 }; assert(!writer::text(ref mut full,ref mut s,b"x") && s.code == 4 && bytes::len(ref full) == 2097152);
    assert(bytes::resize_zeroed(ref mut full,2097153).is_ok);
    var writer::State oversized = writer::State { .code = 0 };
    assert(!writer::text(ref mut full,ref mut oversized,b"") && oversized.code == 4 && bytes::len(ref full) == 2097153);
}
