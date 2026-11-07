//go:build ignore

/* Private ABI adapters only. Command construction, flags, sequencing, deadlines
 * and handle ownership are implemented by windows_process.fy. */
#include <windows.h>
#include <stdint.h>
#include <stddef.h>

void *fairyc_private_windows_job_v1(void) { return CreateJobObjectW(NULL, NULL); }
int32_t fairyc_private_windows_job_info_v1(void *job, uint32_t kind, const uint8_t *data, uint32_t size) {
    return SetInformationJobObject(job, (JOBOBJECTINFOCLASS)kind, (void *)data, size);
}
int32_t fairyc_private_windows_create_v1(const uint8_t *app, uint8_t *line, const uint8_t *cwd,
                                   int32_t inherit, uint32_t flags, void *startup, void *process) {
    return CreateProcessW((LPCWSTR)app, (LPWSTR)line, NULL, NULL, inherit, flags,
                          NULL, (LPCWSTR)cwd, (LPSTARTUPINFOW)startup, (LPPROCESS_INFORMATION)process);
}
int32_t fairyc_private_windows_assign_v1(void *job, void *process) { return AssignProcessToJobObject(job, process); }
uint32_t fairyc_private_windows_resume_v1(void *thread) { return ResumeThread(thread); }
uint32_t fairyc_private_windows_wait_v1(void *process, uint32_t timeout) { return WaitForSingleObject(process, timeout); }
int32_t fairyc_private_windows_exit_v1(void *process, uint32_t *exit) { return GetExitCodeProcess(process, (LPDWORD)exit); }
int32_t fairyc_private_windows_kill_job_v1(void *job, uint32_t code) { return TerminateJobObject(job, code); }
int32_t fairyc_private_windows_kill_process_v1(void *process, uint32_t code) { return TerminateProcess(process, code); }
int32_t fairyc_private_windows_close_v1(void *handle) { return CloseHandle(handle); }
void *fairyc_private_windows_std_handle_v1(uint32_t kind) { return GetStdHandle(kind); }
uint32_t fairyc_private_windows_error_v1(void) { return GetLastError(); }
size_t fairyc_private_windows_ticks_v1(void) { return (size_t)GetTickCount64(); }
size_t fairyc_private_windows_startup_size_v1(void) { return sizeof(STARTUPINFOW); }
size_t fairyc_private_windows_job_size_v1(void) { return sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION); }
void *fairyc_private_windows_cancel_create_v1(const uint8_t *name) {
    SetLastError(0);
    HANDLE event = CreateEventW(NULL, TRUE, FALSE, (LPCWSTR)name);
    if (event != NULL && GetLastError() == ERROR_ALREADY_EXISTS) {
        CloseHandle(event);
        SetLastError(ERROR_ALREADY_EXISTS);
        return NULL;
    }
    return event;
}
void *fairyc_private_windows_cancel_open_v1(const uint8_t *name) {
    return OpenEventW(SYNCHRONIZE | EVENT_MODIFY_STATE, FALSE, (LPCWSTR)name);
}
int32_t fairyc_private_windows_cancel_signal_v1(void *event) { return SetEvent(event); }

/* A process owns at most one console registration. The shared lock covers the
 * entire callback use of the duplicate. Close takes it exclusively to unpublish
 * and drain all such uses before closing/freeing anything. No callback retains
 * an owner pointer after releasing the lock. Console attach/alloc/free resets
 * the OS handler table; callers must not change console attachment while bound.
 * In particular, binding never changes the inherited CTRL+C ignore attribute. */
struct fy_windows_console_cancel_registration { HANDLE event; };
static SRWLOCK fy_windows_console_cancel_lock = SRWLOCK_INIT;
static struct fy_windows_console_cancel_registration *fy_windows_console_cancel_active = NULL;
static BOOL fy_windows_console_cancel_closing = FALSE;
static BOOL fy_windows_console_cancel_poisoned = FALSE;

static BOOL WINAPI fy_windows_console_cancel_handler(DWORD control) {
    if (control != CTRL_C_EVENT && control != CTRL_BREAK_EVENT) { return FALSE; }
    BOOL handled = FALSE;
    AcquireSRWLockShared(&fy_windows_console_cancel_lock);
    if (fy_windows_console_cancel_active != NULL) {
        (void)SetEvent(fy_windows_console_cancel_active->event);
        handled = TRUE;
    }
    ReleaseSRWLockShared(&fy_windows_console_cancel_lock);
    return handled;
}

void *fairyc_private_windows_cancel_local_v1(void) {
    return CreateEventW(NULL, TRUE, FALSE, NULL);
}

void *fairyc_private_windows_cancel_console_bind_v1(void *event) {
    DWORD processes[1];
    /* Headless is a real native ERROR_INVALID_HANDLE, before ownership changes
     * or duplication. Other failures retain their own cause, never a no-op. */
    if (GetConsoleProcessList(processes, 1) == 0) { return NULL; }
    AcquireSRWLockExclusive(&fy_windows_console_cancel_lock);
    if (fy_windows_console_cancel_active != NULL || fy_windows_console_cancel_closing ||
        fy_windows_console_cancel_poisoned) {
        ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
        SetLastError(ERROR_BUSY);
        return NULL;
    }
    struct fy_windows_console_cancel_registration *owner = HeapAlloc(GetProcessHeap(), 0, sizeof(*owner));
    if (owner == NULL) {
        ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
        SetLastError(ERROR_NOT_ENOUGH_MEMORY);
        return NULL;
    }
    if (!DuplicateHandle(GetCurrentProcess(), event, GetCurrentProcess(), &owner->event,
                         EVENT_MODIFY_STATE | SYNCHRONIZE, FALSE, 0)) {
        DWORD error = GetLastError();
        (void)HeapFree(GetProcessHeap(), 0, owner);
        ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
        SetLastError(error);
        return NULL;
    }
    if (!SetConsoleCtrlHandler(fy_windows_console_cancel_handler, TRUE)) {
        DWORD error = GetLastError();
        (void)CloseHandle(owner->event);
        (void)HeapFree(GetProcessHeap(), 0, owner);
        ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
        SetLastError(error);
        return NULL;
    }
    fy_windows_console_cancel_active = owner;
    ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
    return owner;
}

