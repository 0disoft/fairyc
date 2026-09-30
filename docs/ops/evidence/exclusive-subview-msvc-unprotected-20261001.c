/* fairyc constrained C transport: x86_64-windows-msvc, ISO C11 */
#include <stdint.h>

typedef struct fy_byte_slice {
    const uint8_t *data;
    uintptr_t length;
} fy_byte_slice;

typedef struct fy_mut_byte_slice {
    uint8_t *data;
    uintptr_t length;
} fy_mut_byte_slice;

typedef struct { uint8_t data[UINT64_C(4)]; } fy_array_0;

#define FY_TRAP_ASSERTION_FAILED UINT32_C(6)
#define FY_TRAP_BOUNDS UINT32_C(1)
#define FY_TRAP_INTEGER_OVERFLOW UINT32_C(2)
#define FY_TRAP_UNREACHABLE UINT32_C(7)
_Noreturn void fy_runtime_trap(uint32_t category);

static void fy_fn_0(fy_mut_byte_slice fy_l_5);
static int32_t fy_fn_1(void);
static void fy_fn_2(fy_byte_slice fy_l_6);
static fy_mut_byte_slice fy_fn_3(fy_mut_byte_slice fy_l_4);

static void fy_fn_0(fy_mut_byte_slice fy_l_5) {
    (void)fy_l_5;
    fy_mut_byte_slice fy_l_6;
    (void)&fy_l_6;
    uintptr_t fy_l_7;
    (void)&fy_l_7;
    goto fy_b_0;
fy_b_0:
    fy_l_6 = fy_l_5;
    fy_l_7 = ((uintptr_t)UINT64_C(0));
    if (fy_l_7 >= (fy_l_6).length) { goto fy_b_2; }
    goto fy_b_1;
fy_b_1:
    (fy_l_6).data[fy_l_7] = UINT8_C(9);
    return;
fy_b_2:
    fy_runtime_trap(FY_TRAP_BOUNDS);
}

