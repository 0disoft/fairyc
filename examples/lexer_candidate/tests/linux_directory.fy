module tests::linux_directory;
import linux_directory as directory;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("directory_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("directory_probe_counts") unsafe extern C fn u32 counts();
@effects(filesystem) fn rooted::Directory root() {
    return switch(rooted::open(b"/work")) { case .ok(p): { p } case .err(_): { assert(false); return root(); } };
}
@effects(filesystem) fn directory::Entries scanned(ref rooted::Directory root,usize maximum,ref mut usize work) {
    return switch(directory::scan(root,maximum,work)) { case .ok(p): { p } case .err(_): { assert(false); return scanned(root,maximum,work); } };
}
@test @effects(filesystem) fn void owns_names_metadata_and_restarts_each_scan() {
    let root=root();unsafe { mode(0); }var usize work=0;
    let first=scanned(ref root,3,ref mut work);
    assert(work==2212 && directory::count(ref first)==3 && unsafe { counts() }==1071);
    assert(bytes::equal(directory::name(ref first,0),b"alpha") && bytes::equal(directory::name(ref first,1),b"한"));
    let info=directory::metadata(ref first,2);
    assert(info.device==71 && info.identity==127 && info.modified_seconds==-2 && info.modified_nanos==42 && !rooted::regular(info) && info.mode==40960);
    let second=scanned(ref root,3,ref mut work);
    assert(work==4424 && directory::count(ref second)==3 && unsafe { counts() }==2142);
    assert(bytes::equal(directory::name(ref first,2),b"link"));
    assert(rooted::close_directory(root).is_ok);
    assert(bytes::equal(directory::name(ref first,0),b"alpha") && directory::metadata(ref second,0).identity==125);
}
@effects(filesystem) fn void rejected(ref rooted::Directory root,u32 scenario,u32 kind,usize completed,u32 close_code) {
    unsafe { mode(scenario); }var usize work=0;
    switch(directory::scan(root,4096,ref mut work)) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(e.kind==kind && e.completed==completed && e.close_code==close_code); }
    };
}
@test @effects(filesystem) fn void preserves_primary_failure_and_closes_cursor_once() {
    let root=root();
    rejected(ref root,1,4,0,0);assert(unsafe { counts() }==1011);
    rejected(ref root,2,5,3,0);assert(unsafe { counts() }==1071);
    rejected(ref root,3,4,1,4);assert(unsafe { counts() }==1041);
    rejected(ref root,4,3,0,0);assert(unsafe { counts() }==1000);
    rejected(ref root,5,6,0,0);rejected(ref root,6,6,0,0);rejected(ref root,7,8,0,0);
    rejected(ref root,8,8,0,0);rejected(ref root,9,6,0,0);rejected(ref root,10,6,0,0);
    rejected(ref root,11,6,0,0);rejected(ref root,16,8,0,0);rejected(ref root,17,6,0,0);rejected(ref root,18,6,0,0);
    assert(unsafe { counts() }==1011);assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void enforces_exact_item_and_cumulative_work_limits() {
    let root=root();unsafe { mode(0); }var usize work=0;
    assert(!directory::scan(ref root,0,ref mut work).is_ok && !directory::scan(ref root,4097,ref mut work).is_ok && work==0 && unsafe { counts() }==0);
    work=directory::work_limit()+1;assert(!directory::scan(ref root,3,ref mut work).is_ok && unsafe { counts() }==0);
    work=directory::work_limit()-2212;let exact=scanned(ref root,3,ref mut work);
    assert(work==directory::work_limit() && directory::count(ref exact)==3);
    unsafe { mode(0); }work=directory::work_limit()-2211;
    switch(directory::scan(ref root,3,ref mut work)) { case .ok(_): { assert(false); } case .err(e): { assert(e.kind==7 && e.completed==3); } };
    assert(unsafe { counts() }==1061);
    unsafe { mode(12); }work=directory::work_limit()-1-297*3;
    switch(directory::scan(ref root,3,ref mut work)) { case .ok(_): { assert(false); } case .err(e): { assert(e.kind==7 && e.completed==0); } };
    assert(work==directory::work_limit() && unsafe { counts() }==1031);
    unsafe { mode(13); }work=0;let full=scanned(ref root,4096,ref mut work);assert(directory::count(ref full)==4096);
    assert(bytes::equal(directory::name(ref full,4095),b"entry4095"));
    rejected(ref root,14,7,4096,0);
    unsafe { mode(15); }work=0;let empty=scanned(ref root,1,ref mut work);assert(directory::count(ref empty)==0 && work==298);
    assert(rooted::close_directory(root).is_ok);
}
