//go:build ignore

/* Appended after linux_rooted_file.c. The caller owns an exclusive edit lock
 * and a private staging directory. Local trusted directories only: hostile
 * namespace writers and network filesystem commit ambiguity are not admitted. */
static uint32_t fy_linux_edit_stat(int fd, struct stat *info) {
    int result;
    do { result = fstat(fd, info); } while (result < 0 && errno == EINTR);
    return result == 0 ? 0 : (uint32_t)errno;
}

uint32_t fairyc_private_linux_edit_file_prepare_v1(uint32_t handle, uint32_t mode) {
    int fd = fy_linux_root_fd(handle);
    if (fd < 0 || mode > 0777u) { return EINVAL; }
    struct stat info;
    uint32_t code = fy_linux_edit_stat(fd, &info);
    if (code != 0) { return code; }
    if (!S_ISREG(info.st_mode) || info.st_nlink != 0) { return EINVAL; }
    int result;
    do { result = fchmod(fd, (mode_t)mode); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    return fairyc_private_linux_root_file_sync_v1(handle);
}

static uint32_t fy_linux_edit_current(int root, const char *leaf,
        const struct fy_linux_root_metadata_v1 *expected) {
    struct stat info;
    int result;
    do { result = fstatat(root, leaf, &info, AT_SYMLINK_NOFOLLOW); }
    while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISREG(info.st_mode) || info.st_nlink != 1) { return ESTALE; }
    if (info.st_size < 0 || (size_t)info.st_dev != expected->device ||
        (size_t)info.st_ino != expected->identity || (size_t)info.st_size != expected->size ||
        (int64_t)info.st_mtim.tv_sec != expected->modified_seconds ||
        info.st_mtim.tv_nsec < 0 || info.st_mtim.tv_nsec >= 1000000000 ||
        (uint32_t)info.st_mtim.tv_nsec != expected->modified_nanos ||
        (uint32_t)info.st_mode != expected->mode) { return ESTALE; }
    return 0;
}

/* Remove only the stage link which still names our held file. Cleanup never
 * retries unlink or replaces the primary error; leftovers require inspection. */
static uint32_t fy_linux_edit_unlink(int stage, const struct stat *held) {
    struct stat named;
    int result;
    do { result = fstatat(stage, "body", &named, AT_SYMLINK_NOFOLLOW); }
    while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISREG(named.st_mode) || named.st_dev != held->st_dev || named.st_ino != held->st_ino) {
        return ESTALE;
    }
    return unlinkat(stage, "body", 0) == 0 ? 0 : (uint32_t)errno;
}

uint32_t fairyc_private_linux_edit_file_publish_v1(uint32_t handle, uint32_t stage,
        uint32_t root, const uint8_t *leaf, size_t size, const void *metadata,
        uint32_t *cleanup) {
    if (metadata == NULL || cleanup == NULL || metadata == (const void *)cleanup) { return EINVAL; }
    *cleanup = 0;
    int fd = fy_linux_root_fd(handle), from = fy_linux_root_fd(stage), to = fy_linux_root_fd(root);
    if (fd < 0 || from < 0 || to < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    struct fy_linux_root_metadata_v1 expected;
    memcpy(&expected, metadata, sizeof expected);
    if (!S_ISREG(expected.mode) || expected.modified_nanos >= 1000000000u) { return EINVAL; }
    if ((expected.mode & 0200u) == 0) { return EACCES; }
    struct stat file, source, destination;
    uint32_t code = fy_linux_edit_stat(fd, &file);
    if (code != 0) { return code; }
    if (!S_ISREG(file.st_mode) || file.st_nlink != 0 ||
        ((uint32_t)file.st_mode & 0777u) != (expected.mode & 0777u)) { return EINVAL; }
    code = fy_linux_edit_stat(from, &source);
    if (code != 0) { return code; }
    code = fy_linux_edit_stat(to, &destination);
    if (code != 0) { return code; }
    if (!S_ISDIR(source.st_mode) || !S_ISDIR(destination.st_mode) ||
        (source.st_dev == destination.st_dev && source.st_ino == destination.st_ino)) { return EINVAL; }
    if (source.st_dev != destination.st_dev) { return EXDEV; }
    code = fy_linux_edit_current(to, (const char *)leaf, &expected);
    if (code != 0) { return code; }
    code = fy_linux_root_publish_new(handle, stage, (const uint8_t *)"body", 5);
    if (code != 0) { return code; }
    code = fy_linux_edit_current(to, (const char *)leaf, &expected);
    if (code == 0 && renameat(from, "body", to, (const char *)leaf) == 0) { return 0; }
    if (code == 0) { code = (uint32_t)errno; }
    *cleanup = fy_linux_edit_unlink(from, &file);
    return code;
}
