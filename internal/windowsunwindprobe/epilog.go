package windowsunwindprobe

const EpilogFairySource = `module main;
@link_name("fairyc_fixture_v2_epilogs")
unsafe extern C fn i32 verify_epilogs();
fn i32 main() {
    return unsafe { verify_epilogs() };
}
`

// EpilogCompanionC asks the system unwinder to restore controlled contexts at
// instruction boundaries in the executable's real linked version-2 epilogs.
// It does not execute arbitrary PCs or fabricate function-table registrations.
const EpilogCompanionC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

static void fy_v2_return_target(void) { }

static DWORD64 *fy_v2_register(CONTEXT *context, unsigned int reg) {
    switch (reg) {
        case 3: return &context->Rbx;
        case 5: return &context->Rbp;
        case 6: return &context->Rsi;
        case 7: return &context->Rdi;
        case 12: return &context->R12;
        case 13: return &context->R13;
        case 14: return &context->R14;
        case 15: return &context->R15;
        default: return NULL;
    }
}

static int fy_v2_epilog(DWORD64 base, PRUNTIME_FUNCTION entry, DWORD start,
    unsigned int size, unsigned int *contexts, unsigned int *pops) {
    const BYTE *code = (const BYTE *)(uintptr_t)(base + start);
    unsigned int registers[8], offsets[9], count = 0, position = 0;
    /* Closed decoder for the pop/ret epilogs in the current linked CRT. An
       unobserved instruction fails the probe instead of becoming a skip. */
    while (position < size) {
        unsigned int at = position, reg;
        BYTE prefix = 0;
        if (code[position] == 0x41) {
            prefix = 8;
            if (++position == size) { return 10; }
        }
        if (code[position] == 0xc3 && !prefix && position + 1 == size) {
            offsets[count] = at;
            break;
        }
        if (code[position] < 0x58 || code[position] > 0x5f || count == 8) { return 11; }
        reg = code[position] - 0x58u + prefix;
        if (reg != 3 && reg != 5 && reg != 6 && reg != 7 && reg < 12) { return 12; }
        for (unsigned int prior = 0; prior < count; ++prior) {
            if (registers[prior] == reg) { return 13; }
        }
        offsets[count] = at;
        registers[count++] = reg;
        ++position;
    }
    if (position >= size) { return 14; }
    __declspec(align(16)) DWORD64 stack[16];
    CONTEXT expected;
    memset(&expected, 0, sizeof(expected));
    expected.ContextFlags = CONTEXT_FULL;
    for (unsigned int reg = 0; reg < 16; ++reg) {
        DWORD64 *value = fy_v2_register(&expected, reg);
        if (value) { *value = UINT64_C(0x1122334400000000) + reg; }
    }
    for (unsigned int vector = 0; vector < 10; ++vector) {
        (&expected.Xmm6)[vector].Low = UINT64_C(0x8877665500000000) + vector;
        (&expected.Xmm6)[vector].High = (LONGLONG)(UINT64_C(0x3344556600000000) + vector);
    }
    for (unsigned int index = 0; index < count; ++index) {
        stack[index] = *fy_v2_register(&expected, registers[index]);
    }
    stack[count] = (DWORD64)(uintptr_t)&fy_v2_return_target;
    expected.Rip = stack[count];
    expected.Rsp = (DWORD64)(uintptr_t)&stack[count + 1];
    for (unsigned int step = 0; step <= count; ++step) {
        CONTEXT context = expected;
        context.Rip = base + start + offsets[step];
        context.Rsp = (DWORD64)(uintptr_t)&stack[step];
        for (unsigned int index = step; index < count; ++index) {
            *fy_v2_register(&context, registers[index]) = UINT64_C(0xabcdef0000000000) + index;
        }
        PVOID data = NULL;
        DWORD64 frame = 0;
        PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(UNW_FLAG_NHANDLER, base,
            context.Rip, entry, &context, &data, &frame, NULL);
        if (handler != NULL || context.Rip != expected.Rip || context.Rsp != expected.Rsp) { return 15; }
        for (unsigned int reg = 0; reg < 16; ++reg) {
            DWORD64 *actual = fy_v2_register(&context, reg);
            if (actual && *actual != *fy_v2_register(&expected, reg)) { return 16; }
        }
        if (memcmp(&context.Xmm6, &expected.Xmm6, 10 * sizeof(M128A)) != 0) { return 17; }
        ++*contexts;
    }
    *pops += count;
    return 0;
}

int32_t fairyc_fixture_v2_epilogs(void) {
    BYTE *image = (BYTE *)GetModuleHandleW(NULL);
    const IMAGE_DOS_HEADER *dos = (const IMAGE_DOS_HEADER *)(const void *)image;
    const IMAGE_NT_HEADERS64 *nt = (const IMAGE_NT_HEADERS64 *)(const void *)(image + dos->e_lfanew);
    IMAGE_DATA_DIRECTORY directory = nt->OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_EXCEPTION];
    PRUNTIME_FUNCTION entries = (PRUNTIME_FUNCTION)(void *)(image + directory.VirtualAddress);
    unsigned int functions = 0, epilogs = 0, contexts = 0, pops = 0, explicit_offsets = 0;
    for (DWORD index = 0; index < directory.Size / sizeof(RUNTIME_FUNCTION); ++index) {
        PRUNTIME_FUNCTION entry = &entries[index];
        const BYTE *info = image + entry->UnwindData;
        if ((info[0] & 7) != 2) { continue; }
        if ((info[0] >> 3) != 0 || info[2] == 0 || (info[5] & 15) != 6) { return 20; }
        ++functions;
        unsigned int size = info[4], first = 1;
        if (size == 0 || (info[5] >> 4) > 1) { return 21; }
        if (info[5] >> 4) {
            int result = fy_v2_epilog((DWORD64)(uintptr_t)image, entry,
                entry->EndAddress - size, size, &contexts, &pops);
            if (result) { return result; }
            ++epilogs;
        }
        for (; first < info[2] && (info[5 + first * 2] & 15) == 6; ++first) {
            DWORD offset = info[4 + first * 2] + (DWORD)(info[5 + first * 2] >> 4) * 256;
            if (offset == 0) { break; }
            if (offset < size || offset > entry->EndAddress - entry->BeginAddress) { return 22; }
            int result = fy_v2_epilog((DWORD64)(uintptr_t)image, entry,
                entry->EndAddress - offset, size, &contexts, &pops);
            if (result) { return result; }
            ++epilogs;
            ++explicit_offsets;
        }
    }
    if (!functions || epilogs < 3 || contexts <= epilogs || pops < 2 || !explicit_offsets) { return 23; }
    printf("{\"functions\":%u,\"epilogs\":%u,\"contexts\":%u,\"pops\":%u,\"explicitOffsets\":%u}\n",
        functions, epilogs, contexts, pops, explicit_offsets);
    return 42;
}
`
