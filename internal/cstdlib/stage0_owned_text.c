enum fy_std_owned_text_status_v0 {
    FY_STD_OWNED_TEXT_OK_V0 = 0,
    FY_STD_OWNED_TEXT_ALLOCATION_FAILED_V0 = 1,
    FY_STD_OWNED_TEXT_INVALID_UTF8_V0 = 2,
    FY_STD_OWNED_TEXT_INTERIOR_NUL_V0 = 3,
    FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0 = 4,
    FY_STD_OWNED_TEXT_SIZE_OVERFLOW_V0 = 5
};

struct fy_std_owned_text_result_v0 {
    uint32_t status;
    uint32_t allocation_status;
};

struct fy_std_string_v0 {
    struct fy_std_buffer_v0 bytes;
};

struct fy_std_cstr_view_v0 {
    const uint8_t *data;
    size_t length;
};

struct fy_std_cstring_v0 {
    struct fy_std_buffer_v0 bytes;
};

static struct fy_std_owned_text_result_v0 fy_std_owned_text_ok_v0(void) {
    struct fy_std_owned_text_result_v0 result = {FY_STD_OWNED_TEXT_OK_V0, FY_STD_ALLOC_OK_V0};
    return result;
}

static struct fy_std_owned_text_result_v0 fy_std_owned_text_error_v0(
    uint32_t status,
    uint32_t allocation_status
) {
    struct fy_std_owned_text_result_v0 result = {status, allocation_status};
    return result;
}

static struct fy_std_owned_text_result_v0 fy_std_owned_text_from_alloc_v0(uint32_t status) {
    if (status == FY_STD_ALLOC_OK_V0) {
        return fy_std_owned_text_ok_v0();
    }
    if (status == FY_STD_ALLOC_SIZE_OVERFLOW_V0) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_SIZE_OVERFLOW_V0, status);
    }
    if (status == FY_STD_ALLOC_INVALID_ARGUMENT_V0) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, status);
    }
    return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_ALLOCATION_FAILED_V0, status);
}

static int fy_std_string_valid_v0(const struct fy_std_string_v0 *value) {
    return value != NULL && fy_std_buffer_valid_v0(&value->bytes) &&
           fy_std_text_utf8_valid_v0((const uint8_t *)value->bytes.block.data, value->bytes.length);
}

