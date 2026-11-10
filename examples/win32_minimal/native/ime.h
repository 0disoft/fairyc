#ifndef FAIRY_EXAMPLE_IME_H
#define FAIRY_EXAMPLE_IME_H
#include <windows.h>

typedef struct {
    BOOL composing, open, during_key;
    DWORD language, conversion, starts, updates, ends, results, keys, key, raw_key;
    WCHAR composition[128];
} FairyIME;

BOOL fairy_ime_attach(HWND input);
void fairy_ime_snapshot(FairyIME *state);
UINT fairy_ime_observe(const MSG *message);
#endif
