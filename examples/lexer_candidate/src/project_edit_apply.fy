module project_edit_apply;
import project_edit_preview as preview;
import project_edit_options as options;
import project_edit_snapshot as snapshots;
import project_remote_fs as files;
import windows_rooted_file as host;
import std::bytes as bytes;

// Stages: lock, fresh overlay, target, temporary, write/sync, closure recheck,
// original-byte recheck, rename, published-handle close, lock cleanup.
pub struct Problem{u32 stage;u32 code;u32 cleanup_code;bool published;}
fn Problem fail(u32 stage,u32 code,bool published){return Problem{.stage,.code,.cleanup_code=0,.published};}
@effects(filesystem) fn Result<bool,Problem> writable(ref host::Directory parent,ref u8[] leaf){
    let file=switch(host::open_read(parent,leaf)){case .ok(p):{p}case .err(e):{return .err(fail(3,e.os_code,false));}};
    let info=switch(host::observe(ref file)){case .ok(p):{p}case .err(e):{discard host::close(file);return .err(fail(3,e.os_code,false));}};
    switch(host::close(file)){case .ok(_):{}case .err(e):{return .err(fail(3,e.os_code,false));}};
    // Windows permissions represented by Go FileMode are writable/read-only.
    // A writable new file preserves that mode; read-only sources cannot apply.
    if((info.attributes&1)!=0){return .err(fail(3,4,false));}
    return .ok(true);
}
@effects(filesystem) fn Result<host::File,Problem> temporary(ref host::Directory parent){
    var usize attempt=0;
    while(attempt<16){
        var name=bytes::empty();
        if(!bytes::append(ref mut name,b".fairyc-edit-stage-").is_ok || !bytes::append_usize(ref mut name,attempt).is_ok || !bytes::append(ref mut name,b".tmp").is_ok){return .err(fail(4,2,false));}
        switch(host::create_new(parent,bytes::view(ref name))){case .ok(p):{return .ok(p);}case .err(e):{if(!host::collision(e)){return .err(fail(4,e.os_code,false));}}};
        attempt+=1;
    }
    return .err(fail(4,3,false));
}
@effects(filesystem) fn Result<bool,Problem> publish(ref u8[] root_path,ref host::Directory parent,ref mut host::File stage,ref u8[] leaf,ref preview::Preview proposal){
    if(!host::write_all(stage,preview::after(proposal)).is_ok || !host::sync(stage).is_ok){return .err(fail(5,1,false));}
    let current=switch(snapshots::capture_path(root_path)){case .ok(p):{p}case .err(code):{return .err(fail(6,code,false));}};
    if(!bytes::equal(snapshots::digest(ref current),preview::fingerprint(proposal))){return .err(fail(6,5,false));}
    let original=switch(files::read(parent,leaf,8388608)){case .ok(p):{p}case .err(code):{return .err(fail(7,code,false));}};
    if(!bytes::equal(bytes::view(ref original),preview::before(proposal))){return .err(fail(7,5,false));}
    // One opened-directory rename: no removal of the destination, no retry.
    switch(host::publish(parent,stage,leaf)){case .ok(_):{return .ok(true);}case .err(e):{return .err(fail(8,e.os_code,false));}};
    return .err(fail(8,1,false));
}
@effects(filesystem,environment) fn Result<bool,Problem> locked(ref host::Directory root,ref u8[] root_path,ref options::Options o,ref preview::Preview proposal){
    switch(preview::recheck(root_path,options::uid(o),options::symbol(o),options::expected(o),proposal)){case .ok(_):{}case .err(p):{return .err(fail(2,p.code,false));}};
    let target=preview::path(proposal);let at=files::split(target);
    if(at==0){return .err(fail(3,4,false));}
    let parent=switch(files::descend(root,target[..at-1],false)){case .ok(p):{p}case .err(code):{return .err(fail(3,code,false));}};
    discard writable(ref parent,target[at..])?;
    var stage=temporary(ref parent)?;
    let outcome=publish(root_path,ref parent,ref mut stage,target[at..],proposal);
    let closed=switch(host::close(stage)){case .ok(_):{true}case .err(_):{false}};let parent_closed=switch(host::close_directory(parent)){case .ok(_):{true}case .err(_):{false}};
    return switch(outcome){
        case .ok(_):{if(!closed || !parent_closed){.err(fail(9,1,true))}else{.ok(true)}}
        case .err(p):{.err(Problem{.stage=p.stage,.code=p.code,.cleanup_code=if(!closed || !parent_closed){1}else{p.cleanup_code},.published=p.published})}
    };
}
// Cooperating Go/Fairy editors share this exclusive directory lock. Resources
// own cleanup on every ordinary exit; publication errors retain precedence.
@effects(filesystem,environment) pub fn Result<bool,Problem> apply(ref host::Directory root,ref u8[] root_path,ref options::Options o,ref preview::Preview proposal){
    let lock=switch(files::create(root,b".fairyc-edit.lock")){case .ok(p):{p}case .err(code):{return .err(fail(1,code,false));}};
    let outcome=locked(root,root_path,o,proposal);
    let erased=files::erase(ref lock);let closed=switch(host::close_directory(lock)){case .ok(_):{true}case .err(_):{false}};
    let u32 cleanup_code=if(erased!=0){erased}else if(!closed){1}else{0};
    return switch(outcome){
        case .ok(_):{if(cleanup_code!=0){.err(Problem{.stage=10,.code=cleanup_code,.cleanup_code,.published=true})}else{.ok(true)}}
        case .err(p):{.err(Problem{.stage=p.stage,.code=p.code,.cleanup_code=if(p.cleanup_code!=0){p.cleanup_code}else{cleanup_code},.published=p.published})}
    };
}
