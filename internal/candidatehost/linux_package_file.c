//go:build ignore

/* Private package reads reject hard links. Generic rooted I/O remains unchanged.
 * One held-file observation supplies both identity and the package-only check. */
_Static_assert(EMLINK == 31, "private Linux package hard-link error");
uint32_t fairyc_private_linux_package_file_info_v1(uint32_t handle, void *output) {
    if (output == NULL) { return EINVAL; }
    memset(output, 0, sizeof(struct fy_linux_root_metadata_v1));
    int fd = fy_linux_root_fd(handle);
    if (fd < 0) { return EINVAL; }
    struct stat info;
    int result;
    do { result = fstat(fd, &info); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISREG(info.st_mode) || info.st_size < 0 ||
        info.st_mtim.tv_nsec < 0 || info.st_mtim.tv_nsec >= 1000000000) { return EINVAL; }
    if (info.st_nlink != 1) { return EMLINK; }
    struct fy_linux_root_metadata_v1 value = { (size_t)info.st_dev, (size_t)info.st_ino,
        (size_t)info.st_size, (int64_t)info.st_mtim.tv_sec, (uint32_t)info.st_mtim.tv_nsec, (uint32_t)info.st_mode };
    memcpy(output, &value, sizeof value);
    return 0;
}
