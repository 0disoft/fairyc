#include <limits.h>
#include <stdint.h>
#include <string.h>

struct fy_std_file_storage_v0 {
    uintptr_t handle;
    uint8_t open;
};

struct fy_std_file_open_options_v0 {
    uint8_t read;
    uint8_t write;
    uint8_t create;
    uint8_t truncate;
    uint8_t exclusive;
    uint8_t append;
};

struct fy_std_file_position_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t position;
};

struct fy_std_file_size_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t size;
};

_Static_assert(sizeof(off_t) == sizeof(int64_t), "Fairy C Linux File requires 64-bit off_t");
_Static_assert((off_t)-1 < (off_t)0, "Fairy C Linux File requires signed off_t");
_Static_assert(
    sizeof(struct fy_std_file_position_result_v0) == (size_t)16,
    "Fairy C Linux File position result layout drift"
);
_Static_assert(
    sizeof(struct fy_std_file_size_result_v0) == (size_t)16,
    "Fairy C Linux File size result layout drift"
);

enum { FY_STD_FILE_PATH_CAPACITY_V0 = 4096 };

static int fy_std_file_storage_valid_v0(const struct fy_std_file_storage_v0 *file) {
    return file != NULL &&
           ((file->open == UINT8_C(0) && file->handle == (uintptr_t)0) ||
            (file->open == UINT8_C(1) && file->handle > (uintptr_t)0 &&
             file->handle - (uintptr_t)1 <= (uintptr_t)INT_MAX));
}

static int fy_std_file_storage_load_v0(
    const void *storage,
    struct fy_std_file_storage_v0 *file
) {
    if (storage == NULL || file == NULL) {
        return 0;
    }
    memcpy(file, storage, sizeof *file);
    return fy_std_file_storage_valid_v0(file);
}

static void fy_std_file_storage_store_v0(
    void *storage,
    const struct fy_std_file_storage_v0 *file
) {
    memcpy(storage, file, sizeof *file);
}

static int fy_std_file_path_storage_length_v0(const uint8_t *path, size_t *storage_length) {
    if (path == NULL || storage_length == NULL) {
        return 0;
    }
    for (size_t index = 0; index < (size_t)FY_STD_FILE_PATH_CAPACITY_V0; ++index) {
        if (path[index] == UINT8_C(0)) {
            if (index == 0) {
                return 0;
            }
            *storage_length = index + (size_t)1;
            return 1;
        }
    }
    return 0;
}

static int fy_std_file_open_flags_v0(
    const struct fy_std_file_open_options_v0 *options,
    int *flags
) {
    if (options == NULL || flags == NULL || options->read > UINT8_C(1) ||
        options->write > UINT8_C(1) || options->create > UINT8_C(1) ||
        options->truncate > UINT8_C(1) || options->exclusive > UINT8_C(1) ||
        options->append > UINT8_C(1) ||
        (options->read == UINT8_C(0) && options->write == UINT8_C(0)) ||
        (options->truncate != UINT8_C(0) && options->write == UINT8_C(0)) ||
        (options->exclusive != UINT8_C(0) &&
         (options->write == UINT8_C(0) || options->create == UINT8_C(0) ||
          options->truncate != UINT8_C(0))) ||
        (options->append != UINT8_C(0) &&
         (options->write == UINT8_C(0) || options->truncate != UINT8_C(0) ||
          options->exclusive != UINT8_C(0)))) {
        return 0;
    }
    *flags = options->read != UINT8_C(0) && options->write != UINT8_C(0) ?
        O_RDWR : (options->write != UINT8_C(0) ? O_WRONLY : O_RDONLY);
    if (options->create != UINT8_C(0)) {
        *flags |= O_CREAT;
    }
    if (options->truncate != UINT8_C(0)) {
        *flags |= O_TRUNC;
    }
    if (options->exclusive != UINT8_C(0)) {
        *flags |= O_EXCL;
    }
    if (options->append != UINT8_C(0)) {
        *flags |= O_APPEND;
    }
    return 1;
}

static int fy_std_file_storage_to_linux_v0(
    const void *storage,
    struct fy_std_linux_file_v0 *file
) {
    struct fy_std_file_storage_v0 stored;
    if (!fy_std_file_storage_load_v0(storage, &stored) || file == NULL) {
        return 0;
    }
    file->descriptor = stored.open != UINT8_C(0) ? (int)(stored.handle - (uintptr_t)1) : -1;
    file->open = stored.open;
    return 1;
}

static void fy_std_file_storage_from_linux_v0(
    void *storage,
    const struct fy_std_linux_file_v0 *file
) {
    struct fy_std_file_storage_v0 stored = {
        file->open != UINT8_C(0) ? (uintptr_t)(unsigned int)file->descriptor + (uintptr_t)1 :
                                  (uintptr_t)0,
        file->open
    };
    fy_std_file_storage_store_v0(storage, &stored);
}

static struct fy_std_file_position_result_v0 fy_std_file_position_result_v0(
    uint32_t status,
    int os_code,
    size_t position
) {
    struct fy_std_file_position_result_v0 result = {
        status,
        (uint32_t)os_code,
        position
    };
    return result;
}

static struct fy_std_file_size_result_v0 fy_std_file_size_result_v0(
    uint32_t status,
    int os_code,
    size_t size
) {
    struct fy_std_file_size_result_v0 result = {
        status,
        (uint32_t)os_code,
        size
    };
    return result;
}

