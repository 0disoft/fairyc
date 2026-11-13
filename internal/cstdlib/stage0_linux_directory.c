//go:build ignore

#include <dirent.h>
#include <stdlib.h>

uint32_t fy_std_text_utf8_valid_v0(const uint8_t *data, size_t length);

struct fy_std_directory_v0 { void *state; uint8_t open; };
struct fy_std_directory_result_v0 { uint32_t status, os_code; size_t count; };
struct fy_std_directory_entry_result_v0 {
    uint32_t status, os_code;
    size_t name_length;
    uint32_t kind;
    size_t size;
    uint32_t reparse;
};
struct fy_std_linux_directory_state_v0 {
    DIR *iterator;
    char name[4096];
    size_t length;
    int pending, ended;
};

static struct fy_std_directory_result_v0 fy_std_directory_result_v0(uint32_t status, int code) {
    struct fy_std_directory_result_v0 result = {status, (uint32_t)code, 0}; return result;
}
static struct fy_std_directory_entry_result_v0 fy_std_directory_entry_result_v0(uint32_t status, int code) {
    struct fy_std_directory_entry_result_v0 result = {status, (uint32_t)code, 0, 0, 0, 0}; return result;
}

/* Open every prefix relative to an already-opened directory. No component may
   redirect through a symbolic link. Relative dot/parent components keep their
   ordinary filesystem meaning; this API is not a workspace sandbox. */
static int fy_std_linux_directory_open_path_v0(
    const uint8_t *path, size_t length, int create, int *code
) {
    *code = 0;
    if (path == NULL || length == 0 || length >= 4096) { *code = EINVAL; return -1; }
    for (size_t i = 0; i < length; ++i) { if (path[i] == 0) { *code = EINVAL; return -1; } }
    int flags = O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC;
    int descriptor = open(path[0] == '/' ? "/" : ".", flags);
    if (descriptor < 0) { *code = errno; return -1; }
    size_t at = 0;
    while (at < length) {
        while (at < length && path[at] == '/') { ++at; }
        if (at == length) { break; }
        size_t start = at;
        while (at < length && path[at] != '/') { ++at; }
        char component[4096]; size_t count = at - start;
        memcpy(component, path + start, count); component[count] = 0;
        if (create && mkdirat(descriptor, component, (mode_t)0777) != 0 && errno != EEXIST) {
            *code = errno; (void)close(descriptor); return -1;
        }
        int next = openat(descriptor, component, flags);
        if (next < 0) { *code = errno; (void)close(descriptor); return -1; }
        if (close(descriptor) != 0) { *code = errno; (void)close(next); return -1; }
        descriptor = next;
    }
    return descriptor;
}

struct fy_std_directory_result_v0 fy_std_windows_read_dir_open_v0(
    const uint8_t *path, size_t length, void *output
) {
    if (output == NULL) { return fy_std_directory_result_v0(2, EINVAL); }
    struct fy_std_directory_v0 empty = {NULL, 0}; memcpy(output, &empty, sizeof empty);
    if (path == NULL || length == 0) { return fy_std_directory_result_v0(2, EINVAL); }
    if (length >= 4096) { return fy_std_directory_result_v0(6, ENAMETOOLONG); }
    if (!fy_std_text_utf8_valid_v0(path, length)) { return fy_std_directory_result_v0(5, EILSEQ); }
    int code;
    int descriptor = fy_std_linux_directory_open_path_v0(path, length, 0, &code);
    if (descriptor < 0) { return fy_std_directory_result_v0(code == EINVAL ? 2u : 1u, code); }
    struct fy_std_linux_directory_state_v0 *state = calloc(1, sizeof *state);
    if (state == NULL) { (void)close(descriptor); return fy_std_directory_result_v0(4, ENOMEM); }
    state->iterator = fdopendir(descriptor);
    if (state->iterator == NULL) { code = errno; (void)close(descriptor); free(state); return fy_std_directory_result_v0(1, code); }
    struct fy_std_directory_v0 opened = {state, 1}; memcpy(output, &opened, sizeof opened);
    return fy_std_directory_result_v0(0, 0);
}

