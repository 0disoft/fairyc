package windowsunwindprobe

// MixedFairySource exercises the admitted synchronous callback boundary. A trap
// exits the process; foreign SEH is caught before returning through Fairy.
const MixedFairySource = `module main;
@link_name("fairyc_fixture_mixed_outer")
unsafe extern C fn i32 outer(i32 value, unsafe extern C fn i32(i32, ptr mut void) callback, ptr mut void data);
@link_name("fairyc_fixture_mixed_inspect")
unsafe extern C fn i32 inspect(i32 value, ptr mut void data);
@link_name("fairyc_fixture_mixed_observe")
unsafe extern C fn i32 observe();
unsafe fn i32 callback(i32 value, ptr mut void data) {
    assert(value != 13);
    let i32 inspected = inspect(value, data);
    return inspected + 1;
}
fn i32 main() {
    let i32 result = unsafe { outer(INPUT_VALUE, callback, null) };
    assert(result == 42);
    assert(unsafe { observe() } == 1);
    return result;
}
`

const MixedCompanionC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <intrin.h>
#pragma intrinsic(_ReturnAddress)
#pragma intrinsic(_AddressOfReturnAddress)

#ifndef FY_MIXED_HANDLER_SHIFT
#define FY_MIXED_HANDLER_SHIFT 0
#endif
#ifndef FY_MIXED_DATA_SHIFT
#define FY_MIXED_DATA_SHIFT 0
#endif
#ifndef FY_MIXED_STACK_SHIFT
#define FY_MIXED_STACK_SHIFT 0
#endif

typedef int32_t (*fy_mixed_callback)(int32_t, void *);
__declspec(noinline) int32_t fairyc_fixture_mixed_outer(int32_t, fy_mixed_callback, void *);
__declspec(noinline) int32_t fairyc_fixture_mixed_inspect(int32_t, void *);

static DWORD64 fy_mixed_main_pc, fy_mixed_main_sp;
static int fy_mixed_walked, fy_mixed_handler_checked, fy_mixed_caught, fy_mixed_abnormal, fy_mixed_returned;
#define FY_MIXED_EXCEPTION ((DWORD)0xE042F10Eu)

static int fy_mixed_contains(DWORD64 pc, DWORD64 anchor) {
    DWORD64 base = 0;
    PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(anchor, &base, NULL);
    return entry != NULL && pc >= base + entry->BeginAddress && pc < base + entry->EndAddress;
}

/* Check the actual suspended C outer frame, not an unrelated image record.
   Requesting its handler returns metadata; it does not invoke that handler. */
static int fy_mixed_handler(const CONTEXT *before, const CONTEXT *after,
    DWORD64 base, PRUNTIME_FUNCTION entry, DWORD64 walked_frame) {
    const BYTE *info = (const BYTE *)(uintptr_t)(base + entry->UnwindData);
    unsigned int flags = info[0] >> 3;
    if ((info[0] & 7) != 1 || (flags & UNW_FLAG_CHAININFO) != 0 ||
        (flags & (UNW_FLAG_EHANDLER | UNW_FLAG_UHANDLER)) == 0 ||
        before->Rip < base + entry->BeginAddress + info[1]) { return 0; }
    const DWORD *tail = (const DWORD *)(const void *)(info + 4 + ((info[2] + 1u) & ~1u) * 2u);
    CONTEXT context = *before;
    PVOID data = NULL;
    DWORD64 frame = 0;
    DWORD kind = (flags & UNW_FLAG_EHANDLER) ? UNW_FLAG_EHANDLER : UNW_FLAG_UHANDLER;
    PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(kind, base, before->Rip, entry,
        &context, &data, &frame, NULL);
    DWORD64 expected_handler = base + *tail + FY_MIXED_HANDLER_SHIFT;
    const BYTE *expected_data = (const BYTE *)(tail + 1) + FY_MIXED_DATA_SHIFT;
    return (DWORD64)(uintptr_t)handler == expected_handler &&
        data == (const void *)expected_data && frame == walked_frame && frame != 0 &&
        context.Rip == after->Rip && context.Rsp == after->Rsp &&
        context.Rbx == after->Rbx && context.Rbp == after->Rbp &&
        context.Rsi == after->Rsi && context.Rdi == after->Rdi &&
        context.R12 == after->R12 && context.R13 == after->R13 &&
        context.R14 == after->R14 && context.R15 == after->R15 &&
        context.Rip == fy_mixed_main_pc &&
        context.Rsp == fy_mixed_main_sp + FY_MIXED_STACK_SHIFT;
}

