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
