package windowsunwindprobe

const ChainFairySource = `module main;
@link_name("fairyc_fixture_chained_unwind")
unsafe extern C fn i32 verify_chains();
fn i32 main() {
    return unsafe { verify_chains() };
}
`

// ChainCompanionC inverts the linked metadata onto a bounded controlled stack,
// then asks the real OS unwinder to recover independent caller sentinels. These
// are not live CRT captures. Zero-prolog epilog contexts instead come from a
// closed instruction decoder; other forms are explicitly counted as untested.
const ChainCompanionC = chainContextC + chainEpilogC + chainRunnerC

const chainContextC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

static void fy_chain_return_target(void) { }

static DWORD64 *fy_chain_register(CONTEXT *context, unsigned int reg) {
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

static unsigned int fy_chain_slots(unsigned int op, unsigned int info) {
    switch (op) {
        case 0: case 2: return 1;
        case 1: return info == 0 ? 2 : info == 1 ? 3 : 0;
        case 4: case 8: return 2;
        case 5: case 9: return 3;
        default: return 0;
    }
}

static unsigned int fy_chain_u16(const BYTE *p) { return p[0] + (unsigned int)p[1] * 256; }
static DWORD fy_chain_u32(const BYTE *p) { DWORD value; memcpy(&value, p, sizeof(value)); return value; }

static int fy_chain_context(BYTE *image, PRUNTIME_FUNCTION entry, const BYTE **records,
    unsigned int links, unsigned int pc_offset, unsigned int *integer_saves, unsigned int *vector_saves) {
    __declspec(align(16)) BYTE stack[32768];
    CONTEXT expected, context;
    memset(stack, 0, sizeof(stack));
    memset(&expected, 0, sizeof(expected));
    expected.ContextFlags = CONTEXT_FULL;
    for (unsigned int reg = 0; reg < 16; ++reg) {
        DWORD64 *value = fy_chain_register(&expected, reg);
        if (value) { *value = UINT64_C(0x1122334400000000) + reg; }
    }
    for (unsigned int vector = 0; vector < 10; ++vector) {
        (&expected.Xmm6)[vector].Low = UINT64_C(0x8877665500000000) + vector;
        (&expected.Xmm6)[vector].High = (LONGLONG)(UINT64_C(0x3344556600000000) + vector);
    }
    size_t cursor = sizeof(stack) - 120; // Entry RSP is 8 mod 16, caller RSP is aligned.
    expected.Rip = (DWORD64)(uintptr_t)&fy_chain_return_target;
    memcpy(stack + cursor, &expected.Rip, 8);
    expected.Rsp = (DWORD64)(uintptr_t)(stack + cursor + 8);
    context = expected;
    for (unsigned int level = links + 1; level-- > 0;) {
        const BYTE *record = records[level];
        unsigned int offsets[255], count = 0;
        for (unsigned int slot = 0; slot < record[2];) {
            offsets[count++] = slot;
            slot += fy_chain_slots(record[5 + slot * 2] & 15, record[5 + slot * 2] >> 4);
        }
        // Reverse prolog operations to construct the input stack. Only the
        // current fragment is partial; every parent record is fully applied.
        while (count) {
            const BYTE *code = record + 4 + offsets[--count] * 2;
            unsigned int op = code[1] & 15, reg = code[1] >> 4;
            if (level == 0 && code[0] > pc_offset) { continue; }
            if (op == 0 || op == 1 || op == 2) {
                size_t size = op == 0 ? 8 : op == 2 ? reg * 8 + 8 :
                    reg == 0 ? (size_t)fy_chain_u16(code + 2) * 8 : fy_chain_u32(code + 2);
                if (size > cursor) { return 30; }
                cursor -= size;
            }
            if (op == 0 || op == 4 || op == 5) {
                size_t offset = op == 0 ? 0 : op == 4 ? (size_t)fy_chain_u16(code + 2) * 8 : fy_chain_u32(code + 2);
                DWORD64 *value = fy_chain_register(&context, reg);
                if (!value || offset > sizeof(stack) - cursor - 8) { return 31; }
                memcpy(stack + cursor + offset, fy_chain_register(&expected, reg), 8);
                *value = UINT64_C(0xabcdef0000000000) + reg;
                ++*integer_saves;
            }
            if (op == 8 || op == 9) {
                size_t offset = op == 8 ? (size_t)fy_chain_u16(code + 2) * 16 : fy_chain_u32(code + 2);
                if (reg < 6 || reg > 15 || offset > sizeof(stack) - cursor - 16) { return 32; }
                memcpy(stack + cursor + offset, &(&expected.Xmm6)[reg - 6], 16);
                (&context.Xmm6)[reg - 6].Low = UINT64_C(0xabcdef0000000000) + reg;
                ++*vector_saves;
            }
        }
    }
    context.Rip = (DWORD64)(uintptr_t)image + entry->BeginAddress + pc_offset;
    context.Rsp = (DWORD64)(uintptr_t)(stack + cursor);
    PVOID data = NULL;
    DWORD64 frame = 0;
    PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(UNW_FLAG_NHANDLER, (DWORD64)(uintptr_t)image,
        context.Rip, entry, &context, &data, &frame, NULL);
    if (handler || context.Rip != expected.Rip || context.Rsp != expected.Rsp ||
        frame != (DWORD64)(uintptr_t)(stack + cursor)) { return 33; }
    for (unsigned int reg = 0; reg < 16; ++reg) {
        DWORD64 *value = fy_chain_register(&context, reg);
        if (value && *value != *fy_chain_register(&expected, reg)) { return 34; }
    }
    if (memcmp(&context.Xmm6, &expected.Xmm6, 10 * sizeof(M128A))) { return 35; }
    return 0;
}
`

const chainRunnerC = `
int32_t fairyc_fixture_chained_unwind(void) {
    BYTE *image = (BYTE *)GetModuleHandleW(NULL);
    const IMAGE_DOS_HEADER *dos = (const IMAGE_DOS_HEADER *)(const void *)image;
    const IMAGE_NT_HEADERS64 *nt = (const IMAGE_NT_HEADERS64 *)(const void *)(image + dos->e_lfanew);
    IMAGE_DATA_DIRECTORY directory = nt->OptionalHeader.DataDirectory[IMAGE_DIRECTORY_ENTRY_EXCEPTION];
    PRUNTIME_FUNCTION entries = (PRUNTIME_FUNCTION)(void *)(image + directory.VirtualAddress);
    unsigned int functions = 0, untested = 0, contexts = 0, max_depth = 0, integer_saves = 0, vector_saves = 0;
    unsigned int chained_functions = 0, zero_functions = 0, zero_frames = 0, zero_operations = 0, zero_calls = 0, zero_other = 0;
    unsigned int zero_epilogs = 0, zero_contexts = 0, zero_pops = 0, zero_wide_adds = 0;
    unsigned int rejected_epilogs = 0, rejected_contexts = 0;
    for (DWORD index = 0; index < directory.Size / sizeof(RUNTIME_FUNCTION); ++index) {
        PRUNTIME_FUNCTION entry = &entries[index];
        const BYTE *record = image + entry->UnwindData, *records[65];
        unsigned int links = 0, frame_form = 0, operation_form = 0;
        if (!((record[0] >> 3) & UNW_FLAG_CHAININFO)) { continue; }
        ++chained_functions;
        unsigned int zero_prolog = record[1] == 0;
        if (zero_prolog) { ++zero_functions; }
        for (;;) {
            if (links >= 65) { return 40; }
            records[links] = record;
            if (record[3]) { frame_form = 1; }
            if ((record[0] & 7) != 1) { operation_form = 1; }
            for (unsigned int slot = 0; slot < record[2];) {
                unsigned int op = record[5 + slot * 2] & 15, info = record[5 + slot * 2] >> 4;
                unsigned int consumed = fy_chain_slots(op, info);
                if (!consumed || slot + consumed > record[2]) { operation_form = 1; break; }
                slot += consumed;
            }
            if (!((record[0] >> 3) & UNW_FLAG_CHAININFO)) { break; }
            const BYTE *tail = record + 4 + ((record[2] + 1u) & ~1u) * 2;
            record = image + fy_chain_u32(tail + 8);
            ++links;
        }
        if (frame_form || operation_form) {
            ++untested;
            if (zero_prolog && frame_form) { ++zero_frames; }
            else if (zero_prolog) { ++zero_operations; }
            continue;
        }
        if (zero_prolog) {
            fy_chain_epilog plan;
            const BYTE *code = image + entry->BeginAddress;
            size_t size = entry->EndAddress - entry->BeginAddress;
            if (!fy_chain_decode_epilog(code, size, &plan)) {
                if (size == 6 && code[0] == 0xe8 && code[5] == 0xcc) { ++zero_calls; }
                else { ++zero_other; }
                ++untested;
                continue;
            }
            int result = fy_chain_epilog_context(image, entry, &plan, &zero_contexts);
            if (result) { fprintf(stderr, "zero-prolog RVA=%lx result=%d\n", entry->BeginAddress, result); return result; }
            if (!rejected_epilogs && plan.pops) {
                result = fy_chain_reject_epilogs(image, entry, code, size, &plan, &rejected_epilogs, &rejected_contexts);
                if (result) { return result; }
            }
            ++zero_epilogs;
            zero_pops += plan.pops;
            zero_wide_adds += plan.add_size == 7;
            if (links > max_depth) { max_depth = links; }
            continue;
        }
        ++functions;
        if (links > max_depth) { max_depth = links; }
        int result = fy_chain_context(image, entry, records, links, 0, &integer_saves, &vector_saves);
        if (result) { fprintf(stderr, "chain RVA=%lx depth=%u offset=0 result=%d\n", entry->BeginAddress, links, result); return result; }
        ++contexts;
        if (records[0][1]) {
            result = fy_chain_context(image, entry, records, links, records[0][1], &integer_saves, &vector_saves);
            if (result) { fprintf(stderr, "chain RVA=%lx depth=%u offset=%u result=%d\n", entry->BeginAddress, links, records[0][1], result); return result; }
            ++contexts;
        }
    }
    if (functions < 10 || contexts <= functions || max_depth < 2 || !integer_saves) { return 41; }
    if (!zero_epilogs || zero_contexts <= zero_epilogs || !zero_pops || !zero_wide_adds ||
        zero_functions != zero_epilogs + zero_frames + zero_operations + zero_calls + zero_other ||
        rejected_epilogs != 6 || rejected_contexts != 2) { return 43; }
    printf("{\"functions\":%u,\"untestedFunctions\":%u,\"contexts\":%u,\"maximumDepth\":%u,\"integerRestores\":%u,\"vectorRestores\":%u,"
        "\"zeroPrologFunctions\":%u,\"zeroPrologEpilogs\":%u,\"zeroPrologContexts\":%u,\"zeroPrologPops\":%u,\"zeroPrologWideAdds\":%u,"
        "\"chainedFunctions\":%u,\"zeroPrologUnsupportedFrames\":%u,\"zeroPrologUnsupportedOperations\":%u,\"zeroPrologCallTails\":%u,\"zeroPrologOtherInstructions\":%u,"
        "\"rejectedEpilogMutations\":%u,\"rejectedEpilogContexts\":%u}\n",
        functions, untested, contexts, max_depth, integer_saves, vector_saves,
        zero_functions, zero_epilogs, zero_contexts, zero_pops, zero_wide_adds, chained_functions, zero_frames, zero_operations, zero_calls, zero_other,
        rejected_epilogs, rejected_contexts);
    return 42;
}
`
