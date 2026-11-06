module linux_remote_cache;
import linux_rooted_file as rooted;
import linux_remote_fs as fs;
import sha256_stream as sha;
import std::bytes as bytes;

pub struct Problem{u32 code;u32 cleanup_code;}
fn Problem fail(u32 code){return Problem{.code,.cleanup_code=0};}
fn bool digest_valid(ref u8[] digest){
    if(digest.len!=64){return false;}var usize i=0;while(i<digest.len){if(!(digest[i]>=48 && digest[i]<=57 || digest[i]>=97 && digest[i]<=102)){return false;}i+=1;}return true;
}
fn bool matches_bytes(ref u8[] data,ref u8[] digest){var state=sha::empty();return sha::update(ref mut state,data) && sha::matches(ref state,digest);}
@effects(filesystem) fn bool matches(ref rooted::Directory root,ref u8[] leaf,ref u8[] digest){
    let data=switch(fs::read(root,leaf,16777216)){case .ok(p):{p}case .err(_):{return false;}};return matches_bytes(bytes::view(ref data),digest);
}
// Caller has validated ZIP/package identity and owns the exclusive fetch lock.
// Blob identity is checked again here, including every existing-cache reuse.
@effects(filesystem) pub fn Result<bool,Problem> store(ref rooted::Directory root,ref u8[] digest,ref u8[] data){
    if(!digest_valid(digest)){return .err(fail(4));}if(data.len>16777216){return .err(fail(3));}if(!matches_bytes(data,digest)){return .err(fail(4));}
    let parent=switch(fs::descend(root,b"build/package-sources",true)){case .ok(p):{p}case .err(p):{return .err(fail(p));}};
    var leaf=bytes::empty();var stage=bytes::empty();
    if(!bytes::append(ref mut leaf,digest).is_ok || !bytes::append(ref mut leaf,b".zip").is_ok || !bytes::append(ref mut stage,b".fairyc-download-").is_ok || !bytes::append(ref mut stage,digest).is_ok || !bytes::append(ref mut stage,b".tmp").is_ok){return .err(fail(2));}
    let existing=switch(fs::kind(ref parent,bytes::view(ref leaf))){case .ok(p):{p}case .err(p):{return .err(fail(p));}};
    if(existing!=0){return if(existing==2 && matches(ref parent,bytes::view(ref leaf),digest)){.ok(true)}else{.err(fail(4))};}
    // Preserve any leftover named stage from an earlier/manual publisher. New
    // Linux writes use O_TMPFILE, so failure cleanup never needs a stage path.
    let leftover=switch(fs::kind(ref parent,bytes::view(ref stage))){case .ok(p):{p}case .err(p):{return .err(fail(p));}};if(leftover!=0){return .err(fail(4));}
    var file=switch(rooted::create(ref parent)){case .ok(p):{p}case .err(_):{return .err(fail(1));}};
    var u32 code=0;if(!rooted::write_all(ref mut file,data).is_ok){code=1;}
    if(code==0){switch(rooted::publish_new(ref parent,ref mut file,bytes::view(ref leaf))){case .ok(_):{}case .err(p):{if(p.stage!=12){code=1;}else if(!rooted::collision(p) || !matches(ref parent,bytes::view(ref leaf),digest)){code=4;}}};}
    let u32 cleanup_code=if(rooted::close(file).is_ok){0}else{1};
    return if(code==0 && cleanup_code==0){.ok(true)}else{.err(Problem{.code=if(code==0){1}else{code},.cleanup_code})};
}
