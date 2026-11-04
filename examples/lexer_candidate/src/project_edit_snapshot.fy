module project_edit_snapshot;
import project_module_visibility as visibility;
import project_workspace_lock as loader;
import project_workspace_graph as graph;
import project_instance_graph as local;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_modules as modules;
import project_source_set as sources;
import project_key_index as keys;
import project_manifest_values as rules;
import project_path as work;
import project_remote_fs as files;
import windows_rooted_file as host;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

// Exact sorted path/NUL/raw SHA-256/NUL contract used by the Go edit owner.
// The graph lock is merged separately so 512 manifests plus 512 sources still
// fit the existing 1024-key index without losing its 1025th fingerprint input.
pub resource Snapshot{keys::Index names;v::Vec<bytes::Buffer> hashes;bytes::Buffer lock_hash;bytes::Buffer digest;cleanup{}}
pub fn ref u8[] digest(ref Snapshot p){return bytes::view(ref p.digest);}
fn Result<bytes::Buffer,u32> hash(ref u8[] input){var state=sha::empty();var out=bytes::empty();if(!sha::update(ref mut state,input) || !sha::append_hex(ref state,ref mut out)){return .err(2);}return .ok(out);}
fn bool feed(ref mut sha::State state,ref u8[] path,ref u8[] value){return sha::update(state,path) && sha::update(state,b"\0") && sha::update(state,value) && sha::update(state,b"\0");}
fn Result<bool,u32> add(ref mut bytes::Buffer names,ref mut v::Vec<keys::Span> spans,ref mut v::Vec<bytes::Buffer> hashes,ref u8[] path,ref u8[] expected,ref mut usize meter){
    if(path.len>4095 || !rules::path(path)){return .err(4);}
    if(!work::charge(meter,path.len+expected.len+1)){return .err(3);}
    let value=hash(expected)?;
    if(!keys::append(names,spans,path)){return .err(2);}
    if(!v::push<bytes::Buffer>(hashes,value).is_ok){return .err(2);}
    return .ok(true);
}
fn Result<bytes::Buffer,u32> path(ref u8[] package,ref u8[] source,ref u8[] file){var out=bytes::empty();if(!bytes::equal(package,b".") && (!bytes::append(ref mut out,package).is_ok || !bytes::append_byte(ref mut out,47).is_ok)){return .err(2);}if(source.len!=0 && (!bytes::append(ref mut out,source).is_ok || !bytes::append_byte(ref mut out,47).is_ok)){return .err(2);}if(!bytes::append(ref mut out,file).is_ok){return .err(2);}return .ok(out);}
// Hash exact owned input bytes instead of reopening every source and repeating
// directory scans. This entry describes the supplied graph; capture below is
// the fresh locked-workspace entry used by application rechecks.
@effects(filesystem) pub fn Result<Snapshot,u32> from_resolved(ref host::Directory root,ref visibility::Resolved resolved){
    let packages=workspace::local_catalog(local::input(graph::local_graph(visibility::workspace_graph(resolved))));
    var names=bytes::empty();var v::Vec<keys::Span> spans=v::empty<keys::Span>();var v::Vec<bytes::Buffer> hashes=v::empty<bytes::Buffer>();var usize meter=0;var usize owner=0;
    while(owner<catalog::count(packages)){
        let s=modules::source_set(catalog::input(packages,owner));let base=catalog::path(packages,owner);
        let manifest=path(base,b"",b"fairy.toml")?;
        discard add(ref mut names,ref mut spans,ref mut hashes,bytes::view(ref manifest),sources::manifest_input(s),ref mut meter)?;
        var usize at=0;while(at<sources::count(s)){
            let relative=path(base,sources::package_value(s,b"source"),sources::path(s,at))?;
            discard add(ref mut names,ref mut spans,ref mut hashes,bytes::view(ref relative),sources::source(s,at),ref mut meter)?;
            at+=1;
        }
        owner+=1;
    }
    let index=keys::build(names,spans,ref mut meter)?;
    let lock=files::read(root,b"fairy.lock",8388608)?;
    if(!work::charge(ref mut meter,bytes::len(ref lock)+1)){return .err(3);}
    let lock_hash=hash(bytes::view(ref lock))?;
    var state=sha::empty();var bool inserted=false;var usize i=0;
    while(i<keys::count(ref index)){
        let at=keys::ordered(ref index,i);let name=keys::key(ref index,at);
        if(bytes::equal(name,b"fairy.lock") || i>0 && !rules::before(keys::key(ref index,keys::ordered(ref index,i-1)),name)){return .err(4);}
        if(!inserted && rules::before(b"fairy.lock",name)){if(!feed(ref mut state,b"fairy.lock",bytes::view(ref lock_hash))){return .err(3);}inserted=true;}
        if(!feed(ref mut state,name,bytes::view(v::at<bytes::Buffer>(ref hashes,at)))){return .err(3);}i+=1;
    }
    if(!inserted && !feed(ref mut state,b"fairy.lock",bytes::view(ref lock_hash))){return .err(3);}
    var digest=bytes::empty();if(!sha::append_hex(ref state,ref mut digest)){return .err(2);}
    return .ok(Snapshot{.names=index,.hashes,.lock_hash,.digest});
}
@effects(filesystem) pub fn Result<Snapshot,u32> capture(ref host::Directory root,ref u8[] path){
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(path,true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(5);}};
    return from_resolved(root,loader::workspace(ref loaded));
}
