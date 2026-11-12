// Isolated test builds only: controlled real I/O gates and explicitly relaxed
// sharing for change-* fixtures. Production keeps FILE_SHARE_READ throughout.
#define UNICODE
#define _UNICODE
#include <windows.h>
#include <wchar.h>

static LPCWSTR change_base(LPCWSTR path) {
    LPCWSTR base = wcsrchr(path, L'\\'); return base ? base+1 : path;
}
static DWORD change_share(LPCWSTR path) {
    return wcsncmp(change_base(path), L"change-", 7) == 0 ? FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE : FILE_SHARE_READ;
}
static BOOL change_gate(LPCWSTR path, LPCWSTR stage) {
    LPCWSTR base = change_base(path);
    BOOL cli = wcsncmp(base, L"cli-", 4) == 0;
    BOOL eof = wcsstr(base, L"-eof") != NULL;
    if ((!wcscmp(stage, L"cli") && !cli) || (!wcscmp(stage, L"eof") && !eof) ||
        (!wcscmp(stage, L"read") && (cli || eof || (wcsncmp(base, L"change-", 7) && wcsncmp(base, L"protected-", 10))))) { return TRUE; }
    WCHAR prefix[256], name[512];
    if (!GetEnvironmentVariableW(L"FAIRY_HASH_CHANGE_PIPE", prefix, ARRAYSIZE(prefix)) ||
        swprintf_s(name, ARRAYSIZE(name), L"%ls-%ls", prefix, base) < 0) { SetLastError(ERROR_INVALID_PARAMETER); return FALSE; }
    HANDLE pipe = CreateFileW(name, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL);
    if (pipe == INVALID_HANDLE_VALUE) { return FALSE; }
    BYTE byte; DWORD count;
    BOOL ok = ReadFile(pipe, &byte, 1, &count, NULL);
    DWORD error = ok ? 0 : GetLastError(); CloseHandle(pipe);
    if (!error || error == ERROR_BROKEN_PIPE) { return TRUE; }
    SetLastError(error); return FALSE;
}
static BOOL change_read(HANDLE file, LPVOID output, DWORD capacity, LPDWORD count, LPOVERLAPPED overlap) {
    WCHAR path[4096]; LARGE_INTEGER zero = {0}, position;
    if (!GetFinalPathNameByHandleW(file, path, ARRAYSIZE(path), 0) ||
        !SetFilePointerEx(file, zero, &position, FILE_CURRENT)) { return FALSE; }
    // The first chunk has already reached Fairy/SHA before this gate.
    if (position.QuadPart == 65536 && !change_gate(path, L"read")) { return FALSE; }
    return ReadFile(file, output, capacity, count, overlap);
}
