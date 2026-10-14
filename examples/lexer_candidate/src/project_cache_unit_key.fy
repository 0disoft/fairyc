module project_cache_unit_key;
import project_cache_environment as environment;
import project_cache_profile as profile;
import project_cache_key as compatibility;
import project_compiler_input as input;
import generic_source_plan as plan;
import generic_source_plan_input as prepared;
import generic_source_catalog as templates;
import generic_package_bindings as bindings;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import sha256_stream as sha;
import std::bytes as bytes;

pub resource Identity{payload::Payload canonical;bytes::Buffer route;cleanup{}}
pub fn ref u8[] canonical(ref Identity p){return payload::content(ref p.canonical);}
pub fn ref u8[] route(ref Identity p){return bytes::view(ref p.route);}
// Lookup retains exact compiler/profile/SDK authority and the selected package
// instance. Dependency source digests are deliberately absent: current semantic
// admission decides whether their consumed contracts still match. Numeric source
// slots and resolved imports remain in the route; reordered topology is a miss.
pub fn Result<Identity,u32> collect(ref input::Input sources,ref plan::Plan original,ref environment::Environment env,usize first,usize stop,ref mut usize work){
    if(first>=stop || stop>input::count(sources)){return .err(1001);}let owner=input::origin(sources,first).owner;var usize i=first;while(i<stop){if(input::origin(sources,i).owner!=owner){return .err(1001);}i+=1;}
    let fields=switch(environment::key_environment(env,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var wire::Status status=wire::Status{.code=0};var out=wire::writer_marker(b"FYPUNIT0",ref mut status,work);wire::number(ref mut out,first,ref mut status,work);wire::number(ref mut out,stop,ref mut status,work);
    i=0;while(i<8){wire::member(ref mut out,compatibility::environment_field(ref fields,i),ref mut status,work);i+=1;}wire::member(ref mut out,profile::target(),ref mut status,work);wire::member(ref mut out,environment::standard_library(env),ref mut status,work);wire::member(ref mut out,input::instance_id(sources,first),ref mut status,work);
    let stamps=templates::packages(prepared::templates(plan::input(original)));wire::number(ref mut out,bindings::count(stamps),ref mut status,work);i=0;
    while(i<bindings::count(stamps)){let row=bindings::module_at(stamps,i);wire::number(ref mut out,row.owner,ref mut status,work);wire::member(ref mut out,bindings::text(stamps,row.namespace),ref mut status,work);wire::member(ref mut out,bindings::text(stamps,row.path),ref mut status,work);i+=1;}
    wire::number(ref mut out,bindings::import_count(stamps),ref mut status,work);i=0;while(i<bindings::import_count(stamps)){let row=bindings::import_at(stamps,i);wire::number(ref mut out,row.module_index,ref mut status,work);wire::number(ref mut out,row.declaration,ref mut status,work);wire::number(ref mut out,row.target,ref mut status,work);wire::member(ref mut out,bindings::text(stamps,row.alias),ref mut status,work);wire::member(ref mut out,bindings::text(stamps,row.binding),ref mut status,work);i+=1;}
    let canonical=switch(wire::finish(out,status)){case .ok(p):{p}case .err(code):{return .err(code);}};let raw=payload::content(ref canonical);if(!wire::charge(work,raw.len+82)){return .err(1002);}var sha::State hash=sha::empty();if(!sha::update(ref mut hash,raw)){return .err(1002);}var bytes::Buffer route=bytes::empty();if(!bytes::append(ref mut route,b"fylookup0-sha256-").is_ok || !sha::append_hex(ref hash,ref mut route)){return .err(1003);}return .ok(Identity{.canonical,.route});
}