static int32_t fy_fn_1(void) {
    fy_array_0 fy_l_7;
    (void)&fy_l_7;
    fy_mut_byte_slice fy_l_8;
    (void)&fy_l_8;
    fy_mut_byte_slice fy_l_9;
    (void)&fy_l_9;
    fy_mut_byte_slice fy_l_10;
    (void)&fy_l_10;
    fy_mut_byte_slice fy_l_11;
    (void)&fy_l_11;
    fy_mut_byte_slice fy_l_12;
    (void)&fy_l_12;
    uintptr_t fy_l_13;
    (void)&fy_l_13;
    fy_mut_byte_slice fy_l_14;
    (void)&fy_l_14;
    uintptr_t fy_l_15;
    (void)&fy_l_15;
    fy_mut_byte_slice fy_l_16;
    (void)&fy_l_16;
    fy_array_0 fy_l_17;
    (void)&fy_l_17;
    fy_mut_byte_slice fy_l_18;
    (void)&fy_l_18;
    fy_mut_byte_slice fy_l_19;
    (void)&fy_l_19;
    fy_mut_byte_slice fy_l_20;
    (void)&fy_l_20;
    uintptr_t fy_l_21;
    (void)&fy_l_21;
    fy_mut_byte_slice fy_l_22;
    (void)&fy_l_22;
    uint8_t fy_l_23;
    (void)&fy_l_23;
    int32_t fy_l_24;
    (void)&fy_l_24;
    uint8_t fy_l_25;
    (void)&fy_l_25;
    int32_t fy_l_26;
    (void)&fy_l_26;
    fy_mut_byte_slice fy_l_27;
    (void)&fy_l_27;
    fy_byte_slice fy_l_28;
    (void)&fy_l_28;
    uintptr_t fy_l_29;
    (void)&fy_l_29;
    fy_mut_byte_slice fy_l_30;
    (void)&fy_l_30;
    uintptr_t fy_l_31;
    (void)&fy_l_31;
    fy_mut_byte_slice fy_l_32;
    (void)&fy_l_32;
    uintptr_t fy_l_33;
    (void)&fy_l_33;
    int32_t fy_l_34;
    (void)&fy_l_34;
    fy_mut_byte_slice fy_l_35;
    (void)&fy_l_35;
    uintptr_t fy_l_36;
    (void)&fy_l_36;
    uint8_t fy_l_37;
    (void)&fy_l_37;
    int32_t fy_l_38;
    (void)&fy_l_38;
    uintptr_t fy_l_39;
    (void)&fy_l_39;
    uintptr_t fy_l_40;
    (void)&fy_l_40;
    uintptr_t fy_l_41;
    (void)&fy_l_41;
    uintptr_t fy_l_42;
    (void)&fy_l_42;
    fy_mut_byte_slice fy_l_43;
    (void)&fy_l_43;
    int32_t fy_l_44;
    (void)&fy_l_44;
    uint8_t fy_l_45;
    (void)&fy_l_45;
    int32_t fy_l_46;
    (void)&fy_l_46;
    goto fy_b_0;
fy_b_0:
    fy_l_17 = (fy_array_0){ .data = { UINT8_C(1), UINT8_C(2), UINT8_C(3), UINT8_C(4) } };
    fy_l_7 = fy_l_17;
    fy_l_18 = (fy_mut_byte_slice){ .data = fy_l_7.data, .length = ((uintptr_t)UINT64_C(4)) };
    fy_l_8 = fy_l_18;
    if ((((uintptr_t)UINT64_C(1))) > (((uintptr_t)UINT64_C(3))) || (((uintptr_t)UINT64_C(3))) > (fy_l_8).length) { goto fy_b_2; }
    goto fy_b_1;
fy_b_1:
    fy_l_19 = (fy_mut_byte_slice){ .data = (fy_l_8).data + (((uintptr_t)UINT64_C(1))), .length = (((uintptr_t)UINT64_C(3))) - (((uintptr_t)UINT64_C(1))) };
    fy_l_9 = fy_l_19;
    fy_l_20 = fy_l_9;
    fy_l_21 = ((uintptr_t)UINT64_C(0));
    if (fy_l_21 >= (fy_l_20).length) { goto fy_b_4; }
    goto fy_b_3;
fy_b_3:
    (fy_l_20).data[fy_l_21] = UINT8_C(7);
    if ((((uintptr_t)UINT64_C(1))) > (fy_l_9).length) { goto fy_b_6; }
    goto fy_b_5;
fy_b_5:
    fy_l_22 = (fy_mut_byte_slice){ .data = (fy_l_9).data, .length = (((uintptr_t)UINT64_C(1))) };
    fy_l_10 = fy_l_22;
    fy_fn_0(fy_l_10);
    goto fy_b_7;
fy_b_7:
    if (((uintptr_t)UINT64_C(0)) >= (fy_l_9).length) { goto fy_b_9; }
    goto fy_b_8;
fy_b_8:
    fy_l_23 = (fy_l_9).data[((uintptr_t)UINT64_C(0))];
    fy_l_24 = ((fy_l_23) == (UINT8_C(9))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_24 == INT32_C(0)) { goto fy_b_11; }
    goto fy_b_10;
fy_b_10:
    if (((uintptr_t)UINT64_C(1)) >= (fy_l_8).length) { goto fy_b_13; }
    goto fy_b_12;
fy_b_12:
    fy_l_25 = (fy_l_8).data[((uintptr_t)UINT64_C(1))];
    fy_l_26 = ((fy_l_25) == (UINT8_C(9))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_26 == INT32_C(0)) { goto fy_b_15; }
    goto fy_b_14;
fy_b_14:
    fy_l_27 = fy_fn_3(fy_l_8);
    goto fy_b_16;
fy_b_16:
    fy_l_11 = fy_l_27;
    fy_fn_0(fy_l_11);
    goto fy_b_17;
fy_b_17:
    fy_l_28 = (fy_byte_slice){ .data = (fy_l_11).data, .length = (fy_l_11).length };
    fy_fn_2(fy_l_28);
    goto fy_b_18;
fy_b_18:
    fy_l_29 = (fy_l_8).length;
    if ((fy_l_29) > (fy_l_8).length) { goto fy_b_20; }
    goto fy_b_19;
fy_b_19:
    fy_l_30 = (fy_mut_byte_slice){ .data = (fy_l_8).data, .length = (fy_l_29) };
    fy_l_12 = fy_l_30;
    fy_l_31 = (fy_l_12).length;
    fy_l_13 = fy_l_31;
    if ((fy_l_13) > (fy_l_13) || (fy_l_13) > (fy_l_12).length) { goto fy_b_22; }
    goto fy_b_21;
fy_b_21:
    fy_l_32 = (fy_mut_byte_slice){ .data = (fy_l_12).data, .length = (fy_l_13) - (fy_l_13) };
    if ((fy_l_13) != ((uintptr_t)UINT64_C(0))) { fy_l_32.data += (fy_l_13); }
    fy_l_14 = fy_l_32;
    fy_l_33 = (fy_l_14).length;
    fy_l_34 = ((fy_l_33) == (((uintptr_t)UINT64_C(0)))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_34 == INT32_C(0)) { goto fy_b_24; }
    goto fy_b_23;
fy_b_23:
    fy_l_35 = fy_l_12;
    fy_l_36 = ((uintptr_t)UINT64_C(3));
    if (fy_l_36 >= (fy_l_35).length) { goto fy_b_26; }
    goto fy_b_25;
fy_b_25:
    (fy_l_35).data[fy_l_36] = UINT8_C(8);
    if (((uintptr_t)UINT64_C(3)) >= (fy_l_8).length) { goto fy_b_28; }
    goto fy_b_27;
fy_b_27:
    fy_l_37 = (fy_l_8).data[((uintptr_t)UINT64_C(3))];
    fy_l_38 = ((fy_l_37) == (UINT8_C(8))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_38 == INT32_C(0)) { goto fy_b_30; }
    goto fy_b_29;
fy_b_29:
    fy_l_15 = ((uintptr_t)UINT64_C(0));
    if ((((uintptr_t)UINT64_C(10))) != 0 && (fy_l_15) > UINTPTR_MAX / (((uintptr_t)UINT64_C(10)))) { goto fy_b_32; }
    goto fy_b_31;
fy_b_31:
    fy_l_39 = (fy_l_15) * (((uintptr_t)UINT64_C(10)));
    if ((fy_l_39) > UINTPTR_MAX - (((uintptr_t)UINT64_C(1)))) { goto fy_b_34; }
    goto fy_b_33;
fy_b_33:
    fy_l_40 = (fy_l_39) + (((uintptr_t)UINT64_C(1)));
    fy_l_15 = fy_l_40;
    if ((((uintptr_t)UINT64_C(10))) != 0 && (fy_l_15) > UINTPTR_MAX / (((uintptr_t)UINT64_C(10)))) { goto fy_b_36; }
    goto fy_b_35;
fy_b_35:
    fy_l_41 = (fy_l_15) * (((uintptr_t)UINT64_C(10)));
    if ((fy_l_41) > UINTPTR_MAX - (((uintptr_t)UINT64_C(2)))) { goto fy_b_38; }
    goto fy_b_37;
fy_b_37:
    fy_l_42 = (fy_l_41) + (((uintptr_t)UINT64_C(2)));
    fy_l_15 = fy_l_42;
    if ((((uintptr_t)UINT64_C(1))) > (((uintptr_t)UINT64_C(3))) || (((uintptr_t)UINT64_C(3))) > (fy_l_8).length) { goto fy_b_40; }
    goto fy_b_39;
fy_b_39:
    fy_l_43 = (fy_mut_byte_slice){ .data = (fy_l_8).data + (((uintptr_t)UINT64_C(1))), .length = (((uintptr_t)UINT64_C(3))) - (((uintptr_t)UINT64_C(1))) };
    fy_l_16 = fy_l_43;
    fy_l_44 = ((fy_l_15) == (((uintptr_t)UINT64_C(12)))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_44 == INT32_C(0)) { goto fy_b_42; }
    goto fy_b_41;
fy_b_41:
    fy_fn_0(fy_l_16);
    goto fy_b_43;
fy_b_43:
    if (((uintptr_t)UINT64_C(1)) >= (fy_l_8).length) { goto fy_b_45; }
    goto fy_b_44;
fy_b_44:
    fy_l_45 = (fy_l_8).data[((uintptr_t)UINT64_C(1))];
    fy_l_46 = ((fy_l_45) == (UINT8_C(9))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_46 == INT32_C(0)) { goto fy_b_47; }
    goto fy_b_46;
fy_b_46:
    return INT32_C(0);
fy_b_11:
fy_b_15:
fy_b_24:
fy_b_30:
fy_b_42:
fy_b_47:
    fy_runtime_trap(FY_TRAP_ASSERTION_FAILED);
fy_b_2:
fy_b_4:
fy_b_6:
fy_b_9:
fy_b_13:
fy_b_20:
fy_b_22:
fy_b_26:
fy_b_28:
fy_b_40:
fy_b_45:
    fy_runtime_trap(FY_TRAP_BOUNDS);
fy_b_32:
fy_b_34:
fy_b_36:
fy_b_38:
    fy_runtime_trap(FY_TRAP_INTEGER_OVERFLOW);
}

