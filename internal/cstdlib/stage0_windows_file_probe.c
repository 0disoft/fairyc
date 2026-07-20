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
    FY_STD_WINDOWS_FILE_PATH_TOO_LONG_V0 = 6
};

struct fy_std_windows_file_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t count;
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
};

struct fy_std_windows_file_result_v0 fy_std_windows_file_open_v0(
    const uint8_t *,
    const struct fy_std_windows_open_options_v0 *,
    void *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_read_v0(void *, void *, size_t);
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
        UINT8_C(0), UINT8_C(1), UINT8_C(1), UINT8_C(1)
    };
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0)
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
    if (fy_std_windows_file_sync_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 3;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0 ||
        !fy_std_windows_file_closed_v0(&file)) return 4;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_CLOSED_V0) return 5;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 6;
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
    if (total != sizeof payload || memcmp(received, payload, sizeof payload) != 0) return 9;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 10;

    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 11;
    HANDLE stolen = (HANDLE)file.handle;
    if (!CloseHandle(stolen)) return 12;
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
    return 0;
}
