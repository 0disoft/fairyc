@TRAP_RECEIPT_INCLUDE@#include <stdint.h>
#include <stdlib.h>
#include <stddef.h>

#define FY_RUNTIME_TRAP_EXIT_BASE UINT32_C(@TRAP_EXIT_BASE@)

static _Noreturn void fy_runtime_dispatch_v0(uint32_t category, const unsigned char *fy_trap_receipt, size_t fy_trap_receipt_length) {
    if (category < UINT32_C(1) || category > UINT32_C(7)) {
        _Exit(255);
    }
    (void)fy_trap_receipt;
    (void)fy_trap_receipt_length;
@TRAP_RECEIPT_WRITE@    _Exit((int)(FY_RUNTIME_TRAP_EXIT_BASE + category));
}

_Noreturn void fy_runtime_trap(uint32_t category) {
    unsigned char receipt = (unsigned char)category;
    fy_runtime_dispatch_v0(category, &receipt, (size_t)1);
}

_Noreturn void fy_runtime_trap_at_v0(uint32_t category, const char *source, uint32_t source_length, uint64_t start_byte, uint64_t end_byte) {
    unsigned char receipt[276];
    if (source == NULL || source_length == 0 || source_length > 256 || start_byte > end_byte) {
        _Exit(255);
    }
    receipt[0] = (unsigned char)category;
    receipt[1] = 1;
    receipt[2] = (unsigned char)(source_length & UINT32_C(255));
    receipt[3] = (unsigned char)(source_length >> 8);
    for (uint32_t index = 0; index < 8; ++index) {
        receipt[4 + index] = (unsigned char)(start_byte >> (8 * index));
        receipt[12 + index] = (unsigned char)(end_byte >> (8 * index));
    }
    for (uint32_t index = 0; index < source_length; ++index) {
        receipt[20 + index] = (unsigned char)source[index];
    }
    fy_runtime_dispatch_v0(category, receipt, (size_t)20 + source_length);
}
