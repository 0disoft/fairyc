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

@effects(filesystem) fn u8 readback(ref CStr path) {
    var file=switch(fs::open_read(path)) {
        case .ok(value): { value }
        case .err(_): { assert(false); return 0; }
    };
    var u8[1] data=[0;1];
    assert(fs::read_exact(ref mut file,ref mut data).is_ok);
    assert(fs::close(file).is_ok);
    return data[0];
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
    rejected(fs::rename_file(c"missing.bin",c"absent.bin"),1,2);
    rejected(fs::rename_file(c"",c"absent.bin"),2,0);
    store(c"source-λ.bin",b"A");
    store(c"destination.bin",b"B");
    rejected(fs::rename_file(c"source-λ.bin",c""),2,0);
    rejected(fs::rename_file(c"source-λ.bin",c"destination.bin"),1,183);
    assert(readback(c"source-λ.bin")==65 && readback(c"destination.bin")==66);
    assert(fs::create_dir(c"directory").is_ok);
    rejected(fs::rename_file(c"directory",c"new-directory"),2,0);
    assert(fs::remove_dir(c"directory").is_ok);
    var held=switch(fs::open_read(c"source-λ.bin")) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    rejected(fs::rename_file(c"source-λ.bin",c"renamed-λ.bin"),1,32);
    var u8[1] data=[0;1];
    assert(fs::read_exact(ref mut held,ref mut data).is_ok && data[0]==65);
    assert(fs::close(held).is_ok);
    assert(switch(fs::rename_file(c"source-λ.bin",c"renamed-λ.bin")) {
        case .ok(value): { value==0 }
        case .err(_): { false }
    });
    rejected(fs::rename_file(c"source-λ.bin",c"again.bin"),1,2);
    assert(readback(c"renamed-λ.bin")==65);
    assert(fs::remove_file(c"destination.bin").is_ok);
    assert(fs::remove_file(c"renamed-λ.bin").is_ok);
    return 0;
}
