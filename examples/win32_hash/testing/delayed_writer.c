// Test-build-only fault gates, never linked into the distributed EXE.
#define UNICODE
#define _UNICODE
#include <windows.h>
#include <wchar.h>

static BOOL delayed_save_gate(LPCWSTR path, LPCWSTR stage) {
    LPCWSTR base = wcsrchr(path, L'\\'); base = base ? base+1 : path;
    WCHAR marker[80];
    if (swprintf_s(marker, ARRAYSIZE(marker), L"save-%ls", stage) < 0) { return FALSE; }
    if (!wcsstr(base, marker)) { return TRUE; }
    if (wcsstr(base, L"fail-")) { SetLastError(ERROR_DISK_FULL); return FALSE; }
    // A metadata/check operation may be requested twice on one target.
    static WCHAR completed[32768];
    if (!wcscmp(path, completed)) { return TRUE; }
    WCHAR prefix[256], name[512];
    if (!GetEnvironmentVariableW(L"FAIRY_HASH_SAVE_TEST_PIPE", prefix, ARRAYSIZE(prefix)) ||
        swprintf_s(name, ARRAYSIZE(name), L"%ls-%ls", prefix, base) < 0) { SetLastError(ERROR_INVALID_PARAMETER); return FALSE; }
    HANDLE pipe = CreateFileW(name, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL);
    if (pipe == INVALID_HANDLE_VALUE) { return FALSE; }
    BYTE byte; DWORD count, error;
    do {
        BOOL ok = ReadFile(pipe, &byte, 1, &count, NULL);
        error = ok ? ERROR_SUCCESS : GetLastError();
    } while (error == ERROR_OPERATION_ABORTED && wcsstr(base, L"retain"));
    CloseHandle(pipe);
    if (error == ERROR_SUCCESS || error == ERROR_BROKEN_PIPE) {
        wcscpy_s(completed, ARRAYSIZE(completed), path); return TRUE;
    }
    SetLastError(error); return FALSE;
}