/* Virtual unwinding reads metadata and captured registers. It never transfers
   exception control through a Fairy frame. Both generated frames must occur. */
static int fy_mixed_walk(const CONTEXT *initial, DWORD64 callback_pc) {
    CONTEXT context = *initial;
    DWORD64 anchors[4] = {
        (DWORD64)(uintptr_t)&fairyc_fixture_mixed_inspect, callback_pc,
        (DWORD64)(uintptr_t)&fairyc_fixture_mixed_outer, fy_mixed_main_pc
    };
    unsigned int stage = 0;
    for (unsigned int depth = 0; depth < 64 && context.Rip && context.Rsp; ++depth) {
        int outer_frame = stage == 2 && fy_mixed_contains(context.Rip, anchors[stage]);
        if (fy_mixed_contains(context.Rip, anchors[stage])) {
            if (++stage == 4) { return 1; }
        }
        DWORD64 base = 0;
        PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(context.Rip, &base, NULL);
        if (entry != NULL) {
            CONTEXT before = context;
            PVOID handler_data = NULL;
            DWORD64 frame = 0;
            (void)RtlVirtualUnwind(UNW_FLAG_NHANDLER, base, context.Rip, entry,
                &context, &handler_data, &frame, NULL);
            if (outer_frame) {
                if (!fy_mixed_handler(&before, &context, base, entry, frame)) { return 0; }
                ++fy_mixed_handler_checked;
            }
        } else {
            context.Rip = *(const DWORD64 *)(uintptr_t)context.Rsp;
            context.Rsp += sizeof(DWORD64);
        }
    }
    return 0;
}

__declspec(noinline) static void fy_mixed_raise(void) {
    ULONG_PTR payload = 37;
    __try {
        RaiseException(FY_MIXED_EXCEPTION, EXCEPTION_NONCONTINUABLE, 1, &payload);
    } __finally {
        if (AbnormalTermination()) { ++fy_mixed_abnormal; }
    }
}

static LONG fy_mixed_filter(EXCEPTION_POINTERS *information) {
    EXCEPTION_RECORD *record = information->ExceptionRecord;
    if (record->ExceptionCode != FY_MIXED_EXCEPTION || record->NumberParameters != 1 ||
        record->ExceptionInformation[0] != 37) { return EXCEPTION_CONTINUE_SEARCH; }
    return EXCEPTION_EXECUTE_HANDLER;
}

__declspec(noinline) int32_t fairyc_fixture_mixed_inspect(int32_t value, void *data) {
    CONTEXT context;
    RtlCaptureContext(&context);
    fy_mixed_walked = fy_mixed_walk(&context, (DWORD64)(uintptr_t)_ReturnAddress());
    __try { fy_mixed_raise(); }
    __except (fy_mixed_filter(GetExceptionInformation())) { ++fy_mixed_caught; }
    return data == NULL && fy_mixed_walked && fy_mixed_handler_checked == 1 &&
        fy_mixed_caught == 1 && fy_mixed_abnormal == 1 ? value : -100;
}

__declspec(noinline) int32_t fairyc_fixture_mixed_outer(int32_t value, fy_mixed_callback callback, void *data) {
    fy_mixed_main_pc = (DWORD64)(uintptr_t)_ReturnAddress();
    fy_mixed_main_sp = (DWORD64)(uintptr_t)_AddressOfReturnAddress() + sizeof(DWORD64);
    int32_t result = 0;
    __try { result = callback(value, data); }
    __finally {
        if (value == 13 || AbnormalTermination()) {
            fputs("unexpected unwind through Fairy\n", stderr);
            ExitProcess(90);
        }
        ++fy_mixed_returned;
    }
    return result + 1;
}

int32_t fairyc_fixture_mixed_observe(void) {
    return fy_mixed_walked == 1 && fy_mixed_handler_checked == 1 && fy_mixed_caught == 1 &&
        fy_mixed_abnormal == 1 && fy_mixed_returned == 1;
}
`
