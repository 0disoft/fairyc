package cbackendnative

const linuxFileSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_file_result_v0 {
    u32 status;
    i32 error_number;
    c::size_t count;
}
@repr(C)
struct fy_std_linux_file_v0 {
    c::int descriptor;
    u8 open;
}
@repr(C)
struct fy_std_linux_open_options_v0 {
    u8 read;
    u8 write;
    u8 create;
    u8 truncate;
    u32 mode;
}

@link_name("fairyc_fixture_file_prepare_path")
unsafe extern C fn u32 prepare_path() ;
@link_name("fairyc_fixture_file_path")
unsafe extern C fn ptr u8 file_path() ;
@link_name("fairyc_fixture_file_path_storage_length")
unsafe extern C fn c::size_t file_path_storage_length() ;
@link_name("fairyc_fixture_file_payload")
unsafe extern C fn ptr u8 file_payload() ;
@link_name("fairyc_fixture_file_payload_length")
unsafe extern C fn c::size_t file_payload_length() ;
@link_name("fairyc_fixture_file_three_size_t")
unsafe extern C fn c::size_t three_size_t() ;
@link_name("fairyc_fixture_file_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_file_closed_descriptor")
unsafe extern C fn c::int closed_descriptor() ;
@link_name("fairyc_fixture_file_read_segment_one")
unsafe extern C fn ptr mut u8 read_segment_one() ;
@link_name("fairyc_fixture_file_read_segment_two")
unsafe extern C fn ptr mut u8 read_segment_two() ;
@link_name("fairyc_fixture_file_read_segment_three")
unsafe extern C fn ptr mut u8 read_segment_three() ;

