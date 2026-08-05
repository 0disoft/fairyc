#define WIN32_LEAN_AND_MEAN
#include <windows.h>

#include <limits.h>
#include <process.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <wchar.h>

struct fy_std_process_args_result_v0 {
    uint32_t status;
    size_t needed;
};

enum {
    FY_STD_PROCESS_ARGS_OK_V0 = 0,
    FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0 = 1,
    FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0 = 2,
    FY_STD_PROCESS_ARGS_BUFFER_TOO_SMALL_V0 = 3
};

static struct fy_std_process_args_result_v0 fy_std_process_arg_wide_v0(
    size_t index,
    size_t count,
    const wchar_t **argv,
    uint8_t *data,
    size_t capacity
) {
    struct fy_std_process_args_result_v0 result = {
        FY_STD_PROCESS_ARGS_OK_V0,
        (size_t)0
    };
    const wchar_t *argument;
    size_t character_count;
    int needed;
    size_t byte_length;

    if (index >= count) {
        result.status = FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0;
        return result;
    }
    if (argv == NULL || argv[index] == NULL) {
        result.status = FY_STD_PROCESS_ARGS_INDEX_OUT_OF_RANGE_V0;
        return result;
    }
    argument = argv[index];
    character_count = wcslen(argument);
    if (character_count == (size_t)0) {
        return result;
    }
    if (character_count > (size_t)INT_MAX) {
        result.status = FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0;
        return result;
    }
    needed = WideCharToMultiByte(
        CP_UTF8,
        WC_ERR_INVALID_CHARS,
        argument,
        (int)character_count,
        NULL,
        0,
        NULL,
        NULL
    );
    if (needed == 0) {
        /* Unpaired UTF-16 surrogates fail with ERROR_NO_UNICODE_TRANSLATION; all
           undecodable input reports status 2 and leaves the caller buffer untouched. */
        result.status = FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0;
        return result;
    }
    byte_length = (size_t)needed;
    if (data == NULL || byte_length > capacity) {
        result.status = FY_STD_PROCESS_ARGS_BUFFER_TOO_SMALL_V0;
        result.needed = byte_length;
        return result;
    }
    {
        int written = WideCharToMultiByte(
            CP_UTF8,
            WC_ERR_INVALID_CHARS,
            argument,
            (int)character_count,
            (char *)data,
            (int)byte_length,
            NULL,
            NULL
        );
        /* The second pass reuses the same immutable UCRT wide string and the exact
           size the first pass computed, so it cannot fail or write a different byte
           count. The defensive check therefore never becomes a public failure path
           that has already touched the caller buffer. */
        if (written != needed) {
            result.status = FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0;
            return result;
        }
    }
    result.needed = byte_length;
    return result;
}

size_t fy_std_process_arg_count_v0(void) {
    int *argc = __p___argc();
    if (argc == NULL || *argc < 0) {
        return (size_t)0;
    }
    /* The narrow mainCRTStartup policy populates __argc even though it leaves
       *__p___wargv() NULL; the count is identical after wide argv is materialized.
       If wide argv cannot be materialized, arg() reports status 2 while the count
       still reflects the same startup argument set. */
    return (size_t)*argc;
}

struct fy_std_process_args_result_v0 fy_std_process_arg_v0(
    size_t index,
    uint8_t *data,
    size_t capacity
) {
    struct fy_std_process_args_result_v0 result = {
        FY_STD_PROCESS_ARGS_OK_V0,
        (size_t)0
    };
    int *argc = __p___argc();
    wchar_t ***wargv = __p___wargv();
    if (argc == NULL || wargv == NULL || *argc < 0) {
        result.status = FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0;
        return result;
    }
    if (*wargv == NULL) {
        /* mainCRTStartup under int main(void) configures narrow argv only, so the
           wide argv must be materialized lazily with the same unexpanded mode the
           wmain startup would use. No wildcard expansion is introduced. */
        if (_configure_wide_argv(_crt_argv_unexpanded_arguments) != 0 ||
            *wargv == NULL) {
            /* Unicode argv could not be materialized safely. This stays inside the
               existing status table (invalid encoding) with an empty needed value
               and never touches the caller buffer. */
            result.status = FY_STD_PROCESS_ARGS_INVALID_ENCODING_V0;
            result.needed = (size_t)0;
            return result;
        }
    }
    return fy_std_process_arg_wide_v0(
        index,
        (size_t)*argc,
        (const wchar_t **)*wargv,
        data,
        capacity
    );
}
