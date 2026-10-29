module tests::linux_retention;
import linux_retention as retention;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("retention_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("retention_probe_mask") unsafe extern C fn u32 mask();
@link_name("retention_probe_counts") unsafe extern C fn u32 counts();
@link_name("retention_probe_name") unsafe extern C fn void native_name(ptr mut u8 data,u32 index);
@effects(filesystem) fn rooted::Directory opened() { return switch(rooted::open(b"/work")) { case .ok(p): { p } case .err(_): { assert(false);return opened(); } }; }
fn bytes::Buffer name(u32 index) {
    var data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,145).is_ok);
    unsafe { native_name(bytes::view_mut(ref mut data) as ptr mut u8,index); }
    assert(bytes::truncate(ref mut data,144).is_ok);return data;
}
fn retention::Limits limits() { return retention::Limits{.entries=2,.bytes=20,.record=16}; }
@effects(filesystem) fn retention::Stats swept(ref rooted::Directory root,retention::Limits p,ref u8[] preserve,ref mut usize work) {
    return switch(retention::sweep(root,p,preserve,work)) { case .ok(p): { p } case .err(_): { assert(false);return swept(root,p,preserve,work); } };
}
@test @effects(filesystem) fn void evicts_oldest_and_oversize_but_preserves_selected_and_unrelated_files() {
    let root=opened();var usize work=0;unsafe { mode(0); }
    let stats=swept(ref root,limits(),b"",ref mut work);assert(stats.entries==2 && stats.bytes==18 && stats.evicted==1 && unsafe { mask() }==2);
    let protected=name(1);unsafe { mode(0); }work=0;
    let kept=swept(ref root,limits(),bytes::view(ref protected),ref mut work);assert(kept.entries==2 && kept.bytes==20 && kept.evicted==1 && unsafe { mask() }==4);
    unsafe { mode(6); }work=0;let large=swept(ref root,retention::Limits{.entries=3,.bytes=64,.record=16},b"",ref mut work);
    assert(large.entries==2 && large.bytes==18 && large.evicted==1 && unsafe { mask() }==2);
    unsafe { mode(8); }work=0;let empty=swept(ref root,limits(),b"",ref mut work);assert(empty.entries==0 && empty.bytes==0 && empty.evicted==0 && unsafe { mask() }==0);
    unsafe { mode(9); }work=0;let tied=swept(ref root,retention::Limits{.entries=2,.bytes=64,.record=16},b"",ref mut work);assert(tied.evicted==1 && unsafe { mask() }==1);
    unsafe { mode(10); }work=0;let nanos=swept(ref root,retention::Limits{.entries=2,.bytes=64,.record=16},b"",ref mut work);assert(nanos.evicted==1 && unsafe { mask() }==4);
    assert(rooted::close_directory(root).is_ok);
}
@effects(filesystem) fn void failed(ref rooted::Directory root,u32 scenario,u32 kind,usize evicted,u32 os_code,u32 close_code) {
    unsafe { mode(scenario); }var usize work=0;
    let p=if(scenario==4){retention::Limits{.entries=1,.bytes=8,.record=8}}else{limits()};
    switch(retention::sweep(root,p,b"",ref mut work)) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(e.kind==kind && e.evicted==evicted && e.os_code==os_code && e.close_code==close_code); }
    };
}
@test @effects(filesystem) fn void reports_stale_busy_and_partial_mutation_without_claiming_success() {
    let root=opened();failed(ref root,1,3,0,11,0);assert(unsafe { mask() }==0);
    failed(ref root,2,4,0,0,0);assert(unsafe { mask() }==0);
    failed(ref root,3,3,1,0,4);assert(unsafe { mask() }==2);
    failed(ref root,4,3,1,5,0);assert(unsafe { mask() }==2);
    failed(ref root,5,3,0,0,0);assert(unsafe { mask() }==0);
    assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void validates_before_mutation_and_enforces_shared_work_budget() {
    let root=opened();unsafe { mode(0); }var usize work=0;
    assert(!retention::sweep(ref root,retention::Limits{.entries=0,.bytes=20,.record=16},b"",ref mut work).is_ok);
    assert(!retention::sweep(ref root,limits(),b"notes",ref mut work).is_ok && unsafe { counts() }==0);
    let protected=name(1);unsafe { mode(6); }
    switch(retention::sweep(ref root,limits(),bytes::view(ref protected),ref mut work)) { case .ok(_): { assert(false); } case .err(e): { assert(e.kind==7 && e.evicted==0 && unsafe { mask() }==0); } };
    unsafe { mode(0); }work=16777216;assert(!retention::sweep(ref root,limits(),b"",ref mut work).is_ok && unsafe { counts() }==0);
    work=0;let measured=swept(ref root,limits(),b"",ref mut work);let cost=work;assert(measured.evicted==1);
    unsafe { mode(0); }work=16777216-cost;let exact=swept(ref root,limits(),b"",ref mut work);assert(exact.evicted==1 && work==16777216);
    unsafe { mode(0); }work=16777216-cost+1;
    switch(retention::sweep(ref root,limits(),b"",ref mut work)) { case .ok(_): { assert(false); } case .err(e): { assert(e.kind==5 && e.evicted==0 && unsafe { mask() }==0); } };
    let two=retention::Limits{.entries=1,.bytes=8,.record=8};unsafe { mode(0); }work=0;
    let applied=swept(ref root,two,b"",ref mut work);let full_cost=work;assert(applied.evicted==2);
    unsafe { mode(0); }work=16777216-full_cost+1;
    switch(retention::sweep(ref root,two,b"",ref mut work)) { case .ok(_): { assert(false); }case .err(e): { assert(e.kind==5 && e.evicted==1 && unsafe { mask() }==2); } };
    assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void sorts_full_inventory_and_refuses_unprovable_protected_bounds() {
    let root=opened();unsafe { mode(11); }var usize work=0;
    let full=swept(ref root,retention::Limits{.entries=1,.bytes=8,.record=8},b"",ref mut work);
    assert(full.entries==1 && full.bytes==8 && full.evicted==4095 && unsafe { counts() }==4195);
    let protected=name(1);unsafe { mode(12); }work=0;
    switch(retention::sweep(ref root,retention::Limits{.entries=1,.bytes=8,.record=8},bytes::view(ref protected),ref mut work)) {
        case .ok(_): { assert(false); }case .err(e): { assert(e.kind==7 && e.evicted==0 && unsafe { mask() }==0); }
    };
    assert(rooted::close_directory(root).is_ok);
}
