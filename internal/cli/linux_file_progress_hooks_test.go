package cli

// The scripts replace POSIX calls, not File algorithms or SDK result conversion.
const linuxFileProgressHooks = `
#define _POSIX_C_SOURCE 200809L
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>

struct fy_file_step { size_t offset, length; ssize_t result; int error; };
static const struct fy_file_step fy_file_success[] = {
    {0,6,-1,EINTR}, {0,6,2,0}, {2,4,-1,EINTR}, {2,4,1,0}, {3,3,3,0}
};
static const struct fy_file_step fy_file_partial_error[] = {{0,6,2,0}, {2,4,-1,EIO}};
static const struct fy_file_step fy_file_disk_full[] = {{0,6,2,0}, {2,4,-1,ENOSPC}};
static const struct fy_file_step fy_file_partial_zero[] = {{0,6,2,0}, {2,4,0,0}};
static const struct fy_file_step fy_file_first_error[] = {{0,6,-1,EIO}};
static const struct fy_file_step fy_file_first_zero[] = {{0,6,0,0}};
static const struct fy_file_step fy_file_empty[] = {{0,0,0,0}};
static const struct fy_file_step fy_file_single[] = {{0,6,-1,EINTR}, {0,6,2,0}};
static const struct fy_file_step fy_file_sync_ok[] = {{0,0,-1,EINTR}, {0,0,0,0}};
static const struct fy_file_step fy_file_sync_error[] = {{0,0,-1,EINTR}, {0,0,-1,EIO}};
static const struct fy_file_step fy_file_close_error[] = {{0,0,-1,EINTR}};
static const uint8_t fy_file_payload[] = {65,0,255,98,99,100};
static const struct fy_file_step *fy_file_steps;
static size_t fy_file_count, fy_file_calls, fy_file_progress;
static uint32_t fy_file_mode;
static int fy_file_descriptor;
static const uint8_t *fy_file_base;
static uint8_t fy_file_observed[6];

void fairyc_file_progress_begin(uint32_t mode) {
    assert(fy_file_mode == 0 && mode >= 1 && mode <= 17);
    fy_file_mode = mode;
    fy_file_calls = fy_file_progress = 0;
    fy_file_descriptor = -1;
    fy_file_base = NULL;
    memset(fy_file_observed,165,sizeof fy_file_observed);
    switch (mode) {
        case 1: case 8: fy_file_steps=fy_file_success; fy_file_count=5; break;
        case 2: fy_file_steps=fy_file_disk_full; fy_file_count=2; break;
        case 3: case 10: fy_file_steps=fy_file_partial_zero; fy_file_count=2; break;
        case 4: case 11: fy_file_steps=fy_file_first_error; fy_file_count=1; break;
        case 5: case 12: fy_file_steps=fy_file_first_zero; fy_file_count=1; break;
        case 6: fy_file_steps=NULL; fy_file_count=0; break;
        case 7: case 14: fy_file_steps=fy_file_single; fy_file_count=2; break;
        case 9: fy_file_steps=fy_file_partial_error; fy_file_count=2; break;
        case 13: fy_file_steps=fy_file_empty; fy_file_count=1; break;
        case 15: fy_file_steps=fy_file_sync_ok; fy_file_count=2; break;
        case 16: fy_file_steps=fy_file_sync_error; fy_file_count=2; break;
        case 17: fy_file_steps=fy_file_close_error; fy_file_count=1; break;
    }
}

static const struct fy_file_step *fy_file_next(int descriptor, const void *data, size_t length) {
    assert(fy_file_calls < fy_file_count);
    const struct fy_file_step *step = &fy_file_steps[fy_file_calls];
    if (fy_file_calls == 0) { fy_file_descriptor=descriptor; fy_file_base=data; }
    assert(descriptor == fy_file_descriptor && length == step->length);
    assert(step->offset == fy_file_progress);
    if (length != 0) {
        assert(step->offset <= sizeof fy_file_payload);
        assert(length <= sizeof fy_file_payload-step->offset);
        assert(data == fy_file_base+step->offset);
    }
    fy_file_calls++;
    errno=step->error;
    return step;
}

static ssize_t fy_file_progress_write(int descriptor, const void *data, size_t length) {
    if (fy_file_mode == 0) return write(descriptor,data,length);
    assert(fy_file_mode <= 7);
    const struct fy_file_step *step=fy_file_next(descriptor,data,length);
    assert(memcmp(data,fy_file_payload+step->offset,length) == 0);
    if (step->result > 0) {
        memcpy(fy_file_observed+step->offset,data,(size_t)step->result);
        fy_file_progress+=(size_t)step->result;
    }
    return step->result;
}

static ssize_t fy_file_progress_read(int descriptor, void *data, size_t length) {
    if (fy_file_mode == 0) return read(descriptor,data,length);
    assert(fy_file_mode >= 8 && fy_file_mode <= 14);
    const struct fy_file_step *step=fy_file_next(descriptor,data,length);
    if (step->result > 0) {
        memcpy(data,fy_file_payload+step->offset,(size_t)step->result);
        fy_file_progress+=(size_t)step->result;
    }
    return step->result;
}

static int fy_file_progress_sync(int descriptor) {
    if (fy_file_mode == 0) return fsync(descriptor);
    assert(fy_file_mode == 15 || fy_file_mode == 16);
    return (int)fy_file_next(descriptor,NULL,0)->result;
}

static int fy_file_progress_close(int descriptor) {
    if (fy_file_mode == 0) return close(descriptor);
    assert(fy_file_mode == 17);
    const struct fy_file_step *step=fy_file_next(descriptor,NULL,0);
    assert(close(descriptor) == 0);
    errno=step->error;
    return (int)step->result;
}

void fairyc_file_progress_end(void) {
    assert(fy_file_mode != 0 && fy_file_calls == fy_file_count);
    if (fy_file_mode <= 7) {
        assert(memcmp(fy_file_observed,fy_file_payload,fy_file_progress) == 0);
        for (size_t i=fy_file_progress; i<sizeof fy_file_observed; i++) assert(fy_file_observed[i] == 165);
    }
    fy_file_mode=0;
}
`
