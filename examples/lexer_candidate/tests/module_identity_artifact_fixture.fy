module tests::module_identity_artifact_fixture;
import module_identity_artifact_model as model;
import module_source_specializations as selection;
import module_public_interface_model as records;
import module_unit_cleanup_mir_model as clean;
import module_types as types;
import candidate_artifact_wire as wire;
import project_workspace_lock as loader;
import project_module_visibility as visibility;
import project_coordinate_graph as coordinate_graph;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import project_source_set as source_set;
import project_compiler_core as core;
import module_project_identified_ir as project;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;
pub resource Data{bytes::Buffer golden;bytes::Buffer workspace;cleanup{}}
pub fn ref u8[] golden(ref Data p){return bytes::view(ref p.golden);}
pub fn ref u8[] workspace(ref Data p){return bytes::view(ref p.workspace);}
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
fn bytes::Buffer field(ref u8[] raw,ref mut wire::Cursor c,ref mut usize work){let n=wire::count_from(raw,c,65536,1,work);assert(wire::status(c)==0);let value=copied(raw[c.offset..c.offset+n]);c.offset+=n;return value;}
@effects(filesystem) pub fn Data data(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(p):{p}case .err(_):{assert(false);0}};let ref u8[] view=ref path;let input=switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);return data();}};let raw=bytes::view(ref input);var usize work=0;var c=switch(wire::cursor(raw,b"FYIDFIX0",ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return data();}};let golden=field(raw,ref mut c,ref mut work);let workspace=field(raw,ref mut c,ref mut work);assert(wire::end(raw,ref mut c,ref mut work)==0);return Data{.golden,.workspace};}
pub fn clean::Program context(usize mode){var v::Vec<types::Type> graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok && v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok && v::push<types::Type>(ref mut graph,types::Type{.kind=if(mode==1){3}else{13},.scalar=0,.module_index=0,.declaration=3,.first=if(mode==1){0}else{1},.second=0}).is_ok);
    var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();var usize i=0;while(i<3){assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=if(i==2){1}else{0},.frame=if(i==2){2}else{0},.nonempty=i==2,.effects=0,.abilities=if(i==2){1}else{0},.field_start=0,.field_count=0}).is_ok);i+=1;}
    var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();assert(v::push<clean::Frame>(ref mut frames,clean::Frame{.kind=if(mode==2){3}else{1},.context=1,.module_index=0,.declaration=4,.result=1,.effects=0,.receiver=0,.receiver_type=0}).is_ok && v::push<clean::Frame>(ref mut frames,clean::Frame{.kind=2,.context=1,.module_index=0,.declaration=3,.result=2,.effects=0,.receiver=1,.receiver_type=3}).is_ok && v::push<clean::Frame>(ref mut frames,clean::Frame{.kind=1,.context=2,.module_index=0,.declaration=5,.result=1,.effects=0,.receiver=0,.receiver_type=0}).is_ok);
    return switch(clean::assemble(graph,frames,glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(_):{assert(false);return context(mode);}};
}
fn selection::Row row(usize node,usize start,usize end){return selection::Row{.node,.id=records::Text{.start,.end}};}
pub fn model::Snapshot sample(ref clean::Program context){var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();assert(v::push<selection::Row>(ref mut types,row(0,0,0)).is_ok && v::push<selection::Row>(ref mut types,row(0,0,0)).is_ok && v::push<selection::Row>(ref mut types,row(17,0,67)).is_ok && v::push<selection::Row>(ref mut frames,row(23,67,134)).is_ok && v::push<selection::Row>(ref mut frames,row(17,134,201)).is_ok && v::push<selection::Row>(ref mut frames,row(0,0,0)).is_ok);
    let names=copied(b"g0_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaag0_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbg0_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");var usize work=0;return switch(model::assemble(names,types,frames,context,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return sample(context);}};
}
pub resource Loaded{input::Input input;coordinates::Coordinates coordinates;cleanup{}}
pub fn ref input::Input inputs(ref Loaded p){return ref p.input;}
pub fn ref coordinates::Coordinates coordinate_table(ref Loaded p){return ref p.coordinates;}
@effects(filesystem) pub fn Loaded loaded(ref u8[] path){var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize work=0;let workspace_loaded=switch(loader::load(path,true,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return loaded(path);}};let coords=switch(coordinate_graph::collect(visibility::workspace_graph(loader::workspace(ref workspace_loaded)),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return loaded(path);}};let input=switch(input::project(loader::workspace(ref workspace_loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return loaded(path);}};return Loaded{.input,.coordinates=coords};}
@effects(console) pub fn project::Program compiled(ref Loaded p){var usize source_work=0;var usize link_work=0;return switch(core::build_identified(inputs(p),coordinate_table(p),ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return compiled(p);}};}
