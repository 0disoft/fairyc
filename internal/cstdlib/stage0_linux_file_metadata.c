//go:build ignore

struct fy_std_file_kind_result_v0 { uint32_t status, os_code, kind; };
struct fy_std_file_attribute_result_v0 { uint32_t status, os_code, enabled; };
struct fy_std_file_id_v0 {
    uint32_t volume_low, volume_high, file_low_0, file_low_1, file_high_0, file_high_1;
};

static uint32_t fy_std_linux_file_descriptor_v0(const void *storage, int *descriptor) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(storage, &file)) { return 2; }
    if (!file.open) { return 3; }
    *descriptor = file.descriptor;
    return 0;
}

static struct fy_std_file_result_v0 fy_std_linux_file_observe_v0(
    const void *storage, struct stat *observed
) {
    int descriptor;
    uint32_t status = fy_std_linux_file_descriptor_v0(storage, &descriptor);
    if (status != 0) { return fy_std_file_result_v0(status, 0, 0); }
    int sampled;
    do { sampled = fstat(descriptor, observed); } while (sampled != 0 && errno == EINTR);
    return fy_std_file_result_v0(sampled == 0 ? 0u : 1u, sampled == 0 ? 0 : errno, 0);
}

struct fy_std_file_kind_result_v0 fy_std_windows_file_kind_v0(const void *storage) {
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    struct fy_std_file_kind_result_v0 result = {sampled.status, (uint32_t)sampled.error_number, 0};
    if (sampled.status != 0) { return result; }
    result.kind = S_ISREG(observed.st_mode) || S_ISDIR(observed.st_mode) || S_ISBLK(observed.st_mode) ? 1u :
        (S_ISCHR(observed.st_mode) ? 2u : (S_ISFIFO(observed.st_mode) || S_ISSOCK(observed.st_mode) ? 3u : 0u));
    return result;
}

struct fy_std_file_kind_result_v0 fy_std_windows_file_path_kind_v0(const uint8_t *path) {
    struct fy_std_file_kind_result_v0 result = {0, 0, 0};
    size_t length;
    if (!fy_std_file_path_storage_length_v0(path, &length)) { result.status = 2; return result; }
    struct stat observed;
    if (lstat((const char *)path, &observed) != 0) { result.status = 1; result.os_code = (uint32_t)errno; return result; }
    result.kind = S_ISREG(observed.st_mode) ? 1u : (S_ISDIR(observed.st_mode) ? 2u : 3u);
    return result;
}

struct fy_std_file_attribute_result_v0 fy_std_windows_file_read_only_v0(const void *storage) {
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    struct fy_std_file_attribute_result_v0 result = {sampled.status, (uint32_t)sampled.error_number, 0};
    if (sampled.status == 0) { result.enabled = (observed.st_mode & (S_IWUSR | S_IWGRP | S_IWOTH)) == 0 ? 1u : 0u; }
    return result;
}

struct fy_std_file_attribute_result_v0 fy_std_windows_file_set_readonly_v0(void *storage, uint32_t enabled) {
    struct fy_std_file_attribute_result_v0 result = {2, 0, 0};
    if (enabled > 1) { return result; }
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    result.status = sampled.status; result.os_code = (uint32_t)sampled.error_number;
    if (sampled.status != 0) { return result; }
    int descriptor;
    result.status = fy_std_linux_file_descriptor_v0(storage, &descriptor);
    if (result.status != 0) { return result; }
    mode_t permissions = observed.st_mode & (mode_t)07777;
    permissions = enabled != 0 ? permissions & (mode_t)~(S_IWUSR | S_IWGRP | S_IWOTH) : permissions | S_IWUSR;
    int changed;
    do { changed = fchmod(descriptor, permissions); } while (changed != 0 && errno == EINTR);
    if (changed != 0) { result.status = 1; result.os_code = (uint32_t)errno; return result; }
    result.enabled = enabled;
    return result;
}

struct fy_std_file_size_result_v0 fy_std_windows_file_link_count_v0(const void *storage) {
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    if (sampled.status != 0) { return fy_std_file_size_result_v0(sampled.status, sampled.error_number, 0); }
    _Static_assert(sizeof observed.st_nlink <= sizeof(size_t), "Linux link count must fit the ABI");
    return fy_std_file_size_result_v0(0, 0, (size_t)observed.st_nlink);
}

struct fy_std_file_size_result_v0 fy_std_windows_file_alloc_size_v0(const void *storage) {
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    if (sampled.status != 0) { return fy_std_file_size_result_v0(sampled.status, sampled.error_number, 0); }
    if (observed.st_blocks < 0 || (uintmax_t)observed.st_blocks > SIZE_MAX / 512) {
        return fy_std_file_size_result_v0(1, EOVERFLOW, 0);
    }
    return fy_std_file_size_result_v0(0, 0, (size_t)observed.st_blocks * 512);
}

struct fy_std_file_result_v0 fy_std_windows_file_id_v0(const void *storage, struct fy_std_file_id_v0 *output) {
    if (output == NULL) { return fy_std_file_result_v0(2, 0, 0); }
    struct stat observed;
    struct fy_std_file_result_v0 sampled = fy_std_linux_file_observe_v0(storage, &observed);
    if (sampled.status != 0) { return sampled; }
    _Static_assert(sizeof observed.st_dev <= sizeof(uint64_t), "Linux device identity must fit the ABI");
    _Static_assert(sizeof observed.st_ino <= sizeof(uint64_t), "Linux inode identity must fit the ABI");
    uint64_t device = (uint64_t)observed.st_dev, inode = (uint64_t)observed.st_ino;
    struct fy_std_file_id_v0 identity = {
        (uint32_t)device, (uint32_t)(device >> 32), (uint32_t)inode, (uint32_t)(inode >> 32), 0, 0
    };
    memcpy(output, &identity, sizeof identity);
    return sampled;
}

struct fy_std_file_size_result_v0 fy_std_windows_file_set_len_v0(void *storage, uint64_t length) {
    int descriptor;
    uint32_t status = fy_std_linux_file_descriptor_v0(storage, &descriptor);
    if (status != 0) { return fy_std_file_size_result_v0(status, 0, 0); }
    if (length > INT64_MAX) { return fy_std_file_size_result_v0(2, 0, 0); }
    int changed;
    do { changed = ftruncate(descriptor, (off_t)length); } while (changed != 0 && errno == EINTR);
    return fy_std_file_size_result_v0(changed == 0 ? 0u : 1u, changed == 0 ? 0 : errno, changed == 0 ? (size_t)length : 0);
}
