module main;
import std::fs as fs;
import std::bytes as bytes;

fn void rejected(Result<usize,fs::FileError> result,u32 status,u32 os) {
    switch(result) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(fs::error_status(e)==status && fs::error_os_code(e)==os && fs::error_completed(e)==0); }
    };
}

@effects(filesystem) fn void store(ref CStr path,ref u8[] data,usize size) {
    var file=switch(fs::create_new(path)) { case .ok(v): { v } case .err(_): { assert(false); return; } };
    assert(fs::write_all(ref mut file,data).is_ok);
    assert(fs::set_len(ref mut file,size).is_ok);
    assert(fs::close(file).is_ok);
}

@effects(filesystem) fn void readback(ref CStr path,ref u8[] prefix,usize size) {
    var file=switch(fs::open_read(path)) { case .ok(v): { v } case .err(_): { assert(false); return; } };
    assert(switch(fs::size(ref file)) { case .ok(n): { n==size } case .err(_): { false } });
    var u8[256] data=[0;256];
    let u8[256] zero=[0;256];
    assert(fs::read_exact(ref mut file,ref mut data[..prefix.len]).is_ok);
    assert(bytes::equal(ref data[..prefix.len],prefix));
    var usize at=prefix.len;
    while(at<size) {
        let usize length=if(size-at>256) { 256 } else { size-at };
        assert(fs::read_exact(ref mut file,ref mut data[..length]).is_ok);
        assert(bytes::equal(ref data[..length],ref zero[..length]));
        at+=length;
    }
    assert(fs::close(file).is_ok);
}

@effects(filesystem) fn i32 main() {
    rejected(fs::copy_file_bytes(c"missing.bin",c"absent.bin"),1,2);
    rejected(fs::copy_file_bytes(c"",c"invalid.bin"),2,0);
    store(c"source-λ.bin",b"A\0λZ",65537);
    store(c"blocked.bin",b"KEEP",4);
    rejected(fs::copy_file_bytes(c"source-λ.bin",c""),2,0);
    rejected(fs::copy_file_bytes(c"source-λ.bin",c"blocked.bin"),1,80);
    readback(c"blocked.bin",b"KEEP",4);
    readback(c"source-λ.bin",b"A\0λZ",65537);
    var live=switch(fs::open_read(c"source-λ.bin")) { case .ok(v): { v } case .err(_): { return 1; } };
    rejected(fs::copy_file_bytes(c"source-λ.bin",c"busy.bin"),1,32);
    assert(switch(fs::size(ref live)) { case .ok(n): { n==65537 } case .err(_): { false } });
    assert(fs::close(live).is_ok);
    assert(switch(fs::copy_file_bytes(c"source-λ.bin",c"copy-λ.bin")) { case .ok(n): { n==65537 } case .err(_): { false } });
    rejected(fs::copy_file_bytes(c"source-λ.bin",c"copy-λ.bin"),1,80);
    readback(c"copy-λ.bin",b"A\0λZ",65537);
    var append=switch(fs::open_append(c"source-λ.bin")) { case .ok(v): { v } case .err(_): { return 1; } };
    assert(fs::write_all(ref mut append,b"!").is_ok);
    assert(fs::close(append).is_ok);
    assert(fs::remove_file(c"source-λ.bin").is_ok);
    readback(c"copy-λ.bin",b"A\0λZ",65537);
    store(c"empty.bin",b"",0);
    assert(switch(fs::copy_file_bytes(c"empty.bin",c"empty-copy.bin")) { case .ok(n): { n==0 } case .err(_): { false } });
    readback(c"empty-copy.bin",b"",0);
    assert(fs::remove_file(c"blocked.bin").is_ok);
    assert(fs::remove_file(c"copy-λ.bin").is_ok);
    assert(fs::remove_file(c"empty.bin").is_ok);
    assert(fs::remove_file(c"empty-copy.bin").is_ok);
    return 0;
}
