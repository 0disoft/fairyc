module main;
import std::fs as fs;

@effects(filesystem) fn i32 main() {
    assert(switch(fs::create_dir_all(c"nested-λ\\child\\leaf")) {
        case .ok(value): { value==0 }
        case .err(_): { false }
    });
    assert(fs::create_dir_all(c"nested-λ\\child\\leaf").is_ok);
    assert(fs::remove_dir(c"nested-λ\\child\\leaf").is_ok);
    assert(fs::remove_dir(c"nested-λ\\child").is_ok);
    assert(fs::remove_dir(c"nested-λ").is_ok);

    switch(fs::create_dir_all(c"")) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==2 && fs::error_os_code(e)==0 && fs::error_completed(e)==0);
        }
    };
    switch(fs::create_dir_all(c"blocked\\..\\escape")) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==2 && fs::error_os_code(e)==0 && fs::error_completed(e)==0);
        }
    };
    assert(!fs::remove_dir(c"blocked").is_ok);
    assert(!fs::remove_dir(c"escape").is_ok);

    switch(fs::create_dir_all(c"partial\\created\\bad?name")) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==1 && fs::error_os_code(e)==123 && fs::error_completed(e)==0);
        }
    };
    assert(fs::remove_dir(c"partial\\created").is_ok);
    assert(fs::remove_dir(c"partial").is_ok);

    let file=switch(fs::create_new(c"file.bin")) {
        case .ok(value): { value }
        case .err(_): { return 1; }
    };
    assert(fs::close(file).is_ok);
    switch(fs::create_dir_all(c"file.bin")) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==1 && fs::error_os_code(e)==183 && fs::error_completed(e)==0);
        }
    };
    switch(fs::create_dir_all(c"file.bin\\child")) {
        case .ok(_): { assert(false); }
        case .err(e): {
            assert(fs::error_status(e)==1 && fs::error_os_code(e)==183 && fs::error_completed(e)==0);
        }
    };
    assert(fs::remove_file(c"file.bin").is_ok);
    return 42;
}
