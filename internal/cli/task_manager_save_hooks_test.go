package cli

// Win32 wrappers are test fixtures. Fault selection never enters public SDK code
// or the shipped CLI. Only a handle whose resolved name ends in .tmp is targeted.
const taskSaveNativeHooks = `
#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <wchar.h>
static int task_save_calls;
static int task_save_mode(const char *name) {
 char mode[64]; DWORD n=GetEnvironmentVariableA("FAIRYC_TASK_SAVE_FAULT",mode,sizeof mode);
 return n>0 && n<sizeof mode && !strcmp(mode,name);
}
static int task_save_temp(HANDLE h) {
 WCHAR path[32768]; DWORD n=GetFinalPathNameByHandleW(h,path,32768,FILE_NAME_NORMALIZED);
 return n>4 && n<32768 && !wcscmp(path+n-4,L".tmp");
}
static void task_save_checkpoint(void) {
 fputc('R',stdout); fflush(stdout); for (;;) Sleep(1000);
}
static BOOL task_save_write(HANDLE h,LPCVOID b,DWORD n,LPDWORD done,LPOVERLAPPED o) {
 if (task_save_temp(h)) {
  if(task_save_mode("first-write-full")){*done=0;SetLastError(ERROR_DISK_FULL);return FALSE;}
  if(task_save_mode("partial-full")||task_save_mode("during-write")) {
   if(task_save_calls++==0)return WriteFile(h,b,n<2?n:2,done,o);
   if(task_save_mode("during-write"))task_save_checkpoint();
   *done=0;SetLastError(ERROR_DISK_FULL);return FALSE;
  }
 }
 return WriteFile(h,b,n,done,o);
}
static BOOL task_save_sync(HANDLE h) {
 if(task_save_temp(h)&&task_save_mode("sync-failure")){SetLastError(ERROR_WRITE_FAULT);return FALSE;}
 BOOL result=FlushFileBuffers(h);
 if(result&&task_save_temp(h)&&task_save_mode("after-sync"))task_save_checkpoint();
 return result;
}
static BOOL task_save_close(HANDLE h) {
 int fail=task_save_temp(h)&&task_save_mode("close-failure");
 BOOL result=CloseHandle(h);
 if(fail){SetLastError(ERROR_WRITE_FAULT);return FALSE;}
 return result;
}
static BOOL task_save_move(LPCWSTR a,LPCWSTR b,DWORD flags) {
 if(flags&MOVEFILE_REPLACE_EXISTING) {
  if(task_save_mode("before"))task_save_checkpoint();
  if(task_save_mode("replace-full")){SetLastError(ERROR_DISK_FULL);return FALSE;}
 }
 BOOL result=MoveFileExW(a,b,flags);
 if(result&&(flags&MOVEFILE_REPLACE_EXISTING)&&task_save_mode("after"))task_save_checkpoint();
 return result;
}
`
