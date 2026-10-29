module tests::linux_workspace_fs;
import linux_workspace_fs as fs;
import std::bytes as bytes;
@link_name("workspace_probe_reset") unsafe extern C fn void reset(u32 mode);
@link_name("workspace_probe_closes") unsafe extern C fn u32 closes();

@test @effects(filesystem) fn void owns_snapshots_and_read_handles() {
    unsafe { reset(0); }
    var listing=switch(fs::read_dir_utf8("src")) { case .ok(p):{p} case .err(_):{assert(false);return;} };
    var u8[2] small=[0;2]; assert(!fs::next_dir(ref mut listing,ref mut small).is_ok);
    var u8[256] name=[0;256];
    let row=switch(fs::next_dir(ref mut listing,ref mut name)) { case .ok(p):{p} case .err(_):{assert(false);return;} };
    let ref u8[] data=ref name;
    assert(row.has_entry && row.name_length==7 && bytes::equal(data[..7],b"main.fy") && row.size==3 && !row.is_reparse && fs::path_kind_is_file(row.kind));
    let eof=switch(fs::next_dir(ref mut listing,ref mut name)) { case .ok(p):{p} case .err(_):{assert(false);return;} };
    assert(!eof.has_entry); assert(fs::close_dir(listing).is_ok);
    var file=switch(fs::open_read_utf8("src/main.fy")) { case .ok(p):{p} case .err(_):{assert(false);return;} };
    let kind=switch(fs::kind(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::kind_is_disk(kind));
    let size=switch(fs::size(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(size==3);
    let a=switch(fs::file_id(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let b=switch(fs::file_id(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::file_id_equal(ref a,ref b));
    var u8[4] output=[0;4];let count=switch(fs::read(ref mut file,ref mut output)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ref u8[] view=ref output;assert(count==3 && bytes::equal(view[..3],b"abc"));
    assert(fs::close(file).is_ok && unsafe { closes() }==1);
}
@test @effects(filesystem) fn void rejects_link_prefixes_and_preserves_close_failure() {
    unsafe { reset(1); }
    switch(fs::read_dir_utf8("src")){case .ok(_):{assert(false);}case .err(e):{assert(e.os_code==40);}};
    unsafe { reset(2); }
    var file=switch(fs::open_read_utf8("src/main.fy")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(fs::close(file)){case .ok(_):{assert(false);}case .err(e):{assert(e.os_code==5);}};
    assert(unsafe { closes() }==1);
    assert(!fs::read_dir_utf8("../src").is_ok && !fs::open_read_utf8("src/..").is_ok);
}
