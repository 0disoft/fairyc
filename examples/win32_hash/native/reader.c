#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <string.h>

// Win32 I/O transport only. Fairy owns chunk consumption, hashing and policy.
// The worker never holds a Fairy buffer, callback or window handle.
typedef struct {
    uint32_t phase, error, kind, directory;
    size_t count, size;
    uint32_t volume_low, volume_high, file_low_0, file_low_1, file_high_0, file_high_1;
    size_t created, written, changed;
} observation;

typedef struct {
    LONG references, cancelled;
    HANDLE request, ready, stop, thread, file;
    WCHAR *path;
    observation result;
    uint8_t buffer[65536];
} reader;

// Bound retired reads even if a filesystem driver never honors cancellation.
static LONG live_readers;

static void release(reader *value) {
    if (InterlockedDecrement(&value->references) != 0) { return; }
    CloseHandle(value->request); CloseHandle(value->ready);
    CloseHandle(value->stop);
    if (value->thread) { CloseHandle(value->thread); }
    HeapFree(GetProcessHeap(), 0, value->path);
    HeapFree(GetProcessHeap(), 0, value);
    InterlockedDecrement(&live_readers);
}

static BOOL cancelled(reader *value) { return InterlockedCompareExchange(&value->cancelled, 0, 0) != 0; }

static BOOL stamp(HANDLE file, observation *result) {
    LARGE_INTEGER size;
    FILE_ID_INFO id;
    FILE_BASIC_INFO times;
    BY_HANDLE_FILE_INFORMATION basic;
    result->kind = GetFileType(file);
    if (result->kind != FILE_TYPE_DISK) { return TRUE; }
    if (!GetFileInformationByHandle(file, &basic)) { return FALSE; }
    result->directory = (basic.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0;
    if (result->directory) { return TRUE; }
    if (!GetFileSizeEx(file, &size) || size.QuadPart < 0 ||
        !GetFileInformationByHandleEx(file, FileIdInfo, &id, sizeof(id))) { return FALSE; }
    if (!GetFileInformationByHandleEx(file, FileBasicInfo, &times, sizeof(times))) { return FALSE; }
    result->created = (size_t)times.CreationTime.QuadPart;
    result->written = (size_t)times.LastWriteTime.QuadPart;
    result->changed = (size_t)times.ChangeTime.QuadPart;
    result->size = (size_t)size.QuadPart;
    result->volume_low = (uint32_t)id.VolumeSerialNumber;
    result->volume_high = (uint32_t)(id.VolumeSerialNumber >> 32);
    uint32_t words[4]; memcpy(words, id.FileId.Identifier, sizeof(words));
    result->file_low_0 = words[0]; result->file_low_1 = words[1];
    result->file_high_0 = words[2]; result->file_high_1 = words[3];
    return TRUE;
}

static BOOL metadata(reader *value) { return stamp(value->file, &value->result); }

static DWORD WINAPI run_reader(void *context) {
    reader *value = context;
    value->file = CreateFileW(value->path, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING,
        FILE_FLAG_SEQUENTIAL_SCAN | FILE_FLAG_BACKUP_SEMANTICS, NULL);
    if (value->file == INVALID_HANDLE_VALUE || !metadata(value)) { goto failed; }
    value->result.phase = 1; // Open/metadata complete; Fairy decides whether to read.
    SetEvent(value->ready);
    for (;;) {
        if (WaitForSingleObject(value->request, INFINITE) != WAIT_OBJECT_0 || cancelled(value)) { break; }
        DWORD count = 0;
        if (!ReadFile(value->file, value->buffer, sizeof(value->buffer), &count, NULL)) { goto failed; }
        if (cancelled(value)) { break; }
        value->result.count = count;
        if (count == 0) {
            if (!metadata(value)) { goto failed; }
            CloseHandle(value->file); value->file = INVALID_HANDLE_VALUE;
            value->result.phase = 3; SetEvent(value->ready); break;
        }
        value->result.phase = 2; SetEvent(value->ready);
    }
    goto finished;
failed:
    value->result.error = GetLastError();
    if (value->file != INVALID_HANDLE_VALUE) { CloseHandle(value->file); value->file = INVALID_HANDLE_VALUE; }
    value->result.phase = 4; SetEvent(value->ready);
finished:
    if (value->file != INVALID_HANDLE_VALUE) { CloseHandle(value->file); }
    SetEvent(value->stop);
    release(value);
    return 0;
}

static DWORD WINAPI cancel_reader(void *context) {
    reader *value = context;
    WaitForSingleObject(value->stop, INFINITE);
    // Even the cancellation request stays off the GUI thread. A driver may
    // delay its cancellation routine or immediately reissue synchronous I/O.
    if (cancelled(value)) { CancelSynchronousIo(value->thread); }
    release(value);
    return 0;
}

void *hash_reader_open(const uint8_t *path, size_t length) {
    if (!path || !length || length > 131071 || memchr(path, 0, length)) { return NULL; }
    if (InterlockedIncrement(&live_readers) > 2) { InterlockedDecrement(&live_readers); return NULL; }
    reader *value = HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(*value));
    if (!value) { InterlockedDecrement(&live_readers); return NULL; }
    value->references = 1; value->file = INVALID_HANDLE_VALUE;
    value->path = HeapAlloc(GetProcessHeap(), 0, (length + 1) * sizeof(WCHAR));
    value->request = CreateEventW(NULL, FALSE, FALSE, NULL);
    value->ready = CreateEventW(NULL, TRUE, FALSE, NULL);
    value->stop = CreateEventW(NULL, TRUE, FALSE, NULL);
    if (!value->path || !value->request || !value->ready || !value->stop) { release(value); return NULL; }
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)path, (int)length, value->path, (int)length);
    if (!count) { release(value); return NULL; }
    value->path[count] = 0;
    value->references = 2;
    value->thread = CreateThread(NULL, 0, run_reader, value, 0, NULL);
    if (!value->thread) { value->references = 1; release(value); return NULL; }
    InterlockedIncrement(&value->references);
    HANDLE canceller = CreateThread(NULL, 0, cancel_reader, value, 0, NULL);
    if (!canceller) {
        release(value); // Cancel worker was not started.
        InterlockedExchange(&value->cancelled, 1); SetEvent(value->request);
        release(value); // Read worker keeps storage until its I/O returns.
        return NULL;
    }
    CloseHandle(canceller);
    return value;
}

