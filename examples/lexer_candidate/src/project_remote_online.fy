module project_remote_online;
import project_remote_offline as offline;
import project_remote_plan as plan;
import project_remote_bind as bind;
import project_remote_archive as archive;
import project_remote_cache as cache;
import project_remote_https as https;
import project_remote_fs as fs;
import project_manifest as manifest;
import windows_rooted_file as host;
import sha256_stream as sha;
import std::bytes as bytes;

@effects(clock) @link_name("fairyc_private_windows_ticks_v1") unsafe extern C fn usize ticks();
fn offline::Problem fail(u32 stage,u32 code){return offline::Problem{.stage,.code,.cleanup_code=0};}
@effects(clock) fn usize remaining(usize started){let usize now=unsafe{ticks()};return if(now<started || now-started>=120000){0}else{120000-(now-started)};}
@effects(clock) fn bool expired(usize started){return remaining(started)==0;}
@effects(filesystem,network,clock) fn Result<offline::Outcome,offline::Problem> restore_all(ref host::Directory root,ref plan::Plan sources,ref u8[] language,usize started){
    var offline::Outcome result=offline::Outcome{.packages=0,.downloaded=0,.cache_hits=0,.installed=0,.reused=0};var usize compressed=0;var usize expanded=0;var usize files=0;var usize index=0;
    while(index<plan::count(sources)){if(expired(started)){return .err(fail(9,3));}if(compressed>=67108864 || expanded>=67108864 || files>=4096){return .err(fail(4,3));}
        var bytes::Buffer path=bytes::empty();if(!bytes::append(ref mut path,b"build/package-sources/").is_ok || !bytes::append(ref mut path,plan::value(sources,index,4)).is_ok || !bytes::append(ref mut path,b".zip").is_ok){return .err(fail(4,2));}
        let usize available=if(67108864-compressed<16777216){67108864-compressed}else{16777216};var bool hit=true;
        var bytes::Buffer data=switch(fs::read(root,bytes::view(ref path),available)){case .ok(value):{value}case .err(e):{
            if(e!=5){return .err(fail(4,e));}hit=false;let usize time=remaining(started);if(time==0){return .err(fail(9,3));}
            switch(https::download(plan::value(sources,index,3),available,if(time>30000){30000}else{time})){case .ok(value):{value}case .err(code):{return .err(fail(10,code));}}
        }};
        var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref data)) || !sha::matches(ref state,plan::value(sources,index,4))){return .err(fail(5,1));}
        let bundle=switch(archive::unpack(bytes::view(ref data),plan::value(sources,index,1),plan::value(sources,index,2),language,67108864-expanded,4096-files)){case .ok(value):{value}case .err(e):{return .err(fail(6,e));}};
        if(expired(started)){return .err(fail(9,3));}if(!hit){switch(cache::store(root,plan::value(sources,index,4),bytes::view(ref data))){case .ok(_):{}case .err(e):{return .err(offline::Problem{.stage=11,.code=e.code,.cleanup_code=e.cleanup_code});}};}
        if(expired(started)){return .err(fail(9,3));}let bool installed=switch(offline::publish(root,sources,index,ref bundle,started)){case .ok(value):{value}case .err(e):{return .err(e);}};
        compressed+=bytes::len(ref data);expanded+=archive::expanded(ref bundle);files+=archive::files(ref bundle);result.packages+=1;if(hit){result.cache_hits+=1;}else{result.downloaded+=1;}if(installed){result.installed+=1;}else{result.reused+=1;}index+=1;
    }return .ok(result);
}
// Online preparation is explicit. Stable compilation and offline restoration do
// not call this path; only an absent exact cache blob can trigger transport.
@effects(filesystem,network,clock) pub fn Result<offline::Outcome,offline::Problem> fetch(ref u8[] workspace){
    let usize started=unsafe{ticks()};let root=switch(host::open(workspace)){case .ok(value):{value}case .err(_):{return .err(fail(1,1));}};
    let data=switch(fs::read(ref root,b"fairy.sources.json",65536)){case .ok(value):{value}case .err(e):{return .err(fail(1,e));}};var usize meter=0;
    let sources=switch(plan::parse(data,ref mut meter)){case .ok(value):{value}case .err(p):{return .err(fail(2,p.code));}};
    let manifest_data=switch(fs::read(ref root,b"fairy.toml",1048576)){case .ok(value):{value}case .err(e):{return .err(fail(1,e));}};let root_manifest=switch(manifest::parse(bytes::view(ref manifest_data))){case .ok(value):{value}case .err(_):{return .err(fail(2,1));}};
    let u32 bound=bind::validate(ref sources,ref root_manifest,ref mut meter);if(bound!=0){return .err(fail(2,bound));}
    if(expired(started)){return .err(fail(9,3));}let lock=switch(fs::create(ref root,b".fairyc-fetch.lock")){case .ok(value):{value}case .err(e):{return .err(fail(3,e));}};
    let result=restore_all(ref root,ref sources,manifest::package_value(ref root_manifest,b"language"),started);let u32 cleanup_code=fs::erase(ref lock);
    return switch(result){case .ok(value):{if(cleanup_code==0){.ok(value)}else{.err(fail(8,cleanup_code))}}case .err(p):{.err(offline::Problem{.stage=p.stage,.code=p.code,.cleanup_code=if(p.cleanup_code!=0){p.cleanup_code}else{cleanup_code}})}};
}
