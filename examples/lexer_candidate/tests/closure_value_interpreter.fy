module tests::closure_value_interpreter;
import project_compiler_core as core;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interpreter as interpreter;
import module_unit_interpreter_values as runtime;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import sha256_stream as sha;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

// Original source -> production Complete -> checked CFG -> Fairy interpreter.
// No generated C consumer, manufactured helper coordinates or native pointers.
pub struct Failure {u32 phase;u32 stage;u32 code;usize module_index;usize start;usize end;}
resource Input {v::Vec<syntax::Parsed> sources;v::Vec<usize> owners;v::Vec<catalog::Binding> imports;coordinates::Coordinates table;cleanup{}}
fn Failure failed(u32 phase,u32 code,usize module_index,usize start,usize end){return Failure{.phase,.stage=0,.code,.module_index,.start,.end};}
fn Result<syntax::Parsed,Failure> parsed(ref u8[] text,usize module_index){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{return .err(failed(1,11,module_index,0,0));}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(failed(1,9,module_index,0,text.len));}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return .err(failed(1,9,module_index,0,text.len));}};
    if(syntax::problem_count(ref p)!=0){let e=syntax::problem_at(ref p,0);return .err(failed(1,e.code,module_index,e.start,e.end));}return .ok(p);
}
fn Result<Input,Failure> original(ref u8[] root,ref u8[] first,ref u8[] second){
    var sources=v::empty<syntax::Parsed>();var owners=v::empty<usize>();var imports=v::empty<catalog::Binding>();var usize m=0;
    while(m<3){let text=if(m==0){root}else if(m==1){first}else{second};if(text.len!=0){let p=switch(parsed(text,m)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!v::push<syntax::Parsed>(ref mut sources,p).is_ok || !v::push<usize>(ref mut owners,0).is_ok){return .err(failed(1,11,m,0,0));}}m+=1;}
    let declarations=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(failed(1,e.code,e.module_index,e.start,e.end));}};
    m=0;while(m<v::len<syntax::Parsed>(ref sources)){let p=v::at<syntax::Parsed>(ref sources,m);var usize d=1;while(d<syntax::declaration_count(p)){let row=syntax::declaration_at(p,d);if(row.kind==2){var usize probes=0;let alias=syntax::source_view(p)[row.name_start..row.name_end];let resolved=catalog::import_target_measured(ref declarations,m,alias,ref mut probes);if(resolved.code!=0 || resolved.item==0){return .err(failed(1,9,m,row.start,row.end));}if(!v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=m,.declaration=d+1,.target=resolved.item-1}).is_ok){return .err(failed(1,11,m,row.start,row.end));}}d+=1;}m+=1;}
    // A fixed private package identity, independent of source content. Source
    // stamps and interface hashes are derived by production owners separately.
    var digest=sha::empty();var text=bytes::empty();if(!sha::update(ref mut digest,b"closure-value-interpreter-original-source-fixture/v1") || !bytes::append(ref mut text,b"fypc1-sha256-").is_ok || !sha::append_hex(ref digest,ref mut text)){return .err(failed(1,11,0,0,0));}
    var rows=v::empty<coordinates::Coordinate>();if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref text)}).is_ok){return .err(failed(1,11,0,0,0));}return .ok(Input{.sources,.owners,.imports,.table=coordinates::assemble(text,rows)});
}
@effects(console) fn void failure(ref u8[] name,Failure e){
    discard io::write_stderr("closure-interpreter case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" phase=");discard io::write_stderr_u32(e.phase);discard io::write_stderr(" stage=");discard io::write_stderr_u32(e.stage);discard io::write_stderr(" code=");discard io::write_stderr_u32(e.code);discard io::write_stderr(" module=");discard io::write_stderr_usize(e.module_index);discard io::write_stderr(" span=");discard io::write_stderr_usize(e.start);discard io::write_stderr("..");discard io::write_stderr_usize(e.end);discard io::write_stderr("\n");
}
@effects(console) fn void outcome(ref u8[] name,u32 phase,runtime::Outcome done,u32 error,u32 trap,usize bits){
    if(done.error!=error || done.trap!=trap || done.bits!=bits){
        failure(name,failed(phase,done.error,0,0,0));discard io::write_stderr("actual trap/bits=");discard io::write_stderr_u32(done.trap);discard io::write_stderr("/");discard io::write_stderr_usize(done.bits);discard io::write_stderr(" expected error/trap/bits=");discard io::write_stderr_u32(error);discard io::write_stderr("/");discard io::write_stderr_u32(trap);discard io::write_stderr("/");discard io::write_stderr_usize(bits);discard io::write_stderr("\n");assert(false);
    }
}
@effects(console) pub fn void exercise(ref u8[] name,ref u8[] root,ref u8[] first,ref u8[] second,usize bits,u32 trap,bool bounds){
    let source=switch(original(root,first,second)){case .ok(p):{p}case .err(e):{failure(name,e);assert(false);return;}};
    var usize source_work=0;var usize link_work=0;
    let complete=switch(core::build_complete_sources(ref source.sources,ref source.owners,ref source.imports,ref source.table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{var error=failed(10+e.phase,e.code,e.module_index,e.start,e.end);error.stage=e.stage;failure(name,error);assert(false);return;}};
    let owned=core::checked_program(ref complete);let p=identified::ir_program(owned);let cleaned=ir::cleanup_ir(p);var usize entry=0;var usize frame=1;
    while(frame<=ir::count(p)){let f=clean::frame_at(cleaned,frame);if(f.module_index==0 && f.kind==3 && bytes::equal(syntax::declaration_name(v::at<syntax::Parsed>(ref source.sources,0),f.declaration),b"main")){entry=frame;break;}frame+=1;}
    if(entry==0 || !clean::closures(cleaned) || source_work==0 || link_work==0){failure(name,failed(30,4,0,0,0));assert(false);return;}
    let plan=switch(interpreter::prepare(p)){case .ok(p):{p}case .err(e):{failure(name,failed(31,e,0,0,0));assert(false);return;}};
    let args=v::empty<usize>();var heap=v::empty<usize>();var usize steps=1000000;
    let done=interpreter::execute(owned,ref plan,entry,ref args,ref mut heap,ref mut steps,0);outcome(name,32,done,0,trap,bits);
    // Native replay returns the actual semantic result, including caught traps.
    assert(io::write_bytes(name).is_ok && io::write("\n").is_ok && io::write_u32(done.error).is_ok && io::write("/").is_ok && io::write_u32(done.trap).is_ok && io::write("/").is_ok && io::write_usize(done.bits).is_ok && io::write("\n").is_ok);
    if(bounds){
        var empty=v::empty<usize>();var usize zero=0;outcome(name,33,interpreter::execute(owned,ref plan,entry,ref args,ref mut empty,ref mut zero,0),2,0,0);
        var usize depth_steps=1000000;outcome(name,34,interpreter::execute(owned,ref plan,entry,ref args,ref mut empty,ref mut depth_steps,256),3,0,0);
        depth_steps=1000000;outcome(name,35,interpreter::execute(owned,ref plan,entry,ref args,ref mut empty,ref mut depth_steps,255),3,0,0);
        // Fill with a real immutable aggregate allocation. The next frame's
        // binding root must fail through the existing million-cell heap bound.
        var full=v::empty<usize>();var usize allocation_steps=2000000;let filled=runtime::allocate(ref mut full,999999,ref mut allocation_steps);if(filled.error!=0){failure(name,failed(36,filled.error,0,0,0));assert(false);return;}
        var usize heap_steps=1000000;outcome(name,37,interpreter::execute(owned,ref plan,entry,ref args,ref mut full,ref mut heap_steps,0),2,0,0);
    }
}
