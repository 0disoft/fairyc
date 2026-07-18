#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <unistd.h>

void fy_runtime_trace_drop(uint32_t site_id) {
    uint8_t record[4] = {
        (uint8_t)(site_id & UINT32_C(0xff)),
        (uint8_t)((site_id >> 8u) & UINT32_C(0xff)),
        (uint8_t)((site_id >> 16u) & UINT32_C(0xff)),
        (uint8_t)((site_id >> 24u) & UINT32_C(0xff)),
    };
    size_t offset = 0;
    while (offset < sizeof record) {
        ssize_t written = write(STDERR_FILENO, record + offset, sizeof record - offset);
        if (written > 0) {
            offset += (size_t)written;
            continue;
        }
        if (written < 0 && errno == EINTR) {
            continue;
        }
        _Exit(254);
    }
}
