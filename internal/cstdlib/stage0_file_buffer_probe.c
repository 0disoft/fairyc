#ifdef NDEBUG
#undef NDEBUG
#endif
#include <assert.h>

#ifdef _WIN32
typedef struct fy_std_windows_file_v0 fy_probe_file;
typedef struct fy_std_windows_open_options_v0 fy_probe_options;
#else
typedef struct fy_std_file_storage_v0 fy_probe_file;
typedef struct fy_std_file_open_options_v0 fy_probe_options;
#endif

_Static_assert(sizeof(struct fy_std_file_buffer_result_v0) == 24, "file-buffer result size");
_Static_assert(offsetof(struct fy_std_file_buffer_result_v0, count) == 16, "file-buffer count offset");

int main(void) {
    const uint8_t path[] = "file-buffer-input.dat";
    fy_probe_file file = {0};
    fy_probe_options options = {1, 1, 1, 1, 0, 0};
    assert(fy_std_file_open_v0(path, &options, &file).status == 0);
    assert(fy_std_file_write_all_v0(&file, (const uint8_t *)"abcdef", 6).status == 0);
    assert(fy_std_file_seek_start_v0(&file, 0).status == 0);
    struct fy_std_buffer_v0 buffer;
    assert(fy_std_buffer_initialize_v0(0, &buffer) == 0);
    assert(fy_std_buffer_append_v0(&buffer, (const uint8_t *)"pre", 3) == 0);
    const void *original = buffer.block.data;
    size_t capacity = buffer.block.size;
    struct fy_std_file_buffer_result_v0 result = fy_std_file_read_append_v0(&file, &buffer, 0);
    assert(result.file_status == 0 && result.allocation_status == 0 && result.os_code == 0 && result.count == 0);
    assert(buffer.block.data == original && buffer.length == 3 && buffer.block.size == capacity);
    assert(fy_probe_reads == 0 && fy_std_file_position_v0(&file).position == 0);

    result = fy_std_file_read_append_v0(&file, NULL, 4);
    assert(result.file_status == 0 && result.allocation_status == FY_STD_ALLOC_INVALID_ARGUMENT_V0 && result.os_code == 0);
    result = fy_std_file_read_append_v0(&file, &buffer, SIZE_MAX);
    assert(result.file_status == 0 && result.allocation_status == FY_STD_ALLOC_SIZE_OVERFLOW_V0 && result.count == 0);
    fy_std_system_heap_probe_fail_after_allocations_v0(0);
    result = fy_std_file_read_append_v0(&file, &buffer, capacity);
    assert(result.file_status == 0 && result.os_code == 0 && result.allocation_status == FY_STD_ALLOC_OUT_OF_MEMORY_V0 && result.count == 0);
    assert(buffer.block.data == original && buffer.block.size == capacity && buffer.length == 3);
    assert(memcmp(buffer.block.data, "pre", 3) == 0);
    assert(fy_probe_reads == 0 && fy_std_file_position_v0(&file).position == 0);

#ifndef _WIN32
    fy_probe_interrupt = 1;
#endif
    result = fy_std_file_read_append_v0(&file, &buffer, 4);
    assert(result.file_status == 0 && result.os_code == 0 && result.allocation_status == 0 && result.count == 4);
    assert(buffer.length == 7 && memcmp(buffer.block.data, "preabcd", 7) == 0);
#ifdef _WIN32
    assert(fy_probe_reads == 1);
    struct fy_std_windows_file_buffer_result_v0 legacy = fy_std_windows_file_read_append_v0(&file, &buffer, 4);
    assert(legacy.file_status == 0 && legacy.allocation_status == 0 && legacy.count == 2);
#else
    assert(fy_probe_reads == 2 && fy_probe_interrupt == 0);
    result = fy_std_file_read_append_v0(&file, &buffer, 4);
    assert(result.file_status == 0 && result.allocation_status == 0 && result.count == 2);
#endif
    assert(buffer.length == 9 && memcmp(buffer.block.data, "preabcdef", 9) == 0);
    result = fy_std_file_read_append_v0(&file, &buffer, 4);
    assert(result.file_status == 0 && result.count == 0 && buffer.length == 9);
    assert(fy_std_file_close_v0(&file).status == 0);
    result = fy_std_file_read_append_v0(&file, NULL, 0);
    assert(result.file_status == 3 && result.allocation_status == 0 && result.count == 0);

    options.read = 0; options.write = 1; options.create = 0; options.truncate = 0;
    assert(fy_std_file_open_v0(path, &options, &file).status == 0);
    result = fy_std_file_read_append_v0(&file, &buffer, 32);
    assert(result.file_status == 1 && result.os_code != 0 && result.allocation_status == 0 && result.count == 0);
    assert(buffer.length == 9 && memcmp(buffer.block.data, "preabcdef", 9) == 0);
    assert(fy_std_file_position_v0(&file).position == 0);
    assert(fy_std_file_close_v0(&file).status == 0);
    assert(fy_std_file_remove_v0(path).status == 0);
    assert(fy_std_buffer_drop_v0(&buffer) == 0);
    assert(fy_std_system_heap_probe_live_allocations_v0() == 0);
    return 0;
}
