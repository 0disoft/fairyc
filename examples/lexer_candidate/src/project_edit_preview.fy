module project_edit_preview;
import project_edit_body as edit;
import project_workspace_lock as loader;
import project_source_set as sources;
import project_workspace_graph as graph;
import project_instance_graph as local;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_modules as modules;
import project_manifest_values as manifest;
import project_module_visibility as visibility;
import project_lock_document as locks;
import project_coordinate_graph as coordinates;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_compiler_image_hash as image;
import project_cache_environment as environment;
import module_project_identified_ir as identified;
import project_atlas_facts as facts;
import project_atlas_declarations as declarations;
import project_atlas_wire as wire;
import project_atlas_write as w;
import std::bytes as bytes;
import std::vec as v;
import syntax as syntax;

pub resource Preview{edit::Change change;bytes::Buffer path;bytes::Buffer report;cleanup{}}
pub struct Problem{u32 stage;u32 code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code};}
pub fn ref u8[] report(ref Preview p){return bytes::view(ref p.report);}
pub fn ref u8[] path(ref Preview p){return bytes::view(ref p.path);}
pub fn ref u8[] before(ref Preview p){return edit::before(ref p.change);}
pub fn ref u8[] after(ref Preview p){return edit::after(ref p.change);}
fn bool vendor(ref u8[] path){let name=b"vendor/";if(path.len<name.len){return false;}var usize i=0;while(i<name.len){let c=path[i];if(c!=name[i] && !(c>=65 && c<=90 && c+32==name[i])){return false;}i+=1;}return true;}
fn Result<bytes::Buffer,u32> render(ref u8[] path,ref u8[] module_name,ref u8[] function,ref edit::Change change,ref mut usize meter){
    var out=bytes::empty();discard w::raw(ref mut out,b"{\"schema\":\"fairyc.structural-body-edit/v1\",",meter)?;
    discard w::pair(ref mut out,b"path",path,meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"module",module_name,meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"function",function,meter)?;discard w::raw(ref mut out,b",\"bodyStartByte\":",meter)?;
    discard w::number(ref mut out,edit::start(change),meter)?;discard w::raw(ref mut out,b",\"bodyEndByte\":",meter)?;
    discard w::number(ref mut out,edit::end(change),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"beforeSha256",edit::before_hash(change),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"afterSha256",edit::after_hash(change),meter)?;
    discard w::raw(ref mut out,b",\"verified\":true,\"applied\":false}",meter)?;
    if(bytes::len(ref out)>8192){return .err(3);}return .ok(out);
}
// This host reads current source but never creates or refreshes the lock. It
// validates the complete immutable edited input before constructing a report.
@effects(filesystem,environment) pub fn Result<Preview,Problem> prepare(ref u8[] root,ref u8[] uid,ref u8[] symbol,ref u8[] expected,ref u8[] body){
    if(!manifest::uid(uid) || (!wire::hex(symbol,b"fysym1-sha256-128-",32,false) && !wire::hex(symbol,b"fysym1-sha256-256-",64,false)) || expected.len!=64 || body.len>edit::maximum_body()){return .err(fail(1,1));}
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(fail(2,1));}};
    let resolved=loader::workspace(ref loaded);let g=visibility::workspace_graph(resolved);
    let root_sources=modules::source_set(catalog::input(workspace::local_catalog(local::input(graph::local_graph(g))),0));
    if(!bytes::equal(sources::package_value(root_sources,b"uid"),uid)){return .err(fail(1,2));}
    let lock=switch(locks::project(resolved,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    let table=switch(coordinates::collect(g,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    let original=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    var usize source_work=0;var usize link_work=0;
    let complete=switch(compiler::build_complete(ref original,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return .err(fail(3,e.code));}};
    var usize image_work=0;let build=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return .err(fail(2,3));}};
    let f=switch(facts::collect(g,ref lock,ref table,compiler::public_interfaces(ref complete),identified::ir_program(compiler::checked_program(ref complete)),image::value(ref build),0,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    let rows=switch(declarations::collect(ref original,0,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    var declarations::Row selected=declarations::Row{.module_index=0,.declaration=0};var usize matches=0;var usize i=0;
    while(i<v::len<declarations::Row>(ref rows)){
        let row=*v::at<declarations::Row>(ref rows,i);
        let d=switch(declarations::declaration(ref original,row,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
        let digest=switch(declarations::digest(ref f,bytes::view(ref d),ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
        let full=bytes::view(ref digest);
        if(bytes::equal(full,symbol) || symbol.len==50 && bytes::equal(full[18..50],symbol[18..])){matches+=1;selected=row;}
        i+=1;
    }
    if(matches!=1){return .err(fail(1,3));}
    let parsed=input::source(ref original,selected.module_index);
    let change=switch(edit::replace(parsed,selected.declaration,expected,body,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};
    let edited=switch(input::with_source(ref original,selected.module_index,edit::parsed(ref change),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.code));}};
    source_work=0;link_work=0;
    let verified=switch(compiler::build_complete(ref edited,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.code));}};
    var path=bytes::empty();
    if(!bytes::append(ref mut path,sources::package_value(root_sources,b"source")).is_ok || !bytes::append_byte(ref mut path,47).is_ok || !bytes::append(ref mut path,input::file(ref original,selected.module_index)).is_ok){return .err(fail(6,2));}
    if(vendor(bytes::view(ref path))){return .err(fail(1,4));}
    let report=switch(render(bytes::view(ref path),graph::module_name(g,0,input::origin(ref original,selected.module_index).module_index),syntax::declaration_name(parsed,selected.declaration+1),ref change,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(6,code));}};
    return .ok(Preview{.change,.path,.report});
}
