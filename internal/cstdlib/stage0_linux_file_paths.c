//go:build ignore

#include <linux/fs.h>
#include <sys/syscall.h>

/* The public source bundle may already have selected POSIX headers before this
   component. Declare the GNU syscall entry without changing feature macros. */
extern long syscall(long number, ...);

/* Compatibility ABI spellings used by the target-neutral std::fs facade. */
struct fy_std_file_result_v0 fy_std_windows_file_rename_v0(
    const uint8_t *source, const uint8_t *destination
) {
    size_t source_length, destination_length;
    if (!fy_std_file_path_storage_length_v0(source, &source_length) ||
        !fy_std_file_path_storage_length_v0(destination, &destination_length)) {
        return fy_std_file_result_v0(2, 0, 0);
    }
    struct stat observed;
    if (lstat((const char *)source, &observed) != 0) {
        return fy_std_file_result_v0(1, errno, 0);
    }
    if (S_ISDIR(observed.st_mode)) { return fy_std_file_result_v0(2, 0, 0); }
#if defined(SYS_renameat2)
    if (syscall(SYS_renameat2, AT_FDCWD, source, AT_FDCWD, destination, RENAME_NOREPLACE) != 0) {
        return fy_std_file_result_v0(1, errno, 0);
    }
    return fy_std_file_result_v0(0, 0, 0);
#else
    return fy_std_file_result_v0(1, ENOSYS, 0);
#endif
}

struct fy_std_file_result_v0 fy_std_windows_file_hard_link_v0(
    const uint8_t *source, const uint8_t *destination
) {
    size_t source_length, destination_length;
    if (!fy_std_file_path_storage_length_v0(source, &source_length) ||
        !fy_std_file_path_storage_length_v0(destination, &destination_length)) {
        return fy_std_file_result_v0(2, 0, 0);
    }
    if (link((const char *)source, (const char *)destination) != 0) {
        return fy_std_file_result_v0(1, errno, 0);
    }
    return fy_std_file_result_v0(0, 0, 0);
}

/* Exclusive destination creation never truncates an existing path. On an I/O
   failure, count reports bytes written and the partial new destination remains.
   File creation/copying does not promise an atomic snapshot of a changing file. */
struct fy_std_file_result_v0 fy_std_windows_file_copy_bytes_v0(
    const uint8_t *source, const uint8_t *destination
) {
    size_t source_length, destination_length;
    if (!fy_std_file_path_storage_length_v0(source, &source_length) ||
        !fy_std_file_path_storage_length_v0(destination, &destination_length)) {
        return fy_std_file_result_v0(2, 0, 0);
    }
    int input;
    do { input = open((const char *)source, O_RDONLY | O_CLOEXEC | O_NONBLOCK); }
    while (input < 0 && errno == EINTR);
    if (input < 0) { return fy_std_file_result_v0(1, errno, 0); }
    struct stat observed;
    int sampled;
    do { sampled = fstat(input, &observed); } while (sampled != 0 && errno == EINTR);
    if (sampled != 0 || !S_ISREG(observed.st_mode)) {
        int code = sampled != 0 ? errno : EINVAL;
        (void)close(input);
        return fy_std_file_result_v0(sampled != 0 ? 1u : 2u, code, 0);
    }
    int output;
    do { output = open((const char *)destination, O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC, (mode_t)0666); }
    while (output < 0 && errno == EINTR);
    if (output < 0) {
        int code = errno;
        (void)close(input);
        return fy_std_file_result_v0(1, code, 0);
    }
    struct fy_std_file_result_v0 result = fy_std_file_result_v0(0, 0, 0);
    uint8_t buffer[65536];
    for (;;) {
        ssize_t received;
        do { received = read(input, buffer, sizeof buffer); } while (received < 0 && errno == EINTR);
        if (received < 0) { result.status = 1; result.error_number = errno; break; }
        if (received == 0) { break; }
        size_t written = 0;
        while (written < (size_t)received) {
            ssize_t count;
            do { count = write(output, buffer + written, (size_t)received - written); }
            while (count < 0 && errno == EINTR);
            if (count <= 0) {
                result.status = count < 0 ? 1u : 4u;
                result.error_number = count < 0 ? errno : 0;
                break;
            }
            if ((size_t)count > SIZE_MAX - result.count) {
                result.status = 1; result.error_number = EOVERFLOW; break;
            }
            written += (size_t)count;
            result.count += (size_t)count;
        }
        if (result.status != 0) { break; }
    }
    if (close(output) != 0 && result.status == 0) { result.status = 1; result.error_number = errno; }
    if (close(input) != 0 && result.status == 0) { result.status = 1; result.error_number = errno; }
    return result;
}
