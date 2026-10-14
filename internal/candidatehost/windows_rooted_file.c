/* Private handle-relative file ABI only. Fairy owns cache keys, byte limits,
 * I/O loops, flush/publication order, miss policy and resource cleanup. */
#include <windows.h>
#include <winternl.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>

typedef NTSTATUS (NTAPI *fy_root_nt_create)(PHANDLE, ACCESS_MASK, POBJECT_ATTRIBUTES,
    PIO_STATUS_BLOCK, PLARGE_INTEGER, ULONG, ULONG, ULONG, ULONG, PVOID, ULONG);
typedef ULONG (NTAPI *fy_root_nt_error)(NTSTATUS);
typedef NTSTATUS (NTAPI *fy_root_nt_set)(HANDLE, PIO_STATUS_BLOCK, PVOID, ULONG, FILE_INFORMATION_CLASS);

static int fy_root_leaf(const uint8_t *data, size_t bytes) {
    if (data == NULL || bytes == 0 || bytes > 510 || bytes % 2 != 0) {
        SetLastError(ERROR_INVALID_NAME); return 0;
    }
    const WCHAR *name = (const WCHAR *)data;
    size_t count = bytes / 2;
    if ((count == 1 && name[0] == L'.') ||
        (count == 2 && name[0] == L'.' && name[1] == L'.')) {
        SetLastError(ERROR_INVALID_NAME); return 0;
    }
    for (size_t i = 0; i < count; i++) {
        if (name[i] == 0 || name[i] == L'/' || name[i] == L'\\' || name[i] == L':') {
            SetLastError(ERROR_INVALID_NAME); return 0;
        }
    }
    return 1;
}

static int fy_root_regular(HANDLE file) {
    BY_HANDLE_FILE_INFORMATION info;
    if (!GetFileInformationByHandle(file, &info)) { return 0; }
    if (GetFileType(file) != FILE_TYPE_DISK ||
        (info.dwFileAttributes & (FILE_ATTRIBUTE_DIRECTORY | FILE_ATTRIBUTE_REPARSE_POINT)) != 0) {
        SetLastError(ERROR_ACCESS_DENIED); return 0;
    }
    return 1;
}

size_t fairyc_private_windows_directory_open_v1(const uint8_t *path, size_t bytes) {
    if (path == NULL || bytes < 4 || bytes > 65534 || bytes % 2 != 0 ||
        ((const WCHAR *)path)[bytes / 2 - 1] != 0) {
        SetLastError(ERROR_INVALID_NAME); return 0;
    }
    for (size_t i = 0; i + 1 < bytes / 2; i++) {
        if (((const WCHAR *)path)[i] == 0) { SetLastError(ERROR_INVALID_NAME); return 0; }
    }
    HANDLE directory = CreateFileW((LPCWSTR)path, FILE_LIST_DIRECTORY | FILE_TRAVERSE | FILE_READ_ATTRIBUTES | SYNCHRONIZE,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE, NULL, OPEN_EXISTING,
        FILE_FLAG_BACKUP_SEMANTICS | FILE_FLAG_OPEN_REPARSE_POINT, NULL);
    if (directory == INVALID_HANDLE_VALUE) { return 0; }
    BY_HANDLE_FILE_INFORMATION info;
    if (!GetFileInformationByHandle(directory, &info)) {
        DWORD error = GetLastError(); CloseHandle(directory); SetLastError(error); return 0;
    }
    if ((info.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) == 0 ||
        (info.dwFileAttributes & FILE_ATTRIBUTE_REPARSE_POINT) != 0 || GetFileType(directory) != FILE_TYPE_DISK) {
        CloseHandle(directory); SetLastError(ERROR_ACCESS_DENIED); return 0;
    }
    return (size_t)(uintptr_t)directory;
}

