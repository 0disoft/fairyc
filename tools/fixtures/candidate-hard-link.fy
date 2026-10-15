module main;
import std::fs as fs;

@effects(filesystem) fn void store(ref CStr path,ref u8[] data) {
    var file=switch(fs::create_new(path)) {
        case .ok(value): { value }
        case .err(_): { assert(false); return; }
    };
    assert(fs::write_all(ref mut file,data).is_ok);
    assert(fs::close(file).is_ok);
}

@effects(filesystem) fn void readback(ref CStr path,ref u8[] expected) {
    var file=switch(fs::open_read(path)) {
        case .ok(value): { value }
        case .err(_): { assert(false); return; }
    };
    var u8[2] data=[0;2];
    assert(switch(fs::read_exact(ref mut file,ref mut data[..expected.len])) {
        case .ok(count): { count==expected.len }
        case .err(_): { false }
    });
    var usize i=0;
    while(i<expected.len) { assert(data[i]==expected[i]); i+=1; }
    assert(switch(fs::size(ref file)) {
        case .ok(count): { count==expected.len }
        case .err(_): { false }
    });
    assert(fs::close(file).is_ok);
}

fn void rejected(Result<i32,fs::FileError> result,u32 status,u32 os) {
    switch(result) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==status && fs::error_os_code(e)==os && fs::error_completed(e)==0);
        }
    };
}

@effects(filesystem) fn i32 main() {
    rejected(fs::hard_link(c"missing.bin",c"missing-alias.bin"),1,2);
    rejected(fs::remove_file(c"missing-alias.bin"),1,2);
    rejected(fs::hard_link(c"",c"absent.bin"),2,0);
    store(c"source-λ.bin",b"A");
    store(c"blocked.bin",b"B");
    rejected(fs::hard_link(c"source-λ.bin",c""),2,0);
    rejected(fs::hard_link(c"source-λ.bin",c"blocked.bin"),1,183);
    readback(c"source-λ.bin",b"A");
    readback(c"blocked.bin",b"B");
    assert(switch(fs::hard_link(c"source-λ.bin",c"alias-λ.bin")) {
        case .ok(value): { value==0 }
        case .err(_): { false }
    });
    rejected(fs::hard_link(c"source-λ.bin",c"alias-λ.bin"),1,183);
    var file=switch(fs::open_append(c"source-λ.bin")) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    assert(fs::write_all(ref mut file,b"!").is_ok);
    assert(fs::close(file).is_ok);
    readback(c"alias-λ.bin",b"A!");
    assert(fs::remove_file(c"source-λ.bin").is_ok);
    rejected(fs::remove_file(c"source-λ.bin"),1,2);
    readback(c"alias-λ.bin",b"A!");
    assert(fs::remove_file(c"alias-λ.bin").is_ok);
    assert(fs::remove_file(c"blocked.bin").is_ok);
    return 0;
}
