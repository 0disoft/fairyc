/* Appended after rooted I/O. Fixed five-file starter; never recursive deletion. */
struct fy_linux_starter_entry { int fd, created, known; struct stat info; };
static int fy_linux_starter_parent(struct fy_linux_starter_entry *entries, unsigned slot, int root) {
    return slot == 0 ? root : slot == 6 ? entries[1].fd : slot == 7 ? entries[2].fd : entries[0].fd;
}
static uint32_t fy_linux_starter_sync(int fd) {
    int result; do { result = fsync(fd); } while (result < 0 && errno == EINTR);
    return result == 0 ? 0 : (uint32_t)errno;
}
static uint32_t fy_linux_starter_close(int *fd) {
    if (*fd < 0) { return 0; }
    int value = *fd; *fd = -1;
    return close(value) == 0 ? 0 : (uint32_t)errno;
}
static int fy_linux_starter_same(const struct stat *left, const struct stat *right, int directory) {
    return left->st_dev == right->st_dev && left->st_ino == right->st_ino &&
        (directory ? S_ISDIR(right->st_mode) : S_ISREG(right->st_mode));
}
static int fy_linux_starter_rollback(struct fy_linux_starter_entry *entries, const char **names, int root) {
    int failed = 0;
    for (int i = 7; i >= 0; --i) {
        if (!entries[i].created) { continue; }
        int parent = fy_linux_starter_parent(entries, (unsigned)i, root);
        struct stat current;
        if (!entries[i].known || parent < 0) { failed = 1; continue; }
        if (fstatat(parent, names[i], &current, AT_SYMLINK_NOFOLLOW) < 0) {
            if (errno != ENOENT) { failed = 1; } continue;
        }
        if (!fy_linux_starter_same(&entries[i].info, &current, i < 3)) { failed = 1; continue; }
        if (unlinkat(parent, names[i], i < 3 ? AT_REMOVEDIR : 0) < 0) { failed = 1; }
    }
    return failed;
}
uint32_t fairyc_private_linux_starter_create_v1(uint32_t root, const uint8_t *leaf, size_t leaf_size,
        const uint8_t *data, size_t size, const void *sizes, uint32_t *slot, uint32_t *rollback) {
    if (slot == NULL || rollback == NULL) { return EINVAL; }
    *slot = 0; *rollback = 0;
    const size_t *lengths = sizes;
    int parent = fy_linux_root_fd(root);
    if (parent < 0 || !fy_linux_root_path(leaf, leaf_size, 1) || lengths == NULL ||
        size > 65536 || (size != 0 && data == NULL)) { return EINVAL; }
    size_t total = 0;
    for (unsigned i = 0; i < 5; ++i) { if (lengths[i] > 65536 - total) { return EINVAL; } total += lengths[i]; }
    if (total != size) { return EINVAL; }
    const char *names[8] = {(const char *)leaf, "src", "tests", ".gitignore", "README.md", "fairy.toml", "main.fy", "main.fy"};
    struct fy_linux_starter_entry entries[8]; memset(entries, 0, sizeof entries);
    for (unsigned i = 0; i < 8; ++i) { entries[i].fd = -1; }
    uint32_t code = 0;
    for (unsigned i = 0; i < 3; ++i) {
        *slot = i; int fd = fy_linux_starter_parent(entries, i, parent);
        if (mkdirat(fd, names[i], 0755) < 0) { code = (uint32_t)errno; goto failed; }
        entries[i].created = 1;
        entries[i].fd = openat(fd, names[i], O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (entries[i].fd < 0) { code = (uint32_t)errno; goto failed; }
        if (fstat(entries[i].fd, &entries[i].info) < 0) { code = (uint32_t)errno; goto failed; }
        if (!S_ISDIR(entries[i].info.st_mode)) { code = EINVAL; goto failed; }
        entries[i].known = 1;
    }
    size_t offset = 0;
    for (unsigned i = 3; i < 8; ++i) {
        *slot = i; int fd = fy_linux_starter_parent(entries, i, parent);
        entries[i].fd = openat(fd, names[i], O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0644);
        if (entries[i].fd < 0) { code = (uint32_t)errno; goto failed; }
        entries[i].created = 1;
        if (fstat(entries[i].fd, &entries[i].info) < 0) { code = (uint32_t)errno; goto failed; }
        if (!S_ISREG(entries[i].info.st_mode)) { code = EINVAL; goto failed; }
        entries[i].known = 1;
        size_t length = lengths[i-3], written = 0;
        while (written < length) {
            ssize_t n = write(entries[i].fd, data + offset + written, length - written);
            if (n < 0 && errno == EINTR) { continue; }
            if (n <= 0 || (size_t)n > length-written) { code = n < 0 ? (uint32_t)errno : EIO; goto failed; }
            written += (size_t)n;
        }
        offset += length;
        code = fy_linux_starter_sync(entries[i].fd); if (code != 0) { goto failed; }
        code = fy_linux_starter_close(&entries[i].fd); if (code != 0) { goto failed; }
    }
    /* Recheck the fixed recorded identities, including the named root, before success. */
    for (unsigned i = 0; i < 8; ++i) {
        *slot = i; struct stat current;
        if (fstatat(fy_linux_starter_parent(entries, i, parent), names[i], &current, AT_SYMLINK_NOFOLLOW) < 0) { code = (uint32_t)errno; goto failed; }
        if (!fy_linux_starter_same(&entries[i].info, &current, i < 3)) { code = ESTALE; goto failed; }
    }
    for (unsigned i = 3; i > 0; --i) { *slot = i-1; code = fy_linux_starter_sync(entries[i-1].fd); if (code != 0) { goto failed; } }
    *slot = 0; code = fy_linux_starter_sync(parent); if (code != 0) { goto failed; }
    for (unsigned i = 3; i > 0; --i) {
        uint32_t close_code = fy_linux_starter_close(&entries[i-1].fd);
        if (close_code != 0 && code == 0) { code = close_code; *slot = i-1; }
    }
    /* A completed tree is retained if final descriptor closing fails. */
    if (code != 0) { *rollback = 1; }
    return code;
failed:
    for (unsigned i = 3; i < 8; ++i) { if (fy_linux_starter_close(&entries[i].fd) != 0) { *rollback = 1; } }
    if (fy_linux_starter_rollback(entries, names, parent)) { *rollback = 1; }
    for (unsigned i = 3; i > 0; --i) { if (fy_linux_starter_close(&entries[i-1].fd) != 0) { *rollback = 1; } }
    return code;
}
