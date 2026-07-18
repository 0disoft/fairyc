#define _POSIX_C_SOURCE 200809L

#include <errno.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

enum fy_std_file_status_v0 { FY_STD_FILE_OK_V0 = 0, FY_STD_FILE_IO_ERROR_V0 = 1, FY_STD_FILE_INVALID_ARGUMENT_V0 = 2, FY_STD_FILE_CLOSED_V0 = 3, FY_STD_FILE_NO_PROGRESS_V0 = 4 };
struct fy_std_file_result_v0 { uint32_t status; int32_t error_number; size_t count; };
struct fy_std_linux_file_v0 { int descriptor; uint8_t open; };
struct fy_std_linux_open_options_v0 { uint8_t read; uint8_t write; uint8_t create; uint8_t truncate; uint32_t mode; };
struct fy_std_file_result_v0 fy_std_linux_file_open_v0(const uint8_t *, size_t, const struct fy_std_linux_open_options_v0 *, struct fy_std_linux_file_v0 *);
struct fy_std_file_result_v0 fy_std_linux_file_read_v0(struct fy_std_linux_file_v0 *, uint8_t *, size_t);
struct fy_std_file_result_v0 fy_std_linux_file_write_all_v0(struct fy_std_linux_file_v0 *, const uint8_t *, size_t);
struct fy_std_file_result_v0 fy_std_linux_file_sync_v0(struct fy_std_linux_file_v0 *);
struct fy_std_file_result_v0 fy_std_linux_file_close_v0(struct fy_std_linux_file_v0 *);
void fy_std_linux_file_drop_v0(struct fy_std_linux_file_v0 *);

int main(void) {
    char path[] = "/tmp/fairyc-file-probe-XXXXXX";
    int seed = mkstemp(path);
    if (seed < 0 || close(seed) != 0) return 1;
    struct fy_std_linux_open_options_v0 write_options = {0, 1, 0, 1, 0600};
    struct fy_std_linux_file_v0 file = {-1, 0};
    struct fy_std_file_result_v0 result = fy_std_linux_file_open_v0((const uint8_t *)path, sizeof path, &write_options, &file);
    if (result.status != FY_STD_FILE_OK_V0 || !file.open) return 2;
    static const uint8_t payload[] = "partial-io-contract";
    result = fy_std_linux_file_write_all_v0(&file, payload, sizeof payload - 1);
    if (result.status != FY_STD_FILE_OK_V0 || result.count != sizeof payload - 1) return 3;
    if (fy_std_linux_file_sync_v0(&file).status != FY_STD_FILE_OK_V0) return 4;
    if (fy_std_linux_file_close_v0(&file).status != FY_STD_FILE_OK_V0 || file.open || file.descriptor != -1) return 5;
    if (fy_std_linux_file_close_v0(&file).status != FY_STD_FILE_CLOSED_V0) return 6;

    struct fy_std_linux_open_options_v0 read_options = {1, 0, 0, 0, 0};
    if (fy_std_linux_file_open_v0((const uint8_t *)path, sizeof path, &read_options, &file).status != FY_STD_FILE_OK_V0) return 7;
    uint8_t received[32] = {0};
    size_t total = 0;
    while (total < sizeof received) {
        result = fy_std_linux_file_read_v0(&file, received + total, 3);
        if (result.status != FY_STD_FILE_OK_V0) return 8;
        if (result.count == 0) break;
        total += result.count;
    }
    if (total != sizeof payload - 1 || memcmp(received, payload, total) != 0) return 9;
    int stolen = file.descriptor;
    if (close(stolen) != 0) return 10;
    result = fy_std_linux_file_close_v0(&file);
    if (result.status != FY_STD_FILE_IO_ERROR_V0 || result.error_number != EBADF || file.open || file.descriptor != -1) return 11;
    fy_std_linux_file_drop_v0(&file);
    if (unlink(path) != 0) return 12;
    return 0;
}
