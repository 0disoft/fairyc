/* Private GNU/Linux rooted I/O. Fairy owns path admission and resource lifetime. */
#ifndef _GNU_SOURCE
#define _GNU_SOURCE 1
#endif
#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <stdint.h>
#include <stddef.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

/* Handles encode fd+1: zero is empty, and an OS descriptor of zero is owned. */
static int fy_linux_root_fd(uint32_t handle) {
    return handle != 0 && handle <= (uint32_t)INT_MAX + 1u ? (int)(handle - 1u) : -1;
}
static int fy_linux_root_path(const uint8_t *data, size_t size, int leaf) {
    if (data == NULL || size < 2 || size > (leaf ? 256u : 4096u) ||
        data[size-1] != 0 || memchr(data, 0, size-1) != NULL) { return 0; }
    /* A trailing slash would make O_NOFOLLOW follow a final symlink to a directory. */
    if (!leaf && size > 2 && data[size-2] == '/') { return 0; }
    if (leaf && (memchr(data, '/', size-1) != NULL ||
        (size == 2 && data[0] == '.') || (size == 3 && data[0] == '.' && data[1] == '.'))) { return 0; }
    return 1;
}
static uint32_t fy_linux_root_open(int parent, const char *path, int flags, int directory, uint32_t *output) {
    int fd;
    do { fd = openat(parent, path, flags, 0600); } while (fd < 0 && errno == EINTR);
    if (fd < 0) { return (uint32_t)errno; }
    struct stat info;
    int result;
    do { result = fstat(fd, &info); } while (result < 0 && errno == EINTR);
    uint32_t error = result < 0 ? (uint32_t)errno :
        (directory ? !S_ISDIR(info.st_mode) : !S_ISREG(info.st_mode)) ? EINVAL : 0;
    if (error != 0) { (void)close(fd); return error; }
    *output = (uint32_t)fd + 1u;
    return 0;
}
uint32_t fairyc_private_linux_directory_open_v1(const uint8_t *path, size_t size, uint32_t *output) {
    if (output == NULL) { return EINVAL; } *output = 0;
    if (!fy_linux_root_path(path, size, 0)) { return EINVAL; }
    return fy_linux_root_open(AT_FDCWD, (const char *)path,
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC, 1, output);
}
uint32_t fairyc_private_linux_root_directory_open_v1(uint32_t root, const uint8_t *leaf, size_t size, uint32_t *output) {
    if (output == NULL) { return EINVAL; } *output = 0;
    int fd = fy_linux_root_fd(root);
    if (fd < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    return fy_linux_root_open(fd, (const char *)leaf,
        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC, 1, output);
}
uint32_t fairyc_private_linux_root_file_open_v1(uint32_t root, const uint8_t *leaf, size_t size, uint32_t *output) {
    if (output == NULL) { return EINVAL; } *output = 0;
    int fd = fy_linux_root_fd(root);
    if (fd < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    /* A substituted FIFO must not block before the regular-file check. */
    return fy_linux_root_open(fd, (const char *)leaf,
        O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_NOCTTY | O_CLOEXEC, 0, output);
}
struct fy_linux_root_metadata_v1 {
    size_t device, identity, size;
    int64_t modified_seconds;
    uint32_t modified_nanos, mode;
};
_Static_assert(sizeof(struct fy_linux_root_metadata_v1) == 40, "private Linux amd64 metadata");
_Static_assert(sizeof(size_t) == 8 && sizeof(((struct stat *)0)->st_size) <= 8, "private Linux size representation");
_Static_assert(offsetof(struct fy_linux_root_metadata_v1, mode) == 36, "private Linux metadata mode");
uint32_t fairyc_private_linux_root_file_info_v1(uint32_t handle, void *output) {
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
    struct fy_linux_root_metadata_v1 value = { (size_t)info.st_dev, (size_t)info.st_ino,
        (size_t)info.st_size, (int64_t)info.st_mtim.tv_sec, (uint32_t)info.st_mtim.tv_nsec, (uint32_t)info.st_mode };
    memcpy(output, &value, sizeof value); return 0;
}
uint32_t fairyc_private_linux_root_file_read_v1(uint32_t handle, uint8_t *data, size_t size, size_t *copied) {
    if (copied == NULL) { return EINVAL; } *copied = 0;
    int fd = fy_linux_root_fd(handle);
    if (fd < 0 || size > 65536 || (data == NULL && size != 0)) { return EINVAL; }
    if (size == 0) { return 0; }
    ssize_t result;
    do { result = read(fd, data, size); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if ((size_t)result > size) { return EIO; }
    *copied = (size_t)result; return 0;
}
/* Linux releases a descriptor even on close EINTR; never retry an uncertain close. */
uint32_t fairyc_private_linux_root_file_close_v1(uint32_t handle) {
    if (handle == 0) { return 0; }
    int fd = fy_linux_root_fd(handle);
    if (fd < 0) { return EINVAL; }
    return close(fd) == 0 ? 0 : (uint32_t)errno;
}

/* No staging pathname can be replaced underneath cleanup or publication. */
uint32_t fairyc_private_linux_root_file_create_v1(uint32_t root, uint32_t *output) {
    if (output == NULL) { return EINVAL; } *output = 0;
    int fd = fy_linux_root_fd(root);
    if (fd < 0) { return EINVAL; }
    return fy_linux_root_open(fd, ".", O_TMPFILE | O_RDWR | O_CLOEXEC, 0, output);
}
uint32_t fairyc_private_linux_root_file_write_v1(uint32_t handle, const uint8_t *data, size_t size, size_t *copied) {
    if (copied == NULL) { return EINVAL; } *copied = 0;
    int fd = fy_linux_root_fd(handle);
    if (fd < 0 || size > 65536 || (data == NULL && size != 0)) { return EINVAL; }
    if (size == 0) { return 0; }
    ssize_t result;
    do { result = write(fd, data, size); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if ((size_t)result > size) { return EIO; }
    *copied = (size_t)result; return 0;
}
uint32_t fairyc_private_linux_root_file_sync_v1(uint32_t handle) {
    int fd = fy_linux_root_fd(handle);
    if (fd < 0) { return EINVAL; }
    int result;
    do { result = fsync(fd); } while (result < 0 && errno == EINTR);
    return result == 0 ? 0 : (uint32_t)errno;
}
uint32_t fairyc_private_linux_root_file_publish_new_v1(uint32_t handle, uint32_t root, const uint8_t *leaf, size_t size) {
    int fd = fy_linux_root_fd(handle), parent = fy_linux_root_fd(root);
    if (fd < 0 || parent < 0 || !fy_linux_root_path(leaf, size, 1)) { return EINVAL; }
    struct stat info;
    int result;
    do { result = fstat(fd, &info); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (!S_ISREG(info.st_mode) || info.st_nlink != 0) { return EINVAL; }
    if (linkat(fd, "", parent, (const char *)leaf, AT_EMPTY_PATH) == 0) { return 0; }
    int failure = errno;
    if (failure != EPERM && failure != EINVAL && failure != ENOENT) { return (uint32_t)failure; }
    /* Unprivileged documented O_TMPFILE publication, requiring procfs. */
    char path[64];
    int length = snprintf(path, sizeof path, "/proc/self/fd/%d", fd);
    if (length < 0 || (size_t)length >= sizeof path) { return EINVAL; }
    return linkat(AT_FDCWD, path, parent, (const char *)leaf, AT_SYMLINK_FOLLOW) == 0 ? 0 : (uint32_t)errno;
}
