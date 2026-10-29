/* Appended after linux_process.c; private helpers and feature macros are shared. */
#include <fcntl.h>

typedef struct {
    uint32_t pid;
    int readers[2];
    int collected;
} fy_linux_capture;

/* On Linux close releases the descriptor even on EINTR: never retry a reused fd. */
static uint32_t fy_linux_capture_close(int *fd) {
    if (*fd < 0) { return 0; }
    int value = *fd;
    *fd = -1;
    return close(value) == 0 ? 0 : (uint32_t)errno;
}

static uint32_t fy_linux_capture_pipe(int pair[2]) {
    if (pipe2(pair, O_CLOEXEC) != 0) { return (uint32_t)errno; }
    for (int i = 0; i < 2; ++i) {
        /* Keep dup2 sources away from stdin/stdout/stderr even if those are closed. */
        if (pair[i] <= 2) {
            int moved = fcntl(pair[i], F_DUPFD_CLOEXEC, 3);
            if (moved < 0) { return (uint32_t)errno; }
            uint32_t code = fy_linux_capture_close(&pair[i]);
            pair[i] = moved;
            if (code != 0) { return code; }
        }
    }
    int flags = fcntl(pair[0], F_GETFL);
    if (flags < 0 || fcntl(pair[0], F_SETFL, flags | O_NONBLOCK) < 0) { return (uint32_t)errno; }
    return 0;
}

uint32_t fairyc_private_linux_capture_open_v1(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, void **output) {
    if (output == NULL) { return EINVAL; }
    *output = NULL;
    fy_linux_capture *capture = calloc(1, sizeof *capture);
    if (capture == NULL) { return ENOMEM; }
    capture->readers[0] = -1; capture->readers[1] = -1;
    int out[2] = {-1, -1}, err[2] = {-1, -1};
    uint32_t code = fy_linux_capture_pipe(out);
    if (code == 0) { code = fy_linux_capture_pipe(err); }
    if (code == 0) { code = fy_linux_process_spawn(data, size, argc, cwd, cwd_size, out[1], err[1], &capture->pid); }
    uint32_t out_closed = fy_linux_capture_close(&out[1]);
    uint32_t err_closed = fy_linux_capture_close(&err[1]);
    if (code == 0) { code = out_closed != 0 ? out_closed : err_closed; }
    if (code != 0) {
        if (capture->pid != 0) { (void)fairyc_private_linux_process_discard_v1(capture->pid); }
        (void)fy_linux_capture_close(&out[0]);
        (void)fy_linux_capture_close(&err[0]);
        free(capture);
        return code;
    }
    capture->readers[0] = out[0]; capture->readers[1] = err[0];
    *output = capture;
    return 0;
}

uint32_t fairyc_private_linux_capture_drop_v1(void *raw) {
    if (raw == NULL) { return 0; }
    fy_linux_capture *capture = raw;
    uint32_t code = fairyc_private_linux_process_discard_v1(capture->pid);
    uint32_t a = fy_linux_capture_close(&capture->readers[0]);
    uint32_t b = fy_linux_capture_close(&capture->readers[1]);
    free(capture);
    return code != 0 ? code : a != 0 ? a : b;
}
