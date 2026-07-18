#include <stddef.h>
#include <stdint.h>
#include <string.h>

enum fy_std_text_status_v0 {
    FY_STD_TEXT_OK_V0 = 0,
    FY_STD_TEXT_INVALID_UTF8_V0 = 1,
    FY_STD_TEXT_INVALID_BOUNDARY_V0 = 2,
    FY_STD_TEXT_INVALID_ARGUMENT_V0 = 3,
    FY_STD_TEXT_INTERIOR_NUL_V0 = 4
};

struct fy_std_text_view_v0 {
    const uint8_t *data;
    size_t length;
};

uint32_t fy_std_text_validate_utf8_v0(
    const uint8_t *data,
    size_t length,
    struct fy_std_text_view_v0 *output
);
uint32_t fy_std_text_slice_v0(
    const struct fy_std_text_view_v0 *input,
    size_t start,
    size_t end,
    struct fy_std_text_view_v0 *output
);

static int rejects(const uint8_t *data, size_t length) {
    struct fy_std_text_view_v0 sentinel = {(const uint8_t *)(uintptr_t)1, 99};
    uint32_t status = fy_std_text_validate_utf8_v0(data, length, &sentinel);
    return status == FY_STD_TEXT_INVALID_UTF8_V0 &&
           sentinel.data == (const uint8_t *)(uintptr_t)1 && sentinel.length == 99;
}

int main(void) {
    static const uint8_t valid[] = {
        'A', 0x00, 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x9f, 0x98, 0x80
    };
    struct fy_std_text_view_v0 view = {NULL, 0};
    if (fy_std_text_validate_utf8_v0(valid, sizeof valid, &view) != FY_STD_TEXT_OK_V0 ||
        view.data != valid || view.length != sizeof valid) {
        return 1;
    }
    struct fy_std_text_view_v0 slice = {NULL, 0};
    if (fy_std_text_slice_v0(&view, 2, 7, &slice) != FY_STD_TEXT_OK_V0 ||
        slice.length != 5 || memcmp(slice.data, valid + 2, 5) != 0) {
        return 2;
    }
    slice.data = (const uint8_t *)(uintptr_t)1;
    slice.length = 99;
    if (fy_std_text_slice_v0(&view, 3, 7, &slice) != FY_STD_TEXT_INVALID_BOUNDARY_V0 ||
        slice.data != (const uint8_t *)(uintptr_t)1 || slice.length != 99) {
        return 3;
    }
    if (fy_std_text_slice_v0(&view, 7, 6, &slice) != FY_STD_TEXT_INVALID_BOUNDARY_V0) {
        return 4;
    }
    if (fy_std_text_slice_v0(&view, 0, sizeof valid + 1, &slice) != FY_STD_TEXT_INVALID_BOUNDARY_V0) {
        return 5;
    }
    if (fy_std_text_validate_utf8_v0(NULL, 0, &view) != FY_STD_TEXT_OK_V0 ||
        view.data != NULL || view.length != 0) {
        return 6;
    }

    static const uint8_t stray_continuation[] = {0x80};
    static const uint8_t overlong_two[] = {0xc0, 0x80};
    static const uint8_t overlong_three[] = {0xe0, 0x80, 0x80};
    static const uint8_t surrogate[] = {0xed, 0xa0, 0x80};
    static const uint8_t over_maximum[] = {0xf4, 0x90, 0x80, 0x80};
    static const uint8_t truncated[] = {0xf0, 0x9f, 0x98};
    static const uint8_t obsolete_five[] = {0xf8, 0x88, 0x80, 0x80, 0x80};
    if (!rejects(stray_continuation, sizeof stray_continuation) ||
        !rejects(overlong_two, sizeof overlong_two) ||
        !rejects(overlong_three, sizeof overlong_three) ||
        !rejects(surrogate, sizeof surrogate) ||
        !rejects(over_maximum, sizeof over_maximum) ||
        !rejects(truncated, sizeof truncated) ||
        !rejects(obsolete_five, sizeof obsolete_five)) {
        return 7;
    }
    if (fy_std_text_validate_utf8_v0(NULL, 1, &view) != FY_STD_TEXT_INVALID_ARGUMENT_V0 ||
        fy_std_text_validate_utf8_v0(valid, sizeof valid, NULL) != FY_STD_TEXT_INVALID_ARGUMENT_V0) {
        return 8;
    }
    return 0;
}
