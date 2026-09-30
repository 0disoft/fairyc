#include <stddef.h>
#include <stdint.h>
#include <string.h>
#include <windows.h>

/* Selected only by directory symbols. One bounded allocation per explicit open. */
struct fy_std_directory_v0 { void *state; uint8_t open; };
struct fy_std_directory_state_v0 {
    HANDLE search;
    WIN32_FIND_DATAW entry;
    int pending;
    int ended;
};
struct fy_std_directory_result_v0 { uint32_t status; uint32_t os_code; size_t count; };
struct fy_std_directory_entry_result_v0 {
    uint32_t status;
    uint32_t os_code;
    size_t name_length;
    uint32_t kind;
    size_t size;
    uint32_t reparse;
};

static struct fy_std_directory_result_v0 fy_std_directory_result_v0(uint32_t status, DWORD code) {
    struct fy_std_directory_result_v0 result = {status, (uint32_t)code, 0};
    return result;
}

static struct fy_std_directory_entry_result_v0 fy_std_directory_entry_result_v0(uint32_t status, DWORD code) {
    struct fy_std_directory_entry_result_v0 result = {status, (uint32_t)code, 0, 0, 0, 0};
    return result;
}

/* GetFullPathName resolves dot segments, not links. Inspect every directory prefix. */
static DWORD fy_std_directory_check_path_v0(WCHAR *path) {
    size_t length = (size_t)lstrlenW(path);
    size_t start = 3;
    if (path[0] == L'\\' && path[1] == L'\\') {
        start = 2;
        while (start < length && path[start] != L'\\') { ++start; }
        if (start == length) { return ERROR_INVALID_NAME; }
        ++start;
        while (start < length && path[start] != L'\\') { ++start; }
    }
    for (size_t index = start; index <= length; ++index) {
        if (index != length && path[index] != L'\\') { continue; }
        WCHAR saved = path[index];
        path[index] = L'\0';
        DWORD attributes = GetFileAttributesW(path);
        DWORD code = attributes == INVALID_FILE_ATTRIBUTES ? GetLastError() : ERROR_SUCCESS;
        path[index] = saved;
        if (code != ERROR_SUCCESS) { return code; }
        if ((attributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0) { return ERROR_CANT_ACCESS_FILE; }
        if ((attributes & FILE_ATTRIBUTE_DIRECTORY) == 0) { return ERROR_DIRECTORY; }
    }
    return ERROR_SUCCESS;
}

struct fy_std_directory_result_v0 fy_std_windows_read_dir_open_v0(
    const uint8_t *path, size_t length, void *output
) {
    struct fy_std_directory_v0 directory = {NULL, 0};
    if (output == NULL) { return fy_std_directory_result_v0(2, ERROR_INVALID_PARAMETER); }
    memcpy(output, &directory, sizeof directory);
    if (path == NULL || length == 0) { return fy_std_directory_result_v0(2, ERROR_INVALID_PARAMETER); }
    if (length > 4095) { return fy_std_directory_result_v0(6, ERROR_FILENAME_EXCED_RANGE); }
    if (length >= 4 && (path[0] == '\\' || path[0] == '/') &&
        (path[1] == '\\' || path[1] == '/') && path[2] == '.' &&
        (path[3] == '\\' || path[3] == '/')) {
        return fy_std_directory_result_v0(2, ERROR_INVALID_NAME);
    }
    for (size_t index = 0; index < length; ++index) {
        if (path[index] == 0 || path[index] == '*' || path[index] == '?' ||
            (path[index] == ':' && index != 1)) {
            return fy_std_directory_result_v0(2, ERROR_INVALID_NAME);
        }
    }
    WCHAR input[4096];
    int converted = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)path,
        (int)length, input, 4095);
    if (converted == 0) { return fy_std_directory_result_v0(5, GetLastError()); }
    input[converted] = L'\0';
    WCHAR absolute[4096];
    DWORD full_length = GetFullPathNameW(input, 4096, absolute, NULL);
    if (full_length == 0) { return fy_std_directory_result_v0(1, GetLastError()); }
    if (full_length > 4093) { return fy_std_directory_result_v0(6, ERROR_FILENAME_EXCED_RANGE); }
    DWORD code = fy_std_directory_check_path_v0(absolute);
    if (code != ERROR_SUCCESS) { return fy_std_directory_result_v0(1, code); }
    if (absolute[full_length - 1] != L'\\') { absolute[full_length++] = L'\\'; }
    absolute[full_length++] = L'*';
    absolute[full_length] = L'\0';
    struct fy_std_directory_state_v0 *state = (struct fy_std_directory_state_v0 *)
        HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof *state);
    if (state == NULL) { return fy_std_directory_result_v0(1, ERROR_NOT_ENOUGH_MEMORY); }
    state->search = FindFirstFileW(absolute, &state->entry);
    if (state->search == INVALID_HANDLE_VALUE) {
        code = GetLastError();
        if (code != ERROR_FILE_NOT_FOUND) {
            HeapFree(GetProcessHeap(), 0, state);
            return fy_std_directory_result_v0(1, code);
        }
        state->ended = 1;
    } else { state->pending = 1; }
    directory.state = state;
    directory.open = 1;
    memcpy(output, &directory, sizeof directory);
    return fy_std_directory_result_v0(0, ERROR_SUCCESS);
}

