module project_remote_cache;
import windows_rooted_file as host;
import project_remote_fs as fs;
import sha256_stream as sha;
import std::bytes as bytes;

pub struct Problem{u32 code;u32 cleanup_code;}
fn Problem fail(u32 code){return Problem{.code,.cleanup_code=0};}
@effects(filesystem) fn bool matches(ref host::Directory root,ref u8[] leaf,ref u8[] digest){
    let data=switch(fs::read(root,leaf,16777216)){case .ok(p):{p}case .err(_):{return false;}};var sha::State state=sha::empty();return sha::update(ref mut state,bytes::view(ref data)) && sha::matches(ref state,digest);
}
// Called only after SHA-256, complete ZIP and package identity validation, while
// holding the root fetch lock. No replacement or fallback on link failure.
@effects(filesystem) pub fn Result<bool,Problem> store(ref host::Directory root,ref u8[] digest,ref u8[] data){
    let parent=switch(fs::descend(root,b"build/package-sources",true)){case .ok(p):{p}case .err(e):{return .err(fail(e));}};
    var bytes::Buffer leaf=bytes::empty();var bytes::Buffer stage=bytes::empty();
    if(!bytes::append(ref mut leaf,digest).is_ok || !bytes::append(ref mut leaf,b".zip").is_ok || !bytes::append(ref mut stage,b".fairyc-download-").is_ok || !bytes::append(ref mut stage,digest).is_ok || !bytes::append(ref mut stage,b".tmp").is_ok){return .err(fail(2));}
    let usize kind=switch(fs::kind(ref parent,bytes::view(ref leaf))){case .ok(k):{k}case .err(e):{return .err(fail(e));}};
    if(kind!=0){return if(kind==2 && matches(ref parent,bytes::view(ref leaf),digest)){.ok(true)}else{.err(fail(4))};}
    var file=switch(host::create_new(ref parent,bytes::view(ref stage))){case .ok(p):{p}case .err(_):{return .err(fail(1));}};
    var u32 code=0;if(!host::write_all(ref mut file,data).is_ok || !host::sync(ref mut file).is_ok){code=1;}
    if(code==0){switch(host::link_new(ref parent,ref mut file,bytes::view(ref leaf))){case .ok(_):{}case .err(p):{if(!host::publication_collision(p) || !matches(ref parent,bytes::view(ref leaf),digest)){code=4;}}};}
    let u32 cleanup_code=if(host::close(file).is_ok){0}else{1};
    return if(code==0 && cleanup_code==0){.ok(true)}else{.err(Problem{.code=if(code==0){1}else{code},.cleanup_code})};
}
