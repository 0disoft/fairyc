/* Private Linux path queries. UTF-8 validation and path ownership live in Fairy. */
#ifndef _POSIX_C_SOURCE
#define _POSIX_C_SOURCE 200809L
#endif
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

/* Kinds: 1 current directory, 2 TMPDIR (empty if unset), 3 executable image.
 * Zero means success; otherwise return errno. Length excludes the trailing NUL.
 * On failure length stays zero; callers must not publish any buffer contents. */
uint32_t fairyc_private_linux_host_path_v1(uint32_t kind, uint8_t *data,
                                         uint32_t capacity, uint32_t *length) {
    if (length == NULL) { return EINVAL; }
    *length = 0;
    if (data == NULL || capacity == 0) { return EINVAL; }
    size_t count = 0;
    if (kind == 1) {
        if (getcwd((char *)data, capacity) == NULL) { return (uint32_t)errno; }
        count = strlen((const char *)data);
    } else if (kind == 2) {
        const char *value = getenv("TMPDIR");
        if (value == NULL) { value = ""; }
        while (count < capacity && value[count] != '\0') { ++count; }
        if (count == capacity) { return ERANGE; }
        memcpy(data, value, count + 1);
    } else if (kind == 3) {
        ssize_t result = readlink("/proc/self/exe", (char *)data, capacity);
        if (result < 0) { return (uint32_t)errno; }
        count = (size_t)result;
        if (count >= capacity) { return ERANGE; }
        data[count] = 0;
    } else {
        return EINVAL;
    }
    *length = (uint32_t)count;
    return 0;
}
