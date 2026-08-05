#include <stddef.h>
#include <stdint.h>
#include <string.h>
#include <windows.h>

enum fy_std_windows_file_status_v0 {
    FY_STD_WINDOWS_FILE_OK_V0 = 0,
    FY_STD_WINDOWS_FILE_IO_ERROR_V0 = 1,
    FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 = 2,
    FY_STD_WINDOWS_FILE_CLOSED_V0 = 3,
    FY_STD_WINDOWS_FILE_NO_PROGRESS_V0 = 4,
    FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 = 5,
    FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0 = 6,
    FY_STD_WINDOWS_FILE_UNEXPECTED_EOF_V0 = 7
};

struct fy_std_windows_file_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t count;
};

struct fy_std_windows_file_position_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint64_t position;
};

struct fy_std_windows_file_size_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint64_t size;
};

struct fy_std_windows_file_kind_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint32_t kind;
};

struct fy_std_windows_file_v0 {
    void *handle;
    uint8_t open;
};

struct fy_std_windows_open_options_v0 {
    uint8_t read;
    uint8_t write;
    uint8_t create;
    uint8_t truncate;
    uint8_t exclusive;
    uint8_t append;
};

struct fy_std_windows_file_result_v0 fy_std_windows_file_open_v0(
    const uint8_t *,
    const struct fy_std_windows_open_options_v0 *,
    void *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_read_v0(void *, uint8_t *, size_t);
struct fy_std_windows_file_result_v0 fy_std_windows_file_read_exact_v0(
    void *,
    uint8_t *,
    size_t
);
struct fy_std_windows_file_position_result_v0 fy_std_windows_file_position_v0(void *);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_size_v0(const void *);
struct fy_std_windows_file_kind_result_v0 fy_std_windows_file_kind_v0(const void *);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_set_len_v0(void *, uint64_t);
struct fy_std_windows_file_position_result_v0 fy_std_windows_file_seek_start_v0(
    void *,
    uint64_t
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_write_all_v0(
    void *,
    const uint8_t *,
    size_t
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_sync_v0(void *);
struct fy_std_windows_file_result_v0 fy_std_windows_file_close_v0(void *);
uint32_t fy_std_windows_file_drop_v0(void *);

static int fy_std_windows_file_closed_v0(const struct fy_std_windows_file_v0 *file) {
    return file->handle == NULL && file->open == UINT8_C(0);
}

int main(void) {
    static const uint8_t path[] = "fairyc-windows-file-probe.tmp";
    static const uint8_t payload[] = {
        UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
        UINT8_C(70), UINT8_C(105), UINT8_C(108), UINT8_C(101)
    };
    struct fy_std_windows_open_options_v0 write_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 exclusive_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(1), UINT8_C(0)
    };
    struct fy_std_windows_open_options_v0 append_options = {
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(1)
    };
    static const uint8_t replacement[] = {
        UINT8_C(67), UINT8_C(111), UINT8_C(100), UINT8_C(101)
    };
    static const uint8_t final_payload[] = {
        UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
        UINT8_C(67), UINT8_C(111), UINT8_C(100), UINT8_C(101)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result = fy_std_windows_file_open_v0(
        path,
        &write_options,
        &file
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || file.open != UINT8_C(1)) return 1;
    result = fy_std_windows_file_write_all_v0(&file, payload, sizeof payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload) return 2;
    struct fy_std_windows_file_size_result_v0 size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 31;
    struct fy_std_windows_file_kind_result_v0 kind = fy_std_windows_file_kind_v0(&file);
    if (kind.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS ||
        kind.kind != (uint32_t)FILE_TYPE_DISK) return 71;
    struct fy_std_windows_file_position_result_v0 position =
        fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 18;
    size = fy_std_windows_file_set_len_v0(&file, UINT64_MAX);
    if (size.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        size.os_code != (uint32_t)ERROR_SUCCESS || size.size != UINT64_C(0)) return 36;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 37;
    size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 38;
    size = fy_std_windows_file_set_len_v0(&file, UINT64_C(5));
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 || size.size != UINT64_C(5)) return 39;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 40;
    size = fy_std_windows_file_set_len_v0(&file, (uint64_t)sizeof payload);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 41;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 42;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(5));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(5)) return 43;
    result = fy_std_windows_file_write_all_v0(&file, replacement, sizeof replacement);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof replacement) return 44;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_MAX);
    if (position.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        position.os_code != (uint32_t)ERROR_SUCCESS || position.position != UINT64_C(0)) return 19;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 20;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(5));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(5)) return 21;
    size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 32;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(5)) return 22;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(0));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 23;
    if (fy_std_windows_file_sync_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 3;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0 ||
        !fy_std_windows_file_closed_v0(&file)) return 4;
    if (fy_std_windows_file_position_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 24;
    if (fy_std_windows_file_size_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 33;
    if (fy_std_windows_file_kind_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 72;
    if (fy_std_windows_file_set_len_v0(&file, UINT64_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 45;
    if (fy_std_windows_file_seek_start_v0(&file, UINT64_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 25;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 5;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 6;
    size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 34;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 26;
    size = fy_std_windows_file_set_len_v0(&file, (uint64_t)sizeof payload);
    if (size.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        size.os_code != (uint32_t)ERROR_ACCESS_DENIED || size.size != UINT64_C(0)) return 46;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 47;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(5));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(5)) return 27;
    uint8_t tail[4] = {0};
    result = fy_std_windows_file_read_v0(&file, tail, sizeof tail);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof tail ||
        memcmp(tail, replacement, sizeof tail) != 0) return 28;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof payload) return 29;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(0));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 30;
    uint8_t received[sizeof payload] = {0};
    size_t total = 0;
    while (total < sizeof received) {
        size_t requested = sizeof received - total;
        if (requested > 3) requested = 3;
        result = fy_std_windows_file_read_v0(&file, received + total, requested);
        if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 7;
        if (result.count == 0) break;
        total += result.count;
    }
    result = fy_std_windows_file_read_v0(&file, received, sizeof received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 0) return 8;
    if (total != sizeof final_payload ||
        memcmp(received, final_payload, sizeof final_payload) != 0) return 9;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(0));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 77;
    uint8_t exact[sizeof final_payload] = {0};
    result = fy_std_windows_file_read_exact_v0(&file, exact, sizeof exact);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof exact ||
        memcmp(exact, final_payload, sizeof exact) != 0) return 78;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(0));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 79;
    uint8_t oversized[12] = {
        UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5),
        UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5),
        UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5), UINT8_C(0xa5)
    };
    result = fy_std_windows_file_read_exact_v0(&file, oversized, sizeof oversized);
    if (result.status != FY_STD_WINDOWS_FILE_UNEXPECTED_EOF_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS ||
        result.count != sizeof final_payload ||
        memcmp(oversized, final_payload, sizeof final_payload) != 0 ||
        oversized[9] != UINT8_C(0xa5) || oversized[10] != UINT8_C(0xa5) ||
        oversized[11] != UINT8_C(0xa5)) return 80;
    result = fy_std_windows_file_read_exact_v0(&file, NULL, 0);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 0) return 81;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        position.position != (uint64_t)sizeof final_payload) return 82;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 10;
    if (fy_std_windows_file_read_exact_v0(&file, NULL, 0).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 83;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 11;
    HANDLE stolen = (HANDLE)file.handle;
    if (!CloseHandle(stolen)) return 12;
    size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        size.os_code != (uint32_t)ERROR_INVALID_HANDLE || size.size != UINT64_C(0)) return 35;
    kind = fy_std_windows_file_kind_v0(&file);
    if (kind.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        kind.os_code != (uint32_t)ERROR_INVALID_HANDLE ||
        kind.kind != (uint32_t)FILE_TYPE_UNKNOWN) return 73;
    uint8_t invalid_read = UINT8_C(0xa5);
    result = fy_std_windows_file_read_exact_v0(&file, &invalid_read, 1);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_INVALID_HANDLE || result.count != 0 ||
        invalid_read != UINT8_C(0xa5)) return 84;
    size = fy_std_windows_file_set_len_v0(&file, UINT64_C(0));
    if (size.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        size.os_code != (uint32_t)ERROR_INVALID_HANDLE || size.size != UINT64_C(0)) return 48;
    result = fy_std_windows_file_close_v0(&file);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_INVALID_HANDLE ||
        !fy_std_windows_file_closed_v0(&file)) return 13;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 14;
    if (fy_std_windows_file_drop_v0(&file) != UINT32_C(0) ||
        !fy_std_windows_file_closed_v0(&file)) return 15;
    if (fy_std_windows_file_drop_v0(&file) != UINT32_C(0)) return 16;
    if (!DeleteFileW(L"fairyc-windows-file-probe.tmp")) return 17;

    static const uint8_t exclusive_path[] = "fairyc-windows-file-create-new-probe.tmp";
    static const uint8_t exclusive_payload[] = {
        UINT8_C(78), UINT8_C(101), UINT8_C(119)
    };
    result = fy_std_windows_file_open_v0(exclusive_path, &exclusive_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.os_code != ERROR_SUCCESS ||
        result.count != 0 || file.open != UINT8_C(1)) return 49;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 50;
    result = fy_std_windows_file_write_all_v0(&file, exclusive_payload, sizeof exclusive_payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof exclusive_payload) return 51;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 52;
    result = fy_std_windows_file_open_v0(exclusive_path, &exclusive_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_EXISTS || result.count != 0 ||
        !fy_std_windows_file_closed_v0(&file)) return 53;
    result = fy_std_windows_file_open_v0(exclusive_path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 54;
    uint8_t exclusive_received[sizeof exclusive_payload] = {0};
    result = fy_std_windows_file_read_v0(&file, exclusive_received, sizeof exclusive_received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof exclusive_payload ||
        memcmp(exclusive_received, exclusive_payload, sizeof exclusive_payload) != 0) return 55;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 56;
    if (!DeleteFileW(L"fairyc-windows-file-create-new-probe.tmp")) return 57;

    static const uint8_t append_path[] = "fairyc-windows-file-append-probe.tmp";
    static const uint8_t append_first[] = {
        UINT8_C(79), UINT8_C(110), UINT8_C(101)
    };
    static const uint8_t append_second[] = {
        UINT8_C(84), UINT8_C(119), UINT8_C(111)
    };
    static const uint8_t append_expected[] = {
        UINT8_C(79), UINT8_C(110), UINT8_C(101),
        UINT8_C(84), UINT8_C(119), UINT8_C(111), UINT8_C(33)
    };
    result = fy_std_windows_file_open_v0(append_path, &append_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || file.open != UINT8_C(1)) return 58;
    result = fy_std_windows_file_write_all_v0(&file, append_first, sizeof append_first);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof append_first) return 59;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(0));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(0)) return 60;
    result = fy_std_windows_file_write_all_v0(&file, append_second, sizeof append_second);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof append_second) return 61;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 62;
    result = fy_std_windows_file_open_v0(append_path, &append_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 63;
    position = fy_std_windows_file_seek_start_v0(&file, UINT64_C(1));
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != UINT64_C(1)) return 64;
    static const uint8_t exclamation[] = {UINT8_C(33)};
    result = fy_std_windows_file_write_all_v0(&file, exclamation, sizeof exclamation);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != 1) return 65;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 66;
    result = fy_std_windows_file_open_v0(append_path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 67;
    uint8_t append_received[sizeof append_expected] = {0};
    result = fy_std_windows_file_read_v0(&file, append_received, sizeof append_received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof append_expected ||
        memcmp(append_received, append_expected, sizeof append_expected) != 0) return 68;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 69;
    if (!DeleteFileW(L"fairyc-windows-file-append-probe.tmp")) return 70;

    static const uint8_t character_path[] = "NUL";
    result = fy_std_windows_file_open_v0(character_path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 74;
    kind = fy_std_windows_file_kind_v0(&file);
    if (kind.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS ||
        kind.kind != (uint32_t)FILE_TYPE_CHAR) return 75;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 76;
    return 0;
}
