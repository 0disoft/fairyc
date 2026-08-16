#define _POSIX_C_SOURCE 200809L

#include <errno.h>
#include <fcntl.h>
#include <stddef.h>
#include <stdint.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

enum fy_std_file_status_v0 {
    FY_STD_FILE_OK_V0 = 0,
    FY_STD_FILE_IO_ERROR_V0 = 1,
    FY_STD_FILE_INVALID_ARGUMENT_V0 = 2,
    FY_STD_FILE_CLOSED_V0 = 3,
    FY_STD_FILE_NO_PROGRESS_V0 = 4
};

struct fy_std_file_result_v0 {
    uint32_t status;
    int32_t error_number;
    size_t count;
};

struct fy_std_linux_file_v0 {
    int descriptor;
    uint8_t open;
};

struct fy_std_linux_open_options_v0 {
    uint8_t read;
    uint8_t write;
    uint8_t create;
    uint8_t truncate;
    uint32_t mode;
};

static struct fy_std_file_result_v0 fy_std_file_result_v0(uint32_t status, int error_number, size_t count) {
    struct fy_std_file_result_v0 result = {status, (int32_t)error_number, count};
    return result;
}

static int fy_std_linux_file_valid_v0(const struct fy_std_linux_file_v0 *file) {
    return file != NULL && ((file->open == 0 && file->descriptor == -1) ||
                            (file->open == 1 && file->descriptor >= 0));
}

static int fy_std_linux_path_valid_v0(const uint8_t *path, size_t storage_length) {
    if (path == NULL || storage_length == 0 || path[storage_length - 1] != 0) {
        return 0;
    }
    for (size_t index = 0; index + 1 < storage_length; ++index) {
        if (path[index] == 0) {
            return 0;
        }
    }
    return 1;
}

static int fy_std_linux_open_flags_v0(const struct fy_std_linux_open_options_v0 *options, int *flags) {
    if (options == NULL || flags == NULL || options->read > 1 || options->write > 1 ||
        options->create > 1 || options->truncate > 1 || (!options->read && !options->write) ||
        (options->truncate && !options->write) || options->mode > UINT32_C(0777)) {
        return 0;
    }
    *flags = options->read && options->write ? O_RDWR : (options->write ? O_WRONLY : O_RDONLY);
    if (options->create) {
        *flags |= O_CREAT;
    }
    if (options->truncate) {
        *flags |= O_TRUNC;
    }
    return 1;
}

struct fy_std_file_result_v0 fy_std_linux_file_open_v0(
    const uint8_t *path,
    size_t path_storage_length,
    const struct fy_std_linux_open_options_v0 *options,
    struct fy_std_linux_file_v0 *output
) {
    int flags = 0;
    if (!fy_std_linux_path_valid_v0(path, path_storage_length) || output == NULL ||
        !fy_std_linux_open_flags_v0(options, &flags)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    int descriptor;
    do {
        descriptor = open((const char *)path, flags, (mode_t)options->mode);
    } while (descriptor < 0 && errno == EINTR);
    if (descriptor < 0) {
        return fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    output->descriptor = descriptor;
    output->open = 1;
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, 0);
}

struct fy_std_file_result_v0 fy_std_linux_file_read_v0(
    struct fy_std_linux_file_v0 *file,
    uint8_t *buffer,
    size_t length
) {
    if (!fy_std_linux_file_valid_v0(file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (!file->open) {
        return fy_std_file_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    if (length != 0 && buffer == NULL) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    ssize_t count;
    do {
        count = read(file->descriptor, buffer, length);
    } while (count < 0 && errno == EINTR);
    if (count < 0) {
        return fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)count);
}

struct fy_std_file_result_v0 fy_std_linux_file_write_v0(
    struct fy_std_linux_file_v0 *file,
    const uint8_t *buffer,
    size_t length
) {
    if (!fy_std_linux_file_valid_v0(file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (!file->open) {
        return fy_std_file_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    if (length != 0 && buffer == NULL) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    ssize_t count;
    do {
        count = write(file->descriptor, buffer, length);
    } while (count < 0 && errno == EINTR);
    if (count < 0) {
        return fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)count);
}

struct fy_std_file_result_v0 fy_std_linux_file_write_all_v0(
    struct fy_std_linux_file_v0 *file,
    const uint8_t *buffer,
    size_t length
) {
    if (!fy_std_linux_file_valid_v0(file) || !file->open || (length != 0 && buffer == NULL)) {
        if (fy_std_linux_file_valid_v0(file) && !file->open) {
            return fy_std_file_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
        }
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    size_t written = 0;
    while (written < length) {
        struct fy_std_file_result_v0 result = fy_std_linux_file_write_v0(file, buffer + written, length - written);
        if (result.status != FY_STD_FILE_OK_V0) {
            result.count = written;
            return result;
        }
        if (result.count == 0) {
            return fy_std_file_result_v0(FY_STD_FILE_NO_PROGRESS_V0, 0, written);
        }
        written += result.count;
    }
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, written);
}

struct fy_std_file_result_v0 fy_std_linux_file_sync_v0(struct fy_std_linux_file_v0 *file) {
    if (!fy_std_linux_file_valid_v0(file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (!file->open) {
        return fy_std_file_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    int status;
    do {
        status = fsync(file->descriptor);
    } while (status < 0 && errno == EINTR);
    return status == 0 ? fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, 0)
                       : fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
}

struct fy_std_file_result_v0 fy_std_linux_file_close_v0(struct fy_std_linux_file_v0 *file) {
    if (!fy_std_linux_file_valid_v0(file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (!file->open) {
        return fy_std_file_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    int descriptor = file->descriptor;
    file->descriptor = -1;
    file->open = 0;
    if (close(descriptor) != 0) {
        return fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, 0);
}

void fy_std_linux_file_drop_v0(struct fy_std_linux_file_v0 *file) {
    if (fy_std_linux_file_valid_v0(file) && file->open) {
        int descriptor = file->descriptor;
        file->descriptor = -1;
        file->open = 0;
        (void)close(descriptor);
    }
}
