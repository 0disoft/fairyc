#define UNICODE
#define _UNICODE
#include <windows.h>
#include <shellapi.h>
#include <stdint.h>

// Consume exactly one Shell payload; no file I/O or window creation.
size_t hash_drop_read(void *payload, uint8_t *output, size_t capacity) {
    HDROP drop = (HDROP)payload;
    size_t result = 131073;
    WCHAR *wide = NULL;
    if (!drop) { return result; }
    if (!output || !capacity || capacity > 131072 || DragQueryFileW(drop,0xffffffff,NULL,0) != 1) { goto done; }
    UINT length = DragQueryFileW(drop,0,NULL,0);
    if (!length || length >= 32768) { goto done; }
    wide = HeapAlloc(GetProcessHeap(),0,((size_t)length+1)*sizeof(WCHAR));
    if (!wide || DragQueryFileW(drop,0,wide,length+1) != length) { goto done; }
    int written = WideCharToMultiByte(CP_UTF8,WC_ERR_INVALID_CHARS,wide,(int)length,
        (char *)output,(int)capacity,NULL,NULL);
    if (written > 0 && (size_t)written < capacity) { result = (size_t)written; }
done:
    if (wide) { HeapFree(GetProcessHeap(),0,wide); }
    DragFinish(drop);
    return result;
}
