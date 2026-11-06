module linux_edit_apply;
import linux_rooted_file as rooted;
import linux_package_directory as lease;
import linux_edit_file as file;
import linux_edit_fs as files;
import project_edit_preview as preview;
import project_edit_snapshot as snapshots;
import project_edit_options as options;
import std::bytes as bytes;

// Stages match the shared command: lock, fresh overlay, target, temporary,
// write/prepare, closure recheck, original recheck, publish, close, lock cleanup.
pub struct Problem{u32 stage;u32 code;u32 cleanup_code;bool published;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.cleanup_code=0,.published=false};}
fn u32 cause(u32 code,u32 fallback){return if(code!=0){code}else{fallback};}
fn Problem io(u32 stage,files::Problem e){var p=fail(stage,cause(e.os_code,e.kind));p.cleanup_code=cause(e.close_code,e.directory_close_code);return p;}
// Only the first cleanup cause supplements a failed operation. A successful
// operation is already published; cleanup errors must preserve that fact.
fn Result<bool,Problem> cleaned(Result<bool,Problem> outcome,u32 stage,u32 primary,u32 secondary){
    return switch(outcome){
        case .ok(_):{if(primary!=0 || secondary!=0){return .err(Problem{.stage,.code=cause(primary,secondary),.cleanup_code=if(primary!=0){secondary}else{0},.published=true});};.ok(true)}
        case .err(original):{var p=original;if(p.cleanup_code==0){p.cleanup_code=cause(primary,secondary);};.err(p)}
    };
}
// These are definitely existing-name collisions, never publication retries.
// A secondary acquisition error prevents even another unused-name attempt.
@effects(filesystem) fn Result<lease::Directory,Problem> temporary(ref rooted::Directory parent){
    var usize attempt=0;
    while(attempt<16){
        var name=bytes::empty();if(!bytes::append(ref mut name,b".fairyc-edit-stage-").is_ok || !bytes::append_usize(ref mut name,attempt).is_ok || !bytes::append(ref mut name,b".tmp").is_ok){return .err(fail(4,2));}
        switch(lease::create(parent,bytes::view(ref name))){
            case .ok(p):{return .ok(p);}
            case .err(e):{if(e.os_code!=17 || e.cleanup_code!=0){var p=fail(4,cause(e.os_code,e.kind));p.cleanup_code=e.cleanup_code;return .err(p);}}
        };attempt+=1;
    }return .err(fail(4,3));
}
@effects(filesystem) fn Result<bool,Problem> publish(ref u8[] root_path,ref rooted::Directory parent,ref lease::Directory stage,ref mut file::File output,ref u8[] leaf,rooted::Metadata original,ref preview::Preview proposal){
    switch(file::write_all(output,preview::after(proposal))){case .ok(_):{}case .err(e):{return .err(fail(5,cause(e.os_code,e.stage)));}};
    switch(file::prepare(output)){case .ok(_):{}case .err(e):{return .err(fail(5,cause(e.os_code,e.stage)));}};
    let current=switch(snapshots::capture_path(root_path)){case .ok(p):{p}case .err(code):{return .err(fail(6,code));}};
    if(!bytes::equal(snapshots::digest(ref current),preview::fingerprint(proposal))){return .err(fail(6,5));}
    var info=files::empty_metadata();
    let old=switch(files::read_observed(parent,leaf,files::maximum(),ref mut info)){case .ok(p):{p}case .err(e):{return .err(io(7,e));}};
    if(!bytes::equal(bytes::view(ref old),preview::before(proposal)) || !files::same(info,original)){return .err(fail(7,5));}
    // One rename attempt. Never remove the destination or replay publication.
    return switch(file::publish(output,lease::directory(stage),parent,leaf)){
        case .ok(_):{.ok(true)}
        case .err(e):{.err(Problem{.stage=8,.code=cause(e.os_code,cause(e.cleanup_code,e.stage)),.cleanup_code=e.cleanup_code,.published=e.published})}
    };
}
@effects(filesystem) fn Result<bool,Problem> staged(ref u8[] root_path,ref rooted::Directory parent,ref lease::Directory stage,ref u8[] leaf,rooted::Metadata original,ref preview::Preview proposal){
    var output=switch(file::create(lease::directory(stage),original)){case .ok(p):{p}case .err(e):{return .err(fail(4,cause(e.os_code,e.stage)));}};
    let outcome=publish(root_path,parent,stage,ref mut output,leaf,original,proposal);
    let u32 closed=switch(file::close(output)){case .ok(_):{0}case .err(e):{e.os_code}};
    return cleaned(outcome,9,closed,0);
}
@effects(filesystem) fn Result<bool,Problem> prepared(ref u8[] root_path,ref rooted::Directory parent,ref u8[] leaf,ref preview::Preview proposal){
    var original=files::empty_metadata();
    let old=switch(files::read_observed(parent,leaf,files::maximum(),ref mut original)){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};
    if(!bytes::equal(bytes::view(ref old),preview::before(proposal)) || !file::writable(original)){return .err(fail(3,4));}
    var stage=temporary(parent)?;
    let outcome=staged(root_path,parent,ref stage,leaf,original,proposal);
    let u32 erased=switch(lease::erase(ref mut stage)){case .ok(_):{0}case .err(e):{cause(e.os_code,e.kind)}};
    let closed=lease::close(stage);
    let finished=cleaned(outcome,9,erased,0);
    return switch(closed){case .ok(_):{finished}case .err(e):{cleaned(finished,9,e.os_code,e.cleanup_code)}};
}
@effects(filesystem,environment) fn Result<bool,Problem> locked(ref rooted::Directory root,ref u8[] root_path,ref options::Options o,ref preview::Preview proposal){
    switch(preview::recheck(root_path,options::uid(o),options::symbol(o),options::expected(o),proposal)){case .ok(_):{}case .err(e):{return .err(fail(2,e.code));}};
    let target=preview::path(proposal);let at=files::split(target);if(at==0){return .err(fail(3,4));}
    let parent=switch(files::open_directory(root,target[..at-1])){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};
    let outcome=prepared(root_path,ref parent,target[at..],proposal);
    let u32 closed=switch(rooted::close_directory(parent)){case .ok(_):{0}case .err(e):{e.os_code}};
    return cleaned(outcome,9,closed,0);
}
@effects(filesystem,environment) pub fn Result<bool,Problem> apply(ref rooted::Directory root,ref u8[] root_path,ref options::Options o,ref preview::Preview proposal){
    var lock=switch(lease::create(root,b".fairyc-edit.lock")){case .ok(p):{p}case .err(e):{var p=fail(1,cause(e.os_code,e.kind));p.cleanup_code=e.cleanup_code;return .err(p);}};
    let outcome=locked(root,root_path,o,proposal);
    let u32 erased=switch(lease::erase(ref mut lock)){case .ok(_):{0}case .err(e):{cause(e.os_code,e.kind)}};
    let closed=lease::close(lock);
    let finished=cleaned(outcome,10,erased,0);
    return switch(closed){case .ok(_):{finished}case .err(e):{cleaned(finished,10,e.os_code,e.cleanup_code)}};
}
