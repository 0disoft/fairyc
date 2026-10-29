module tests::linux_rooted_file;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("rooted_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("rooted_probe_counts") unsafe extern C fn u32 counts();
@link_name("rooted_probe_written") unsafe extern C fn usize written();
@link_name("rooted_probe_steps") unsafe extern C fn u32 steps();

@effects(filesystem) fn rooted::Directory opened() {
    return switch(rooted::open(b"/work")) { case .ok(p): { p } case .err(_): { assert(false); return opened(); } };
}
@test @effects(filesystem) fn void validates_leaves_before_native_calls() {
    unsafe { mode(0); }
    assert(!rooted::valid_leaf(b"") && !rooted::valid_leaf(b".") && !rooted::valid_leaf(b".."));
    assert(!rooted::valid_leaf(b"a/b") && !rooted::valid_leaf(b"a\0b") && !rooted::valid_leaf(b"\xff"));
    assert(rooted::valid_leaf(b"한 😀\\:x"));
    var u8[256] large=[65;256];let ref u8[] view=ref large;
    assert(rooted::valid_leaf(view[..255]) && !rooted::valid_leaf(view));
    assert(!rooted::open(b"").is_ok && !rooted::open(b"x\0y").is_ok && !rooted::open(b"/work/").is_ok);
    assert(unsafe { counts() }==0);
    let root=opened();
    assert(!rooted::open_read(ref root,b"../x").is_ok && !rooted::directory(ref root,b"..").is_ok);
    assert(unsafe { counts() }==100);
    assert(rooted::close_directory(root).is_ok && unsafe { counts() }==101);
}
@test @effects(filesystem) fn void preserves_metadata_binary_reads_and_consuming_close() {
    unsafe { mode(0); }
    let root=opened();
    let child=switch(rooted::directory(ref root,b"child")) { case .ok(p): { p } case .err(_): { assert(false); return; } };
    assert(rooted::close_directory(child).is_ok);
    var file=switch(rooted::open_read(ref root,b"input")) { case .ok(p): { p } case .err(_): { assert(false); return; } };
    let info=switch(rooted::observe(ref file)) { case .ok(p): { p } case .err(_): { assert(false); return; } };
    assert(info.device==71 && info.identity==123 && info.size==3 && info.modified_seconds==91 && info.modified_nanos==42 && rooted::regular(info));
    var u8[4] data=[0;4];
    switch(rooted::read(ref mut file,ref mut data)) { case .ok(n): { assert(n==3 && data[0]==97 && data[1]==0 && data[2]==255); } case .err(_): { assert(false); } };
    unsafe { mode(1); }
    switch(rooted::close(file)) { case .ok(_): { assert(false); } case .err(e): { assert(e.stage==7 && e.os_code==4); } };
    assert(unsafe { counts() }==1);
    unsafe { mode(0); } assert(rooted::close_directory(root).is_ok && unsafe { counts() }==1);
}
@test @effects(filesystem) fn void separates_missing_io_and_malformed_native_results() {
    unsafe { mode(0); } let root=opened();
    unsafe { mode(2); }
    switch(rooted::open_read(ref root,b"input")) { case .ok(_): { assert(false); } case .err(e): { assert(rooted::missing(e)); } };
    unsafe { mode(0); }
    var file=switch(rooted::open_read(ref root,b"input")) { case .ok(p): { p } case .err(_): { assert(false); return; } };
    var u8[4] data=[0;4];unsafe { mode(3); }
    switch(rooted::read(ref mut file,ref mut data)) { case .ok(_): { assert(false); } case .err(e): { assert(e.stage==8); } };
    unsafe { mode(4); }
    switch(rooted::read(ref mut file,ref mut data)) { case .ok(_): { assert(false); } case .err(e): { assert(e.stage==6 && e.os_code==5); } };
    unsafe { mode(5); } assert(!rooted::observe(ref file).is_ok);
    unsafe { mode(0); } assert(rooted::close(file).is_ok); assert(rooted::close_directory(root).is_ok);
}
@effects(filesystem) fn rooted::File created(ref rooted::Directory root) {
    return switch(rooted::create(root)) { case .ok(p): { p } case .err(_): { assert(false); return created(root); } };
}
@test @effects(filesystem) fn void writes_partial_chunks_syncs_before_exclusive_publication_and_freezes() {
    unsafe { mode(0); } let root=opened();var file=created(ref root);
    var data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,131079).is_ok);
    switch(rooted::write_all(ref mut file,bytes::view(ref data))) { case .ok(n): { assert(n==131079); } case .err(_): { assert(false); } };
    assert(unsafe { written() }==131079 && unsafe { steps() }==300);
    assert(!rooted::publish_new(ref root,ref mut file,b"../x").is_ok && unsafe { steps() }==300);
    assert(rooted::publish_new(ref root,ref mut file,b"new").is_ok && unsafe { steps() }==311);
    assert(!rooted::write_all(ref mut file,b"x").is_ok && !rooted::publish_new(ref root,ref mut file,b"new").is_ok);
    assert(unsafe { steps() }==311);assert(rooted::close(file).is_ok);assert(rooted::close_directory(root).is_ok);
}
@effects(filesystem) fn void failed_write(ref rooted::Directory root,u32 scenario) {
    unsafe { mode(scenario); } var file=created(root);
    switch(rooted::write_all(ref mut file,b"0123456789")) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(e.stage==if(scenario==12){10}else{8});assert(e.completed==if(scenario==12){7}else{0}); }
    };
}
@effects(filesystem) fn void failed_publish(ref rooted::Directory root,u32 scenario) {
    unsafe { mode(scenario); } var file=created(root);
    switch(rooted::publish_new(root,ref mut file,b"new")) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(e.stage==if(scenario==13){11}else{12});assert(rooted::collision(e)==(scenario==14)); }
    };
    assert(unsafe { steps() }==if(scenario==13){10}else{11});
}
@test @effects(filesystem) fn void reports_partial_progress_and_cleans_up_failed_stages() {
    unsafe { mode(0); } let root=opened();unsafe { mode(10); }
    switch(rooted::create(ref root)) { case .ok(_): { assert(false); } case .err(e): { assert(e.stage==9 && e.os_code==95); } };
    assert(unsafe { counts() }==100);
    failed_write(ref root,11);assert(unsafe { counts() }==101);
    failed_write(ref root,12);assert(unsafe { counts() }==101);
    failed_write(ref root,15);assert(unsafe { counts() }==101);
    failed_publish(ref root,13);assert(unsafe { counts() }==101);
    failed_publish(ref root,14);assert(unsafe { counts() }==101);
    failed_publish(ref root,16);assert(unsafe { counts() }==101);
    unsafe { mode(0); }assert(rooted::close_directory(root).is_ok);
}
