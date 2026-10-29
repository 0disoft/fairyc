//go:build ignore

/* Appended after rooted Linux I/O; this observation never authorizes mutation. */
uint32_t fairyc_private_linux_workspace_path_kind_v1(const uint8_t *path, uint32_t *kind) {
    if (kind == NULL) { return EINVAL; }
    *kind = 0;
    if (path == NULL) { return EINVAL; }
    size_t length = strnlen((const char *)path, 4096);
    if (length == 0 || length >= 4096) { return EINVAL; }
    struct stat info;
    int result;
    do { result = lstat((const char *)path, &info); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    *kind = (uint32_t)info.st_mode;
    return 0;
}
