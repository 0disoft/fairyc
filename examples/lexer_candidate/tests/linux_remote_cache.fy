module tests::linux_remote_cache;
import linux_remote_cache as cache;
import linux_remote_fs as fs;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("remote_fs_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("remote_fs_probe_live") unsafe extern C fn u32 live();
@link_name("remote_fs_probe_mutations") unsafe extern C fn u32 mutations();
@link_name("remote_cache_probe_case") unsafe extern C fn void scenario(u32 value);
@link_name("remote_cache_probe_alter") unsafe extern C fn void alter(u32 value);
@link_name("remote_cache_probe_attempts") unsafe extern C fn u32 attempts();
fn ref u8[] digest(){return b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad";}
fn ref u8[] cached(){return b"build/package-sources/ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad.zip";}
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@effects(filesystem) fn bytes::Buffer read(ref rooted::Directory root,ref u8[] path){return switch(fs::read(root,path,16777216)){case .ok(p):{p}case .err(_):{assert(false);return read(root,path);}};}
@effects(filesystem) fn cache::Problem problem(ref rooted::Directory root,ref u8[] key,ref u8[] data){return switch(cache::store(root,key,data)){case .ok(_):{assert(false);return problem(root,key,data);}case .err(p):{p}};}
@test @effects(filesystem) fn void validates_identity_and_size_before_creating_cache_directories(){
    unsafe{mode(0);}{let root=opened();assert(problem(ref root,b"abc",b"abc").code==4 && problem(ref root,b"BA7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",b"abc").code==4 && problem(ref root,digest(),b"abd").code==4);
        var oversized=bytes::empty();assert(bytes::resize_zeroed(ref mut oversized,16777217).is_ok);assert(problem(ref root,digest(),bytes::view(ref oversized)).code==3 && unsafe{mutations()}==0);
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void stores_and_rehashes_hits_without_replacing_cache_bytes(){
    unsafe{mode(0);}{let root=opened();assert(cache::store(ref root,digest(),b"abc").is_ok);let count=unsafe{mutations()};let content=read(ref root,cached());assert(bytes::equal(bytes::view(ref content),b"abc"));
        assert(cache::store(ref root,digest(),b"abc").is_ok && unsafe{mutations()}==count && unsafe{attempts()}==101);
        let blank=b"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";
        assert(cache::store(ref root,blank,b"").is_ok && cache::store(ref root,blank,b"").is_ok);
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void rejects_corrupt_linked_aliased_and_leftover_cache_entries(){
    var u32 value=1;while(value<=4){unsafe{mode(0);}{let root=opened();assert(cache::store(ref root,digest(),b"abc").is_ok);unsafe{alter(value);}let before=unsafe{mutations()};assert(problem(ref root,digest(),b"abc").code==4 && unsafe{mutations()}==before);}assert(unsafe{live()}==0);value+=1;}
    unsafe{mode(0);}{let root=opened();let parent=switch(fs::descend(ref root,b"build/package-sources",true)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        let old=b".fairyc-download-ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad.tmp";assert(fs::write(ref parent,old,b"old")==0);let before=unsafe{mutations()};assert(problem(ref root,digest(),b"abc").code==4 && unsafe{mutations()}==before);let kept=read(ref parent,old);assert(bytes::equal(bytes::view(ref kept),b"old"));
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void preserves_collision_and_close_results_without_replaying_publication(){
    var u32 failed_case=7;while(failed_case<=9){unsafe{mode(failed_case);}{let root=opened();let failed=problem(ref root,digest(),b"abc");assert(failed.code==if(failed_case==8){4}else{1});assert(failed.cleanup_code==0 && unsafe{attempts()}==if(failed_case==8){101}else{100});let parent=switch(fs::descend(ref root,b"build/package-sources",false)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(switch(fs::kind(ref parent,b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad.zip")){case .ok(p):{p==0}case .err(_):{false}});}assert(unsafe{live()}==0);failed_case+=1;}
    unsafe{mode(0);}{let root=opened();unsafe{scenario(6);}let failed=problem(ref root,digest(),b"abc");assert(failed.code==1 && failed.cleanup_code==1);unsafe{scenario(0);}let kept=read(ref root,cached());assert(bytes::equal(bytes::view(ref kept),b"abc") && cache::store(ref root,digest(),b"abc").is_ok);}assert(unsafe{live()}==0);
    var u32 value=14;while(value<=15){unsafe{mode(0);}{let root=opened();unsafe{scenario(value);}
        if(value==14){assert(cache::store(ref root,digest(),b"abc").is_ok);}else{assert(problem(ref root,digest(),b"abc").code==4);}
        assert(unsafe{attempts()}==101);let kept=read(ref root,cached());assert(bytes::equal(bytes::view(ref kept),b"abc")== (value==14));
    }assert(unsafe{live()}==0);value+=1;}
}
