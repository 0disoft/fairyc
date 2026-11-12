// Test-build-only gates. Every gate blocks in an actual Win32 ReadFile on an
// owned named pipe; releasing the server resumes the original disk operation.
#define UNICODE
#define _UNICODE
#include <windows.h>
#include <wchar.h>

static BOOL delayed_gate(LPCWSTR path, LPCWSTR marker) {
    LPCWSTR base = wcsrchr(path, L'\\'); base = base ? base+1 : path;
    if (!wcsstr(base, marker)) { return TRUE; }
    WCHAR prefix[256], name[512];
    if (!GetEnvironmentVariableW(L"FAIRY_HASH_TEST_PIPE", prefix, ARRAYSIZE(prefix)) ||
        swprintf_s(name, ARRAYSIZE(name), L"%ls-%ls", prefix, base) < 0) { SetLastError(ERROR_INVALID_PARAMETER); return FALSE; }
    HANDLE pipe = CreateFileW(name, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL);
    if (pipe == INVALID_HANDLE_VALUE) { return FALSE; }
    BYTE byte; DWORD count, error;
    do {
        BOOL ok = ReadFile(pipe, &byte, 1, &count, NULL);
        error = ok ? ERROR_SUCCESS : GetLastError();
    } while (error == ERROR_OPERATION_ABORTED && wcsstr(base, L"retain"));
    CloseHandle(pipe);
    if (error == ERROR_SUCCESS || error == ERROR_BROKEN_PIPE) { return TRUE; }
    SetLastError(error); return FALSE;
}

static HANDLE delayed_open(LPCWSTR path, DWORD access, DWORD share, LPSECURITY_ATTRIBUTES security,
                           DWORD disposition, DWORD flags, HANDLE template_file) {
    if (!delayed_gate(path, L"slow-open")) { return INVALID_HANDLE_VALUE; }
    return CreateFileW(path, access, share, security, disposition, flags, template_file);
}
static BOOL delayed_read(HANDLE file, LPVOID output, DWORD capacity, LPDWORD count, LPOVERLAPPED overlap) {
    WCHAR path[4096]; LARGE_INTEGER zero = {0}, position;
    if (!GetFinalPathNameByHandleW(file, path, ARRAYSIZE(path), 0) ||
        !SetFilePointerEx(file, zero, &position, FILE_CURRENT)) { return FALSE; }
    if (position.QuadPart == 0 && !delayed_gate(path, L"slow-read")) { return FALSE; }
    return ReadFile(file, output, capacity, count, overlap);
}
static BOOL delayed_metadata(HANDLE file, FILE_INFO_BY_HANDLE_CLASS kind, LPVOID output, DWORD capacity) {
    WCHAR path[4096];
    if (!GetFinalPathNameByHandleW(file, path, ARRAYSIZE(path), 0) || !delayed_gate(path, L"slow-meta")) { return FALSE; }
    return GetFileInformationByHandleEx(file, kind, output, capacity);
}
