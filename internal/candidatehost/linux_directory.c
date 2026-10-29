/* Appended after linux_rooted_file.c; private DIR ownership and one-row transport. */
#include <dirent.h>

uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root, size_t *output) {
    if (output == NULL) { return EINVAL; } *output = 0;
    int fd = fy_linux_root_fd(root);
    if (fd < 0) { return EINVAL; }
    /* dup would share the root's directory offset. A fresh open description does not. */
    uint32_t owned = 0;
    uint32_t code = fy_linux_root_open(fd, ".", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC, 1, &owned);
    if (code != 0) { return code; }
    fd = fy_linux_root_fd(owned);
    DIR *stream = fdopendir(fd);
    if (stream == NULL) { code = (uint32_t)errno; (void)close(fd); return code; }
    *output = (size_t)(uintptr_t)stream; return 0;
}

/* state 0 EOF, 1 row + nofollow metadata, 2 dot entry or vanished name.
 * Exactly one readdir result per call: skipped entries still cost caller work. */
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t handle, uint8_t *name,
        size_t capacity, uint32_t *length, uint32_t *state, void *metadata) {
    if (length == NULL || state == NULL || metadata == NULL) { return EINVAL; }
    *length = 0; *state = 0;
    memset(metadata, 0, sizeof(struct fy_linux_root_metadata_v1));
    if (handle == 0 || name == NULL || capacity != 256) { return EINVAL; }
    errno = 0;
    struct dirent *entry = readdir((DIR *)(uintptr_t)handle);
    if (entry == NULL) { return (uint32_t)errno; }
    size_t offset = offsetof(struct dirent, d_name);
    if (entry->d_reclen <= offset) { return EIO; }
    size_t available = entry->d_reclen - offset;
    size_t bound = available < capacity ? available : capacity;
    const char *end = (const char *)memchr(entry->d_name, 0, bound);
    if (end == NULL) { return available >= capacity ? ENAMETOOLONG : EIO; }
    size_t size = (size_t)(end - entry->d_name);
    if (size == 0 || memchr(entry->d_name, '/', size) != NULL) { return EIO; }
    if ((size == 1 && entry->d_name[0] == '.') ||
        (size == 2 && entry->d_name[0] == '.' && entry->d_name[1] == '.')) {
        memcpy(name, entry->d_name, size + 1); *length = (uint32_t)size; *state = 2; return 0;
    }
    struct stat info;
    int result;
    do { result = fstatat(dirfd((DIR *)(uintptr_t)handle), entry->d_name, &info, AT_SYMLINK_NOFOLLOW); }
    while (result < 0 && errno == EINTR);
    if (result < 0 && errno != ENOENT) { return (uint32_t)errno; }
    if (result < 0) {
        memcpy(name, entry->d_name, size + 1); *length = (uint32_t)size; *state = 2; return 0;
    }
    if (info.st_size < 0 || info.st_mtim.tv_nsec < 0 || info.st_mtim.tv_nsec >= 1000000000) { return EIO; }
    struct fy_linux_root_metadata_v1 value = { (size_t)info.st_dev, (size_t)info.st_ino,
        (size_t)info.st_size, (int64_t)info.st_mtim.tv_sec, (uint32_t)info.st_mtim.tv_nsec, (uint32_t)info.st_mode };
    memcpy(name, entry->d_name, size + 1); memcpy(metadata, &value, sizeof value);
    *length = (uint32_t)size; *state = 1; return 0;
}

uint32_t fairyc_private_linux_directory_scan_close_v1(size_t handle) {
    if (handle == 0) { return 0; }
    /* closedir owns its descriptor; consume once even if it reports an error. */
    return closedir((DIR *)(uintptr_t)handle) == 0 ? 0 : (uint32_t)errno;
}
