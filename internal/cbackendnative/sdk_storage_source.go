package cbackendnative

// Only the allocator control is foreign; all observations use public SDK calls.
const sdkStorageSource = `module main;
import std::alloc as alloc;
import std::bytes as bytes;
import std::text as text;

@link_name("fairyc_sdk_probe_fail_after")
unsafe extern C fn void fail_after(usize count) ;

fn usize size_once(ref mut usize calls) { *calls = *calls + 1; return 4; }

fn ref u8[] source_once(ref mut usize calls, ref u8[] source) { *calls = *calls + 1; return source; }

fn void check_arena(alloc::Arena input) {
    var alloc::Arena arena = input;
    var usize calls = 0;
    var usize copy_calls = 0;
    region scratch in ref mut arena {
        var u8[4] source = [65, 0, 255, 90];
        let i32 copy_code = switch (scratch.copy_bytes(source_once(ref mut copy_calls, ref source))) {
            case .ok(copied): {
                assert(copy_calls == 1 && bytes::equal(copied, b"A\0\xffZ"));
                source[0] = 66;
                copied[3] = 88;
                assert(copied[0] == 65 && source[3] == 90);
                unsafe { fail_after(0); }
                assert(switch (scratch.copy_bytes(copied)) {
                    case .ok(_): { false }
                    case .err(error): { alloc::error_status(error) == 1 }
                });
                unsafe { fail_after(1); }
                assert(switch (scratch.copy_bytes(copied)) {
                    case .ok(_): { false }
                    case .err(error): { alloc::error_status(error) == 1 }
                });
                assert(bytes::equal(copied, b"A\0\xffX"));
                assert(source[0] == 66 && source[3] == 90);
                let i32 recovery = switch (scratch.copy_bytes(copied[1..3])) {
                    case .ok(next): {
                        assert(bytes::equal(next, b"\0\xff"));
                        next[0] = 7;
                        assert(copied[1] == 0);
                        0
                    }
                    case .err(_): { 1 }
                };
                assert(recovery == 0);
                0
            }
            case .err(_): { 1 }
        };
        assert(copy_code == 0);
        unsafe { fail_after(0); }
        assert(switch (scratch.copy_bytes(b"")) {
            case .ok(empty): { empty.len == 0 && empty.is_empty }
            case .err(_): { false }
        });
        assert(switch (scratch.copy_bytes(b"x")) {
            case .ok(_): { false }
            case .err(error): { alloc::error_status(error) == 1 }
        });
        let i32 bytes_code = switch (scratch.alloc_zeroed_bytes(size_once(ref mut calls))) {
            case .ok(value): {
                assert(calls == 1 && value.len == 4 && value[0] == 0 && value[1] == 0 && value[2] == 0 && value[3] == 0);
                value[0] = 65;
                value[3] = 90;
                unsafe { fail_after(0); }
                assert(switch (scratch.alloc_zeroed_bytes(8)) {
                    case .ok(_): { false }
                    case .err(error): { alloc::error_status(error) == 1 }
                });
                unsafe { fail_after(1); }
                assert(switch (scratch.alloc_zeroed_bytes(8)) {
                    case .ok(_): { false }
                    case .err(error): { alloc::error_status(error) == 1 }
                });
                assert(value[0] == 65 && value[3] == 90 && value.len == 4);
                assert(switch (scratch.alloc_zeroed_bytes(2)) {
                    case .ok(next): { next.len == 2 && next[0] == 0 && next[1] == 0 }
                    case .err(_): { false }
                });
                0
            }
            case .err(_): { 1 }
        };
        assert(bytes_code == 0);
        unsafe { fail_after(0); }
        assert(switch (scratch.alloc_zeroed_bytes(0)) {
            case .ok(empty): { empty.is_empty && empty.len == 0 }
            case .err(_): { false }
        });
        // Empty allocation must not consume the armed allocation failure.
        assert(switch (scratch.alloc_zeroed_bytes(1)) {
            case .ok(_): { false }
            case .err(error): { alloc::error_status(error) == 1 }
        });
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
    assert(switch (bytes::resize_zeroed(ref mut value, 6)) {
        case .ok(n): { n == 6 } case .err(_): { false }
    });
    let usize grown_capacity = bytes::capacity(ref value);
    let ref mut u8[] writable = bytes::view_mut(ref mut value);
    assert(writable.len == 6 && writable[3] == 0 && writable[4] == 0 && writable[5] == 0);
    writable[4] = 88;
    writable[5] = 89;
    assert(bytes::equal(bytes::view(ref value), b"ABC\0XY"));
    assert(switch (bytes::truncate(ref mut value, 4)) {
        case .ok(n): { n == 4 } case .err(_): { false }
    });
    assert(bytes::capacity(ref value) == grown_capacity);
    assert(bytes::equal(bytes::view(ref value), b"ABC\0"));
    assert(switch (bytes::resize_zeroed(ref mut value, 6)) {
        case .ok(n): { n == 6 } case .err(_): { false }
    });
    assert(bytes::capacity(ref value) == grown_capacity);
    assert(bytes::equal(bytes::view(ref value), b"ABC\0\0\0"));
    assert(switch (bytes::clear(ref mut value)) {
        case .ok(n): { n == 0 } case .err(_): { false }
    });
    assert(bytes::is_empty(ref value));
}

fn void check_empty_buffer() {
    var bytes::Buffer value = bytes::empty();
    assert(bytes::is_empty(ref value) && bytes::len(ref value) == 0);
    assert(bytes::capacity(ref value) == 0 && bytes::view(ref value).is_empty);
    let ref mut u8[] writable = bytes::view_mut(ref mut value);
    assert(writable.is_empty && writable.len == 0);
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

fn void check_text_round_trip(bytes::Buffer input, ref u8[] expected) {
    let usize original_capacity = bytes::capacity(ref input);
    let i32 code = switch (text::from_buffer(input)) {
        case .ok(value): {
            assert(text::string_len(ref value) == expected.len);
            assert(text::string_is_empty(ref value) == expected.is_empty);
            assert(bytes::equal(text::as_bytes(text::string_as_str(ref value)), expected));
            let bytes::Buffer returned = text::into_buffer(value);
            assert(bytes::capacity(ref returned) == original_capacity);
            assert(bytes::equal(bytes::view(ref returned), expected));
            0
        }
        case .err(_): { 1 }
    };
    assert(code == 0);
}

fn void check_text_edges() {
    assert(switch (text::copy("")) {
        case .ok(value): {
            text::string_is_empty(ref value) && text::string_len(ref value) == 0 &&
                text::as_bytes(text::string_as_str(ref value)).is_empty
        }
        case .err(_): { false }
    });
    assert(switch (text::cstring_from_text("")) {
        case .ok(value): {
            text::cstring_is_empty(ref value) && text::cstring_len(ref value) == 0 &&
                consume_cstr(text::cstring_as_cstr(ref value)) == 0
        }
        case .err(_): { false }
    });
    let ref u8[] unicode = b"\xed\x95\x9c\xea\xb8\x80\xce\xbb";
    let i32 unicode_code = switch (text::validate_utf8(unicode)) {
        case .ok(view): {
            assert(view.len == 8 && bytes::equal(text::as_bytes(view), unicode));
            switch (text::copy(view)) {
                case .ok(value): {
                    assert(text::string_len(ref value) == 8);
                    assert(bytes::equal(text::as_bytes(text::string_as_str(ref value)), unicode));
                    0
                }
                case .err(_): { 2 }
            }
        }
        case .err(_): { 3 }
    };
    assert(unicode_code == 0);
    let i32 round_trip_code = switch (bytes::copy(unicode)) {
        case .ok(value): { check_text_round_trip(value, unicode); 0 }
        case .err(_): { 4 }
    };
    assert(round_trip_code == 0);
    check_text_round_trip(bytes::empty(), b"");
}

fn i32 main() {
    check_empty_buffer();
    check_text_edges();
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
