#include <stddef.h>
#include <stdint.h>
#include <string.h>

enum fy_std_alloc_status_v0 {
    FY_STD_ALLOC_OK_V0 = 0,
    FY_STD_ALLOC_OUT_OF_MEMORY_V0 = 1,
    FY_STD_ALLOC_SIZE_OVERFLOW_V0 = 2,
    FY_STD_ALLOC_UNSUPPORTED_ALIGNMENT_V0 = 3,
    FY_STD_ALLOC_INVALID_ARGUMENT_V0 = 4
};

enum fy_std_text_status_v0 {
    FY_STD_TEXT_OK_V0 = 0,
    FY_STD_TEXT_INVALID_UTF8_V0 = 1,
    FY_STD_TEXT_INVALID_BOUNDARY_V0 = 2,
    FY_STD_TEXT_INVALID_ARGUMENT_V0 = 3,
    FY_STD_TEXT_INTERIOR_NUL_V0 = 4
};

enum fy_std_owned_text_status_v0 {
    FY_STD_OWNED_TEXT_OK_V0 = 0,
    FY_STD_OWNED_TEXT_ALLOCATION_FAILED_V0 = 1,
    FY_STD_OWNED_TEXT_INVALID_UTF8_V0 = 2,
    FY_STD_OWNED_TEXT_INTERIOR_NUL_V0 = 3,
    FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0 = 4,
    FY_STD_OWNED_TEXT_SIZE_OVERFLOW_V0 = 5
};

struct fy_std_system_heap_block_v0 { void *data; size_t size; size_t alignment; };
struct fy_std_buffer_v0 { struct fy_std_system_heap_block_v0 block; size_t length; };
struct fy_std_text_view_v0 { const uint8_t *data; size_t length; };
struct fy_std_owned_text_result_v0 { uint32_t status; uint32_t allocation_status; };
struct fy_std_string_v0 { struct fy_std_buffer_v0 bytes; };
struct fy_std_cstr_view_v0 { const uint8_t *data; size_t length; };
struct fy_std_cstring_v0 { struct fy_std_buffer_v0 bytes; };
struct fy_std_string_storage_v0 { void *data; size_t capacity; size_t alignment; size_t length; };
struct fy_std_cstring_storage_v0 { void *data; size_t capacity; size_t alignment; size_t storage_length; };

