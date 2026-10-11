module module_project_cli;
import candidate_limits as limits;
import module_cli as manifests;
import module_project_compile as compile;
import module_project_link as link;
import module_signatures as signatures;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as emit;
import file_source as files;
import scanner as scanner;
import syntax as syntax;
import token_output as output;
import declaration_output as declarations;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

pub struct Problem {u32 phase;u32 stage;u32 code;usize module_index;usize start;usize end;}
fn Problem fail(u32 phase,u32 stage,u32 code,usize module_index,usize start,usize end){return Problem{.phase,.stage,.code,.module_index,.start,.end};}

// The first manifest source owns the entry. Names in dependencies do not
// select an entry, and the C emitter still checks the exact result/parameters.
pub fn Result<emit::Bundle,Problem> build(ref v::Vec<syntax::Parsed> sources){
    if(v::len<syntax::Parsed>(sources)==0){return .err(fail(1,0,1001,0,0,0));}
    let metadata=switch(signatures::build(sources)){case .ok(p):{p}case .err(e):{return .err(fail(1,0,e.code,e.module_index,e.start,e.end));}};
    let units=switch(compile::build(sources,ref metadata)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.stage,e.code,e.module_index,e.start,e.end));}};
    let linked=switch(link::build(ref metadata,compile::programs(ref units))){case .ok(p):{p}case .err(e):{return .err(fail(3,0,e.code,e.module_index,e.start,e.end));}};
    var usize entry=0;var usize i=1;let root=v::at<syntax::Parsed>(sources,0);
    while(i<=ir::count(ref linked)){
        let frame=clean::frame_at(ir::cleanup_ir(ref linked),i);
        if(frame.module_index==0 && frame.kind==3){
            let declaration=syntax::declaration_at(root,frame.declaration-1);
            if(bytes::equal(syntax::source_view(root)[declaration.name_start..declaration.name_end],b"main")){
                if(entry!=0){return .err(fail(4,0,1001,0,declaration.start,declaration.end));}entry=i;
            }
        }i+=1;
    }
    if(entry==0){return .err(fail(4,0,1001,0,0,0));}
    return switch(emit::partition_entry(linked,entry,128)){case .ok(p):{.ok(p)}case .err(e):{.err(fail(5,0,e.code,e.module_index,e.start,e.end))}};
}

@effects(console)
fn i32 error(ref str message){discard io::write_stderr(message);return 2;}
@effects(console)
fn bool number(usize value){var bytes::Buffer text=bytes::empty();if(!bytes::append_usize(ref mut text,value).is_ok || !bytes::append_byte(ref mut text,10).is_ok){return false;}return output::write(bytes::view(ref text))==0;}
@effects(console)
fn bool part(ref u8[] data){return number(data.len) && output::write(data)==0;}
// No bytes are written until the complete checked bundle exists. Output I/O
// failure may interrupt transport, so consumers must check status and framing.
@effects(console)
pub fn i32 packet(ref emit::Bundle bundle){
    if(output::write(b"FYCPARTS1\n")!=0 || !part(emit::project_header(bundle)) || !part(emit::project_literals(bundle)) || !number(emit::project_unit_count(bundle))){return 2;}
    var usize i=0;while(i<emit::project_unit_count(bundle)){if(!part(emit::project_source(bundle,i))){return 2;}i+=1;}return 0;
}

@effects(filesystem,console)
pub fn i32 run(ref u8[] path){
    let manifest=switch(files::manifest(path)){case .ok(p):{p}case .err(code):{return error(if(code==4){"error: manifest limit (65536 bytes)\n"}else if(code==3){"error: buffer allocation failed\n"}else{"error: manifest file read failed\n"});}};
    let text=bytes::view(ref manifest);var v::Vec<manifests::Path> paths=v::empty<manifests::Path>();
    let u32 status=manifests::paths(text,ref mut paths);
    if(status!=0){return error(if(status==3){"error: buffer allocation failed\n"}else if(status==4){"error: file path limit (4095 UTF-8 bytes)\n"}else if(status==5){"error: module limit (256 files)\n"}else{"error: invalid module manifest\n"});}
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<v::len<manifests::Path>(ref paths)){
        let entry=*v::at<manifests::Path>(ref paths,i);let filename=text[entry.start..entry.end];
        let data=switch(files::load(filename)){case .ok(p):{p}case .err(code):{return error(if(code==4){"error: source limit (65536 bytes)\n"}else if(code==3){"error: buffer allocation failed\n"}else{"error: input file read failed\n"});}};
        let stream=switch(scanner::collect(data,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(code):{return error(if(code==3){"error: buffer allocation failed\n"}else{"error: lexer storage or input limit\n"});}};
        if(scanner::diagnostic_count(ref stream)!=0){return output::errors(ref stream,filename);}
        let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(code):{return error(if(code==3){"error: buffer allocation failed\n"}else{"error: parser failed or limit\n"});}};
        if(syntax::problem_count(ref parsed)!=0){return declarations::errors(ref parsed,filename);}
        if(!v::push<syntax::Parsed>(ref mut sources,parsed).is_ok){return error("error: buffer allocation failed\n");}i+=1;
    }
    let bundle=switch(build(ref sources)){
        case .ok(p):{p}
        case .err(e):{
            if(e.module_index<v::len<manifests::Path>(ref paths)){let entry=*v::at<manifests::Path>(ref paths,e.module_index);output::prefix(syntax::source_view(v::at<syntax::Parsed>(ref sources,e.module_index)),text[entry.start..entry.end],e.start);}
            discard io::write_stderr("project phase ");discard io::write_stderr_u32(e.phase);discard io::write_stderr(" stage ");discard io::write_stderr_u32(e.stage);discard io::write_stderr(" code ");discard io::write_stderr_u32(e.code);output::range(e.start,e.end);return 2;
        }
    };
    return packet(ref bundle);
}