struct fy_std_directory_entry_result_v0 fy_std_windows_read_dir_next_v0(
    void *storage, uint8_t *name, size_t capacity
) {
    if (storage == NULL || (capacity != 0 && name == NULL)) {
        return fy_std_directory_entry_result_v0(2, ERROR_INVALID_PARAMETER);
    }
    struct fy_std_directory_v0 directory;
    memcpy(&directory, storage, sizeof directory);
    if (directory.open == 0 || directory.state == NULL) {
        return fy_std_directory_entry_result_v0(3, ERROR_INVALID_HANDLE);
    }
    struct fy_std_directory_state_v0 *state = (struct fy_std_directory_state_v0 *)directory.state;
    while (!state->ended) {
        if (!state->pending) {
            if (!FindNextFileW(state->search, &state->entry)) {
                DWORD code = GetLastError();
                if (code != ERROR_NO_MORE_FILES) { return fy_std_directory_entry_result_v0(1, code); }
                state->ended = 1;
                break;
            }
            state->pending = 1;
        }
        const WCHAR *wide_name = state->entry.cFileName;
        if (wide_name[0] == L'.' && (wide_name[1] == L'\0' ||
            (wide_name[1] == L'.' && wide_name[2] == L'\0'))) {
            state->pending = 0;
            continue;
        }
        int wide_length = lstrlenW(wide_name);
        int required = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, wide_name,
            wide_length, NULL, 0, NULL, NULL);
        if (required == 0) { return fy_std_directory_entry_result_v0(5, GetLastError()); }
        if ((size_t)required > capacity) {
            struct fy_std_directory_entry_result_v0 result =
                fy_std_directory_entry_result_v0(6, ERROR_INSUFFICIENT_BUFFER);
            result.name_length = (size_t)required;
            return result; /* Keep this item pending so a larger buffer can retry. */
        }
        if (WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, wide_name, wide_length,
            (char *)name, required, NULL, NULL) == 0) {
            return fy_std_directory_entry_result_v0(5, GetLastError());
        }
        DWORD attributes = state->entry.dwFileAttributes;
        uint32_t reparse = (attributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0 ? 1u : 0u;
        uint32_t kind = reparse != 0 || (attributes & FILE_ATTRIBUTE_DEVICE) != 0 ? 3u :
            ((attributes & FILE_ATTRIBUTE_DIRECTORY) != 0 ? 2u : 1u);
        struct fy_std_directory_entry_result_v0 result = {
            0, 0, (size_t)required, kind,
            (size_t)(((uint64_t)state->entry.nFileSizeHigh << 32) | state->entry.nFileSizeLow), reparse
        };
        state->pending = 0;
        return result;
    }
    return fy_std_directory_entry_result_v0(0, ERROR_SUCCESS);
}

struct fy_std_directory_result_v0 fy_std_windows_read_dir_close_v0(void *storage) {
    if (storage == NULL) { return fy_std_directory_result_v0(2, ERROR_INVALID_PARAMETER); }
    struct fy_std_directory_v0 directory;
    memcpy(&directory, storage, sizeof directory);
    struct fy_std_directory_v0 closed = {NULL, 0};
    memcpy(storage, &closed, sizeof closed);
    if (directory.open == 0 || directory.state == NULL) {
        return fy_std_directory_result_v0(3, ERROR_INVALID_HANDLE);
    }
    struct fy_std_directory_state_v0 *state = (struct fy_std_directory_state_v0 *)directory.state;
    DWORD code = ERROR_SUCCESS;
    if (state->search != INVALID_HANDLE_VALUE && !FindClose(state->search)) { code = GetLastError(); }
    HeapFree(GetProcessHeap(), 0, state);
    return fy_std_directory_result_v0(code == ERROR_SUCCESS ? 0u : 1u, code);
}

uint32_t fy_std_windows_read_dir_drop_v0(void *storage) {
    if (storage == NULL) { return 2; }
    struct fy_std_directory_v0 directory;
    memcpy(&directory, storage, sizeof directory);
    if (directory.open == 0) { return 0; }
    return fy_std_windows_read_dir_close_v0(storage).status;
}
