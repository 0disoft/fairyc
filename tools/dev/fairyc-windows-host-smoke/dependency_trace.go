package main

// This debugger is a separate /MT validation process, never linked into a
// production image. LOAD_DLL events cover load-and-unload paths missed by snapshots.
const windowsDependencyTracerC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <psapi.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>

static const WCHAR *skip_token(const WCHAR *p) {
    while (*p == L' ' || *p == L'\t') { ++p; }
    int quoted = 0;
    while (*p) {
        if (*p == L'"') { quoted = !quoted; }
        else if (!quoted && (*p == L' ' || *p == L'\t')) { break; }
        ++p;
    }
    while (*p == L' ' || *p == L'\t') { ++p; }
    return p;
}

static int module_path(HANDLE process, HMODULE base, HANDLE file) {
    WCHAR path[32768];
    DWORD size = file ? GetFinalPathNameByHandleW(file, path, 32768, FILE_NAME_NORMALIZED | VOLUME_NAME_DOS) : 0;
    if (!size || size >= 32768) { size = K32GetModuleFileNameExW(process, base, path, 32768); }
    if (!size || size >= 32768) { return 0; }
    char utf8[98304];
    int bytes = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, path, -1, utf8, (int)sizeof(utf8), NULL, NULL);
    if (!bytes || strchr(utf8,'\t') || strchr(utf8,'\n') || strchr(utf8,'\r')) { return 0; }
    printf("DLL\t%llx\t%s\n", (unsigned long long)(uintptr_t)base, utf8);
    return fflush(stdout) == 0;
}

int wmain(int argc, WCHAR **argv) {
    if (argc < 4) { return 2; }
    SetErrorMode(SEM_FAILCRITICALERRORS|SEM_NOGPFAULTERRORBOX|SEM_NOOPENFILEERRORBOX);
    SECURITY_ATTRIBUTES attributes = { sizeof(attributes), NULL, TRUE };
    HANDLE out = CreateFileW(argv[2], GENERIC_WRITE, FILE_SHARE_READ, &attributes, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    HANDLE err = CreateFileW(argv[3], GENERIC_WRITE, FILE_SHARE_READ, &attributes, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    HANDLE in = CreateFileW(L"NUL", GENERIC_READ, FILE_SHARE_READ|FILE_SHARE_WRITE, &attributes, OPEN_EXISTING, 0, NULL);
    if (out == INVALID_HANDLE_VALUE || err == INVALID_HANDLE_VALUE || in == INVALID_HANDLE_VALUE) { return 3; }
    HANDLE job = CreateJobObjectW(NULL,NULL);
    JOBOBJECT_EXTENDED_LIMIT_INFORMATION limits = {0};
    limits.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
    if (!job || !SetInformationJobObject(job, JobObjectExtendedLimitInformation, &limits, sizeof(limits))) { return 4; }
    const WCHAR *tail = GetCommandLineW();
    for (int i=0; i<4; ++i) { tail=skip_token(tail); }
    size_t length = wcslen(argv[1]) + wcslen(tail) + 5;
    WCHAR *command = (WCHAR *)calloc(length, sizeof(WCHAR));
    if (!command || swprintf_s(command,length,L"\"%ls\" %ls",argv[1],tail)<0) { return 5; }
    STARTUPINFOW startup = {0}; startup.cb=sizeof(startup);
    startup.dwFlags=STARTF_USESTDHANDLES;
    startup.hStdInput=in; startup.hStdOutput=out; startup.hStdError=err;
    PROCESS_INFORMATION process = {0};
    BOOL created=CreateProcessW(argv[1],command,NULL,NULL,TRUE,DEBUG_ONLY_THIS_PROCESS|CREATE_SUSPENDED|CREATE_NO_WINDOW,NULL,NULL,&startup,&process);
    free(command);
    CloseHandle(out); CloseHandle(err); CloseHandle(in);
    if (!created) { CloseHandle(job); return 6; }
    if (!AssignProcessToJobObject(job,process.hProcess) || ResumeThread(process.hThread)==(DWORD)-1) {
        TerminateProcess(process.hProcess,91); CloseHandle(process.hThread); CloseHandle(process.hProcess); CloseHandle(job); return 7;
    }
    ULONGLONG start=GetTickCount64();
    int complete=0, initial_breakpoint=0, count=0, failure=0;
    while (!complete && !failure) {
        if (GetTickCount64()-start>30000 || ++count>8192) { failure=8; break; }
        DEBUG_EVENT event = {0};
        if (!WaitForDebugEvent(&event,1000)) {
            if (GetLastError()==ERROR_SEM_TIMEOUT) { continue; }
            failure=9; break;
        }
        DWORD status=DBG_CONTINUE;
        switch (event.dwDebugEventCode) {
        case CREATE_PROCESS_DEBUG_EVENT:
            if (event.u.CreateProcessInfo.hFile) { CloseHandle(event.u.CreateProcessInfo.hFile); }
            break;
        case LOAD_DLL_DEBUG_EVENT:
            if (!module_path(process.hProcess,(HMODULE)event.u.LoadDll.lpBaseOfDll,event.u.LoadDll.hFile)) { failure=10; }
            if (event.u.LoadDll.hFile) { CloseHandle(event.u.LoadDll.hFile); }
            break;
        case UNLOAD_DLL_DEBUG_EVENT:
            printf("UNLOAD\t%llx\n",(unsigned long long)(uintptr_t)event.u.UnloadDll.lpBaseOfDll); fflush(stdout);
            break;
        case EXCEPTION_DEBUG_EVENT:
            if (!initial_breakpoint && event.u.Exception.ExceptionRecord.ExceptionCode==EXCEPTION_BREAKPOINT && event.u.Exception.dwFirstChance) { initial_breakpoint=1; }
            else { status=DBG_EXCEPTION_NOT_HANDLED; }
            break;
        case EXIT_PROCESS_DEBUG_EVENT:
            printf("EXIT\t%lu\n",event.u.ExitProcess.dwExitCode); fflush(stdout); complete=1; break;
        default: break;
        }
        if (!ContinueDebugEvent(event.dwProcessId,event.dwThreadId,status)) { failure=11; }
    }
    if (!complete || failure) { TerminateJobObject(job,92); }
    WaitForSingleObject(process.hProcess,5000);
    CloseHandle(process.hThread); CloseHandle(process.hProcess); CloseHandle(job);
    return complete && !failure ? 0 : failure ? failure : 12;
}
`