uint32_t fy_std_buffer_initialize_v0(size_t capacity, struct fy_std_buffer_v0 *output);
uint32_t fy_std_buffer_append_v0(struct fy_std_buffer_v0 *buffer, const uint8_t *source, size_t count);
uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *buffer);
struct fy_std_owned_text_result_v0 fy_std_string_copy_v0(const struct fy_std_text_view_v0 *input, struct fy_std_string_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_string_from_buffer_v0(struct fy_std_buffer_v0 *input, struct fy_std_string_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_string_as_view_v0(const struct fy_std_string_v0 *input, struct fy_std_text_view_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_string_append_v0(struct fy_std_string_v0 *value, const struct fy_std_text_view_v0 *suffix);
struct fy_std_owned_text_result_v0 fy_std_string_drop_v0(struct fy_std_string_v0 *value);
uint32_t fy_std_cstr_validate_v0(const uint8_t *storage, size_t storage_length, struct fy_std_cstr_view_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_v0(const struct fy_std_text_view_v0 *input, struct fy_std_cstring_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_cstring_as_view_v0(const struct fy_std_cstring_v0 *input, struct fy_std_cstr_view_v0 *output);
struct fy_std_owned_text_result_v0 fy_std_cstring_drop_v0(struct fy_std_cstring_v0 *value);
struct fy_std_owned_text_result_v0 fy_std_string_copy_storage_v0(const struct fy_std_text_view_v0 *input, void *output_storage);
struct fy_std_owned_text_result_v0 fy_std_string_append_storage_v0(void *value_storage, const struct fy_std_text_view_v0 *suffix);
struct fy_std_owned_text_result_v0 fy_std_string_drop_storage_v0(void *value_storage);
struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_storage_v0(const struct fy_std_text_view_v0 *input, void *output_storage);
struct fy_std_owned_text_result_v0 fy_std_cstring_drop_storage_v0(void *value_storage);
void fy_std_owned_text_probe_fail_next_allocation_v0(void);
size_t fy_std_owned_text_probe_live_allocations_v0(void);

static int ok(struct fy_std_owned_text_result_v0 result) {
    return result.status == FY_STD_OWNED_TEXT_OK_V0 && result.allocation_status == FY_STD_ALLOC_OK_V0;
}

static int invalid_argument(struct fy_std_owned_text_result_v0 result) {
    return result.status == FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0 &&
           result.allocation_status == FY_STD_ALLOC_OK_V0;
}

static int allocation_failed(struct fy_std_owned_text_result_v0 result) {
    return result.status == FY_STD_OWNED_TEXT_ALLOCATION_FAILED_V0 &&
           result.allocation_status == FY_STD_ALLOC_OUT_OF_MEMORY_V0;
}

static int string_storage_empty(const struct fy_std_string_storage_v0 *storage) {
    return storage->data == NULL && storage->capacity == 0 && storage->alignment == 0 &&
           storage->length == 0;
}

static int cstring_storage_empty(const struct fy_std_cstring_storage_v0 *storage) {
    return storage->data == NULL && storage->capacity == 0 && storage->alignment == 0 &&
           storage->storage_length == 0;
}

static int string_storage_equal(
    const struct fy_std_string_storage_v0 *left,
    const struct fy_std_string_storage_v0 *right
) {
    return left->data == right->data && left->capacity == right->capacity &&
           left->alignment == right->alignment && left->length == right->length;
}

static int cstring_storage_equal(
    const struct fy_std_cstring_storage_v0 *left,
    const struct fy_std_cstring_storage_v0 *right
) {
    return left->data == right->data && left->capacity == right->capacity &&
           left->alignment == right->alignment && left->storage_length == right->storage_length;
}

int main(void) {
    static const uint8_t hello[] = {'h', 0xc3, 0xa9};
    static const uint8_t emoji[] = {0xf0, 0x9f, 0x98, 0x80};
    struct fy_std_text_view_v0 hello_view = {hello, sizeof hello};
    struct fy_std_text_view_v0 emoji_view = {emoji, sizeof emoji};
    struct fy_std_string_v0 string = {{{NULL, 0, 0}, 0}};
    if (!ok(fy_std_string_copy_v0(&hello_view, &string)) ||
        !ok(fy_std_string_append_v0(&string, &emoji_view))) {
        return 1;
    }
    struct fy_std_text_view_v0 string_view = {NULL, 0};
    if (!ok(fy_std_string_as_view_v0(&string, &string_view)) || string_view.length != 7 ||
        memcmp(string_view.data, "h\xc3\xa9\xf0\x9f\x98\x80", 7) != 0) {
        return 2;
    }
    struct fy_std_text_view_v0 self_prefix = {string_view.data, 3};
    if (!ok(fy_std_string_append_v0(&string, &self_prefix)) ||
        !ok(fy_std_string_as_view_v0(&string, &string_view)) || string_view.length != 10 ||
        memcmp(string_view.data + 7, hello, 3) != 0) {
        return 3;
    }
    if (!ok(fy_std_string_drop_v0(&string))) {
        return 4;
    }

    static const uint8_t invalid_utf8[] = {0xc0, 0x80};
    struct fy_std_buffer_v0 invalid_buffer = {{NULL, 0, 0}, 0};
    if (fy_std_buffer_initialize_v0(sizeof invalid_utf8, &invalid_buffer) != FY_STD_ALLOC_OK_V0 ||
        fy_std_buffer_append_v0(&invalid_buffer, invalid_utf8, sizeof invalid_utf8) != FY_STD_ALLOC_OK_V0) {
        return 5;
    }
    void *invalid_data = invalid_buffer.block.data;
    struct fy_std_string_v0 untouched = {{{(void *)(uintptr_t)1, 7, 1}, 7}};
    struct fy_std_owned_text_result_v0 conversion = fy_std_string_from_buffer_v0(&invalid_buffer, &untouched);
    if (conversion.status != FY_STD_OWNED_TEXT_INVALID_UTF8_V0 ||
        invalid_buffer.block.data != invalid_data || invalid_buffer.length != sizeof invalid_utf8 ||
        untouched.bytes.block.data != (void *)(uintptr_t)1) {
        return 6;
    }
    if (fy_std_buffer_drop_v0(&invalid_buffer) != FY_STD_ALLOC_OK_V0) {
        return 7;
    }

    struct fy_std_buffer_v0 valid_buffer = {{NULL, 0, 0}, 0};
    if (fy_std_buffer_initialize_v0(sizeof hello, &valid_buffer) != FY_STD_ALLOC_OK_V0 ||
        fy_std_buffer_append_v0(&valid_buffer, hello, sizeof hello) != FY_STD_ALLOC_OK_V0 ||
        !ok(fy_std_string_from_buffer_v0(&valid_buffer, &string)) ||
        valid_buffer.block.data != NULL || valid_buffer.block.size != 0 || valid_buffer.length != 0 ||
        !ok(fy_std_string_drop_v0(&string))) {
        return 8;
    }

    struct fy_std_cstring_v0 cstring = {{{NULL, 0, 0}, 0}};
    if (!ok(fy_std_cstring_from_text_v0(&hello_view, &cstring))) {
        return 9;
    }
    struct fy_std_cstr_view_v0 cview = {NULL, 0};
    if (!ok(fy_std_cstring_as_view_v0(&cstring, &cview)) || cview.length != sizeof hello ||
        memcmp(cview.data, hello, sizeof hello) != 0 || cview.data[cview.length] != 0) {
        return 10;
    }
    if (!ok(fy_std_cstring_drop_v0(&cstring))) {
        return 11;
    }

    static const uint8_t with_nul[] = {'a', 0, 'b'};
    struct fy_std_text_view_v0 nul_view = {with_nul, sizeof with_nul};
    cstring.bytes.block.data = (void *)(uintptr_t)1;
    conversion = fy_std_cstring_from_text_v0(&nul_view, &cstring);
    if (conversion.status != FY_STD_OWNED_TEXT_INTERIOR_NUL_V0 ||
        cstring.bytes.block.data != (void *)(uintptr_t)1) {
        return 12;
    }
    cview.data = (const uint8_t *)(uintptr_t)1;
    cview.length = 99;
    static const uint8_t invalid_cstr[] = {'a', 0, 'b', 0};
    if (fy_std_cstr_validate_v0(invalid_cstr, sizeof invalid_cstr, &cview) != FY_STD_TEXT_INTERIOR_NUL_V0 ||
        cview.data != (const uint8_t *)(uintptr_t)1 || cview.length != 99) {
        return 13;
    }

    struct fy_std_text_view_v0 empty_text = {NULL, 0};
    cstring.bytes.block.data = NULL;
    cstring.bytes.block.size = 0;
    cstring.bytes.block.alignment = 0;
    cstring.bytes.length = 0;
    if (!ok(fy_std_cstring_from_text_v0(&empty_text, &cstring)) ||
        !ok(fy_std_cstring_as_view_v0(&cstring, &cview)) || cview.length != 0 || cview.data[0] != 0 ||
        !ok(fy_std_cstring_drop_v0(&cstring))) {
        return 14;
    }

    struct fy_std_string_storage_v0 string_storage = {NULL, 0, 0, 0};
    struct fy_std_cstring_storage_v0 cstring_storage = {NULL, 0, 0, 0};
    if (!invalid_argument(fy_std_string_copy_storage_v0(&hello_view, NULL)) ||
        !invalid_argument(fy_std_string_append_storage_v0(NULL, &hello_view)) ||
        !invalid_argument(fy_std_string_drop_storage_v0(NULL)) ||
        !invalid_argument(fy_std_cstring_from_text_storage_v0(&hello_view, NULL)) ||
        !invalid_argument(fy_std_cstring_drop_storage_v0(NULL))) {
        return 15;
    }
    if (!invalid_argument(fy_std_string_copy_storage_v0(NULL, &string_storage)) ||
        !string_storage_empty(&string_storage)) {
        return 16;
    }
    struct fy_std_string_storage_v0 nonempty_string_output = {(void *)(uintptr_t)1, 7, 1, 7};
    struct fy_std_string_storage_v0 preserved_nonempty_string_output = nonempty_string_output;
    if (!invalid_argument(fy_std_string_copy_storage_v0(&hello_view, &nonempty_string_output)) ||
        !string_storage_equal(&nonempty_string_output, &preserved_nonempty_string_output)) {
        return 29;
    }
    if (!ok(fy_std_string_copy_storage_v0(&hello_view, &string_storage)) ||
        string_storage.data == NULL || string_storage.length != sizeof hello ||
        memcmp(string_storage.data, hello, sizeof hello) != 0) {
        return 17;
    }
    struct fy_std_string_storage_v0 preserved_string = string_storage;
    uint8_t preserved_string_bytes[sizeof hello];
    memcpy(preserved_string_bytes, string_storage.data, sizeof preserved_string_bytes);
    struct fy_std_text_view_v0 invalid_suffix = {NULL, 1};
    if (!invalid_argument(fy_std_string_append_storage_v0(&string_storage, &invalid_suffix)) ||
        !string_storage_equal(&string_storage, &preserved_string) ||
        memcmp(string_storage.data, preserved_string_bytes, sizeof preserved_string_bytes) != 0) {
        return 18;
    }
    struct fy_std_text_view_v0 storage_self_prefix = {
        (const uint8_t *)string_storage.data,
        string_storage.length
    };
    if (!ok(fy_std_string_append_storage_v0(&string_storage, &storage_self_prefix)) ||
        string_storage.length != sizeof hello * 2 ||
        memcmp(string_storage.data, hello, sizeof hello) != 0 ||
        memcmp((const uint8_t *)string_storage.data + sizeof hello, hello, sizeof hello) != 0) {
        return 19;
    }
    if (!ok(fy_std_string_drop_storage_v0(&string_storage)) ||
        !string_storage_empty(&string_storage) ||
        !ok(fy_std_string_drop_storage_v0(&string_storage)) ||
        !string_storage_empty(&string_storage)) {
        return 20;
    }
    struct fy_std_string_storage_v0 malformed_string = {(void *)(uintptr_t)1, 0, 0, 0};
    preserved_string = malformed_string;
    if (!invalid_argument(fy_std_string_append_storage_v0(&malformed_string, &hello_view)) ||
        !string_storage_equal(&malformed_string, &preserved_string) ||
        !invalid_argument(fy_std_string_drop_storage_v0(&malformed_string)) ||
        !string_storage_equal(&malformed_string, &preserved_string)) {
        return 21;
    }

    if (!invalid_argument(fy_std_cstring_from_text_storage_v0(NULL, &cstring_storage)) ||
        !cstring_storage_empty(&cstring_storage)) {
        return 22;
    }
    struct fy_std_cstring_storage_v0 nonempty_cstring_output = {(void *)(uintptr_t)1, 7, 1, 7};
    struct fy_std_cstring_storage_v0 preserved_nonempty_cstring_output = nonempty_cstring_output;
    if (!invalid_argument(fy_std_cstring_from_text_storage_v0(&hello_view, &nonempty_cstring_output)) ||
        !cstring_storage_equal(&nonempty_cstring_output, &preserved_nonempty_cstring_output)) {
        return 30;
    }
    struct fy_std_cstring_storage_v0 preserved_cstring = cstring_storage;
    conversion = fy_std_cstring_from_text_storage_v0(&nul_view, &cstring_storage);
    if (conversion.status != FY_STD_OWNED_TEXT_INTERIOR_NUL_V0 ||
        conversion.allocation_status != FY_STD_ALLOC_OK_V0 ||
        !cstring_storage_equal(&cstring_storage, &preserved_cstring)) {
        return 23;
    }
    struct fy_std_text_view_v0 overflow_text = {hello, SIZE_MAX};
    conversion = fy_std_cstring_from_text_storage_v0(&overflow_text, &cstring_storage);
    if (conversion.status != FY_STD_OWNED_TEXT_SIZE_OVERFLOW_V0 ||
        conversion.allocation_status != FY_STD_ALLOC_SIZE_OVERFLOW_V0 ||
        !cstring_storage_equal(&cstring_storage, &preserved_cstring)) {
        return 24;
    }
    if (!ok(fy_std_cstring_from_text_storage_v0(&empty_text, &cstring_storage)) ||
        cstring_storage.data == NULL || cstring_storage.storage_length != 1 ||
        ((const uint8_t *)cstring_storage.data)[0] != 0 ||
        !ok(fy_std_cstring_drop_storage_v0(&cstring_storage)) ||
        !cstring_storage_empty(&cstring_storage) ||
        !ok(fy_std_cstring_drop_storage_v0(&cstring_storage)) ||
        !cstring_storage_empty(&cstring_storage)) {
        return 25;
    }
    if (!ok(fy_std_cstring_from_text_storage_v0(&hello_view, &cstring_storage)) ||
        cstring_storage.data == NULL || cstring_storage.storage_length != sizeof hello + 1 ||
        memcmp(cstring_storage.data, hello, sizeof hello) != 0 ||
        ((const uint8_t *)cstring_storage.data)[sizeof hello] != 0) {
        return 26;
    }
    if (!ok(fy_std_cstring_drop_storage_v0(&cstring_storage)) ||
        !cstring_storage_empty(&cstring_storage)) {
        return 27;
    }
    struct fy_std_cstring_storage_v0 malformed_cstring = {(void *)(uintptr_t)1, 0, 0, 0};
    preserved_cstring = malformed_cstring;
    if (!invalid_argument(fy_std_cstring_drop_storage_v0(&malformed_cstring)) ||
        !cstring_storage_equal(&malformed_cstring, &preserved_cstring)) {
        return 28;
    }

    if (fy_std_owned_text_probe_live_allocations_v0() != 0) {
        return 31;
    }
    fy_std_owned_text_probe_fail_next_allocation_v0();
    conversion = fy_std_string_copy_storage_v0(&hello_view, &string_storage);
    if (!allocation_failed(conversion) || !string_storage_empty(&string_storage) ||
        fy_std_owned_text_probe_live_allocations_v0() != 0) {
        return 32;
    }

    if (!ok(fy_std_string_copy_storage_v0(&hello_view, &string_storage)) ||
        fy_std_owned_text_probe_live_allocations_v0() != 1) {
        return 33;
    }
    preserved_string = string_storage;
    memcpy(preserved_string_bytes, string_storage.data, sizeof preserved_string_bytes);
    fy_std_owned_text_probe_fail_next_allocation_v0();
    conversion = fy_std_string_append_storage_v0(&string_storage, &emoji_view);
    if (!allocation_failed(conversion) ||
        !string_storage_equal(&string_storage, &preserved_string) ||
        memcmp(string_storage.data, preserved_string_bytes, sizeof preserved_string_bytes) != 0 ||
        fy_std_owned_text_probe_live_allocations_v0() != 1) {
        return 34;
    }
    if (!ok(fy_std_string_drop_storage_v0(&string_storage)) ||
        !string_storage_empty(&string_storage) ||
        fy_std_owned_text_probe_live_allocations_v0() != 0) {
        return 35;
    }

    fy_std_owned_text_probe_fail_next_allocation_v0();
    conversion = fy_std_cstring_from_text_storage_v0(&hello_view, &cstring_storage);
    if (!allocation_failed(conversion) || !cstring_storage_empty(&cstring_storage) ||
        fy_std_owned_text_probe_live_allocations_v0() != 0) {
        return 36;
    }
    return 0;
}