@link_name("fy_std_linux_file_open_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_open(ptr u8 path, c::size_t path_storage_length, ptr fy_std_linux_open_options_v0 options, ptr mut fy_std_linux_file_v0 output) ;
@link_name("fy_std_linux_file_read_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_read(ptr mut fy_std_linux_file_v0 file, ptr mut u8 buffer, c::size_t length) ;
@link_name("fy_std_linux_file_write_all_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_write_all(ptr mut fy_std_linux_file_v0 file, ptr u8 buffer, c::size_t length) ;
@link_name("fy_std_linux_file_sync_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_sync(ptr mut fy_std_linux_file_v0 file) ;
@link_name("fy_std_linux_file_close_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_close(ptr mut fy_std_linux_file_v0 file) ;

@link_name("fairyc_fixture_file_result_matches")
unsafe extern C fn u32 result_matches(ptr fy_std_file_result_v0 result, u32 status, i32 error_number, c::size_t count) ;
@link_name("fairyc_fixture_file_is_closed")
unsafe extern C fn u32 file_is_closed(ptr fy_std_linux_file_v0 file) ;
@link_name("fairyc_fixture_file_buffers_match_and_cleanup")
unsafe extern C fn u32 buffers_match_and_cleanup() ;

fn i32 main() {
    let u32 prepared = unsafe { prepare_path() };
    let ptr u8 path = unsafe { file_path() };
    let c::size_t path_length = unsafe { file_path_storage_length() };
    let ptr u8 payload = unsafe { file_payload() };
    let c::size_t payload_length = unsafe { file_payload_length() };
    let c::size_t three = unsafe { three_size_t() };
    let c::size_t zero = unsafe { zero_size_t() };
    let c::int closed = unsafe { closed_descriptor() };

    var fy_std_linux_file_v0 file = fy_std_linux_file_v0 {
        .descriptor = closed,
        .open = 0,
    };
    let fy_std_linux_open_options_v0 write_options = fy_std_linux_open_options_v0 {
        .read = 0,
        .write = 1,
        .create = 0,
        .truncate = 1,
        .mode = 384,
    };
    let fy_std_file_result_v0 opened_for_write = unsafe {
        linux_file_open(
            path,
            path_length,
            ref write_options as ptr fy_std_linux_open_options_v0,
            ref mut file as ptr mut fy_std_linux_file_v0
        )
    };
    let u32 opened_for_write_ok = unsafe {
        result_matches(ref opened_for_write as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 written = unsafe {
        linux_file_write_all(ref mut file as ptr mut fy_std_linux_file_v0, payload, payload_length)
    };
    let u32 written_ok = unsafe {
        result_matches(ref written as ptr fy_std_file_result_v0, 0, 0, payload_length)
    };
    let fy_std_file_result_v0 synced = unsafe {
        linux_file_sync(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 synced_ok = unsafe {
        result_matches(ref synced as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 closed_after_write = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_after_write_ok = unsafe {
        result_matches(ref closed_after_write as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let u32 closed_state_after_write = unsafe {
        file_is_closed(ref file as ptr fy_std_linux_file_v0)
    };
    let fy_std_file_result_v0 closed_twice = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_twice_ok = unsafe {
        result_matches(ref closed_twice as ptr fy_std_file_result_v0, 3, 0, zero)
    };

    let fy_std_linux_open_options_v0 read_options = fy_std_linux_open_options_v0 {
        .read = 1,
        .write = 0,
        .create = 0,
        .truncate = 0,
        .mode = 0,
    };
    let fy_std_file_result_v0 opened_for_read = unsafe {
        linux_file_open(
            path,
            path_length,
            ref read_options as ptr fy_std_linux_open_options_v0,
            ref mut file as ptr mut fy_std_linux_file_v0
        )
    };
    let u32 opened_for_read_ok = unsafe {
        result_matches(ref opened_for_read as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let ptr mut u8 first_buffer = unsafe { read_segment_one() };
    let ptr mut u8 second_buffer = unsafe { read_segment_two() };
    let ptr mut u8 third_buffer = unsafe { read_segment_three() };
    let fy_std_file_result_v0 read_first = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, first_buffer, three)
    };
    let u32 read_first_ok = unsafe {
        result_matches(ref read_first as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_second = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, second_buffer, three)
    };
    let u32 read_second_ok = unsafe {
        result_matches(ref read_second as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_third = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, third_buffer, three)
    };
    let u32 read_third_ok = unsafe {
        result_matches(ref read_third as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_eof = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, first_buffer, three)
    };
    let u32 read_eof_ok = unsafe {
        result_matches(ref read_eof as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 closed_after_read = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_after_read_ok = unsafe {
        result_matches(ref closed_after_read as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let u32 closed_state_after_read = unsafe {
        file_is_closed(ref file as ptr fy_std_linux_file_v0)
    };
    let u32 observed = unsafe { buffers_match_and_cleanup() };

    return if (prepared == 1 && opened_for_write_ok == 1 && written_ok == 1 && synced_ok == 1 &&
        closed_after_write_ok == 1 && closed_state_after_write == 1 && closed_twice_ok == 1 &&
        opened_for_read_ok == 1 && read_first_ok == 1 && read_second_ok == 1 &&
        read_third_ok == 1 && read_eof_ok == 1 && closed_after_read_ok == 1 &&
        closed_state_after_read == 1 && observed == 1) {
        42
    } else {
        0
    };
}
`

const linuxFileSourceCallBridge = `
#include <stdlib.h>

static char fairyc_fixture_file_path_storage[] = "/tmp/fairyc-file-source-XXXXXX";
static const uint8_t fairyc_fixture_file_payload_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
    UINT8_C(70), UINT8_C(105), UINT8_C(108), UINT8_C(101)
};
static uint8_t fairyc_fixture_file_read_bytes[9];

uint32_t fairyc_fixture_file_prepare_path(void) {
    int descriptor = mkstemp(fairyc_fixture_file_path_storage);
    if (descriptor < 0) {
        return UINT32_C(0);
    }
    if (close(descriptor) != 0) {
        (void)unlink(fairyc_fixture_file_path_storage);
        return UINT32_C(0);
    }
    return UINT32_C(1);
}

const uint8_t *fairyc_fixture_file_path(void) {
    return (const uint8_t *)fairyc_fixture_file_path_storage;
}

size_t fairyc_fixture_file_path_storage_length(void) {
    return sizeof(fairyc_fixture_file_path_storage);
}

const uint8_t *fairyc_fixture_file_payload(void) {
    return fairyc_fixture_file_payload_bytes;
}

size_t fairyc_fixture_file_payload_length(void) {
    return sizeof(fairyc_fixture_file_payload_bytes);
}

size_t fairyc_fixture_file_three_size_t(void) { return (size_t)3; }
size_t fairyc_fixture_file_zero_size_t(void) { return (size_t)0; }
int fairyc_fixture_file_closed_descriptor(void) { return -1; }
uint8_t *fairyc_fixture_file_read_segment_one(void) { return fairyc_fixture_file_read_bytes; }
uint8_t *fairyc_fixture_file_read_segment_two(void) { return fairyc_fixture_file_read_bytes + 3; }
uint8_t *fairyc_fixture_file_read_segment_three(void) { return fairyc_fixture_file_read_bytes + 6; }

uint32_t fairyc_fixture_file_result_matches(
    const struct fy_std_file_result_v0 *result,
    uint32_t status,
    int32_t error_number,
    size_t count
) {
    return result != NULL && result->status == status && result->error_number == error_number &&
           result->count == count ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_file_is_closed(const struct fy_std_linux_file_v0 *file) {
    return file != NULL && file->descriptor == -1 && file->open == UINT8_C(0) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_file_buffers_match_and_cleanup(void) {
    uint32_t matches = UINT32_C(1);
    for (size_t index = 0; index < sizeof(fairyc_fixture_file_payload_bytes); ++index) {
        if (fairyc_fixture_file_read_bytes[index] != fairyc_fixture_file_payload_bytes[index]) {
            matches = UINT32_C(0);
        }
    }
    if (unlink(fairyc_fixture_file_path_storage) != 0) {
        return UINT32_C(0);
    }
    return matches;
}
`

var linuxFileSourceCallExpectedCFragments = []string{
	"struct fy_std_file_result_v0 {",
	"uint32_t status;",
	"int32_t error_number;",
	"size_t count;",
	"_Static_assert(sizeof(struct fy_std_file_result_v0) == UINT64_C(16)",
	"struct fy_std_linux_file_v0 {",
	"int descriptor;",
	"uint8_t open;",
	"struct fy_std_linux_open_options_v0 {",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_open_v0(const uint8_t * fy_a_0, size_t fy_a_1, const struct fy_std_linux_open_options_v0 * fy_a_2, struct fy_std_linux_file_v0 * fy_a_3);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_read_v0(struct fy_std_linux_file_v0 * fy_a_0, uint8_t * fy_a_1, size_t fy_a_2);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_write_all_v0(struct fy_std_linux_file_v0 * fy_a_0, const uint8_t * fy_a_1, size_t fy_a_2);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_sync_v0(struct fy_std_linux_file_v0 * fy_a_0);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_close_v0(struct fy_std_linux_file_v0 * fy_a_0);",
	" = fy_std_linux_file_open_v0(",
	" = fy_std_linux_file_read_v0(",
	" = fy_std_linux_file_write_all_v0(",
	" = fy_std_linux_file_sync_v0(",
	" = fy_std_linux_file_close_v0(",
}
