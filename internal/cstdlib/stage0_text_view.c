#include <stddef.h>
#include <stdint.h>

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

static int fy_std_text_is_continuation_v0(uint8_t byte) {
    return (byte & UINT8_C(0xc0)) == UINT8_C(0x80);
}

uint32_t fy_std_text_utf8_valid_v0(const uint8_t *data, size_t length) {
    if (data == NULL) {
        return length == 0;
    }
    size_t index = 0;
    while (index < length) {
        uint8_t first = data[index];
        if (first <= UINT8_C(0x7f)) {
            index++;
            continue;
        }
        if (first >= UINT8_C(0xc2) && first <= UINT8_C(0xdf)) {
            if (length - index < 2 || !fy_std_text_is_continuation_v0(data[index + 1])) {
                return 0;
            }
            index += 2;
            continue;
        }
        if (first >= UINT8_C(0xe0) && first <= UINT8_C(0xef)) {
            if (length - index < 3 || !fy_std_text_is_continuation_v0(data[index + 2])) {
                return 0;
            }
            uint8_t second = data[index + 1];
            if (first == UINT8_C(0xe0)) {
                if (second < UINT8_C(0xa0) || second > UINT8_C(0xbf)) {
                    return 0;
                }
            } else if (first == UINT8_C(0xed)) {
                if (second < UINT8_C(0x80) || second > UINT8_C(0x9f)) {
                    return 0;
                }
            } else if (!fy_std_text_is_continuation_v0(second)) {
                return 0;
            }
            index += 3;
            continue;
        }
        if (first >= UINT8_C(0xf0) && first <= UINT8_C(0xf4)) {
            if (length - index < 4 ||
                !fy_std_text_is_continuation_v0(data[index + 2]) ||
                !fy_std_text_is_continuation_v0(data[index + 3])) {
                return 0;
            }
            uint8_t second = data[index + 1];
            if (first == UINT8_C(0xf0)) {
                if (second < UINT8_C(0x90) || second > UINT8_C(0xbf)) {
                    return 0;
                }
            } else if (first == UINT8_C(0xf4)) {
                if (second < UINT8_C(0x80) || second > UINT8_C(0x8f)) {
                    return 0;
                }
            } else if (!fy_std_text_is_continuation_v0(second)) {
                return 0;
            }
            index += 4;
            continue;
        }
        return 0;
    }
    return 1;
}

uint32_t fy_std_text_validate_utf8_v0(
    const uint8_t *data,
    size_t length,
    struct fy_std_text_view_v0 *output
) {
    if (output == NULL || (data == NULL && length != 0)) {
        return FY_STD_TEXT_INVALID_ARGUMENT_V0;
    }
    if (!fy_std_text_utf8_valid_v0(data, length)) {
        return FY_STD_TEXT_INVALID_UTF8_V0;
    }
    output->data = length == 0 ? NULL : data;
    output->length = length;
    return FY_STD_TEXT_OK_V0;
}

uint32_t fy_std_text_slice_v0(
    const struct fy_std_text_view_v0 *input,
    size_t start,
    size_t end,
    struct fy_std_text_view_v0 *output
) {
    if (input == NULL || output == NULL ||
        !fy_std_text_utf8_valid_v0(input->data, input->length)) {
        return FY_STD_TEXT_INVALID_ARGUMENT_V0;
    }
    if (start > end || end > input->length) {
        return FY_STD_TEXT_INVALID_BOUNDARY_V0;
    }
    if ((start != 0 && start != input->length && fy_std_text_is_continuation_v0(input->data[start])) ||
        (end != 0 && end != input->length && fy_std_text_is_continuation_v0(input->data[end]))) {
        return FY_STD_TEXT_INVALID_BOUNDARY_V0;
    }
    output->data = start == end ? NULL : input->data + start;
    output->length = end - start;
    return FY_STD_TEXT_OK_V0;
}
