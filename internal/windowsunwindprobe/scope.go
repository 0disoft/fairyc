package windowsunwindprobe

const ScopeFairySource = `module main;
@link_name("fairyc_fixture_scope_cases")
unsafe extern C fn i32 scope_cases();
fn i32 main() { return unsafe { scope_cases() }; }
`

// All SEH dispatch and cleanup finish inside C before returning to Fairy.
const ScopeCompanionC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#define FY_SCOPE_EXCEPTION ((DWORD)0xE042F11Eu)
static unsigned int fy_scope_trace, fy_scope_abnormal, fy_scope_valid;
static DWORD fy_scope_data_rva, fy_scope_handler_rva, fy_scope_pc, fy_scope_count;
__declspec(noinline) int fy_scope_body(int kind);
static void fy_scope_event(unsigned int value) { fy_scope_trace = fy_scope_trace * 10 + value; }
static LONG fy_scope_filter(EXCEPTION_POINTERS *p, int outer) {
    EXCEPTION_RECORD *r = p->ExceptionRecord;
    if (r->ExceptionCode != FY_SCOPE_EXCEPTION || r->NumberParameters != 2 ||
        r->ExceptionInformation[0] < 1 || r->ExceptionInformation[0] > 2 ||
        r->ExceptionInformation[1] != 37) { fy_scope_valid = 0; return EXCEPTION_CONTINUE_SEARCH; }
    fy_scope_event(outer ? 4 : 1);
    return outer || r->ExceptionInformation[0] == 1 ? EXCEPTION_EXECUTE_HANDLER : EXCEPTION_CONTINUE_SEARCH;
}
__declspec(noinline) static void fy_scope_touch(volatile char *p) { p[0] = 17; }
__declspec(noinline) static int fy_scope_capture(void) {
    CONTEXT context;
    RtlCaptureContext(&context);
    DWORD64 wanted_base = 0;
    PRUNTIME_FUNCTION wanted = RtlLookupFunctionEntry((DWORD64)(uintptr_t)&fy_scope_body, &wanted_base, NULL);
    if (!wanted) { return 0; }
    for (unsigned int depth = 0; depth < 32 && context.Rip && context.Rsp; ++depth) {
        DWORD64 base = 0, frame = 0;
        PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(context.Rip, &base, NULL);
        if (!entry) { context.Rip = *(DWORD64 *)(uintptr_t)context.Rsp; context.Rsp += 8; continue; }
        int body = base == wanted_base && entry->BeginAddress == wanted->BeginAddress;
        DWORD64 pc = context.Rip;
        PVOID data = NULL;
        PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(body ? UNW_FLAG_EHANDLER : UNW_FLAG_NHANDLER,
            base, pc, entry, &context, &data, &frame, NULL);
        if (body) {
            const BYTE *info = (const BYTE *)(uintptr_t)(base + entry->UnwindData);
            const DWORD *tail = (const DWORD *)(const void *)(info + 4 + ((info[2] + 1u) & ~1u) * 2u);
            if (!handler || (DWORD64)(uintptr_t)handler != base + *tail || data != tail + 1 || !frame) { return 0; }
            const SCOPE_TABLE_AMD64 *scopes = (const SCOPE_TABLE_AMD64 *)data;
            if (!scopes->Count || scopes->Count > 32) { return 0; }
            unsigned int covering = 0;
            for (DWORD i = 0; i < scopes->Count; ++i) {
                if (pc >= base + scopes->ScopeRecord[i].BeginAddress && pc < base + scopes->ScopeRecord[i].EndAddress) { ++covering; }
            }
            if (covering < 2) { return 0; }
            fy_scope_data_rva = (DWORD)((DWORD64)(uintptr_t)data - base);
            fy_scope_handler_rva = *tail;
            fy_scope_pc = (DWORD)(pc - base);
            fy_scope_count = scopes->Count;
            return 1;
        }
    }
    return 0;
}
__declspec(noinline) int fy_scope_body(int kind) {
    char cookie_storage[48];
    fy_scope_touch(cookie_storage);
    __try {
        __try {
            if (!fy_scope_capture()) { fy_scope_valid = 0; }
            if (kind) { ULONG_PTR args[2] = {(ULONG_PTR)kind, 37}; RaiseException(FY_SCOPE_EXCEPTION, EXCEPTION_NONCONTINUABLE, 2, args); }
        } __except (fy_scope_filter(GetExceptionInformation(), 0)) { fy_scope_event(2); }
    } __finally { fy_scope_event(3); fy_scope_abnormal = AbnormalTermination() != 0; }
    return cookie_storage[0] == 17;
}
__declspec(noinline) int fy_scope_outer(int kind) {
    __try { return fy_scope_body(kind); }
    __except (fy_scope_filter(GetExceptionInformation(), 1)) { fy_scope_event(5); return 1; }
}
__declspec(noinline) int fy_scope_constant(void) {
    __try { ULONG_PTR args[2] = {1, 37}; RaiseException(FY_SCOPE_EXCEPTION, EXCEPTION_NONCONTINUABLE, 2, args); }
    __except (EXCEPTION_EXECUTE_HANDLER) { return 1; }
    return 0;
}
int32_t fairyc_fixture_scope_cases(void) {
    unsigned int traces[3], abnormal[3];
    fy_scope_valid = 1;
    for (int kind = 0; kind < 3; ++kind) {
        fy_scope_trace = 0; fy_scope_abnormal = 0;
        if (!fy_scope_outer(kind)) { return 91; }
        traces[kind] = fy_scope_trace; abnormal[kind] = fy_scope_abnormal;
    }
    if (!fy_scope_valid || traces[0] != 3 || traces[1] != 123 || traces[2] != 1435 ||
        abnormal[0] || abnormal[1] || abnormal[2] != 1 || !fy_scope_constant()) { return 92; }
    printf("{\"dataRVA\":%lu,\"handlerRVA\":%lu,\"controlRVA\":%lu,\"count\":%lu,\"normal\":%u,\"inner\":%u,\"outer\":%u,\"abnormal\":%u}\n",
        fy_scope_data_rva, fy_scope_handler_rva, fy_scope_pc, fy_scope_count, traces[0], traces[1], traces[2], abnormal[2]);
    return 42;
}
`
