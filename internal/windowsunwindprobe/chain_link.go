package windowsunwindprobe

const ChainLinkFairySource = `module main;
@link_name("fairyc_fixture_chain_link")
unsafe extern C fn i32 verify_chain_link();
fn i32 main() { return unsafe { verify_chain_link() }; }
`

// This companion checks the platform fixture's code independently, executes it,
// then unwinds controlled contexts at its primary and both secondary entries.
const ChainLinkCompanionC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <string.h>
extern int fy_chain(void);
static void fy_chain_link_return(void) { }
int32_t fairyc_fixture_chain_link(void) {
    static const BYTE expected_code[] = {
        0x48, 0x83, 0xec, 0x28, 0x90, 0x90, 0x90, 0x90,
        0xb8, 42, 0, 0, 0, 0x48, 0x83, 0xc4, 0x28, 0xc3
    };
    if (memcmp((const void *)(uintptr_t)&fy_chain, expected_code, sizeof(expected_code)) || fy_chain() != 42) { return 70; }
    for (unsigned int fragment = 0; fragment < 3; ++fragment) {
        __declspec(align(16)) DWORD64 stack[16] = {0};
        CONTEXT context, expected;
        RtlCaptureContext(&expected);
        stack[5] = (DWORD64)(uintptr_t)&fy_chain_link_return;
        expected.Rip = stack[5];
        expected.Rsp = (DWORD64)(uintptr_t)&stack[6];
        context = expected;
        context.Rip = (DWORD64)(uintptr_t)&fy_chain + fragment * 4;
        context.Rsp = (DWORD64)(uintptr_t)&stack[fragment ? 0 : 5];
        DWORD64 base = 0, frame = 0;
        PRUNTIME_FUNCTION entry = RtlLookupFunctionEntry(context.Rip, &base, NULL);
        if (!entry || entry->BeginAddress + base != context.Rip) { return 71; }
        PVOID data = NULL;
        PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(UNW_FLAG_NHANDLER, base, context.Rip,
            entry, &context, &data, &frame, NULL);
        if (handler || context.Rip != expected.Rip || context.Rsp != expected.Rsp ||
            frame != (DWORD64)(uintptr_t)&stack[fragment ? 0 : 5]) { return 72; }
        if (context.Rbx != expected.Rbx || context.Rbp != expected.Rbp || context.Rsi != expected.Rsi ||
            context.Rdi != expected.Rdi || context.R12 != expected.R12 || context.R13 != expected.R13 ||
            context.R14 != expected.R14 || context.R15 != expected.R15 ||
            memcmp(&context.Xmm6, &expected.Xmm6, 10 * sizeof(M128A))) { return 73; }
    }
    return 42;
}
`