static void fy_fn_2(fy_byte_slice fy_l_6) {
    (void)fy_l_6;
    uintptr_t fy_l_7;
    (void)&fy_l_7;
    int32_t fy_l_8;
    (void)&fy_l_8;
    goto fy_b_0;
fy_b_0:
    fy_l_7 = (fy_l_6).length;
    fy_l_8 = ((fy_l_7) > (((uintptr_t)UINT64_C(0)))) ? INT32_C(1) : INT32_C(0);
    if (fy_l_8 == INT32_C(0)) { goto fy_b_2; }
    goto fy_b_1;
fy_b_1:
    return;
fy_b_2:
    fy_runtime_trap(FY_TRAP_ASSERTION_FAILED);
}

static fy_mut_byte_slice fy_fn_3(fy_mut_byte_slice fy_l_4) {
    (void)fy_l_4;
    uintptr_t fy_l_5;
    (void)&fy_l_5;
    fy_mut_byte_slice fy_l_6;
    (void)&fy_l_6;
    goto fy_b_0;
fy_b_0:
    fy_l_5 = (fy_l_4).length;
    if ((((uintptr_t)UINT64_C(1))) > (fy_l_5) || (fy_l_5) > (fy_l_4).length) { goto fy_b_2; }
    goto fy_b_1;
fy_b_1:
    fy_l_6 = (fy_mut_byte_slice){ .data = (fy_l_4).data + (((uintptr_t)UINT64_C(1))), .length = (fy_l_5) - (((uintptr_t)UINT64_C(1))) };
    return fy_l_6;
fy_b_2:
    fy_runtime_trap(FY_TRAP_BOUNDS);
}

int main(void) {
    (void)&fy_fn_0;
    (void)&fy_fn_2;
    (void)&fy_fn_3;
    return (int)fy_fn_1();
}
