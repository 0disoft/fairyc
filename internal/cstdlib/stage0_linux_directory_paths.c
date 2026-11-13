//go:build ignore

#include <linux/fs.h>
#include <sys/syscall.h>
extern long syscall(long number, ...);

static int fy_std_linux_directory_parent_v0(const uint8_t *path, char *name, int *code) {
    size_t storage_length;
    *code = 0;
    if (!fy_std_file_path_storage_length_v0(path, &storage_length)) { *code = EINVAL; return -1; }
    size_t length = storage_length - 1;
    while (length > 1 && path[length - 1] == '/') { --length; }
    size_t start = length;
    while (start > 0 && path[start - 1] != '/') { --start; }
    if (start == length) { *code = EINVAL; return -1; }
    memcpy(name, path + start, length - start); name[length - start] = 0;
    if (start == 0) { return fy_std_linux_directory_open_path_v0((const uint8_t *)".", 1, 0, code); }
    while (start > 1 && path[start - 1] == '/') { --start; }
    return fy_std_linux_directory_open_path_v0(path, start, 0, code);
}

static struct fy_std_file_result_v0 fy_std_linux_directory_finish_v0(int descriptor, int code, size_t count) {
    if (descriptor >= 0 && close(descriptor) != 0 && code == 0) { code = errno; }
    return fy_std_file_result_v0(code == 0 ? 0u : 1u, code, count);
}

struct fy_std_file_result_v0 fy_std_windows_directory_create_v0(const uint8_t *path) {
    char name[4096]; int code;
    int parent = fy_std_linux_directory_parent_v0(path, name, &code);
    if (parent < 0) { return fy_std_file_result_v0(code == EINVAL ? 2u : 1u, code, 0); }
    code = mkdirat(parent, name, (mode_t)0777) == 0 ? 0 : errno;
    return fy_std_linux_directory_finish_v0(parent, code, 0);
}

struct fy_std_file_result_v0 fy_std_windows_directory_ensure_v0(const uint8_t *path) {
    size_t length;
    if (!fy_std_file_path_storage_length_v0(path, &length)) { return fy_std_file_result_v0(2, EINVAL, 0); }
    int code;
    int directory = fy_std_linux_directory_open_path_v0(path, length - 1, 1, &code);
    if (directory < 0) { return fy_std_file_result_v0(code == EINVAL ? 2u : 1u, code, 0); }
    return fy_std_linux_directory_finish_v0(directory, 0, 0);
}

struct fy_std_file_result_v0 fy_std_windows_directory_remove_v0(const uint8_t *path) {
    char name[4096]; int code;
    int parent = fy_std_linux_directory_parent_v0(path, name, &code);
    if (parent < 0) { return fy_std_file_result_v0(code == EINVAL ? 2u : 1u, code, 0); }
    code = unlinkat(parent, name, AT_REMOVEDIR) == 0 ? 0 : errno;
    return fy_std_linux_directory_finish_v0(parent, code, 0);
}

struct fy_std_file_result_v0 fy_std_windows_directory_rename_v0(const uint8_t *source, const uint8_t *destination) {
    char first[4096], second[4096]; int code;
    int source_parent = fy_std_linux_directory_parent_v0(source, first, &code);
    if (source_parent < 0) { return fy_std_file_result_v0(code == EINVAL ? 2u : 1u, code, 0); }
    int destination_parent = fy_std_linux_directory_parent_v0(destination, second, &code);
    if (destination_parent < 0) { return fy_std_linux_directory_finish_v0(source_parent, code, 0); }
    struct stat observed;
    uint32_t status = 0;
    if (fstatat(source_parent, first, &observed, AT_SYMLINK_NOFOLLOW) != 0) { code = errno; }
    else if (!S_ISDIR(observed.st_mode)) { code = EINVAL; status = 2; }
    else {
#if defined(SYS_renameat2)
        code = syscall(SYS_renameat2, source_parent, first, destination_parent, second, RENAME_NOREPLACE) == 0 ? 0 : errno;
#else
        code = ENOSYS;
#endif
    }
    if (close(destination_parent) != 0 && code == 0) { code = errno; }
    struct fy_std_file_result_v0 result = fy_std_linux_directory_finish_v0(source_parent, code, 0);
    if (status != 0) { result.status = status; }
    return result;
}

