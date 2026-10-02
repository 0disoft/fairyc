#include <stddef.h>
#include <stdint.h>
#include <string.h>

struct fy_std_process_args_result_v0 { uint32_t status; size_t needed; };
static size_t fy_std_process_argc_v0;
static char **fy_std_process_argv_v0;

/* The generated entry borrows the OS argv for the lifetime of this process. */
void fy_std_process_set_args_v0(int argc, char **argv) {
    fy_std_process_argc_v0 = argc > 0 ? (size_t)argc : (size_t)0;
    fy_std_process_argv_v0 = argv;
}

size_t fy_std_process_arg_count_v0(void) { return fy_std_process_argc_v0; }

struct fy_std_process_args_result_v0 fy_std_process_arg_v0(size_t index, uint8_t *data, size_t capacity) {
    struct fy_std_process_args_result_v0 result = {UINT32_C(0), (size_t)0};
    if (index >= fy_std_process_argc_v0 || fy_std_process_argv_v0 == NULL || fy_std_process_argv_v0[index] == NULL) {
        result.status = UINT32_C(1);
        return result;
    }
    const uint8_t *argument = (const uint8_t *)fy_std_process_argv_v0[index];
    size_t length = strlen((const char *)argument);
    if (!fy_std_text_utf8_valid_v0(argument, length)) {
        result.status = UINT32_C(2);
        return result;
    }
    if (length > capacity || (length != 0 && data == NULL)) {
        result.status = UINT32_C(3);
        result.needed = length;
        return result;
    }
    if (length != 0) { memcpy(data, argument, length); }
    result.needed = length;
    return result;
}