struct fy_std_directory_entry_result_v0 fy_std_windows_read_dir_next_v0(void *storage, uint8_t *name, size_t capacity) {
    if (storage == NULL || (name == NULL && capacity != 0)) { return fy_std_directory_entry_result_v0(2, EINVAL); }
    struct fy_std_directory_v0 directory; memcpy(&directory, storage, sizeof directory);
    if (directory.open == 0 && directory.state == NULL) { return fy_std_directory_entry_result_v0(3, EBADF); }
    if (directory.open != 1 || directory.state == NULL) { return fy_std_directory_entry_result_v0(2, EINVAL); }
    struct fy_std_linux_directory_state_v0 *state = directory.state;
    while (!state->ended) {
        if (!state->pending) {
            errno = 0;
            struct dirent *entry = readdir(state->iterator);
            if (entry == NULL) {
                if (errno != 0) { return fy_std_directory_entry_result_v0(1, errno); }
                state->ended = 1; break;
            }
            if (!strcmp(entry->d_name, ".") || !strcmp(entry->d_name, "..")) { continue; }
            size_t length = strlen(entry->d_name);
            if (length == 0 || length >= sizeof state->name) { return fy_std_directory_entry_result_v0(6, ENAMETOOLONG); }
            memcpy(state->name, entry->d_name, length + 1); state->length = length; state->pending = 1;
        }
        if (!fy_std_text_utf8_valid_v0((const uint8_t *)state->name, state->length)) { return fy_std_directory_entry_result_v0(5, EILSEQ); }
        if (capacity < state->length) {
            struct fy_std_directory_entry_result_v0 result = fy_std_directory_entry_result_v0(6, ERANGE);
            result.name_length = state->length; return result;
        }
        struct stat observed;
        if (fstatat(dirfd(state->iterator), state->name, &observed, AT_SYMLINK_NOFOLLOW) != 0) {
            return fy_std_directory_entry_result_v0(1, errno);
        }
        _Static_assert(sizeof observed.st_size <= sizeof(size_t), "Linux entry size must fit the ABI");
        uint32_t kind = S_ISREG(observed.st_mode) ? 1u : (S_ISDIR(observed.st_mode) ? 2u : 3u);
        if (kind == 1 && observed.st_size < 0) {
            return fy_std_directory_entry_result_v0(6, EOVERFLOW);
        }
        memcpy(name, state->name, state->length);
        struct fy_std_directory_entry_result_v0 result = {
            0, 0, state->length, kind, kind == 1 ? (size_t)observed.st_size : 0, S_ISLNK(observed.st_mode) ? 1u : 0u
        };
        state->pending = 0; return result;
    }
    return fy_std_directory_entry_result_v0(0, 0);
}

struct fy_std_directory_result_v0 fy_std_windows_read_dir_close_v0(void *storage) {
    if (storage == NULL) { return fy_std_directory_result_v0(2, EINVAL); }
    struct fy_std_directory_v0 directory; memcpy(&directory, storage, sizeof directory);
    struct fy_std_directory_v0 empty = {NULL, 0}; memcpy(storage, &empty, sizeof empty);
    if (directory.open == 0 && directory.state == NULL) { return fy_std_directory_result_v0(3, EBADF); }
    if (directory.open != 1 || directory.state == NULL) { return fy_std_directory_result_v0(2, EINVAL); }
    struct fy_std_linux_directory_state_v0 *state = directory.state;
    int closed = closedir(state->iterator); int code = closed == 0 ? 0 : errno;
    free(state); return fy_std_directory_result_v0(closed == 0 ? 0u : 1u, code);
}

uint32_t fy_std_windows_read_dir_drop_v0(void *storage) {
    if (storage == NULL) { return 2; }
    struct fy_std_directory_v0 directory; memcpy(&directory, storage, sizeof directory);
    if (directory.open == 0 && directory.state == NULL) { return 0; }
    return fy_std_windows_read_dir_close_v0(storage).status;
}
