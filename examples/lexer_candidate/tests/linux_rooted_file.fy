module tests::linux_rooted_file;
import linux_rooted_file as rooted;
@link_name("rooted_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("rooted_probe_counts") unsafe extern C fn u32 counts();

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
    assert(!rooted::open(b"").is_ok && !rooted::open(b"x\0y").is_ok);
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
