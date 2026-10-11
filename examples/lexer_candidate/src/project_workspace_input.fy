module project_workspace_input;
import project_workspace_catalog as catalog;
import project_source_set as sources;
import project_source_modules as modules;
import project_manifest as manifest;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Edge{usize owner;usize target;usize declaration;}
pub resource Workspace{catalog::Catalog catalog;v::Vec<Edge> edges;cleanup{}}
// 1 nested catalog error; 2 reserved UID; 3 reserved module; 4 reserved binding;
// 5 unknown member path; 6 exact version mismatch; 7 allocation; 8 work bound.
// Scope 0 is an original manifest, 1 an original source file, 2 a catalog error.
pub struct Problem{u32 code;u32 scope;u32 field;usize package_index;usize dependency;usize module_index;usize start;usize end;}
pub resource Error{Problem problem;bytes::Buffer package_path;bytes::Buffer file_path;bytes::Buffer context;bytes::Buffer target;bytes::Buffer input;v::Vec<catalog::Error> catalog;cleanup{}}
fn Problem problem(u32 code,u32 scope,u32 field,usize owner,usize dependency,usize module_index,usize start,usize end){return Problem{.code,.scope,.field,.package_index=owner,.dependency,.module_index,.start,.end};}
fn Error allocation(){return Error{.problem=problem(7,2,0,0,0,0,0,0),.package_path=bytes::empty(),.file_path=bytes::empty(),.context=bytes::empty(),.target=bytes::empty(),.input=bytes::empty(),.catalog=v::empty<catalog::Error>()};}
fn Error capture(Problem p,ref u8[] package_path,ref u8[] file_path,ref u8[] context,ref u8[] target,ref u8[] input){
    let bytes::Buffer a=switch(bytes::copy(package_path)){case .ok(value):{value}case .err(_):{return allocation();}};
    let bytes::Buffer b=switch(bytes::copy(file_path)){case .ok(value):{value}case .err(_):{return allocation();}};
    let bytes::Buffer c=switch(bytes::copy(context)){case .ok(value):{value}case .err(_):{return allocation();}};
    let bytes::Buffer d=switch(bytes::copy(target)){case .ok(value):{value}case .err(_):{return allocation();}};
    let bytes::Buffer e=switch(bytes::copy(input)){case .ok(value):{value}case .err(_):{return allocation();}};
    return Error{.problem=p,.package_path=a,.file_path=b,.context=c,.target=d,.input=e,.catalog=v::empty<catalog::Error>()};
}
fn Error catalog_error(catalog::Error e){var v::Vec<catalog::Error> errors=v::empty<catalog::Error>();if(!v::push<catalog::Error>(ref mut errors,e).is_ok){return allocation();}return Error{.problem=problem(1,2,0,0,0,0,0,0),.package_path=bytes::empty(),.file_path=bytes::empty(),.context=bytes::empty(),.target=bytes::empty(),.input=bytes::empty(),.catalog=errors};}
pub fn Problem error(ref Error e){return e.problem;}
pub fn ref u8[] error_package(ref Error e){return bytes::view(ref e.package_path);}
pub fn ref u8[] error_file(ref Error e){return bytes::view(ref e.file_path);}
pub fn ref u8[] error_context(ref Error e){return bytes::view(ref e.context);}
pub fn ref u8[] error_target(ref Error e){return bytes::view(ref e.target);}
pub fn ref u8[] error_input(ref Error e){return bytes::view(ref e.input);}
pub fn bool has_catalog_error(ref Error e){return v::len<catalog::Error>(ref e.catalog)==1;}
pub fn ref catalog::Error nested_catalog_error(ref Error e){assert(has_catalog_error(e));return v::at<catalog::Error>(ref e.catalog,0);}
pub fn ref str reason(u32 code){if(code>=2 && code<=4){return "invalid-standard-library";}if(code==5){return "unknown-path-dependency";}if(code==6){return "dependency-version-mismatch";}if(code==8){return "source-limit-exceeded";}if(code==7){return "allocation-failed";}return "invalid-workspace-catalog";}
pub fn ref catalog::Catalog local_catalog(ref Workspace workspace){return ref workspace.catalog;}
pub fn usize edge_count(ref Workspace workspace){return v::len<Edge>(ref workspace.edges);}
pub fn Edge edge(ref Workspace workspace,usize at){assert(at<edge_count(workspace));return *v::at<Edge>(ref workspace.edges,at);}
pub fn ref u8[] binding(ref Workspace workspace,usize at){let Edge e=edge(workspace,at);return manifest::dependency_name(catalog::package(ref workspace.catalog,e.owner),e.declaration);}
pub fn usize feature_first(ref Workspace workspace,usize at){let Edge e=edge(workspace,at);return manifest::feature_first(catalog::package(ref workspace.catalog,e.owner),e.declaration);}
pub fn usize next_feature(ref Workspace workspace,usize at,usize id){let Edge e=edge(workspace,at);return manifest::next_value(catalog::package(ref workspace.catalog,e.owner),id);}
pub fn ref u8[] feature(ref Workspace workspace,usize at,usize id){let Edge e=edge(workspace,at);return manifest::value(catalog::package(ref workspace.catalog,e.owner),id);}
fn Error manifest_error(ref catalog::Catalog input,u32 code,u32 field,usize owner,usize dependency,manifest::Span span,ref u8[] context,ref u8[] target){return capture(problem(code,0,field,owner,dependency,0,span.start,span.end),catalog::path(input,owner),b"fairy.toml",context,target,sources::manifest_input(modules::source_set(catalog::input(input,owner))));}
fn Result<v::Vec<Edge>,Error> declarations(ref catalog::Catalog input,ref mut usize meter){
    var v::Vec<Edge> edges=v::empty<Edge>();var usize owner=0;
    while(owner<catalog::count(input)){let ref manifest::Manifest m=catalog::package(input,owner);let ref u8[] uid=manifest::package_value(m,b"uid");if(!work::charge(meter,uid.len+1)){return .err(manifest_error(input,8,3,owner,0,manifest::package_span(m,b"uid"),uid,b""));}
        if(bytes::equal(uid,b"fairyc.std")){return .err(manifest_error(input,2,3,owner,0,manifest::package_span(m,b"uid"),uid,b""));}
        let ref modules::Modules package=catalog::input(input,owner);var usize module_index=0;while(module_index<modules::count(package)){let ref u8[] name=modules::name(package,module_index);if(!work::charge(meter,name.len+1)){return .err(manifest_error(input,8,0,owner,0,manifest::Span{.start=0,.end=0},name,b""));}
            if(bytes::equal(name,b"std") || name.len>=5 && bytes::equal(name[..5],b"std::")){let d=syntax::declaration_at(modules::parsed(package,module_index),0);return .err(capture(problem(3,1,0,owner,0,module_index,d.name_start,d.name_end),catalog::path(input,owner),modules::path(package,module_index),name,b"",sources::source(modules::source_set(package),module_index)));}module_index+=1;
        }
        var usize dependency=0;while(dependency<manifest::dependency_count(m)){let ref u8[] name=manifest::dependency_name(m,dependency);let ref u8[] path=manifest::dependency_value(m,dependency,b"path");let ref u8[] required=manifest::dependency_value(m,dependency,b"version");if(!work::charge(meter,32+name.len+path.len+required.len)){return .err(manifest_error(input,8,10,owner,dependency,manifest::dependency_span(m,dependency,b""),name,path));}
            if(bytes::equal(name,b"stdlib")){return .err(manifest_error(input,4,10,owner,dependency,manifest::dependency_span(m,dependency,b""),name,path));}
            let usize found=switch(catalog::find_metered(input,path,meter)){case .ok(value):{value}case .err(_):{return .err(manifest_error(input,8,11,owner,dependency,manifest::dependency_span(m,dependency,b"path"),name,path));}};
            if(found<=1){return .err(manifest_error(input,5,11,owner,dependency,manifest::dependency_span(m,dependency,b"path"),name,path));}let usize target=found-1;let ref u8[] declared=manifest::package_value(catalog::package(input,target),b"version");if(!work::charge(meter,declared.len+1)){return .err(manifest_error(input,8,5,owner,dependency,manifest::dependency_span(m,dependency,b"version"),name,declared));}
            if(!bytes::equal(required,declared)){return .err(manifest_error(input,6,5,owner,dependency,manifest::dependency_span(m,dependency,b"version"),name,declared));}
            if(!v::push<Edge>(ref mut edges,Edge{.owner,.target,.declaration=dependency}).is_ok){return .err(manifest_error(input,7,10,owner,dependency,manifest::dependency_span(m,dependency,b""),name,path));}dependency+=1;
        }owner+=1;
    }return .ok(edges);
}
@effects(filesystem)
pub fn Result<Workspace,Error> load(ref u8[] root,ref mut sources::Budget budget,ref mut usize meter){
    var sources::Budget staged=*budget;let catalog::Catalog loaded=switch(catalog::load(root,ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(catalog_error(e));}};
    let v::Vec<Edge> edges=switch(declarations(ref loaded,meter)){case .ok(value):{value}case .err(e):{return .err(e);}};*budget=staged;return .ok(Workspace{.catalog=loaded,.edges});
}
