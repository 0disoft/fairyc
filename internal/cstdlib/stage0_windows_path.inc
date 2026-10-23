#ifndef FY_STD_WINDOWS_LONG_PATH_V0
#define FY_STD_WINDOWS_LONG_PATH_V0
#include <stddef.h>
#include <string.h>
#include <wchar.h>
#include <windows.h>

/* Input was already decoded and validated by the owning entry point. */
static DWORD fy_std_windows_long_path_v0(WCHAR *path, size_t capacity) {
    size_t length = (size_t)lstrlenW(path);
    /* File APIs retain their existing explicit device-namespace behavior.
       Directory enumeration rejects these inputs before reaching this helper. */
    if (length >= 4 && (path[0] == L'\\' || path[0] == L'/') &&
        (path[1] == L'\\' || path[1] == L'/') &&
        (path[2] == L'?' || path[2] == L'.') &&
        (path[3] == L'\\' || path[3] == L'/')) {
        return ERROR_SUCCESS;
    }
    WCHAR absolute[4096];
    DWORD full = GetFullPathNameW(path, 4096, absolute, NULL);
    if (full == 0) { return GetLastError(); }
    if (full >= 4096) { return ERROR_FILENAME_EXCED_RANGE; }
    /* Keep ordinary Win32 name handling for paths below the directory limit. */
    if (full < MAX_PATH - 12) {
        if ((size_t)full >= capacity) { return ERROR_FILENAME_EXCED_RANGE; }
        memcpy(path, absolute, ((size_t)full + 1) * sizeof(WCHAR));
        return ERROR_SUCCESS;
    }
    size_t skip = 0, prefix = 4;
    const WCHAR *head = L"\\\\?\\";
    if (absolute[0] == L'\\' && absolute[1] == L'\\') {
        skip = 2; prefix = 8; head = L"\\\\?\\UNC\\";
    } else if (full < 3 || absolute[1] != L':' || absolute[2] != L'\\') {
        return ERROR_INVALID_NAME;
    }
    if ((size_t)full - skip + prefix >= capacity) { return ERROR_FILENAME_EXCED_RANGE; }
    memcpy(path, head, prefix * sizeof(WCHAR));
    memcpy(path + prefix, absolute + skip, ((size_t)full - skip + 1) * sizeof(WCHAR));
    return ERROR_SUCCESS;
}
#endif