size_t fairyc_private_windows_root_file_open_v1(size_t root, const uint8_t *leaf, size_t bytes, uint32_t mode) {
    if (root == 0 || mode < 1 || mode > 3) { SetLastError(ERROR_INVALID_PARAMETER); return 0; }
    if (!fy_root_leaf(leaf, bytes)) { return 0; }
    HMODULE module = GetModuleHandleW(L"ntdll.dll");
    if (module == NULL) { return 0; }
    fy_root_nt_create create = (fy_root_nt_create)GetProcAddress(module, "NtCreateFile");
    fy_root_nt_error error = (fy_root_nt_error)GetProcAddress(module, "RtlNtStatusToDosError");
    if (create == NULL || error == NULL) { SetLastError(ERROR_PROC_NOT_FOUND); return 0; }
    UNICODE_STRING name = { (USHORT)bytes, (USHORT)bytes, (PWSTR)leaf };
    OBJECT_ATTRIBUTES attributes = { sizeof(OBJECT_ATTRIBUTES), (HANDLE)root, &name, 0x40, NULL, NULL };
    IO_STATUS_BLOCK io;
    HANDLE file = NULL;
    ACCESS_MASK access = GENERIC_READ | SYNCHRONIZE;
    if (mode == 2) { access |= GENERIC_WRITE | DELETE; }
    if (mode == 3) { access |= DELETE; }
    NTSTATUS status = create(&file, access, &attributes, &io, NULL, FILE_ATTRIBUTE_NORMAL,
        mode == 3 ? FILE_SHARE_READ : FILE_SHARE_READ | FILE_SHARE_DELETE, mode == 2 ? 2u : 1u,
        0x00000040u | 0x00000020u | 0x00200000u, NULL, 0);
    if (status < 0) { SetLastError(error(status)); return 0; }
    if (!fy_root_regular(file)) {
        DWORD failure = GetLastError(); CloseHandle(file); SetLastError(failure); return 0;
    }
    return (size_t)(uintptr_t)file;
}

int32_t fairyc_private_windows_root_file_size_v1(size_t file, size_t *size) {
    LARGE_INTEGER value;
    if (!GetFileSizeEx((HANDLE)(uintptr_t)file, &value)) { return 0; }
    if (value.QuadPart < 0 || (uint64_t)value.QuadPart > (uint64_t)SIZE_MAX) {
        SetLastError(ERROR_FILE_TOO_LARGE); return 0;
    }
    *size = (size_t)value.QuadPart; return 1;
}
int32_t fairyc_private_windows_root_file_read_v1(size_t file, uint8_t *data, size_t size, size_t *copied) {
    if (size > 65536) { SetLastError(ERROR_INVALID_PARAMETER); return 0; }
    DWORD count = 0;
    int32_t success = ReadFile((HANDLE)(uintptr_t)file, data, (DWORD)size, &count, NULL);
    *copied = count; return success;
}
int32_t fairyc_private_windows_root_file_write_v1(size_t file, const uint8_t *data, size_t size, size_t *written) {
    if (size > 65536) { SetLastError(ERROR_INVALID_PARAMETER); return 0; }
    DWORD count = 0;
    int32_t success = WriteFile((HANDLE)(uintptr_t)file, data, (DWORD)size, &count, NULL);
    *written = count; return success;
}
int32_t fairyc_private_windows_root_file_sync_v1(size_t file) { return FlushFileBuffers((HANDLE)(uintptr_t)file); }
int32_t fairyc_private_windows_root_file_close_v1(size_t handle) { return CloseHandle((HANDLE)(uintptr_t)handle); }
int32_t fairyc_private_windows_root_file_delete_v1(size_t file) {
    FILE_DISPOSITION_INFO information = { TRUE };
    return SetFileInformationByHandle((HANDLE)(uintptr_t)file, FileDispositionInfo, &information, sizeof(information));
}
static int32_t fy_root_file_rename(size_t file, size_t root, const uint8_t *leaf, size_t bytes, BOOLEAN replace) {
    if (root == 0) { SetLastError(ERROR_INVALID_PARAMETER); return 0; }
    if (!fy_root_leaf(leaf, bytes)) { return 0; }
    struct { FILE_RENAME_INFO information; WCHAR tail[255]; } buffer;
    memset(&buffer, 0, sizeof(buffer));
    buffer.information.ReplaceIfExists = replace;
    buffer.information.RootDirectory = (HANDLE)(uintptr_t)root;
    buffer.information.FileNameLength = (DWORD)bytes;
    memcpy(buffer.information.FileName, leaf, bytes);
    HMODULE module = GetModuleHandleW(L"ntdll.dll");
    if (module == NULL) { return 0; }
    fy_root_nt_set set = (fy_root_nt_set)GetProcAddress(module, "NtSetInformationFile");
    fy_root_nt_error error = (fy_root_nt_error)GetProcAddress(module, "RtlNtStatusToDosError");
    if (set == NULL || error == NULL) { SetLastError(ERROR_PROC_NOT_FOUND); return 0; }
    IO_STATUS_BLOCK io;
    NTSTATUS status = set((HANDLE)(uintptr_t)file, &io, &buffer,
        (ULONG)(sizeof(FILE_RENAME_INFO) + bytes), (FILE_INFORMATION_CLASS)10);
    if (status < 0) { SetLastError(error(status)); return 0; }
    return 1;
}

