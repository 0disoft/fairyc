//go:build ignore

/* Appended after linux_rooted_file.c. Private package directories only: callers
 * hold the workspace fetch lock and reject case aliases before entering here.
 * That lock coordinates publishers, not hostile external namespace writers. */
#ifndef RENAME_NOREPLACE
#define RENAME_NOREPLACE 1u
#endif

static void fy_linux_package_close(uint32_t handle, uint32_t *cleanup) {
    uint32_t code = fairyc_private_linux_root_file_close_v1(handle);
    if (*cleanup == 0) { *cleanup = code; }
}

static uint32_t fy_linux_package_open_directory(int parent, const char *name,
        uint32_t *output, uint32_t *cleanup) {
    int fd;
    do { fd = openat(parent, name, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC, 0600); }
    while (fd < 0 && errno == EINTR);
    if (fd < 0) { return (uint32_t)errno; }
    struct stat info;
    int result;
    do { result = fstat(fd, &info); } while (result < 0 && errno == EINTR);
    uint32_t code = result < 0 ? (uint32_t)errno : !S_ISDIR(info.st_mode) ? EINVAL : 0;
    if (code != 0) { fy_linux_package_close((uint32_t)fd + 1u, cleanup); return code; }
    *output = (uint32_t)fd + 1u;
    return 0;
}

static uint32_t fy_linux_package_directory(uint32_t root, const uint8_t *leaf,
        size_t size, int create, uint32_t *child, uint32_t *parent, uint32_t *cleanup) {
    if (child == NULL || parent == NULL || cleanup == NULL ||
        child == parent || child == cleanup || parent == cleanup) { return EINVAL; }
    *child = 0; *parent = 0; *cleanup = 0;
    int fd = fy_linux_root_fd(root);
    if (fd < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    uint32_t retained = 0;
    uint32_t code = fy_linux_package_open_directory(fd, ".", &retained, cleanup);
    if (code != 0) { return code; }
    fd = fy_linux_root_fd(retained);
    /* Do not replay an uncertain namespace mutation, including EINTR. */
    if (create && mkdirat(fd, (const char *)leaf, 0700) != 0) {
        code = (uint32_t)errno;
        fy_linux_package_close(retained, cleanup);
        return code;
    }
    code = fy_linux_package_open_directory(fd, (const char *)leaf, child, cleanup);
    if (code != 0) {
        /* Only a newly created, empty entry may be rolled back. An edit failure
         * never removes the existing name. No retry after uncertain unlink. */
        if (create && unlinkat(fd, (const char *)leaf, AT_REMOVEDIR) != 0) {
            if (*cleanup == 0) { *cleanup = (uint32_t)errno; }
        }
        fy_linux_package_close(retained, cleanup);
        return code;
    }
    *parent = retained;
    return 0;
}

uint32_t fairyc_private_linux_package_directory_create_v1(uint32_t root,
        const uint8_t *leaf, size_t size, uint32_t *child, uint32_t *parent, uint32_t *cleanup) {
    return fy_linux_package_directory(root, leaf, size, 1, child, parent, cleanup);
}

uint32_t fairyc_private_linux_package_directory_edit_v1(uint32_t root,
        const uint8_t *leaf, size_t size, uint32_t *child, uint32_t *parent, uint32_t *cleanup) {
    return fy_linux_package_directory(root, leaf, size, 0, child, parent, cleanup);
}

/* A retained fd survives parent renames. Recheck the final directory entry
 * against that fd before removing or publishing it; never follow a symlink. */
static uint32_t fy_linux_package_same_directory(uint32_t child, uint32_t parent,
        const uint8_t *leaf, size_t size) {
    int fd = fy_linux_root_fd(child), root = fy_linux_root_fd(parent);
    if (fd < 0 || root < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    struct stat held, named;
    int result;
    do { result = fstat(fd, &held); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISDIR(held.st_mode)) { return EINVAL; }
    do { result = fstatat(root, (const char *)leaf, &named, AT_SYMLINK_NOFOLLOW); }
    while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISDIR(named.st_mode) || held.st_dev != named.st_dev || held.st_ino != named.st_ino) {
        return ESTALE;
    }
    return 0;
}

uint32_t fairyc_private_linux_package_directory_publish_v1(uint32_t child,
        uint32_t parent, const uint8_t *from, size_t from_size,
        uint32_t destination, const uint8_t *leaf, size_t size) {
    int root = fy_linux_root_fd(destination);
    if (root < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    uint32_t code = fy_linux_package_same_directory(child, parent, from, from_size);
    if (code != 0) { return code; }
    /* No renameat fallback: it could silently replace an existing empty tree.
     * Unsupported filesystems/kernels fail without weakening no-replacement. */
    return renameat2(fy_linux_root_fd(parent), (const char *)from,
        root, (const char *)leaf, RENAME_NOREPLACE) == 0 ? 0 : (uint32_t)errno;
}

uint32_t fairyc_private_linux_package_directory_erase_v1(uint32_t child,
        uint32_t parent, const uint8_t *leaf, size_t size) {
    uint32_t code = fy_linux_package_same_directory(child, parent, leaf, size);
    if (code != 0) { return code; }
    /* The caller clears the private tree under its own entry/depth budgets.
     * This primitive never recurses and the kernel refuses nonempty trees. */
    return unlinkat(fy_linux_root_fd(parent), (const char *)leaf, AT_REMOVEDIR) == 0
        ? 0 : (uint32_t)errno;
}
