#include "../native/drop.c"
#include <shlobj.h>
#include <stdio.h>
#include <string.h>

static HDROP payload(const WCHAR *names, size_t bytes) {
    HGLOBAL block = GlobalAlloc(GHND,sizeof(DROPFILES)+bytes);
    DROPFILES *data = GlobalLock(block);
    if (!data) { if (block) { GlobalFree(block); } return NULL; }
    data->pFiles = sizeof(DROPFILES); data->fWide = TRUE;
    memcpy((uint8_t *)data+sizeof(DROPFILES),names,bytes);
    GlobalUnlock(block); return (HDROP)block;
}

int main(void) {
    const WCHAR one[] = L"C:\\data\\입력 Ω.bin\0";
    const WCHAR two[] = L"C:\\one.bin\0C:\\two.bin\0";
    const WCHAR empty[] = L"\0";
    const WCHAR invalid[] = {L'C',L':',L'\\',0xd800,0,0};
    uint8_t output[131072];
    const char *expected = "C:\\data\\입력 Ω.bin";
    size_t length = hash_drop_read(payload(one,sizeof(one)),output,sizeof(output));
    if (length != strlen(expected) || memcmp(output,expected,length)) { return 1; }
    if (hash_drop_read(payload(two,sizeof(two)),output,sizeof(output)) != 131073) { return 2; }
    if (hash_drop_read(payload(empty,sizeof(empty)),output,sizeof(output)) != 131073) { return 3; }
    if (hash_drop_read(payload(one,sizeof(one)),output,4) != 131073) { return 4; }
    if (hash_drop_read(payload(invalid,sizeof(invalid)),output,sizeof(output)) != 131073) { return 5; }
    WCHAR *long_name = HeapAlloc(GetProcessHeap(),HEAP_ZERO_MEMORY,32770*sizeof(WCHAR));
    if (!long_name) { return 6; }
    for (size_t i=0;i<32768;i++) { long_name[i]=L'x'; }
    HDROP drop = payload(long_name,32770*sizeof(WCHAR)); HeapFree(GetProcessHeap(),0,long_name);
    if (hash_drop_read(drop,output,sizeof(output)) != 131073) { return 7; }
    puts("{\"ok\":true,\"verification\":\"native-shell-drop-decoder\",\"cases\":6,\"guiLaunched\":false,\"inputInjected\":false}");
    return 0;
}