int32_t fairyc_private_windows_root_file_rename_v1(size_t file, size_t root, const uint8_t *leaf, size_t bytes) {
    return fy_root_file_rename(file, root, leaf, bytes, TRUE);
}
int32_t fairyc_private_windows_root_file_publish_new_v1(size_t file, size_t root, const uint8_t *leaf, size_t bytes) {
    return fy_root_file_rename(file, root, leaf, bytes, FALSE);
}
size_t fairyc_private_windows_root_file_time_v1(void) {
    FILETIME now;
    GetSystemTimeAsFileTime(&now);
    return (size_t)(((uint64_t)now.dwHighDateTime << 32) | now.dwLowDateTime);
}

struct fy_root_file_metadata_v1 { size_t identity; size_t modified; size_t size; uint32_t attributes; };
int32_t fairyc_private_windows_root_file_info_v1(size_t file, void *output) {
    BY_HANDLE_FILE_INFORMATION info;
    if (output == NULL) { SetLastError(ERROR_INVALID_PARAMETER); return 0; }
    if (!GetFileInformationByHandle((HANDLE)(uintptr_t)file, &info)) { return 0; }
    struct fy_root_file_metadata_v1 value = {
        (size_t)(((uint64_t)info.nFileIndexHigh << 32) | info.nFileIndexLow),
        (size_t)(((uint64_t)info.ftLastWriteTime.dwHighDateTime << 32) | info.ftLastWriteTime.dwLowDateTime),
        (size_t)(((uint64_t)info.nFileSizeHigh << 32) | info.nFileSizeLow), info.dwFileAttributes
    };
    memcpy(output, &value, sizeof(value)); return 1;
}
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, FileName) == 104, "private directory name offset");
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, FileId) == 96, "private directory ID offset");
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, LastWriteTime) == 24, "private directory time offset");
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, EndOfFile) == 40, "private directory size offset");
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, FileAttributes) == 56, "private directory attributes offset");
_Static_assert(offsetof(FILE_ID_BOTH_DIR_INFO, FileNameLength) == 60, "private directory length offset");
int32_t fairyc_private_windows_root_directory_query_v1(size_t root, uint8_t *data, size_t size, uint32_t restart) {
    if (root == 0 || data == NULL || size < sizeof(FILE_ID_BOTH_DIR_INFO) || size > 65536 ||
        ((uintptr_t)data & 7u) != 0 || restart > 1) {
        SetLastError(ERROR_INVALID_PARAMETER); return 0;
    }
    return GetFileInformationByHandleEx((HANDLE)(uintptr_t)root,
        restart != 0 ? FileIdBothDirectoryRestartInfo : FileIdBothDirectoryInfo, data, (DWORD)size);
}
