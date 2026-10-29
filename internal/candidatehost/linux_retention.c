/* Appended after linux_rooted_file.c. Requires cooperating private publishers;
 * flock is advisory, not protection against arbitrary external namespace edits. */
uint32_t fairyc_private_linux_root_file_erase_if_v1(uint32_t root, const uint8_t *leaf,
        size_t size, const void *expected, uint32_t *removed, uint32_t *close_code) {
    if (removed == NULL || close_code == NULL) { return EINVAL; }
    *removed = 0; *close_code = 0;
    if (expected == NULL || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    struct fy_linux_root_metadata_v1 observed;
    memcpy(&observed, expected, sizeof observed);
    if (!S_ISREG(observed.mode) || observed.identity == 0 || observed.modified_nanos >= 1000000000) { return EINVAL; }
    uint32_t owned = 0;
    uint32_t code = fy_linux_root_lock(root, &owned);
    if (code != 0) { return code; }
    int fd = fy_linux_root_fd(owned);
    struct stat actual;
    int result;
    do { result = fstatat(fd, (const char *)leaf, &actual, AT_SYMLINK_NOFOLLOW); }
    while (result < 0 && errno == EINTR);
    if (result < 0) {
        code = errno == ENOENT ? 0 : (uint32_t)errno;
    } else if (S_ISREG(actual.st_mode) && actual.st_nlink == 1 && actual.st_size >= 0 &&
        (size_t)actual.st_dev == observed.device && (size_t)actual.st_ino == observed.identity &&
        (size_t)actual.st_size == observed.size && (int64_t)actual.st_mtim.tv_sec == observed.modified_seconds &&
        actual.st_mtim.tv_nsec >= 0 && (uint32_t)actual.st_mtim.tv_nsec == observed.modified_nanos &&
        (uint32_t)actual.st_mode == observed.mode) {
        /* No automatic retry after an uncertain namespace mutation. */
        if (unlinkat(fd, (const char *)leaf, 0) == 0) { *removed = 1; }
        else { code = errno == ENOENT ? 0 : (uint32_t)errno; }
    }
    if (close(fd) != 0) { *close_code = (uint32_t)errno; }
    return code;
}
