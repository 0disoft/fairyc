#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>

// One outstanding Win32 operation. Fairy chooses validation, staging and commit.
// No worker retains Fairy memory, HWNDs or callbacks.
typedef struct {
    uint32_t phase, error, exists;
    uint32_t volume_low, volume_high, file_low_0, file_low_1, file_high_0, file_high_1;
} writer_observation;
typedef struct {
    LONG references, stopping, cancelled;
    HANDLE request, ready, stop, done, thread, file;
    WCHAR *target, *temporary;
    uint8_t *data;
    size_t length;
    uint32_t operation;
    BOOL published, armed;
    writer_observation result;
} writer;
static LONG live_writers;

static WCHAR *writer_path(const uint8_t *path, size_t length) {
    if (!path || !length || length > 131071 || memchr(path, 0, length)) { return NULL; }
    WCHAR *wide = HeapAlloc(GetProcessHeap(), 0, (length+1)*sizeof(WCHAR));
    if (!wide) { return NULL; }
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)path, (int)length, wide, (int)length);
    if (!count) { HeapFree(GetProcessHeap(), 0, wide); return NULL; }
    wide[count] = 0; return wide;
}
static BOOL writer_cancelled(writer *value) { return InterlockedCompareExchange(&value->cancelled, 0, 0) != 0; }
static void writer_release(writer *value) {
    if (InterlockedDecrement(&value->references)) { return; }
    CloseHandle(value->request); CloseHandle(value->ready); CloseHandle(value->stop); CloseHandle(value->done);
    if (value->thread) { CloseHandle(value->thread); }
    HeapFree(GetProcessHeap(), 0, value->target); HeapFree(GetProcessHeap(), 0, value->temporary);
    HeapFree(GetProcessHeap(), 0, value->data); HeapFree(GetProcessHeap(), 0, value);
    InterlockedDecrement(&live_writers);
}
static BOOL writer_disposition(writer *value, BOOL remove) {
    FILE_DISPOSITION_INFO disposition = {(BOOLEAN)(remove != FALSE)};
    BOOL ok = SetFileInformationByHandle(value->file, FileDispositionInfo, &disposition, sizeof(disposition));
    if (ok) { value->armed = remove; }
    return ok;
}
static BOOL writer_check(writer *value) {
    HANDLE file = CreateFileW(value->target, FILE_READ_ATTRIBUTES, FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        NULL, OPEN_EXISTING, FILE_FLAG_BACKUP_SEMANTICS, NULL);
    if (file == INVALID_HANDLE_VALUE) {
        DWORD error = GetLastError();
        if (error == ERROR_FILE_NOT_FOUND || error == ERROR_PATH_NOT_FOUND) { return TRUE; }
        SetLastError(error); return FALSE;
    }
    FILE_ID_INFO id;
    BOOL ok = GetFileInformationByHandleEx(file, FileIdInfo, &id, sizeof(id));
    DWORD error = ok ? 0 : GetLastError();
    if (!CloseHandle(file) && ok) { error = GetLastError(); ok = FALSE; }
    if (!ok) { SetLastError(error); return FALSE; }
    uint32_t words[4]; memcpy(words, id.FileId.Identifier, sizeof(words));
    value->result.exists = 1;
    value->result.volume_low = (uint32_t)id.VolumeSerialNumber;
    value->result.volume_high = (uint32_t)(id.VolumeSerialNumber >> 32);
    value->result.file_low_0 = words[0]; value->result.file_low_1 = words[1];
    value->result.file_high_0 = words[2]; value->result.file_high_1 = words[3];
    return TRUE;
}
static BOOL writer_create(writer *value) {
    value->file = CreateFileW(value->temporary, GENERIC_WRITE | DELETE, 0, NULL, CREATE_NEW, FILE_ATTRIBUTE_NORMAL, NULL);
    if (value->file == INVALID_HANDLE_VALUE) { return FALSE; }
    // Reversible disposition: process/handle closure removes staged bytes.
    // Normal Close waits for worker cleanup; no filesystem call runs on UI.
    return writer_disposition(value, TRUE);
}
static BOOL writer_write(writer *value) {
    size_t offset = 0;
    while (offset < value->length) {
        if (writer_cancelled(value)) { SetLastError(ERROR_OPERATION_ABORTED); return FALSE; }
        DWORD count = 0, remaining = (DWORD)(value->length-offset);
        if (!WriteFile(value->file, value->data+offset, remaining, &count, NULL)) { return FALSE; }
        if (!count || count > remaining) { SetLastError(ERROR_WRITE_FAULT); return FALSE; }
        offset += count;
    }
    return TRUE;
}
static BOOL writer_publish(writer *value) {
    size_t bytes = wcslen(value->target)*sizeof(WCHAR);
    DWORD capacity = (DWORD)(sizeof(FILE_RENAME_INFO)+bytes);
    FILE_RENAME_INFO *rename = HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, capacity);
    if (!rename) { SetLastError(ERROR_NOT_ENOUGH_MEMORY); return FALSE; }
    rename->ReplaceIfExists = TRUE; rename->FileNameLength = (DWORD)bytes;
    memcpy(rename->FileName, value->target, bytes);
    // Once Fairy authorizes commit, cancellation is disabled and normal Close
    // keeps pumping until rename/cleanup finishes. Never delete a committed file.
    BOOL ok = writer_disposition(value, FALSE);
    if (ok) { ok = SetFileInformationByHandle(value->file, FileRenameInfo, rename, capacity); }
    DWORD error = ok ? 0 : GetLastError();
    if (ok) { value->published = TRUE; }
    else { writer_disposition(value, TRUE); }
    HeapFree(GetProcessHeap(), 0, rename); SetLastError(error); return ok;
}
static DWORD WINAPI writer_run(void *context) {
    writer *value = context;
    for (;;) {
        if (WaitForSingleObject(value->request, INFINITE) != WAIT_OBJECT_0 ||
            InterlockedCompareExchange(&value->stopping, 0, 0)) { break; }
        BOOL ok = FALSE;
        switch (value->operation) {
        case 1: ok = writer_check(value); break;
        case 2: ok = writer_create(value); break;
        case 3: ok = writer_write(value); break;
        case 4: ok = FlushFileBuffers(value->file); break;
        case 5: ok = writer_publish(value); break;
        default: SetLastError(ERROR_INVALID_PARAMETER); break;
        }
        value->result.error = ok ? 0 : GetLastError();
        value->result.phase = 1; SetEvent(value->ready);
    }
    DWORD cleanup_error = 0;
    if (value->file != INVALID_HANDLE_VALUE) {
        if (!value->published && !value->armed && !writer_disposition(value, TRUE)) { cleanup_error = GetLastError(); }
        if (!CloseHandle(value->file)) { cleanup_error = GetLastError(); }
        value->file = INVALID_HANDLE_VALUE;
    }
    value->result.error = cleanup_error; value->result.phase = 2;
    SetEvent(value->done); SetEvent(value->stop); writer_release(value); return 0;
}
static DWORD WINAPI writer_cancel(void *context) {
    writer *value = context;
    WaitForSingleObject(value->stop, INFINITE);
    if (writer_cancelled(value)) { CancelSynchronousIo(value->thread); }
    writer_release(value); return 0;
}
void *job_writer_open(const uint8_t *target, size_t length, const uint8_t *data, size_t count) {
    if ((!data && count) || count > 1048576) { return NULL; }
    if (InterlockedIncrement(&live_writers) != 1) { InterlockedDecrement(&live_writers); return NULL; }
    writer *value = HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(*value));
    if (!value) { InterlockedDecrement(&live_writers); return NULL; }
    value->references = 1; value->file = INVALID_HANDLE_VALUE;
    value->target = writer_path(target, length); value->length = count;
    value->data = HeapAlloc(GetProcessHeap(), 0, count ? count : 1);
    value->request = CreateEventW(NULL, FALSE, FALSE, NULL);
    value->ready = CreateEventW(NULL, TRUE, TRUE, NULL);
    value->stop = CreateEventW(NULL, TRUE, FALSE, NULL);
    value->done = CreateEventW(NULL, TRUE, FALSE, NULL);
    if (!value->target || !value->data || !value->request || !value->ready || !value->stop || !value->done) { writer_release(value); return NULL; }
    if (count) { memcpy(value->data, data, count); } value->result.phase = 1;
    value->references = 2; value->thread = CreateThread(NULL, 0, writer_run, value, 0, NULL);
    if (!value->thread) { value->references = 1; writer_release(value); return NULL; }
    InterlockedIncrement(&value->references);
    HANDLE cancel = CreateThread(NULL, 0, writer_cancel, value, 0, NULL);
    if (!cancel) {
        writer_release(value); InterlockedExchange(&value->stopping, 1); SetEvent(value->request);
        writer_release(value); return NULL;
    }
    CloseHandle(cancel); return value;
}
int32_t job_writer_submit(void *context, uint32_t operation, const uint8_t *path, size_t length) {
    writer *value = context;
    if (!value || operation < 1 || operation > 5 || InterlockedCompareExchange(&value->stopping, 0, 0) ||
        WaitForSingleObject(value->ready, 0) != WAIT_OBJECT_0) { return 1; }
    if (operation == 2) {
        WCHAR *wide = writer_path(path, length);
        if (!wide || value->file != INVALID_HANDLE_VALUE) { HeapFree(GetProcessHeap(), 0, wide); return 1; }
        HeapFree(GetProcessHeap(), 0, value->temporary); value->temporary = wide;
    }
    memset(&value->result, 0, sizeof(value->result)); value->operation = operation;
    ResetEvent(value->ready); return SetEvent(value->request) ? 0 : 1;
}
void job_writer_snapshot(void *context, void *output) {
    writer *value = context; writer_observation empty = {0};
    if (WaitForSingleObject(value->done, 0) == WAIT_OBJECT_0 ||
        (!InterlockedCompareExchange(&value->stopping, 0, 0) && WaitForSingleObject(value->ready, 0) == WAIT_OBJECT_0)) {
        memcpy(output, &value->result, sizeof(value->result));
    } else { memcpy(output, &empty, sizeof(empty)); }
}
void job_writer_finish(void *context, uint32_t cancel) {
    writer *value = context;
    if (cancel) { InterlockedExchange(&value->cancelled, 1); }
    InterlockedExchange(&value->stopping, 1); SetEvent(value->request); SetEvent(value->stop);
}
void job_writer_wait(void *context) {
    writer *value = context;
    HANDLE event = InterlockedCompareExchange(&value->stopping, 0, 0) ? value->done : value->ready;
    MsgWaitForMultipleObjectsEx(1, &event, 16, QS_ALLINPUT, MWMO_INPUTAVAILABLE);
}
void job_writer_drop(void *context) {
    if (context) { job_writer_finish(context, 0); writer_release(context); }
}