/* Own descriptor on every path. Descending follows opened directory identities,
   not path strings. Links are unlinked, never traversed; cross-device directories fail.
   Partial deletion is observable through count, not rolled back. */
static int fy_std_linux_directory_remove_children_v0(
    int descriptor, size_t depth, dev_t device, size_t *remaining, size_t *removed
) {
    if (depth >= 64) { (void)close(descriptor); return EOVERFLOW; }
    DIR *iterator = fdopendir(descriptor);
    if (iterator == NULL) { int code = errno; (void)close(descriptor); return code; }
    int code = 0;
    for (;;) {
        errno = 0; struct dirent *entry = readdir(iterator);
        if (entry == NULL) { code = errno; break; }
        if (!strcmp(entry->d_name, ".") || !strcmp(entry->d_name, "..")) { continue; }
        if (*remaining == 0) { code = EOVERFLOW; break; }
        --*remaining;
        struct stat observed;
        if (fstatat(dirfd(iterator), entry->d_name, &observed, AT_SYMLINK_NOFOLLOW) != 0) { code = errno; break; }
        if (S_ISDIR(observed.st_mode)) {
            if (observed.st_dev != device) { code = EXDEV; break; }
            int child = openat(dirfd(iterator), entry->d_name, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
            if (child < 0) { code = errno; break; }
            struct stat opened;
            if (fstat(child, &opened) != 0) { code = errno; (void)close(child); break; }
            if (opened.st_dev != observed.st_dev || opened.st_ino != observed.st_ino || !S_ISDIR(opened.st_mode)) {
                code = ESTALE; (void)close(child); break;
            }
            code = fy_std_linux_directory_remove_children_v0(child, depth + 1, device, remaining, removed);
            if (code != 0) { break; }
            struct stat current;
            if (fstatat(dirfd(iterator), entry->d_name, &current, AT_SYMLINK_NOFOLLOW) != 0) { code = errno; break; }
            if (current.st_dev != observed.st_dev || current.st_ino != observed.st_ino || !S_ISDIR(current.st_mode)) { code = ESTALE; break; }
            if (unlinkat(dirfd(iterator), entry->d_name, AT_REMOVEDIR) != 0) { code = errno; break; }
        } else if (unlinkat(dirfd(iterator), entry->d_name, 0) != 0) { code = errno; break; }
        ++*removed;
    }
    if (closedir(iterator) != 0 && code == 0) { code = errno; }
    return code;
}

struct fy_std_file_result_v0 fy_std_windows_dir_remove_all_v0(const uint8_t *path) {
    char name[4096]; int code;
    int parent = fy_std_linux_directory_parent_v0(path, name, &code);
    if (parent < 0) { return fy_std_file_result_v0(code == EINVAL ? 2u : 1u, code, 0); }
    if (!strcmp(name, ".") || !strcmp(name, "..")) { (void)close(parent); return fy_std_file_result_v0(2, EINVAL, 0); }
    struct stat observed;
    if (fstatat(parent, name, &observed, AT_SYMLINK_NOFOLLOW) != 0) { return fy_std_linux_directory_finish_v0(parent, errno, 0); }
    if (!S_ISDIR(observed.st_mode)) { (void)close(parent); return fy_std_file_result_v0(2, EINVAL, 0); }
    int directory = openat(parent, name, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (directory < 0) { return fy_std_linux_directory_finish_v0(parent, errno, 0); }
    struct stat opened;
    if (fstat(directory, &opened) != 0) { code = errno; (void)close(directory); return fy_std_linux_directory_finish_v0(parent, code, 0); }
    if (opened.st_dev != observed.st_dev || opened.st_ino != observed.st_ino) {
        (void)close(directory); return fy_std_linux_directory_finish_v0(parent, ESTALE, 0);
    }
    size_t remaining = 65536, removed = 0;
    code = fy_std_linux_directory_remove_children_v0(directory, 0, observed.st_dev, &remaining, &removed);
    if (code == 0) {
        struct stat current;
        if (fstatat(parent, name, &current, AT_SYMLINK_NOFOLLOW) != 0) { code = errno; }
        else if (current.st_dev != observed.st_dev || current.st_ino != observed.st_ino || !S_ISDIR(current.st_mode)) { code = ESTALE; }
        else if (unlinkat(parent, name, AT_REMOVEDIR) != 0) { code = errno; }
        else { ++removed; }
    }
    return fy_std_linux_directory_finish_v0(parent, code, removed);
}
