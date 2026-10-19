    char fy_trap_handle_text[32];
    DWORD fy_trap_handle_length = GetEnvironmentVariableA("FAIRYC_TRAP_CONTROL_HANDLE_V1", fy_trap_handle_text, (DWORD)sizeof(fy_trap_handle_text));
    uintptr_t fy_trap_handle_value = (uintptr_t)0;
    if (fy_trap_handle_length == (DWORD)0 || fy_trap_handle_length >= (DWORD)sizeof(fy_trap_handle_text)) {
        _Exit(254);
    }
    for (DWORD fy_index = (DWORD)0; fy_index < fy_trap_handle_length; ++fy_index) {
        unsigned int fy_digit = (unsigned int)(unsigned char)fy_trap_handle_text[fy_index] - (unsigned int)'0';
        if (fy_digit > UINT32_C(9) || fy_trap_handle_value > (UINTPTR_MAX - (uintptr_t)fy_digit) / (uintptr_t)10) {
            _Exit(254);
        }
        fy_trap_handle_value = fy_trap_handle_value * (uintptr_t)10 + (uintptr_t)fy_digit;
    }
    unsigned char fy_trap_receipt = (unsigned char)category;
    DWORD fy_trap_written = (DWORD)0;
    if (fy_trap_handle_value == (uintptr_t)0 || !WriteFile((HANDLE)fy_trap_handle_value, &fy_trap_receipt, (DWORD)1, &fy_trap_written, NULL) || fy_trap_written != (DWORD)1) {
        _Exit(254);
    }
