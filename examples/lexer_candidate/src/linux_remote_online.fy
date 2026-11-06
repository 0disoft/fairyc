module linux_remote_online;
import linux_remote_offline as offline;
import project_remote_plan as plan;
import project_remote_bind as bind;
import project_remote_archive as archive;
import linux_remote_cache as cache;
import linux_remote_https as https;
import linux_remote_fs as fs;
import project_manifest as manifest;
import linux_rooted_file as host;
import linux_package_directory as lease;
import linux_fetch_deadline as deadline;
import sha256_stream as sha;
import std::bytes as bytes;

// Preserve HTTPS native and consuming-close causes through orchestration. The
// existing offline stages and outcomes retain their meanings; no retry occurs.
pub struct Problem{u32 stage;u32 code;u32 native_code;u32 cleanup_code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.native_code=0,.cleanup_code=0};}
fn Problem publication(offline::Problem p){return Problem{.stage=p.stage,.code=p.code,.native_code=0,.cleanup_code=p.cleanup_code};}
pub fn usize archive_limit(usize compressed){return if(compressed>=67108864){0}else if(67108864-compressed<16777216){67108864-compressed}else{16777216};}
pub fn usize request_limit(usize remaining){return if(remaining>30000){30000}else{remaining};}

@effects(filesystem,network,clock) fn Result<offline::Outcome,Problem> restore_all(ref host::Directory root,ref plan::Plan sources,ref u8[] language,deadline::Deadline limit){
    var result=offline::Outcome{.packages=0,.downloaded=0,.cache_hits=0,.installed=0,.reused=0};var usize compressed=0;var usize expanded=0;var usize files=0;var usize index=0;
    while(index<plan::count(sources)){
        if(deadline::expired(limit)){return .err(fail(9,3));}
        let available=archive_limit(compressed);if(available==0 || expanded>=67108864 || files>=4096){return .err(fail(4,3));}
        var path=bytes::empty();if(!bytes::append(ref mut path,b"build/package-sources/").is_ok || !bytes::append(ref mut path,plan::value(sources,index,4)).is_ok || !bytes::append(ref mut path,b".zip").is_ok){return .err(fail(4,2));}
        var bool hit=true;
        let data=switch(fs::read(root,bytes::view(ref path),available)){
            case .ok(p):{p}
            case .err(p):{
                // Corruption, case/link/type conflicts and I/O failures never
                // fall back to transport or replace the observed cache object.
                if(p!=5){return .err(fail(4,p));}hit=false;
                let time=request_limit(deadline::remaining(limit));if(time==0){return .err(fail(9,3));}
                switch(https::download(plan::value(sources,index,3),available,time)){
                    case .ok(p):{p}
                    case .err(p):{return .err(Problem{.stage=10,.code=p.code,.native_code=p.native_code,.cleanup_code=p.cleanup_code});}
                }
            }
        };
        if(deadline::expired(limit)){return .err(fail(9,3));}
        var state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref data)) || !sha::matches(ref state,plan::value(sources,index,4))){return .err(fail(5,1));}
        let bundle=switch(archive::unpack(bytes::view(ref data),plan::value(sources,index,1),plan::value(sources,index,2),language,67108864-expanded,4096-files)){case .ok(p):{p}case .err(p):{return .err(fail(6,p));}};
        if(deadline::expired(limit)){return .err(fail(9,3));}
        if(!hit){switch(cache::store(root,plan::value(sources,index,4),bytes::view(ref data))){case .ok(_):{}case .err(p):{return .err(Problem{.stage=11,.code=p.code,.native_code=0,.cleanup_code=p.cleanup_code});}};}
        if(deadline::expired(limit)){return .err(fail(9,3));}
        let installed=switch(offline::publish(root,sources,index,ref bundle,limit)){case .ok(p):{p}case .err(p):{return .err(publication(p));}};
        compressed+=bytes::len(ref data);expanded+=archive::expanded(ref bundle);files+=archive::files(ref bundle);result.packages+=1;
        if(hit){result.cache_hits+=1;}else{result.downloaded+=1;}if(installed){result.installed+=1;}else{result.reused+=1;}index+=1;
    }
    if(deadline::expired(limit)){return .err(fail(9,3));}return .ok(result);
}

// Explicit online preparation only. Stable compilation and offline restoration
// do not call this module. Acquire the lock only after plan/manifest binding.
@effects(filesystem,network,clock) pub fn Result<offline::Outcome,Problem> fetch(ref u8[] workspace){
    let limit=switch(deadline::start()){case .ok(p):{p}case .err(p):{return .err(Problem{.stage=9,.code=p,.native_code=p,.cleanup_code=0});}};
    let root=switch(host::open(workspace)){case .ok(p):{p}case .err(_):{return .err(fail(1,1));}};
    let data=switch(fs::read(ref root,b"fairy.sources.json",65536)){case .ok(p):{p}case .err(p):{return .err(fail(1,p));}};var usize meter=0;
    let sources=switch(plan::parse(data,ref mut meter)){case .ok(p):{p}case .err(p):{return .err(fail(2,p.code));}};
    let manifest_data=switch(fs::read(ref root,b"fairy.toml",1048576)){case .ok(p):{p}case .err(p):{return .err(fail(1,p));}};
    let root_manifest=switch(manifest::parse(bytes::view(ref manifest_data))){case .ok(p):{p}case .err(_):{return .err(fail(2,1));}};
    let bound=bind::validate(ref sources,ref root_manifest,ref mut meter);if(bound!=0){return .err(fail(2,bound));}
    if(deadline::expired(limit)){return .err(fail(9,3));}
    let kind=switch(fs::kind(ref root,b".fairyc-fetch.lock")){case .ok(p):{p}case .err(p):{return .err(fail(3,p));}};if(kind!=0){return .err(fail(3,4));}
    var lock=switch(lease::create(ref root,b".fairyc-fetch.lock")){case .ok(p):{p}case .err(p):{return .err(Problem{.stage=3,.code=if(p.kind==2){2}else{1},.native_code=p.os_code,.cleanup_code=p.cleanup_code});}};
    let result=restore_all(ref root,ref sources,manifest::package_value(ref root_manifest,b"language"),limit);let cleanup_code=fs::erase(ref mut lock);
    return switch(result){
        case .ok(p):{if(cleanup_code==0){.ok(p)}else{.err(fail(8,cleanup_code))}}
        case .err(p):{.err(Problem{.stage=p.stage,.code=p.code,.native_code=p.native_code,.cleanup_code=if(p.cleanup_code!=0){p.cleanup_code}else{cleanup_code}})}
    };
}
