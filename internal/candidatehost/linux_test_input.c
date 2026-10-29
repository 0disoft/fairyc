/* Appended after linux_capture.c. Only the caller-owned replay file is opened. */
#include <sys/stat.h>

uint32_t fairyc_private_linux_test_capture_open_v1(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, uint32_t input_mode,
        const uint8_t *path, size_t path_size, void **output) {
    if (output == NULL) { return EINVAL; }
    *output = NULL;
    if (input_mode > 2 || (input_mode != 2 && path_size != 0)) { return EINVAL; }
    int input = -1;
    uint32_t code = 0;
    if (input_mode == 1) {
        int eof[2] = {-1, -1};
        code = fy_linux_capture_pipe(eof, 4);
        uint32_t closed = fy_linux_capture_close(&eof[1]);
        input = eof[0];
        if (code == 0) { code = closed; }
    } else if (input_mode == 2) {
        if (path == NULL || path_size < 2 || path_size > 4096 || path[0] != '/' ||
            path[path_size-1] != 0 || memchr(path, 0, path_size-1) != NULL) { return EINVAL; }
        input = open((const char *)path, O_RDONLY | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK);
        if (input < 0) { return (uint32_t)errno; }
        struct stat info;
        if (fstat(input, &info) != 0) { code = (uint32_t)errno; }
        else if (!S_ISREG(info.st_mode) || info.st_size < 0 || info.st_size > 65536) { code = EINVAL; }
        if (code == 0 && input < 4) {
            int moved = fcntl(input, F_DUPFD_CLOEXEC, 4);
            if (moved < 0) { code = (uint32_t)errno; }
            else { code = fy_linux_capture_close(&input); input = moved; }
        }
    }
    if (code == 0) { code = fy_linux_capture_open(data, size, argc, cwd, cwd_size, 1, input, output); }
    uint32_t closed = fy_linux_capture_close(&input);
    if (code == 0 && closed != 0) {
        code = closed;
        (void)fairyc_private_linux_capture_drop_v1(*output);
        *output = NULL;
    }
    return code;
}