struct fy_std_owned_text_result_v0 fy_std_string_copy_v0(
    const struct fy_std_text_view_v0 *input,
    struct fy_std_string_v0 *output
) {
    if (input == NULL || output == NULL || !fy_std_text_utf8_valid_v0(input->data, input->length)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_string_v0 replacement = {{{NULL, 0, 0}, 0}};
    uint32_t status = fy_std_buffer_initialize_v0(input->length, &replacement.bytes);
    if (status != FY_STD_ALLOC_OK_V0) {
        return fy_std_owned_text_from_alloc_v0(status);
    }
    status = fy_std_buffer_append_v0(&replacement.bytes, input->data, input->length);
    if (status != FY_STD_ALLOC_OK_V0) {
        fy_std_buffer_drop_v0(&replacement.bytes);
        return fy_std_owned_text_from_alloc_v0(status);
    }
    *output = replacement;
    return fy_std_owned_text_ok_v0();
}

struct fy_std_owned_text_result_v0 fy_std_string_from_buffer_v0(
    struct fy_std_buffer_v0 *input,
    struct fy_std_string_v0 *output
) {
    if (!fy_std_buffer_valid_v0(input) || output == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    if (!fy_std_text_utf8_valid_v0((const uint8_t *)input->block.data, input->length)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_UTF8_V0, FY_STD_ALLOC_OK_V0);
    }
    output->bytes = *input;
    input->block.data = NULL;
    input->block.size = 0;
    input->block.alignment = 0;
    input->length = 0;
    return fy_std_owned_text_ok_v0();
}

struct fy_std_owned_text_result_v0 fy_std_string_as_view_v0(
    const struct fy_std_string_v0 *input,
    struct fy_std_text_view_v0 *output
) {
    if (!fy_std_string_valid_v0(input) || output == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    output->data = input->bytes.length == 0 ? NULL : (const uint8_t *)input->bytes.block.data;
    output->length = input->bytes.length;
    return fy_std_owned_text_ok_v0();
}

struct fy_std_owned_text_result_v0 fy_std_string_append_v0(
    struct fy_std_string_v0 *value,
    const struct fy_std_text_view_v0 *suffix
) {
    if (!fy_std_string_valid_v0(value) || suffix == NULL ||
        !fy_std_text_utf8_valid_v0(suffix->data, suffix->length)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    return fy_std_owned_text_from_alloc_v0(
        fy_std_buffer_append_v0(&value->bytes, suffix->data, suffix->length)
    );
}

struct fy_std_owned_text_result_v0 fy_std_string_drop_v0(struct fy_std_string_v0 *value) {
    if (!fy_std_string_valid_v0(value)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    return fy_std_owned_text_from_alloc_v0(fy_std_buffer_drop_v0(&value->bytes));
}

uint32_t fy_std_cstr_validate_v0(
    const uint8_t *storage,
    size_t storage_length,
    struct fy_std_cstr_view_v0 *output
) {
    if (storage == NULL || storage_length == 0 || output == NULL) {
        return FY_STD_TEXT_INVALID_ARGUMENT_V0;
    }
    if (storage[storage_length - 1] != 0) {
        return FY_STD_TEXT_INVALID_ARGUMENT_V0;
    }
    for (size_t index = 0; index + 1 < storage_length; ++index) {
        if (storage[index] == 0) {
            return FY_STD_TEXT_INTERIOR_NUL_V0;
        }
    }
    output->data = storage;
    output->length = storage_length - 1;
    return FY_STD_TEXT_OK_V0;
}

static int fy_std_cstring_valid_v0(const struct fy_std_cstring_v0 *value) {
    if (value == NULL || !fy_std_buffer_valid_v0(&value->bytes) || value->bytes.length == 0) {
        return 0;
    }
    struct fy_std_cstr_view_v0 ignored = {NULL, 0};
    return fy_std_cstr_validate_v0(
        (const uint8_t *)value->bytes.block.data,
        value->bytes.length,
        &ignored
    ) == FY_STD_TEXT_OK_V0;
}

struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_v0(
    const struct fy_std_text_view_v0 *input,
    struct fy_std_cstring_v0 *output
) {
    if (input == NULL || output == NULL || (input->data == NULL && input->length != 0)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    if (input->length == SIZE_MAX) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_SIZE_OVERFLOW_V0, FY_STD_ALLOC_SIZE_OVERFLOW_V0);
    }
    if (!fy_std_text_utf8_valid_v0(input->data, input->length)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    for (size_t index = 0; index < input->length; ++index) {
        if (input->data[index] == 0) {
            return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INTERIOR_NUL_V0, FY_STD_ALLOC_OK_V0);
        }
    }
    struct fy_std_cstring_v0 replacement = {{{NULL, 0, 0}, 0}};
    uint32_t status = fy_std_buffer_initialize_v0(input->length + 1, &replacement.bytes);
    if (status != FY_STD_ALLOC_OK_V0) {
        return fy_std_owned_text_from_alloc_v0(status);
    }
    status = fy_std_buffer_append_v0(&replacement.bytes, input->data, input->length);
    static const uint8_t terminator = 0;
    if (status == FY_STD_ALLOC_OK_V0) {
        status = fy_std_buffer_append_v0(&replacement.bytes, &terminator, 1);
    }
    if (status != FY_STD_ALLOC_OK_V0) {
        fy_std_buffer_drop_v0(&replacement.bytes);
        return fy_std_owned_text_from_alloc_v0(status);
    }
    *output = replacement;
    return fy_std_owned_text_ok_v0();
}

struct fy_std_owned_text_result_v0 fy_std_cstring_as_view_v0(
    const struct fy_std_cstring_v0 *input,
    struct fy_std_cstr_view_v0 *output
) {
    if (!fy_std_cstring_valid_v0(input) || output == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    uint32_t status = fy_std_cstr_validate_v0(
        (const uint8_t *)input->bytes.block.data,
        input->bytes.length,
        output
    );
    if (status != FY_STD_TEXT_OK_V0) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    return fy_std_owned_text_ok_v0();
}

struct fy_std_owned_text_result_v0 fy_std_cstring_drop_v0(struct fy_std_cstring_v0 *value) {
    if (!fy_std_cstring_valid_v0(value)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    return fy_std_owned_text_from_alloc_v0(fy_std_buffer_drop_v0(&value->bytes));
}

static int fy_std_owned_text_buffer_empty_v0(const struct fy_std_buffer_v0 *value) {
    return value != NULL && value->block.data == NULL && value->block.size == 0 &&
           value->block.alignment == 0 && value->length == 0;
}

struct fy_std_owned_text_result_v0 fy_std_string_copy_storage_v0(
    const struct fy_std_text_view_v0 *input,
    void *output_storage
) {
    if (output_storage == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_string_v0 output;
    memcpy(&output, output_storage, sizeof output);
    if (!fy_std_owned_text_buffer_empty_v0(&output.bytes)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_string_copy_v0(input, &output);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(output_storage, &output, sizeof output);
    }
    return result;
}

struct fy_std_owned_text_result_v0 fy_std_string_append_storage_v0(
    void *value_storage,
    const struct fy_std_text_view_v0 *suffix
) {
    if (value_storage == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_string_v0 value;
    memcpy(&value, value_storage, sizeof value);
    if (!fy_std_string_valid_v0(&value)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_string_append_v0(&value, suffix);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(value_storage, &value, sizeof value);
    }
    return result;
}

struct fy_std_owned_text_result_v0 fy_std_string_drop_storage_v0(void *value_storage) {
    if (value_storage == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_string_v0 value;
    memcpy(&value, value_storage, sizeof value);
    if (fy_std_owned_text_buffer_empty_v0(&value.bytes)) {
        return fy_std_owned_text_ok_v0();
    }
    if (!fy_std_string_valid_v0(&value)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_string_drop_v0(&value);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(value_storage, &value, sizeof value);
    }
    return result;
}

struct fy_std_owned_text_result_v0 fy_std_string_from_buffer_storage_v0(
    void *input_storage,
    void *output_storage
) {
    if (input_storage == NULL || output_storage == NULL || input_storage == output_storage) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_buffer_v0 input;
    struct fy_std_string_v0 output;
    memcpy(&input, input_storage, sizeof input);
    memcpy(&output, output_storage, sizeof output);
    if (!fy_std_owned_text_buffer_empty_v0(&output.bytes)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_string_from_buffer_v0(&input, &output);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(input_storage, &input, sizeof input);
        memcpy(output_storage, &output, sizeof output);
    }
    return result;
}

struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_storage_v0(
    const struct fy_std_text_view_v0 *input,
    void *output_storage
) {
    if (output_storage == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_cstring_v0 output;
    memcpy(&output, output_storage, sizeof output);
    if (!fy_std_owned_text_buffer_empty_v0(&output.bytes)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_cstring_from_text_v0(input, &output);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(output_storage, &output, sizeof output);
    }
    return result;
}

struct fy_std_owned_text_result_v0 fy_std_cstring_drop_storage_v0(void *value_storage) {
    if (value_storage == NULL) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_cstring_v0 value;
    memcpy(&value, value_storage, sizeof value);
    if (fy_std_owned_text_buffer_empty_v0(&value.bytes)) {
        return fy_std_owned_text_ok_v0();
    }
    if (!fy_std_cstring_valid_v0(&value)) {
        return fy_std_owned_text_error_v0(FY_STD_OWNED_TEXT_INVALID_ARGUMENT_V0, FY_STD_ALLOC_OK_V0);
    }
    struct fy_std_owned_text_result_v0 result = fy_std_cstring_drop_v0(&value);
    if (result.status == FY_STD_OWNED_TEXT_OK_V0) {
        memcpy(value_storage, &value, sizeof value);
    }
    return result;
}