int32_t fairyc_private_windows_cancel_console_close_v1(void *registration) {
    AcquireSRWLockExclusive(&fy_windows_console_cancel_lock);
    if (registration == NULL || fy_windows_console_cancel_active != registration) {
        ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
        SetLastError(ERROR_INVALID_HANDLE);
        return 0;
    }
    struct fy_windows_console_cancel_registration *owner = fy_windows_console_cancel_active;
    fy_windows_console_cancel_active = NULL;
    fy_windows_console_cancel_closing = TRUE;
    ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
    /* Do not hold the SRW lock inside unregister: any OS-delivered callbacks
     * still arriving can finish, observe NULL, and touch no freed state. */
    BOOL removed = SetConsoleCtrlHandler(fy_windows_console_cancel_handler, FALSE);
    DWORD error = removed ? ERROR_SUCCESS : GetLastError();
    if (!removed && error == ERROR_SUCCESS) { error = ERROR_INVALID_STATE; }
    if (!CloseHandle(owner->event) && error == ERROR_SUCCESS) { error = GetLastError(); }
    (void)HeapFree(GetProcessHeap(), 0, owner);
    AcquireSRWLockExclusive(&fy_windows_console_cancel_lock);
    if (!removed) { fy_windows_console_cancel_poisoned = TRUE; }
    fy_windows_console_cancel_closing = FALSE;
    ReleaseSRWLockExclusive(&fy_windows_console_cancel_lock);
    if (error != ERROR_SUCCESS) { SetLastError(error); return 0; }
    return 1;
}
int32_t fairyc_private_windows_create_env_v1(const uint8_t *app, uint8_t *line, const uint8_t *cwd,
                                   int32_t inherit, uint32_t flags, void *startup, void *process, const uint8_t *environment) {
    return CreateProcessW((LPCWSTR)app, (LPWSTR)line, NULL, NULL, inherit, flags,
                          (void *)environment, (LPCWSTR)cwd, (LPSTARTUPINFOW)startup, (LPPROCESS_INFORMATION)process);
}
int32_t fairyc_private_windows_pipe_v1(void *reader, void *writer) {
    return CreatePipe((PHANDLE)reader, (PHANDLE)writer, NULL, 0);
}
int32_t fairyc_private_windows_inherit_v1(void *handle, uint32_t flags) {
    return SetHandleInformation(handle, HANDLE_FLAG_INHERIT, flags);
}
int32_t fairyc_private_windows_peek_v1(void *handle, uint32_t *available) {
    return PeekNamedPipe(handle, NULL, 0, NULL, (LPDWORD)available, NULL);
}
int32_t fairyc_private_windows_read_v1(void *handle, uint8_t *data, uint32_t size, uint32_t *read) {
    return ReadFile(handle, data, size, (LPDWORD)read, NULL);
}
size_t fairyc_private_windows_handle_value_v1(void *handle) { return (size_t)(uintptr_t)handle; }
void *fairyc_private_windows_input_file_v1(const uint8_t *path) {
    SECURITY_ATTRIBUTES attributes = {sizeof(SECURITY_ATTRIBUTES), NULL, TRUE};
    HANDLE handle = CreateFileW((LPCWSTR)path, GENERIC_READ, FILE_SHARE_READ, &attributes,
                                OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
    return handle == INVALID_HANDLE_VALUE ? NULL : handle;
}
uint8_t *fairyc_private_windows_environment_v1(void) { return (uint8_t *)GetEnvironmentStringsW(); }
int32_t fairyc_private_windows_free_environment_v1(uint8_t *environment) { return FreeEnvironmentStringsW((LPWCH)environment); }
uint32_t fairyc_private_windows_environment_word_v1(uint8_t *environment, size_t index) {
    return (uint32_t)((const WCHAR *)environment)[index];
}
int32_t fairyc_private_windows_compare_name_v1(const uint8_t *left, size_t left_size, const uint8_t *right, size_t right_size) {
    return CompareStringOrdinal((LPCWCH)left, (int)left_size, (LPCWCH)right, (int)right_size, TRUE);
}
void *fairyc_private_windows_open_v1(uint32_t access, int32_t inherit, uint32_t pid) { return OpenProcess(access, inherit, pid); }
int32_t fairyc_private_windows_handle_count_v1(uint32_t *count) { return GetProcessHandleCount(GetCurrentProcess(), (PDWORD)count); }
uint32_t fairyc_private_windows_current_path_v1(uint8_t *data, uint32_t capacity) { return GetCurrentDirectoryW(capacity, (LPWSTR)data); }
uint32_t fairyc_private_windows_temp_path_v1(uint8_t *data, uint32_t capacity) { return GetTempPathW(capacity, (LPWSTR)data); }
uint32_t fairyc_private_windows_image_path_v1(uint8_t *data, uint32_t capacity) { return GetModuleFileNameW(NULL, (LPWSTR)data, capacity); }