int32_t hash_reader_available(void) { return InterlockedCompareExchange(&live_readers, 0, 0) < 2; }

observation hash_reader_observe(void *context) {
    reader *value = context;
    observation empty = {0};
    if (!value || WaitForSingleObject(value->ready, 0) != WAIT_OBJECT_0) { return empty; }
    return value->result; // Immutable until the GUI acknowledges with next().
}

void hash_reader_snapshot(void *context, void *output) {
    observation value = hash_reader_observe(context);
    memcpy(output, &value, sizeof(value));
}

int32_t hash_reader_copy(void *context, uint8_t *output, size_t capacity) {
    reader *value = context;
    if (!value || !output || WaitForSingleObject(value->ready, 0) != WAIT_OBJECT_0 ||
        value->result.phase != 2 || value->result.count > capacity) { return 1; }
    memcpy(output, value->buffer, value->result.count);
    return 0;
}

int32_t hash_reader_next(void *context) {
    reader *value = context;
    if (!value || WaitForSingleObject(value->ready, 0) != WAIT_OBJECT_0 ||
        (value->result.phase != 1 && value->result.phase != 2)) { return 1; }
    ResetEvent(value->ready);
    return SetEvent(value->request) ? 0 : 1;
}

void hash_reader_wait(void *context) {
    reader *value = context;
    if (value) { MsgWaitForMultipleObjectsEx(1, &value->ready, 16, QS_ALLINPUT, MWMO_INPUTAVAILABLE); }
}

void hash_reader_drop(void *context) {
    reader *value = context;
    if (!value) { return; }
    InterlockedExchange(&value->cancelled, 1);
    SetEvent(value->request);
    SetEvent(value->stop);
    // No file API, cancellation call or worker join runs on the GUI thread.
    release(value);
}

// Synchronous CLI guard: use only owned Win32 handles, never std::fs internals.
typedef struct { HANDLE file; WCHAR *path; observation initial; } source_guard;
void hash_source_drop(void *context) {
    source_guard *value = context;
    if (!value) { return; }
    if (value->file != INVALID_HANDLE_VALUE) { CloseHandle(value->file); }
    HeapFree(GetProcessHeap(), 0, value->path); HeapFree(GetProcessHeap(), 0, value);
}
void *hash_source_open(const uint8_t *path, size_t length) {
    if (!path || !length || length > 131071 || memchr(path, 0, length)) { return NULL; }
    source_guard *value = HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(*value));
    if (!value) { return NULL; }
    value->file = INVALID_HANDLE_VALUE;
    value->path = HeapAlloc(GetProcessHeap(), 0, (length+1)*sizeof(WCHAR));
    if (!value->path) { hash_source_drop(value); return NULL; }
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)path, (int)length, value->path, (int)length);
    if (!count) { hash_source_drop(value); return NULL; }
    value->path[count] = 0;
    value->file = CreateFileW(value->path, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS, NULL);
    value->initial.phase = 1;
    if (value->file == INVALID_HANDLE_VALUE || !stamp(value->file, &value->initial)) {
        value->initial.error = GetLastError(); value->initial.phase = 4;
    }
    return value;
}
void hash_source_snapshot(void *context, void *output, uint32_t refresh) {
    source_guard *value = context;
    observation result = value->initial;
    if (refresh && result.phase != 4) {
        HANDLE file = value->file;
        if (refresh == 2) { file = CreateFileW(value->path, FILE_READ_ATTRIBUTES, FILE_SHARE_READ, NULL, OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS, NULL); }
        if (file == INVALID_HANDLE_VALUE || !stamp(file, &result)) { result.error = GetLastError(); result.phase = 4; }
        if (refresh == 2 && file != INVALID_HANDLE_VALUE && !CloseHandle(file)) { result.error = GetLastError(); result.phase = 4; }
    }
    memcpy(output, &result, sizeof(result));
}
