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
int32_t fairyc_private_windows_exit_v1(void *process, uint32_t *exit) { return GetExitCodeProcess(process, exit); }
int32_t fairyc_private_windows_kill_job_v1(void *job, uint32_t code) { return TerminateJobObject(job, code); }
int32_t fairyc_private_windows_kill_process_v1(void *process, uint32_t code) { return TerminateProcess(process, code); }
int32_t fairyc_private_windows_close_v1(void *handle) { return CloseHandle(handle); }
void *fairyc_private_windows_std_handle_v1(uint32_t kind) { return GetStdHandle(kind); }
uint32_t fairyc_private_windows_error_v1(void) { return GetLastError(); }
size_t fairyc_private_windows_ticks_v1(void) { return (size_t)GetTickCount64(); }
size_t fairyc_private_windows_startup_size_v1(void) { return sizeof(STARTUPINFOW); }
size_t fairyc_private_windows_job_size_v1(void) { return sizeof(JOBOBJECT_EXTENDED_LIMIT_INFORMATION); }
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
uint8_t *fairyc_private_windows_environment_v1(void) { return (uint8_t *)GetEnvironmentStringsW(); }
int32_t fairyc_private_windows_free_environment_v1(uint8_t *environment) { return FreeEnvironmentStringsW((LPWCH)environment); }
uint32_t fairyc_private_windows_environment_word_v1(uint8_t *environment, size_t index) {
    return (uint32_t)((const WCHAR *)environment)[index];
}
int32_t fairyc_private_windows_compare_name_v1(const uint8_t *left, size_t left_size, const uint8_t *right, size_t right_size) {
    return CompareStringOrdinal((LPCWCH)left, (int)left_size, (LPCWCH)right, (int)right_size, TRUE);
}
void *fairyc_private_windows_open_v1(uint32_t access, int32_t inherit, uint32_t pid) { return OpenProcess(access, inherit, pid); }
int32_t fairyc_private_windows_handle_count_v1(uint32_t *count) { return GetProcessHandleCount(GetCurrentProcess(), count); }
uint32_t fairyc_private_windows_current_path_v1(uint8_t *data, uint32_t capacity) { return GetCurrentDirectoryW(capacity, (LPWSTR)data); }
uint32_t fairyc_private_windows_temp_path_v1(uint8_t *data, uint32_t capacity) { return GetTempPathW(capacity, (LPWSTR)data); }
uint32_t fairyc_private_windows_image_path_v1(uint8_t *data, uint32_t capacity) { return GetModuleFileNameW(NULL, (LPWSTR)data, capacity); }
