package cbackendnative

// Only the allocator control is foreign; all observations use public SDK calls.
const sdkStorageSource = `module main;
import std::alloc as alloc;
import std::bytes as bytes;
import std::text as text;

@link_name("fairyc_sdk_probe_fail_after")
unsafe extern C fn void fail_after(usize count) ;

fn void check_arena(alloc::Arena input) {
    var alloc::Arena arena = input;
    region scratch in ref mut arena {
        assert(switch (scratch.place(42)) {
            case .ok(value): { *value == 42 }
            case .err(_): { false }
        });
        unsafe { fail_after(0); }
        assert(switch (scratch.place(7)) {
            case .ok(_): { false }
            case .err(error): { alloc::error_status(error) == 1 }
        });
        unsafe { fail_after(1); }
        assert(switch (scratch.place(8)) {
            case .ok(_): { false }
            case .err(error): { alloc::error_status(error) == 1 }
        });
        assert(switch (scratch.place(9)) {
            case .ok(value): { *value == 9 }
            case .err(_): { false }
        });
    }
    region reused in ref mut arena {
        assert(switch (reused.place(11)) {
            case .ok(value): { *value == 11 }
            case .err(_): { false }
        });
    }
}

fn void check_buffer(bytes::Buffer input) {
    var bytes::Buffer value = input;
    assert(switch (bytes::append_byte(ref mut value, 65)) {
        case .ok(n): { n == 1 } case .err(_): { false }
    });
    let usize old_capacity = bytes::capacity(ref value);
    unsafe { fail_after(0); }
    assert(switch (bytes::reserve(ref mut value, old_capacity + 16)) {
        case .ok(_): { false } case .err(error): { bytes::error_status(error) == 1 }
    });
    assert(bytes::len(ref value) == 1 && bytes::capacity(ref value) == old_capacity);
    assert(bytes::view(ref value)[0] == 65);
    assert(switch (bytes::append(ref mut value, b"BC")) {
        case .ok(n): { n == 3 } case .err(_): { false }
    });
    assert(bytes::equal(bytes::view(ref value), b"ABC"));
    assert(switch (bytes::clear(ref mut value)) {
        case .ok(n): { n == 0 } case .err(_): { false }
    });
    assert(bytes::is_empty(ref value));
}

fn void check_string(text::String input) {
    var text::String value = input;
    unsafe { fail_after(0); }
    assert(switch (text::append(ref mut value, " C")) {
        case .ok(_): { false }
        case .err(error): {
            text::error_status(error) == 1 && text::error_allocation_status(error) == 1
        }
    });
    assert(bytes::equal(text::as_bytes(text::string_as_str(ref value)), b"Fairy"));
    assert(switch (text::append(ref mut value, " C")) {
        case .ok(n): { n == 7 } case .err(_): { false }
    });
    let bytes::Buffer returned = text::into_buffer(value);
    assert(bytes::equal(bytes::view(ref returned), b"Fairy C"));
}

fn i32 consume_cstr(ref CStr value) { return 0; }

fn void check_cstring(text::CString value) {
    let ref CStr view = text::cstring_as_cstr(ref value);
    assert(consume_cstr(view) == 0 && text::cstring_len(ref value) == 5);
    assert(!text::cstring_is_empty(ref value));
}

fn void check_invalid_buffer(bytes::Buffer input) {
    assert(switch (text::from_buffer(input)) {
        case .ok(_): { false }
        case .err(returned): {
            bytes::len(ref returned) == 1 && bytes::capacity(ref returned) == 1 &&
                bytes::view(ref returned)[0] == 255
        }
    });
}

fn i32 main() {
    let i32 arena_code = switch (alloc::create()) {
        case .ok(value): { check_arena(value); 0 } case .err(_): { 1 }
    };
    assert(arena_code == 0);
    let i32 buffer_code = switch (bytes::with_capacity(1)) {
        case .ok(value): { check_buffer(value); 0 } case .err(_): { 2 }
    };
    assert(buffer_code == 0);
    let i32 string_code = switch (text::copy("Fairy")) {
        case .ok(value): { check_string(value); 0 } case .err(_): { 3 }
    };
    assert(string_code == 0);
    unsafe { fail_after(0); }
    assert(switch (text::copy("failed")) {
        case .ok(_): { false }
        case .err(error): {
            text::error_status(error) == 1 && text::error_allocation_status(error) == 1
        }
    });
    unsafe { fail_after(0); }
    assert(switch (text::cstring_from_text("failed")) {
        case .ok(_): { false }
        case .err(error): {
            text::error_status(error) == 1 && text::error_allocation_status(error) == 1
        }
    });
    let i32 cstring_code = switch (text::cstring_from_text("Fairy")) {
        case .ok(value): { check_cstring(value); 0 } case .err(_): { 4 }
    };
    assert(cstring_code == 0);
    assert(switch (text::validate_utf8(b"\xff")) {
        case .ok(_): { false }
        case .err(error): { text::error_status(error) == 2 }
    });
    assert(switch (text::cstring_from_text("a\0b")) {
        case .ok(_): { false }
        case .err(error): { text::error_status(error) == 3 }
    });
    let i32 invalid_code = switch (bytes::copy(b"\xff")) {
        case .ok(value): { check_invalid_buffer(value); 0 } case .err(_): { 5 }
    };
    assert(invalid_code == 0);
    return 42;
}
`
