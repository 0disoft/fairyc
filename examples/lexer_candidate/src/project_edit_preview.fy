module project_edit_preview;
import project_edit_body as edit;
import project_edit_snapshot as snapshots;
import windows_rooted_file as host;
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
import module_public_type_text as public_type;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_compiler_image_hash as image;
import project_cache_environment as environment;
import module_project_identified_ir as identified;
import project_atlas_facts as facts;
import project_atlas_declarations as declarations;
import project_atlas_schema as schema;
import project_atlas_source_mode as mode;
import project_atlas_wire as wire;
import project_atlas_write as w;
import std::bytes as bytes;
import std::vec as v;
import syntax as syntax;

pub resource Selection{input::Input original;public_type::Coordinates table;declarations::Row selected;bytes::Buffer path;bytes::Buffer module_name;bytes::Buffer function;v::Vec<bytes::Buffer> production_roots;bytes::Buffer fingerprint;cleanup{}}
pub resource Preview{Selection selection;edit::Change change;bytes::Buffer report;cleanup{}}
pub struct Problem{u32 stage;u32 code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code};}
pub fn ref u8[] report(ref Preview p){return bytes::view(ref p.report);}
pub fn ref u8[] path(ref Preview p){return bytes::view(ref p.selection.path);}
pub fn ref u8[] before(ref Preview p){return edit::before(ref p.change);}
pub fn ref u8[] after(ref Preview p){return edit::after(ref p.change);}
pub fn ref u8[] fingerprint(ref Preview p){return bytes::view(ref p.selection.fingerprint);}
fn ref u8[] replacement(ref Preview p){let raw=after(p);let end=raw.len-(before(p).len-edit::end(ref p.change));return raw[edit::start(ref p.change)..end];}
fn bool vendor(ref u8[] path){let name=b"vendor/";if(path.len<name.len){return false;}var usize i=0;while(i<name.len){let c=path[i];if(c!=name[i] && !(c>=65 && c<=90 && c+32==name[i])){return false;}i+=1;}return true;}
fn bool prefix(ref u8[] root,ref u8[] path){if(path.len<root.len || path.len>root.len && path[root.len]!=47){return false;}var usize i=0;while(i<root.len){let a=root[i];let b=path[i];if((if(a>=65 && a<=90){a+32}else{a})!=(if(b>=65 && b<=90){b+32}else{b})){return false;}i+=1;}return true;}
pub fn bool body_allowed(ref Selection p,ref u8[] path){if(!manifest::path(path)){return false;}var usize i=0;while(i<v::len<bytes::Buffer>(ref p.production_roots)){if(prefix(bytes::view(v::at<bytes::Buffer>(ref p.production_roots,i)),path)){return false;}i+=1;}return true;}
fn Result<v::Vec<bytes::Buffer>,u32> roots(ref catalog::Catalog packages){var v::Vec<bytes::Buffer> out=v::empty<bytes::Buffer>();var usize i=0;while(i<catalog::count(packages)){var path=bytes::empty();let base=catalog::path(packages,i);if(!bytes::equal(base,b".") && (!bytes::append(ref mut path,base).is_ok || !bytes::append_byte(ref mut path,47).is_ok)){return .err(2);}if(!bytes::append(ref mut path,sources::package_value(modules::source_set(catalog::input(packages,i)),b"source")).is_ok){return .err(2);}if(!v::push<bytes::Buffer>(ref mut out,path).is_ok){return .err(2);}i+=1;}return .ok(out);}
fn Result<bytes::Buffer,u32> render(ref u8[] path,ref u8[] module_name,ref u8[] function,ref edit::Change change,bool applied,ref mut usize meter){
    var out=bytes::empty();discard w::raw(ref mut out,b"{\"schema\":\"fairyc.structural-body-edit/v1\",",meter)?;
    discard w::pair(ref mut out,b"path",path,meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"module",module_name,meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"function",function,meter)?;discard w::raw(ref mut out,b",\"bodyStartByte\":",meter)?;
    discard w::number(ref mut out,edit::start(change),meter)?;discard w::raw(ref mut out,b",\"bodyEndByte\":",meter)?;
    discard w::number(ref mut out,edit::end(change),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"beforeSha256",edit::before_hash(change),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"afterSha256",edit::after_hash(change),meter)?;
    discard w::raw(ref mut out,if(applied){b",\"verified\":true,\"applied\":true}"}else{b",\"verified\":true,\"applied\":false}"},meter)?;
    if(bytes::len(ref out)>8192){return .err(3);}return .ok(out);
}
// Selection precedes body-file access and owns all data used by the overlay.
@effects(filesystem,environment) pub fn Result<Selection,Problem> select(ref u8[] root,ref u8[] uid,ref u8[] symbol){
    if(!manifest::uid(uid) || (!wire::hex(symbol,b"fysym1-sha256-128-",32,false) && !wire::hex(symbol,b"fysym1-sha256-256-",64,false))){return .err(fail(1,1));}
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(fail(2,1));}};
    let resolved=loader::workspace(ref loaded);let g=visibility::workspace_graph(resolved);
    let root_sources=modules::source_set(catalog::input(workspace::local_catalog(local::input(graph::local_graph(g))),0));
    if(!bytes::equal(sources::package_value(root_sources,b"uid"),uid)){return .err(fail(1,2));}
    let lock=switch(locks::project(resolved,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    let table=switch(coordinates::collect(g,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    let original=switch(input::project(resolved,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    var usize source_work=0;var usize link_work=0;
    let aggregates=switch(mode::aggregates(ref original,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    let complete=switch(if(aggregates){compiler::build_complete_aggregate_sources(input::sources(ref original),input::owners(ref original),input::bindings(ref original),ref table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref original,ref table,ref mut source_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{return .err(fail(3,e.code));}};
    var usize image_work=0;let build=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return .err(fail(2,3));}};
    let f=switch(facts::collect(g,ref lock,ref table,compiler::public_interfaces(ref complete),identified::ir_program(compiler::checked_program(ref complete)),image::value(ref build),0,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    let metadata=switch(if(aggregates){schema::prepare_aggregates(ref original,ref table,ref complete,ref mut meter)}else{schema::prepare(ref original,ref table,ref complete,ref mut meter)}){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    var type_rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    let rows=switch(if(aggregates){declarations::collect_aggregates(ref original,0,ref mut meter)}else{declarations::collect_generics(ref original,0,ref mut meter)}){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    var declarations::Row selected=declarations::Row{.module_index=0,.declaration=0};var usize matches=0;var usize i=0;
    while(i<v::len<declarations::Row>(ref rows)){
        let row=*v::at<declarations::Row>(ref rows,i);
        let d=switch(schema::declaration(ref original,ref table,ref metadata,ref mut type_rows,row,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
        let digest=switch(declarations::digest(ref f,bytes::view(ref d),ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
        let full=bytes::view(ref digest);
        if(bytes::equal(full,symbol) || symbol.len==50 && bytes::equal(full[18..50],symbol[18..])){matches+=1;selected=row;}
        i+=1;
    }
    if(matches!=1){return .err(fail(1,3));}
    let parsed=input::source(ref original,selected.module_index);
    if(syntax::declaration_at(parsed,selected.declaration).kind!=3){return .err(fail(1,3));}
    var path=bytes::empty();
    if(!bytes::append(ref mut path,sources::package_value(root_sources,b"source")).is_ok || !bytes::append_byte(ref mut path,47).is_ok || !bytes::append(ref mut path,input::file(ref original,selected.module_index)).is_ok){return .err(fail(6,2));}
    if(vendor(bytes::view(ref path))){return .err(fail(1,4));}
    let module_name=switch(bytes::copy(graph::module_name(g,0,input::origin(ref original,selected.module_index).module_index))){case .ok(p):{p}case .err(_):{return .err(fail(6,2));}};
    let function=switch(bytes::copy(syntax::declaration_name(parsed,selected.declaration+1))){case .ok(p):{p}case .err(_):{return .err(fail(6,2));}};
    let production_roots=switch(roots(workspace::local_catalog(local::input(graph::local_graph(g))))){case .ok(p):{p}case .err(code):{return .err(fail(6,code));}};
    let opened=switch(host::open(root)){case .ok(p):{p}case .err(_):{return .err(fail(2,1));}};
    let observed=switch(snapshots::from_resolved(ref opened,resolved)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
    let fingerprint=switch(bytes::copy(snapshots::digest(ref observed))){case .ok(p):{p}case .err(_):{return .err(fail(6,2));}};
    return .ok(Selection{.original,.table,.selected,.path,.module_name,.function,.production_roots,.fingerprint});
}
pub fn Result<Preview,Problem> finish(Selection selected,ref u8[] expected,ref u8[] body){
    if(expected.len!=64 || body.len>edit::maximum_body()){return .err(fail(1,1));}
    var usize meter=0;
    let change=switch(edit::replace(input::source(ref selected.original,selected.selected.module_index),selected.selected.declaration,expected,body,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};
    let edited=switch(input::with_source(ref selected.original,selected.selected.module_index,edit::parsed(ref change),ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(5,e.code));}};
    var usize source_work=0;var usize link_work=0;
    let aggregates=switch(mode::aggregates(ref edited,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(5,code));}};
    let verified=switch(if(aggregates){compiler::build_complete_aggregate_sources(input::sources(ref edited),input::owners(ref edited),input::bindings(ref edited),ref selected.table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref edited,ref selected.table,ref mut source_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{return .err(fail(5,e.code));}};
    let report=switch(render(bytes::view(ref selected.path),bytes::view(ref selected.module_name),bytes::view(ref selected.function),ref change,false,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(6,code));}};
    return .ok(Preview{.selection=selected,.change,.report});
}
// Compatibility entry for an already owned body. No workspace write or lock refresh.
@effects(filesystem,environment) pub fn Result<Preview,Problem> prepare(ref u8[] root,ref u8[] uid,ref u8[] symbol,ref u8[] expected,ref u8[] body){
    if(expected.len!=64 || body.len>edit::maximum_body()){return .err(fail(1,1));}
    let selected=select(root,uid,symbol)?;return finish(selected,expected,body);
}
// Rebuild the complete overlay from fresh locked inputs and the already owned
// replacement. Raw manifest/source/lock drift rejects even when semantics agree.
@effects(filesystem,environment) pub fn Result<bool,Problem> recheck(ref u8[] root,ref u8[] uid,ref u8[] symbol,ref u8[] expected,ref Preview proposal){
    let checked=prepare(root,uid,symbol,expected,replacement(proposal))?;
    if(!bytes::equal(fingerprint(ref checked),fingerprint(proposal)) || !bytes::equal(path(ref checked),path(proposal)) || !bytes::equal(before(ref checked),before(proposal)) || !bytes::equal(after(ref checked),after(proposal))){return .err(fail(7,5));}
    return .ok(true);
}
pub fn Result<bytes::Buffer,u32> applied_report(ref Preview p){var usize meter=0;return render(bytes::view(ref p.selection.path),bytes::view(ref p.selection.module_name),bytes::view(ref p.selection.function),ref p.change,true,ref mut meter);}