struct fy_std_file_result_v0 fy_std_file_open_v0(
    const uint8_t *path,
    const struct fy_std_file_open_options_v0 *options,
    void *output_storage
) {
    size_t storage_length = 0;
    int flags = 0;
    struct fy_std_file_storage_v0 output;
    if (!fy_std_file_path_storage_length_v0(path, &storage_length) ||
        !fy_std_file_open_flags_v0(options, &flags) ||
        !fy_std_file_storage_load_v0(output_storage, &output) ||
        output.open != UINT8_C(0)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    (void)storage_length;
    int descriptor;
    do {
        descriptor = open((const char *)path, flags, (mode_t)0666);
    } while (descriptor < 0 && errno == EINTR);
    if (descriptor < 0) {
        return fy_std_file_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    output.handle = (uintptr_t)(unsigned int)descriptor + (uintptr_t)1;
    output.open = UINT8_C(1);
    fy_std_file_storage_store_v0(output_storage, &output);
    return fy_std_file_result_v0(FY_STD_FILE_OK_V0, 0, 0);
}

struct fy_std_file_result_v0 fy_std_file_open_utf8_v0(
    const uint8_t *path,
    size_t length,
    const struct fy_std_file_open_options_v0 *options,
    void *output_storage
) {
    if (path == NULL || length == 0 || length >= (size_t)FY_STD_FILE_PATH_CAPACITY_V0) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    for (size_t index = 0; index < length; ++index) {
        if (path[index] == UINT8_C(0)) {
            return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
        }
    }
    uint8_t terminated[FY_STD_FILE_PATH_CAPACITY_V0];
    memcpy(terminated, path, length);
    terminated[length] = UINT8_C(0);
    return fy_std_file_open_v0(terminated, options, output_storage);
}

struct fy_std_file_result_v0 fy_std_file_read_v0(
    void *file_storage,
    uint8_t *buffer,
    size_t length
) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    return fy_std_linux_file_read_v0(&file, buffer, length);
}

struct fy_std_file_position_result_v0 fy_std_file_position_v0(void *file_storage) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    off_t position;
    do {
        position = lseek(file.descriptor, (off_t)0, SEEK_CUR);
    } while (position < (off_t)0 && errno == EINTR);
    if (position < (off_t)0) {
        return fy_std_file_position_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_position_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)position);
}

struct fy_std_file_size_result_v0 fy_std_file_size_v0(const void *file_storage) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_size_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_file_size_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    struct stat metadata;
    int status;
    do {
        status = fstat(file.descriptor, &metadata);
    } while (status < 0 && errno == EINTR);
    if (status < 0) {
        return fy_std_file_size_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    if (metadata.st_size < (off_t)0) {
        return fy_std_file_size_result_v0(FY_STD_FILE_IO_ERROR_V0, EOVERFLOW, 0);
    }
    return fy_std_file_size_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)metadata.st_size);
}

struct fy_std_file_position_result_v0 fy_std_file_seek_start_v0(
    void *file_storage,
    size_t offset
) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    if (offset > (size_t)INT64_MAX) {
        return fy_std_file_position_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    off_t position;
    do {
        position = lseek(file.descriptor, (off_t)offset, SEEK_SET);
    } while (position < (off_t)0 && errno == EINTR);
    if (position < (off_t)0) {
        return fy_std_file_position_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_position_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)position);
}

static struct fy_std_file_position_result_v0 fy_std_file_seek_relative_v0(
    void *file_storage,
    int32_t offset,
    int origin
) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    if (file.open == UINT8_C(0)) {
        return fy_std_file_position_result_v0(FY_STD_FILE_CLOSED_V0, 0, 0);
    }
    off_t position;
    do {
        position = lseek(file.descriptor, (off_t)offset, origin);
    } while (position < (off_t)0 && errno == EINTR);
    if (position < (off_t)0) {
        return fy_std_file_position_result_v0(FY_STD_FILE_IO_ERROR_V0, errno, 0);
    }
    return fy_std_file_position_result_v0(FY_STD_FILE_OK_V0, 0, (size_t)position);
}

struct fy_std_file_position_result_v0 fy_std_file_seek_current_v0(
    void *file_storage,
    int32_t offset
) {
    return fy_std_file_seek_relative_v0(file_storage, offset, SEEK_CUR);
}

struct fy_std_file_position_result_v0 fy_std_file_seek_end_v0(
    void *file_storage,
    int32_t offset
) {
    return fy_std_file_seek_relative_v0(file_storage, offset, SEEK_END);
}

struct fy_std_file_result_v0 fy_std_file_write_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    return fy_std_linux_file_write_v0(&file, buffer, length);
}

struct fy_std_file_result_v0 fy_std_file_write_all_v0(
    void *file_storage,
    const uint8_t *buffer,
    size_t length
) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    return fy_std_linux_file_write_all_v0(&file, buffer, length);
}

struct fy_std_file_result_v0 fy_std_file_sync_v0(void *file_storage) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    return fy_std_linux_file_sync_v0(&file);
}

struct fy_std_file_result_v0 fy_std_file_close_v0(void *file_storage) {
    struct fy_std_linux_file_v0 file;
    if (!fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        return fy_std_file_result_v0(FY_STD_FILE_INVALID_ARGUMENT_V0, 0, 0);
    }
    struct fy_std_file_result_v0 result = fy_std_linux_file_close_v0(&file);
    fy_std_file_storage_from_linux_v0(file_storage, &file);
    return result;
}

uint32_t fy_std_file_drop_v0(void *file_storage) {
    struct fy_std_linux_file_v0 file;
    if (fy_std_file_storage_to_linux_v0(file_storage, &file)) {
        fy_std_linux_file_drop_v0(&file);
        fy_std_file_storage_from_linux_v0(file_storage, &file);
    }
    return UINT32_C(0);
}
