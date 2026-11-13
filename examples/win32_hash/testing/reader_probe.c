#include "../native/reader.c"
#include <stdio.h>

#define CHECK(test) do { if (!(test)) { fprintf(stderr, "reader check failed at %d: %s\n", __LINE__, #test); return 1; } } while (0)

int wmain(int argc, WCHAR **argv) {
    CHECK(argc == 2);
    uint8_t path[4096], output[65536];
    int length = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, argv[1], -1, (char *)path, sizeof(path), NULL, NULL);
    CHECK(length > 0);
    reader *value = job_reader_open(path, (size_t)length-1);
    CHECK(value != NULL && WaitForSingleObject(value->ready, 3000) == WAIT_OBJECT_0);
    observation first = job_reader_observe(value);
    CHECK(first.phase == 1 && first.kind == FILE_TYPE_DISK && !first.directory && first.size == 1234567);
    size_t total = 0;
    for (;;) {
        CHECK(job_reader_next(value) == 0);
        CHECK(WaitForSingleObject(value->ready, 3000) == WAIT_OBJECT_0);
        observation next = job_reader_observe(value);
        if (next.phase == 3) {
            CHECK(next.size == first.size && total == first.size);
            CHECK(memcmp(&next.volume_low, &first.volume_low, 24) == 0);
            continue;
        }
        if (next.phase == 5) {
            CHECK(next.size == first.size && total == first.size && next.written == first.written && next.changed == first.changed);
            CHECK(memcmp(&next.volume_low, &first.volume_low, 24) == 0);
            break;
        }
        CHECK(next.phase == 2 && next.count > 0 && next.count <= sizeof(output));
        CHECK(job_reader_copy(value, output, next.count-1) == 1);
        CHECK(job_reader_copy(value, output, sizeof(output)) == 0);
        for (size_t at = 0; at < next.count; at++) { CHECK(output[at] == (uint8_t)((total+at)%256)); }
        total += next.count;
    }
    job_reader_drop(value);
    source_guard *guard = job_source_open(path, (size_t)length-1);
    CHECK(guard && guard->initial.phase == 1 && guard->initial.written && guard->initial.changed);
    HANDLE writer = CreateFileW(argv[1], GENERIC_WRITE, FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        NULL, OPEN_EXISTING, 0, NULL);
    CHECK(writer == INVALID_HANDLE_VALUE && GetLastError() == ERROR_SHARING_VIOLATION);
    HANDLE attributes = CreateFileW(argv[1], FILE_WRITE_ATTRIBUTES, FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
        NULL, OPEN_EXISTING, 0, NULL);
    CHECK(attributes != INVALID_HANDLE_VALUE);
    FILETIME original, touched;
    CHECK(GetFileTime(attributes, NULL, NULL, &original));
    ULARGE_INTEGER time; time.LowPart = original.dwLowDateTime; time.HighPart = original.dwHighDateTime;
    time.QuadPart += 10000000; touched.dwLowDateTime = time.LowPart; touched.dwHighDateTime = time.HighPart;
    CHECK(SetFileTime(attributes, NULL, NULL, &touched) && CloseHandle(attributes));
    observation modified;
    job_source_snapshot(guard, &modified, 1);
    CHECK(modified.phase == 1 && modified.size == guard->initial.size && modified.written != guard->initial.written);
    job_source_snapshot(guard, &modified, 2);
    CHECK(modified.phase == 1 && modified.written != guard->initial.written);
    job_source_drop(guard);
    WCHAR pipe[128];
    CHECK(swprintf_s(pipe, ARRAYSIZE(pipe), L"\\\\.\\pipe\\fairy-reader-%lu", GetCurrentProcessId()) > 0);
    HANDLE server = CreateNamedPipeW(pipe, PIPE_ACCESS_OUTBOUND, PIPE_TYPE_BYTE | PIPE_WAIT, 1, 4096, 4096, 0, NULL);
    CHECK(server != INVALID_HANDLE_VALUE);
    length = WideCharToMultiByte(CP_UTF8, 0, pipe, -1, (char *)path, sizeof(path), NULL, NULL);
    value = job_reader_open(path, (size_t)length-1);
    CHECK(value != NULL);
    CHECK(ConnectNamedPipe(server, NULL) || GetLastError() == ERROR_PIPE_CONNECTED);
    CHECK(WaitForSingleObject(value->ready, 3000) == WAIT_OBJECT_0);
    CHECK(job_reader_observe(value).kind == FILE_TYPE_PIPE);
    CHECK(job_reader_next(value) == 0);
    BOOL pending = FALSE; ULONGLONG deadline = GetTickCount64()+3000;
    while (!pending && GetTickCount64() < deadline) {
        CHECK(GetThreadIOPendingFlag(value->thread, &pending)); Sleep(1);
    }
    CHECK(pending);
    HANDLE thread = NULL;
    CHECK(DuplicateHandle(GetCurrentProcess(), value->thread, GetCurrentProcess(), &thread, SYNCHRONIZE, FALSE, 0));
    ULONGLONG started = GetTickCount64();
    job_reader_drop(value);
    ULONGLONG elapsed = GetTickCount64()-started;
    CHECK(elapsed < 500 && WaitForSingleObject(thread, 3000) == WAIT_OBJECT_0);
    CloseHandle(thread); CloseHandle(server);
    deadline = GetTickCount64()+3000;
    while (InterlockedCompareExchange(&live_readers, 0, 0) != 0 && GetTickCount64() < deadline) { Sleep(1); }
    CHECK(InterlockedCompareExchange(&live_readers, 0, 0) == 0);
    // A cancelled worker must not poison the next job in the same process.
    length = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, argv[1], -1, (char *)path, sizeof(path), NULL, NULL);
    CHECK(length > 0 && job_reader_available());
    value = job_reader_open(path, (size_t)length-1);
    CHECK(value && WaitForSingleObject(value->ready,3000) == WAIT_OBJECT_0 && job_reader_next(value) == 0);
    CHECK(WaitForSingleObject(value->ready,3000) == WAIT_OBJECT_0 && job_reader_copy(value,output,sizeof(output)) == 0);
    CHECK(output[0] == 0 && output[255] == 255);
    job_reader_drop(value);
    deadline = GetTickCount64()+3000;
    while (InterlockedCompareExchange(&live_readers,0,0) && GetTickCount64()<deadline) { Sleep(1); }
    CHECK(!InterlockedCompareExchange(&live_readers,0,0));
    printf("{\"ok\":true,\"verification\":\"native-reader-transport\",\"bytes\":%zu,\"actualPendingRead\":true,\"cancelMs\":%llu,\"liveReaders\":0,\"reusedAfterCancel\":true,\"overwriteDenied\":true,\"sameSizeMetadataChange\":true}\n", total, elapsed);
    return 0;
}
