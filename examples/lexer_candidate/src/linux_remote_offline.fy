module linux_remote_offline;
import project_remote_plan as plan;
import project_remote_bind as bind;
import project_remote_archive as archive;
import project_manifest as manifest;
import linux_remote_fs as fs;
import linux_rooted_file as host;
import linux_directory as directory;
import linux_package_directory as lease;
import linux_fetch_deadline as deadline;
import sha256_stream as sha;
import std::bytes as bytes;

// Same private outcome/error stages as the Windows offline owner. This module
// never imports transport, changes source metadata or retries publication.
pub struct Outcome{usize packages;usize downloaded;usize cache_hits;usize installed;usize reused;}
pub struct Problem{u32 stage;u32 code;u32 cleanup_code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.cleanup_code=0};}
fn Problem native_failure(u32 stage,lease::Problem p){return Problem{.stage,.code=if(p.kind==2){2}else{1},.cleanup_code=p.cleanup_code};}
fn usize find(ref archive::Bundle b,ref u8[] name){var usize i=0;while(i<archive::count(b)){if(!archive::directory(b,i) && bytes::equal(archive::name(b,i),name)){return i;}i+=1;}return archive::count(b);}

@effects(filesystem,clock) fn u32 verify(ref host::Directory root,ref archive::Bundle b,ref u8[] prefix,usize depth,ref mut usize remaining,ref mut usize files,deadline::Deadline limit){
    if(depth>32 || deadline::expired(limit)){return 3;}var usize work=0;
    let entries=switch(directory::scan_packages(root,ref mut work)){case .ok(p):{p}case .err(_):{return 1;}};var usize i=0;
    while(i<directory::count(ref entries)){
        if(*remaining==0 || deadline::expired(limit)){return 3;};*remaining-=1;
        let leaf=directory::name(ref entries,i);let info=directory::metadata(ref entries,i);var path=bytes::empty();
        if(!bytes::append(ref mut path,prefix).is_ok || !bytes::append(ref mut path,leaf).is_ok){return 2;}
        if(!archive::portable(bytes::view(ref path))){return 4;}
        let kind=switch(fs::kind(root,leaf)){case .ok(p):{p}case .err(p):{return p;}};
        if(kind==1 && (info.mode&61440)==16384){
            let child=switch(host::directory(root,leaf)){case .ok(p):{p}case .err(_):{return 1;}};
            if(!bytes::append_byte(ref mut path,47).is_ok){return 2;}
            let code=verify(ref child,b,bytes::view(ref path),depth+1,remaining,files,limit);if(code!=0){return code;}
        }else{
            if(kind!=2 || !host::regular(info)){return 4;}let at=find(b,bytes::view(ref path));if(at==archive::count(b)){return 4;}
            let actual=switch(fs::read_observed(root,leaf,info,8388608)){case .ok(p):{p}case .err(p):{return p;}};
            if(!bytes::equal(bytes::view(ref actual),archive::contents(b,at))){return 4;};*files+=1;
        }i+=1;
    }return 0;
}
@effects(filesystem,clock) fn u32 write_stage(ref host::Directory stage,ref archive::Bundle b,deadline::Deadline limit){
    var usize i=0;while(i<archive::count(b)){if(deadline::expired(limit)){return 3;}if(!archive::directory(b,i)){let code=fs::write(stage,archive::name(b,i),archive::contents(b,i));if(code!=0){return code;}}i+=1;}return 0;
}
@effects(filesystem,clock) pub fn Result<bool,Problem> publish(ref host::Directory root,ref plan::Plan sources,usize index,ref archive::Bundle b,deadline::Deadline limit){
    if(deadline::expired(limit)){return .err(fail(9,3));}
    let vendor=switch(fs::descend(root,b"vendor",true)){case .ok(p):{p}case .err(p):{return .err(fail(7,p));}};let parent=ref vendor;let leaf=plan::value(sources,index,0)[7..];
    let kind=switch(fs::kind(parent,leaf)){case .ok(p):{p}case .err(p):{return .err(fail(7,p));}};
    if(kind!=0){
        if(kind!=1){return .err(fail(7,4));}let existing=switch(host::directory(parent,leaf)){case .ok(p):{p}case .err(_):{return .err(fail(7,1));}};
        var usize remaining=8192;var usize files=0;let code=verify(ref existing,b,b"",0,ref mut remaining,ref mut files,limit);
        if(code!=0 || files!=archive::files(b)){return .err(fail(7,if(code==0){4}else{code}));}return .ok(false);
    }
    var stage_name=bytes::empty();if(!bytes::append(ref mut stage_name,b".fairyc-source-").is_ok || !bytes::append(ref mut stage_name,plan::value(sources,index,4)).is_ok){return .err(fail(7,2));}
    let named=switch(fs::kind(parent,bytes::view(ref stage_name))){case .ok(p):{p}case .err(p):{return .err(fail(7,p));}};if(named!=0){return .err(fail(7,4));}
    var stage=switch(lease::create(parent,bytes::view(ref stage_name))){case .ok(p):{p}case .err(p):{return .err(native_failure(7,p));}};
    var code=write_stage(lease::directory(ref stage),b,limit);
    if(code==0 && !deadline::expired(limit)){code=fs::publish(ref mut stage,parent,leaf);}else if(code==0){code=3;}
    if(code==0){return .ok(true);}var usize remaining=8192;var cleanup_code=fs::clear(lease::directory(ref stage),0,ref mut remaining);
    if(cleanup_code==0){cleanup_code=fs::erase(ref mut stage);}return .err(Problem{.stage=7,.code,.cleanup_code});
}
@effects(filesystem,clock) fn Result<Outcome,Problem> restore_all(ref host::Directory root,ref plan::Plan sources,ref u8[] language,deadline::Deadline limit){
    var result=Outcome{.packages=0,.downloaded=0,.cache_hits=0,.installed=0,.reused=0};var usize compressed=0;var usize expanded=0;var usize files=0;var usize index=0;
    while(index<plan::count(sources)){
        if(deadline::expired(limit)){return .err(fail(9,3));}if(compressed>=67108864 || expanded>=67108864 || files>=4096){return .err(fail(4,3));}
        var path=bytes::empty();if(!bytes::append(ref mut path,b"build/package-sources/").is_ok || !bytes::append(ref mut path,plan::value(sources,index,4)).is_ok || !bytes::append(ref mut path,b".zip").is_ok){return .err(fail(4,2));}
        let usize available=if(67108864-compressed<16777216){67108864-compressed}else{16777216};let data=switch(fs::read(root,bytes::view(ref path),available)){case .ok(p):{p}case .err(p):{return .err(fail(4,p));}};
        var state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref data)) || !sha::matches(ref state,plan::value(sources,index,4))){return .err(fail(5,1));}
        let bundle=switch(archive::unpack(bytes::view(ref data),plan::value(sources,index,1),plan::value(sources,index,2),language,67108864-expanded,4096-files)){case .ok(p):{p}case .err(p):{return .err(fail(6,p));}};
        if(deadline::expired(limit)){return .err(fail(9,3));}let installed=switch(publish(root,sources,index,ref bundle,limit)){case .ok(p):{p}case .err(p):{return .err(p);}};
        compressed+=bytes::len(ref data);expanded+=archive::expanded(ref bundle);files+=archive::files(ref bundle);result.packages+=1;result.cache_hits+=1;if(installed){result.installed+=1;}else{result.reused+=1;}index+=1;
    }return .ok(result);
}
@effects(filesystem,clock) pub fn Result<Outcome,Problem> restore(ref u8[] workspace){
    let limit=switch(deadline::start()){case .ok(p):{p}case .err(p):{return .err(fail(9,p));}};
    let root=switch(host::open(workspace)){case .ok(p):{p}case .err(_):{return .err(fail(1,1));}};
    let data=switch(fs::read(ref root,b"fairy.sources.json",65536)){case .ok(p):{p}case .err(p):{return .err(fail(1,p));}};var usize meter=0;
    let sources=switch(plan::parse(data,ref mut meter)){case .ok(p):{p}case .err(p):{return .err(fail(2,p.code));}};
    let manifest_data=switch(fs::read(ref root,b"fairy.toml",1048576)){case .ok(p):{p}case .err(p):{return .err(fail(1,p));}};
    let root_manifest=switch(manifest::parse(bytes::view(ref manifest_data))){case .ok(p):{p}case .err(_):{return .err(fail(2,1));}};
    let bound=bind::validate(ref sources,ref root_manifest,ref mut meter);if(bound!=0){return .err(fail(2,bound));}
    if(deadline::expired(limit)){return .err(fail(9,3));}
    let kind=switch(fs::kind(ref root,b".fairyc-fetch.lock")){case .ok(p):{p}case .err(p):{return .err(fail(3,p));}};if(kind!=0){return .err(fail(3,4));}
    var lock=switch(lease::create(ref root,b".fairyc-fetch.lock")){case .ok(p):{p}case .err(p):{return .err(native_failure(3,p));}};
    let result=restore_all(ref root,ref sources,manifest::package_value(ref root_manifest,b"language"),limit);let cleanup_code=fs::erase(ref mut lock);
    return switch(result){case .ok(p):{if(cleanup_code==0){.ok(p)}else{.err(fail(8,cleanup_code))}}case .err(p):{.err(Problem{.stage=p.stage,.code=p.code,.cleanup_code=if(p.cleanup_code!=0){p.cleanup_code}else{cleanup_code}})}};
}
