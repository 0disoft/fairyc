module tests::linux_cache_file;
import linux_cache_file as cache;
import linux_directory as directory;
import std::bytes as bytes;
@link_name("cache_probe_reset") unsafe extern C fn void reset(u32 mode);
@link_name("cache_probe_counts") unsafe extern C fn u32 counts();
fn bytes::Buffer generation(){var out=bytes::empty();assert(bytes::append(ref mut out,b"fyproject0-").is_ok);var usize i=0;while(i<129){assert(bytes::append_byte(ref mut out,if(i==64){45}else{97}).is_ok);i+=1;}assert(bytes::append(ref mut out,b".fyc").is_ok);return out;}
@test @effects(filesystem) fn void anonymous_stage_publication_and_collision_preserve_ownership(){
    unsafe{reset(0);};let root=switch(cache::open(b".")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let name=generation();
    {var file=switch(cache::create_new(ref root,b".stage-unused")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(cache::write_all(ref mut file,b"FYCACHE0abc").is_ok);assert(cache::sync(ref mut file).is_ok);
    assert(cache::publish_new(ref root,ref mut file,bytes::view(ref name)).is_ok);assert(cache::close(file).is_ok);}
    {var file=switch(cache::create_new(ref root,b".stage-unused")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(cache::write_all(ref mut file,b"FYCACHE0abc").is_ok);
    assert(switch(cache::publish_new(ref root,ref mut file,bytes::view(ref name))){case .ok(_):{false}case .err(e):{cache::publication_collision(e) && e.os_code==17}});}
    assert(unsafe{counts()}==2);
    var file=switch(cache::open_read(ref root,bytes::view(ref name))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let info=switch(cache::observe(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(info.size==11 && info.raw.device==71 && info.raw.identity==123 && info.raw.modified_seconds==0-7 && info.raw.modified_nanos==99);
    var u8[16] data=[0;16];let n=switch(cache::read(ref mut file,ref mut data)){case .ok(p):{p}case .err(_):{assert(false);return;}};let ref u8[] raw=ref data;
    assert(n==11 && bytes::equal(raw[..n],b"FYCACHE0abc"));assert(cache::close(file).is_ok);
}
@test @effects(filesystem) fn void inventory_ignores_named_stages_and_keeps_exact_removal_metadata(){
    unsafe{reset(1);};let root=switch(cache::open(b".")){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;
    let entries=switch(cache::scan(ref root,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let name=generation();assert(cache::count(ref entries)==1 && bytes::equal(cache::name(ref entries,0),bytes::view(ref name)));
    let info=cache::metadata(ref entries,0);assert(cache::regular(info) && cache::earlier(info,cache::empty_metadata()) && cache::same_time(info,info));
    assert(switch(cache::erase_if(ref root,bytes::view(ref name),info)){case .ok(p):{p}case .err(_):{false}});
    unsafe{reset(2);};assert(switch(cache::erase_if(ref root,bytes::view(ref name),info)){case .ok(_):{false}case .err(e):{e.stage==7 && e.os_code==5 && e.completed==1}});
    work=directory::work_limit();assert(!cache::scan(ref root,4096,ref mut work).is_ok);
    assert(!cache::open(b"../unsafe").is_ok);
}
