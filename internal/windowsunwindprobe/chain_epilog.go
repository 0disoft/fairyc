package windowsunwindprobe

// Only admit instruction boundaries reached from the fragment entry through
// simple MOV instructions, then ADD RSP, nonvolatile POPs and RET. Unknown bytes
// stop classification; searching for an opcode inside operands is never allowed.
const chainEpilogC = `
typedef struct {
    unsigned int offsets[10], registers[8], pops, add_size;
    size_t allocation;
} fy_chain_epilog;

static size_t fy_chain_mov_size(const BYTE *code, size_t size) {
    size_t position = 0;
    BYTE rex = 0;
    if (!size) { return 0; }
    if (code[0] >= 0x40 && code[0] <= 0x4f) { rex = code[position++]; }
    if (position == size) { return 0; }
    BYTE op = code[position++];
    if (op >= 0xb0 && op <= 0xbf) {
        size_t immediate = op < 0xb8 ? 1 : (rex & 8) ? 8 : 4;
        return immediate <= size - position ? position + immediate : 0;
    }
    if (op == 0x0f) {
        // MOVZX r32, r8; only a register operand is admitted.
        return position + 2 <= size && code[position] == 0xb6 &&
            (code[position + 1] & 0xc0) == 0xc0 ? position + 2 : 0;
    }
    if (op != 0x8b || position == size) { return 0; }
    BYTE modrm = code[position++];
    unsigned int mod = modrm >> 6, rm = modrm & 7;
    if (mod == 3) { return position; }
    if (mod == 0 && rm != 4 && rm != 5) { return position; } // MOV from [register].
    if ((mod != 1 && mod != 2) || rm != 4 || (rex & 3) || position == size || code[position++] != 0x24) { return 0; }
    size_t displacement = mod == 1 ? 1 : 4; // MOV from [RSP + displacement].
    return displacement <= size - position ? position + displacement : 0;
}

static int fy_chain_decode_epilog(const BYTE *code, size_t size, fy_chain_epilog *plan) {
    memset(plan, 0, sizeof(*plan));
    size_t position = 0;
    for (unsigned int prefix = 0; prefix < 16; ++prefix) {
        size_t consumed = fy_chain_mov_size(code + position, size - position);
        if (!consumed) { break; }
        position += consumed;
    }
    if (size - position < 4 || code[position] != 0x48 || code[position + 2] != 0xc4) { return 0; }
    plan->offsets[0] = (unsigned int)position;
    if (code[position + 1] == 0x83 && code[position + 3] < 0x80) {
        plan->allocation = code[position + 3];
        plan->add_size = 4;
    } else if (code[position + 1] == 0x81 && size - position >= 7) {
        DWORD allocation = fy_chain_u32(code + position + 3);
        if (allocation >= 0x80000000u) { return 0; }
        plan->allocation = allocation;
        plan->add_size = 7;
    } else { return 0; }
    if (!plan->allocation || plan->allocation % 8 || plan->allocation > 2048) { return 0; }
    position += plan->add_size;
    while (position < size) {
        plan->offsets[plan->pops + 1] = (unsigned int)position;
        if (code[position] == 0xc3) { return 1; }
        unsigned int reg = 0;
        if (code[position] == 0x41) {
            reg = 8;
            if (++position == size) { return 0; }
        }
        if (code[position] < 0x58 || code[position] > 0x5f || plan->pops == 8) { return 0; }
        reg += code[position++] - 0x58u;
        CONTEXT scratch;
        if (!fy_chain_register(&scratch, reg)) { return 0; }
        for (unsigned int prior = 0; prior < plan->pops; ++prior) {
            if (plan->registers[prior] == reg) { return 0; }
        }
        plan->registers[plan->pops++] = reg;
    }
    return 0;
}

static int fy_chain_epilog_context(BYTE *image, PRUNTIME_FUNCTION entry,
    const fy_chain_epilog *plan, unsigned int *contexts) {
    __declspec(align(16)) DWORD64 stack[512];
    CONTEXT expected;
    memset(stack, 0, sizeof(stack));
    memset(&expected, 0, sizeof(expected));
    expected.ContextFlags = CONTEXT_FULL;
    for (unsigned int reg = 0; reg < 16; ++reg) {
        DWORD64 *value = fy_chain_register(&expected, reg);
        if (value) { *value = UINT64_C(0x9988776600000000) + reg; }
    }
    for (unsigned int vector = 0; vector < 10; ++vector) {
        (&expected.Xmm6)[vector].Low = UINT64_C(0x5566778800000000) + vector;
        (&expected.Xmm6)[vector].High = (LONGLONG)(UINT64_C(0x2233445500000000) + vector);
    }
    size_t fixed = plan->allocation / 8;
    if (fixed + plan->pops + 1 >= sizeof(stack) / sizeof(stack[0])) { return 50; }
    for (unsigned int pop = 0; pop < plan->pops; ++pop) {
        stack[fixed + pop] = *fy_chain_register(&expected, plan->registers[pop]);
    }
    stack[fixed + plan->pops] = (DWORD64)(uintptr_t)&fy_chain_return_target;
    expected.Rip = stack[fixed + plan->pops];
    expected.Rsp = (DWORD64)(uintptr_t)&stack[fixed + plan->pops + 1];
    for (unsigned int step = 0; step <= plan->pops + 1; ++step) {
        CONTEXT context = expected;
        unsigned int consumed = step ? step - 1 : 0;
        context.Rip = (DWORD64)(uintptr_t)image + entry->BeginAddress + plan->offsets[step];
        context.Rsp = (DWORD64)(uintptr_t)&stack[step ? fixed + consumed : 0];
        // Other registers are already restored before this epilog PC. Poison
        // only the registers whose POP instructions have not executed yet.
        for (unsigned int pop = consumed; pop < plan->pops; ++pop) {
            *fy_chain_register(&context, plan->registers[pop]) = UINT64_C(0xabcdef0000000000) + pop;
        }
        PVOID data = NULL;
        DWORD64 frame = 0;
        PEXCEPTION_ROUTINE handler = RtlVirtualUnwind(UNW_FLAG_NHANDLER, (DWORD64)(uintptr_t)image,
            context.Rip, entry, &context, &data, &frame, NULL);
        if (handler || context.Rip != expected.Rip || context.Rsp != expected.Rsp) { return 51; }
        for (unsigned int reg = 0; reg < 16; ++reg) {
            DWORD64 *value = fy_chain_register(&context, reg);
            if (value && *value != *fy_chain_register(&expected, reg)) { return 52; }
        }
        if (memcmp(&context.Xmm6, &expected.Xmm6, 10 * sizeof(M128A))) { return 53; }
        ++*contexts;
    }
    return 0;
}

static int fy_chain_reject_epilogs(BYTE *image, PRUNTIME_FUNCTION entry,
    const BYTE *code, size_t size, const fy_chain_epilog *plan,
    unsigned int *rejected, unsigned int *rejected_contexts) {
    BYTE copy[256];
    fy_chain_epilog decoded;
    size_t end = plan->offsets[plan->pops + 1] + 1;
    if (end > sizeof(copy) || end > size || !plan->pops) { return 60; }
    for (unsigned int mutation = 0; mutation < 6; ++mutation) {
        memcpy(copy, code, end);
        size_t available = end;
        switch (mutation) {
            case 0: --available; break; // Missing RET.
            case 1: copy[end - 1] = 0xcc; break;
            case 2: // Negative ADD immediate.
                copy[plan->offsets[0] + plan->add_size - 1] |= 0x80; break;
            case 3: copy[plan->offsets[0] + 2] = 0xc0; break; // ADD RAX instead of RSP.
            case 4: // POP a volatile register, accounting for the optional REX.B.
                copy[plan->offsets[1] + (copy[plan->offsets[1]] == 0x41)] = 0x58; break;
            case 5: copy[0] = 0xe9; break; // Unadmitted branch prefix.
        }
        if (fy_chain_decode_epilog(copy, available, &decoded)) { return 61; }
        ++*rejected;
    }
    // Plausible but wrong decoded expectations must disagree with the real OS
    // unwind of the unchanged image. Never execute mutated instructions.
    fy_chain_epilog wrong = *plan;
    unsigned int ignored = 0;
    wrong.allocation += 8;
    if (fy_chain_epilog_context(image, entry, &wrong, &ignored) != 51) { return 62; }
    ++*rejected_contexts;
    wrong = *plan;
    wrong.registers[0] = plan->registers[0] == 5 ? 3 : 5;
    if (fy_chain_epilog_context(image, entry, &wrong, &ignored) != 52) { return 63; }
    ++*rejected_contexts;
    return 0;
}
`
