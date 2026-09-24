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

struct fy_std_windows_file_attribute_result_v0 {
    uint32_t status;
    uint32_t os_code;
    uint32_t enabled;
};

struct fy_std_windows_file_id_v0 {
    uint32_t volume_low;
    uint32_t volume_high;
    uint32_t file_low_0;
    uint32_t file_low_1;
    uint32_t file_high_0;
    uint32_t file_high_1;
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

static int fy_result_is(
    struct fy_std_windows_file_result_v0 value, uint32_t status, DWORD os_code
) {
    return value.status == status && value.os_code == (uint32_t)os_code && value.count == 0;
}

struct fy_std_windows_file_result_v0 fy_std_windows_file_open_v0(
    const uint8_t *,
    const struct fy_std_windows_open_options_v0 *,
    void *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_remove_v0(const uint8_t *);
struct fy_std_windows_file_result_v0 fy_std_windows_directory_create_v0(const uint8_t *);
struct fy_std_windows_file_result_v0 fy_std_windows_directory_ensure_v0(const uint8_t *);
struct fy_std_windows_file_result_v0 fy_std_windows_directory_remove_v0(const uint8_t *);
struct fy_std_windows_file_result_v0 fy_std_windows_dir_remove_all_v0(const uint8_t *);
struct fy_std_windows_file_result_v0 fy_std_windows_directory_rename_v0(
    const uint8_t *,
    const uint8_t *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_rename_v0(
    const uint8_t *,
    const uint8_t *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_hard_link_v0(
    const uint8_t *,
    const uint8_t *
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
struct fy_std_windows_file_result_v0 fy_std_windows_file_id_v0(
    const void *,
    struct fy_std_windows_file_id_v0 *
);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_set_len_v0(void *, uint64_t);
struct fy_std_windows_file_position_result_v0 fy_std_windows_file_seek_start_v0(
    void *,
    uint64_t
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_write_v0(
    void *,
    const uint8_t *,
    size_t
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_write_all_v0(
    void *,
    const uint8_t *,
    size_t
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_sync_v0(void *);
struct fy_std_windows_file_result_v0 fy_std_windows_file_close_v0(void *);
uint32_t fy_std_windows_file_drop_v0(void *);
int fy_std_windows_file_copy_probe_v0(void);
int fy_std_windows_file_path_kind_probe_v0(void);
int fy_std_windows_file_seek_probe_v0(void);
int fy_std_windows_file_metadata_probe_v0(void);
int fy_std_windows_file_directory_probe_v0(void);
int fy_std_windows_file_remove_tree_probe_v0(void);
int fy_std_windows_file_link_probe_v0(void);

static int fy_std_windows_file_closed_v0(const struct fy_std_windows_file_v0 *file) {
    return file->handle == NULL && file->open == UINT8_C(0);
}

static int fy_std_windows_file_ids_equal_v0(
    const struct fy_std_windows_file_id_v0 *left,
    const struct fy_std_windows_file_id_v0 *right
) {
    return left->volume_low == right->volume_low &&
           left->volume_high == right->volume_high &&
           left->file_low_0 == right->file_low_0 &&
           left->file_low_1 == right->file_low_1 &&
           left->file_high_0 == right->file_high_0 &&
           left->file_high_1 == right->file_high_1;
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
    result = fy_std_windows_file_write_v0(&file, payload, UINT64_C(5));
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != UINT64_C(5)) return 2;
    result = fy_std_windows_file_write_all_v0(&file, payload + 5, sizeof payload - 5);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload - 5) return 94;
    struct fy_std_windows_file_size_result_v0 size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        size.size != (uint64_t)sizeof payload) return 31;
    struct fy_std_windows_file_kind_result_v0 kind = fy_std_windows_file_kind_v0(&file);
    if (kind.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        kind.os_code != (uint32_t)ERROR_SUCCESS ||
        kind.kind != (uint32_t)FILE_TYPE_DISK) return 71;
    struct fy_std_windows_file_id_v0 original_id = {0};
    result = fy_std_windows_file_id_v0(&file, &original_id);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 85;
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
    struct fy_std_windows_file_id_v0 closed_id = {
        UINT32_C(1), UINT32_C(2), UINT32_C(3),
        UINT32_C(4), UINT32_C(5), UINT32_C(6)
    };
    const struct fy_std_windows_file_id_v0 closed_id_expected = closed_id;
    result = fy_std_windows_file_id_v0(&file, &closed_id);
    if (result.status != FY_STD_WINDOWS_FILE_CLOSED_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0 ||
        !fy_std_windows_file_ids_equal_v0(&closed_id, &closed_id_expected)) return 86;
    if (fy_std_windows_file_set_len_v0(&file, UINT64_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 45;
    if (fy_std_windows_file_seek_start_v0(&file, UINT64_C(0)).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 25;
    if (fy_std_windows_file_seek_start_v0(&file, UINT64_MAX).status !=
        FY_STD_WINDOWS_FILE_CLOSED_V0) return 219;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 5;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 6;
    struct fy_std_windows_file_id_v0 reopened_id = {0};
    result = fy_std_windows_file_id_v0(&file, &reopened_id);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        !fy_std_windows_file_ids_equal_v0(&original_id, &reopened_id)) return 87;
    result = fy_std_windows_file_id_v0(&file, NULL);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 88;
    static const uint8_t other_path[] = "fairyc-windows-file-id-other-probe.tmp";
    struct fy_std_windows_file_v0 other_file = {NULL, UINT8_C(0)};
    result = fy_std_windows_file_open_v0(other_path, &write_options, &other_file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 89;
    struct fy_std_windows_file_id_v0 other_id = {0};
    result = fy_std_windows_file_id_v0(&other_file, &other_id);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        fy_std_windows_file_ids_equal_v0(&original_id, &other_id)) return 90;
    if (fy_std_windows_file_close_v0(&other_file).status !=
        FY_STD_WINDOWS_FILE_OK_V0) return 91;
    if (!DeleteFileW(L"fairyc-windows-file-id-other-probe.tmp")) return 92;
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
    struct fy_std_windows_file_id_v0 invalid_id = original_id;
    result = fy_std_windows_file_id_v0(&file, &invalid_id);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_INVALID_HANDLE || result.count != 0 ||
        !fy_std_windows_file_ids_equal_v0(&invalid_id, &original_id)) return 93;
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

    static const uint8_t remove_path[] = "fairyc-windows-file-remove-probe.tmp";
    static const uint8_t non_ascii_path[] = {UINT8_C(0x80), UINT8_C(0)};
    result = fy_std_windows_file_remove_v0(NULL);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 95;
    result = fy_std_windows_file_remove_v0(non_ascii_path);
    if (result.status != FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 96;
    result = fy_std_windows_file_open_v0(remove_path, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 97;
    result = fy_std_windows_file_remove_v0(remove_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_SHARING_VIOLATION || result.count != 0) return 98;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 99;
    result = fy_std_windows_file_remove_v0(remove_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 100;
    result = fy_std_windows_file_remove_v0(remove_path);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 101;

    static const uint8_t rename_source[] = "fairyc-windows-file-rename-source-probe.tmp";
    static const uint8_t rename_destination[] =
        "fairyc-windows-file-rename-destination-probe.tmp";
    static const uint8_t rename_missing[] = "fairyc-windows-file-rename-missing-probe.tmp";
    static const uint8_t rename_directory[] = "fairyc-windows-file-rename-directory-probe";
    static const uint8_t rename_directory_destination[] =
        "fairyc-windows-file-rename-directory-destination-probe";
    static const uint8_t rename_source_payload[] = {
        UINT8_C(83), UINT8_C(111), UINT8_C(117), UINT8_C(114), UINT8_C(99), UINT8_C(101)
    };
    static const uint8_t rename_destination_payload[] = {
        UINT8_C(68), UINT8_C(101), UINT8_C(115), UINT8_C(116)
    };
    result = fy_std_windows_file_rename_v0(NULL, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 102;
    result = fy_std_windows_file_rename_v0(rename_missing, non_ascii_path);
    if (result.status != FY_STD_WINDOWS_FILE_PATH_ENCODING_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 103;
    if (!CreateDirectoryW(L"fairyc-windows-file-rename-directory-probe", NULL)) return 104;
    result = fy_std_windows_file_rename_v0(rename_directory, rename_directory_destination);
    if (result.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 105;
    if (!RemoveDirectoryW(L"fairyc-windows-file-rename-directory-probe")) return 106;
    result = fy_std_windows_file_rename_v0(rename_missing, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 107;
    result = fy_std_windows_file_open_v0(rename_source, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 108;
    result = fy_std_windows_file_write_all_v0(
        &file,
        rename_source_payload,
        sizeof rename_source_payload
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof rename_source_payload) return 109;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 110;
    result = fy_std_windows_file_open_v0(rename_destination, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 111;
    result = fy_std_windows_file_write_all_v0(
        &file,
        rename_destination_payload,
        sizeof rename_destination_payload
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof rename_destination_payload) return 112;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 113;
    result = fy_std_windows_file_rename_v0(rename_source, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_ALREADY_EXISTS || result.count != 0) return 114;
    result = fy_std_windows_file_open_v0(rename_source, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 115;
    uint8_t rename_source_received[sizeof rename_source_payload] = {0};
    result = fy_std_windows_file_read_exact_v0(
        &file,
        rename_source_received,
        sizeof rename_source_received
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof rename_source_received ||
        memcmp(rename_source_received, rename_source_payload, sizeof rename_source_payload) != 0) {
        return 116;
    }
    result = fy_std_windows_file_rename_v0(rename_source, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_SHARING_VIOLATION || result.count != 0) return 117;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 118;
    result = fy_std_windows_file_open_v0(rename_destination, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 119;
    uint8_t rename_destination_received[sizeof rename_destination_payload] = {0};
    result = fy_std_windows_file_read_exact_v0(
        &file,
        rename_destination_received,
        sizeof rename_destination_received
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof rename_destination_received ||
        memcmp(
            rename_destination_received,
            rename_destination_payload,
            sizeof rename_destination_payload
        ) != 0) return 120;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 121;
    result = fy_std_windows_file_remove_v0(rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 122;
    result = fy_std_windows_file_open_v0(rename_source, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 123;
    result = fy_std_windows_file_rename_v0(rename_source, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_SHARING_VIOLATION || result.count != 0) return 124;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 125;
    result = fy_std_windows_file_rename_v0(rename_source, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.os_code != (uint32_t)ERROR_SUCCESS || result.count != 0) return 126;
    result = fy_std_windows_file_rename_v0(rename_source, rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        result.os_code != (uint32_t)ERROR_FILE_NOT_FOUND || result.count != 0) return 127;
    result = fy_std_windows_file_open_v0(rename_destination, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 128;
    uint8_t rename_final_received[sizeof rename_source_payload] = {0};
    result = fy_std_windows_file_read_exact_v0(
        &file,
        rename_final_received,
        sizeof rename_final_received
    );
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof rename_final_received ||
        memcmp(rename_final_received, rename_source_payload, sizeof rename_source_payload) != 0) {
        return 129;
    }
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 130;
    result = fy_std_windows_file_remove_v0(rename_destination);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 131;

    static const uint8_t utf8_path[] =
        "fairyc-windows-file-\xED\x95\x9C\xEA\xB8\x80.tmp";
    static const uint8_t invalid_utf8_path[] = {
        UINT8_C(0xc0), UINT8_C(0xaf), UINT8_C(0)
    };
    result = fy_std_windows_file_open_v0(invalid_utf8_path, &write_options, &file);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_PATH_ENCODING_V0, ERROR_SUCCESS) ||
        !fy_std_windows_file_closed_v0(&file)) return 132;
    result = fy_std_windows_file_open_v0(utf8_path, &write_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 133;
    result = fy_std_windows_file_write_all_v0(&file, payload, sizeof payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload) return 134;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 135;
    result = fy_std_windows_file_open_v0(utf8_path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 136;
    uint8_t utf8_received[sizeof payload] = {0};
    result = fy_std_windows_file_read_exact_v0(&file, utf8_received, sizeof utf8_received);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        result.count != sizeof payload ||
        memcmp(utf8_received, payload, sizeof payload) != 0) return 137;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 138;
    result = fy_std_windows_file_remove_v0(utf8_path);
    if (!fy_result_is(result, FY_STD_WINDOWS_FILE_OK_V0, ERROR_SUCCESS)) return 139;

    int directory_code = fy_std_windows_file_directory_probe_v0();
    if (directory_code != 0) return directory_code;
    int remove_tree_code = fy_std_windows_file_remove_tree_probe_v0();
    if (remove_tree_code != 0) return remove_tree_code;
    int link_code = fy_std_windows_file_link_probe_v0();
    if (link_code != 0) return link_code;
    int path_kind_code = fy_std_windows_file_path_kind_probe_v0();
    if (path_kind_code != 0) return path_kind_code;
    int seek_code = fy_std_windows_file_seek_probe_v0();
    if (seek_code != 0) return seek_code;
    int metadata_code = fy_std_windows_file_metadata_probe_v0();
    if (metadata_code != 0) return metadata_code;
    return fy_std_windows_file_copy_probe_v0();
}
