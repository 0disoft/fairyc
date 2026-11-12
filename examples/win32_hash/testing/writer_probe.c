#include "../native/writer.c"
#include <stdio.h>

#define CHECK(test) do { if (!(test)) { fprintf(stderr, "writer check failed at %d: %s (error=%lu)\n", __LINE__, #test, GetLastError()); return 1; } } while (0)
static BOOL operation(writer *value, uint32_t code, const uint8_t *path, size_t count) {
    if (hash_writer_submit(value, code, path, count) || WaitForSingleObject(value->ready, 3000)) { return FALSE; }
    return value->result.error == 0;
}
static BOOL finish(writer *value, BOOL cancel) {
    hash_writer_finish(value, cancel);
    if (WaitForSingleObject(value->done, 3000) || value->result.error) { return FALSE; }
    hash_writer_drop(value); return TRUE;
}
int wmain(int argc, WCHAR **argv) {
    CHECK(argc == 2);
    uint8_t target[4096], temp[4096]; WCHAR temporary[4096];
    int length = WideCharToMultiByte(CP_UTF8, 0, argv[1], -1, (char *)target, sizeof(target), NULL, NULL);
    CHECK(length > 0 && swprintf_s(temporary, ARRAYSIZE(temporary), L"%ls.tmp", argv[1]) > 0);
    int tlength = WideCharToMultiByte(CP_UTF8, 0, temporary, -1, (char *)temp, sizeof(temp), NULL, NULL);
    CHECK(tlength > 0);
    const uint8_t data[] = "prepared new report\n";
    for (int pass = 0; pass < 3; ++pass) {
        writer *value = hash_writer_open(target, (size_t)length-1, data, sizeof(data)-1);
        CHECK(value != NULL);
        CHECK(hash_writer_open(target, (size_t)length-1, data, sizeof(data)-1) == NULL);
        CHECK(operation(value, 1, NULL, 0) && value->result.exists == 1);
        CHECK(operation(value, 2, temp, (size_t)tlength-1));
        CHECK(operation(value, 3, NULL, 0) && operation(value, 4, NULL, 0));
        if (pass == 0) { CHECK(finish(value, TRUE)); }
        else if (pass == 1) {
            CHECK(SetFileAttributesW(argv[1], FILE_ATTRIBUTE_READONLY));
            CHECK(!operation(value, 5, NULL, 0));
            CHECK(finish(value, FALSE));
            CHECK(SetFileAttributesW(argv[1], FILE_ATTRIBUTE_NORMAL));
        } else { CHECK(operation(value, 5, NULL, 0) && finish(value, FALSE)); }
        CHECK(GetFileAttributesW(temporary) == INVALID_FILE_ATTRIBUTES);
        HANDLE file = CreateFileW(argv[1], GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL);
        char output[64] = {0}; DWORD count = 0;
        CHECK(file != INVALID_HANDLE_VALUE && ReadFile(file, output, sizeof(output), &count, NULL) && CloseHandle(file));
        if (pass < 2) { CHECK(count == 3 && !memcmp(output, "old", 3)); }
        else { CHECK(count == sizeof(data)-1 && !memcmp(output, data, count)); }
        ULONGLONG until = GetTickCount64()+3000;
        while (InterlockedCompareExchange(&live_writers, 0, 0) && GetTickCount64() < until) { Sleep(1); }
        CHECK(!InterlockedCompareExchange(&live_writers, 0, 0));
    }
    printf("{\"ok\":true,\"verification\":\"native-writer-transport\",\"cases\":3,\"oldResultProtected\":true,\"temporaryRemoved\":true,\"liveWriters\":0}\n");
    return 0;
}
