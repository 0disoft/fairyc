package cli

// Only production POSIX calls are intercepted. The application remains emitted
// Fairy, and these selectors are never linked into a shipped executable.
const taskSaveLinuxHooks = `
#define _POSIX_C_SOURCE 200809L
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
static int task_save_descriptor = -1;
static int task_save_calls;
static int task_save_mode(const char *name) {
 const char *mode = getenv("FAIRYC_TASK_SAVE_FAULT");
 return mode != NULL && strcmp(mode,name) == 0;
}
static int task_save_is_temp(const char *path) {
 size_t n = strlen(path); return n >= 4 && strcmp(path+n-4,".tmp") == 0;
}
static void task_save_checkpoint(void) {
 fputc('R',stdout); fflush(stdout); for (;;) pause();
}
static int task_save_open(const char *path,int flags,mode_t mode) {
 int fd = open(path,flags,mode);
 if (fd >= 0 && task_save_is_temp(path)) task_save_descriptor = fd;
 return fd;
}
static ssize_t task_save_write(int fd,const void *data,size_t length) {
 if (fd == task_save_descriptor) {
  if (task_save_mode("first-write-full")) { errno=ENOSPC; return -1; }
  if (task_save_mode("partial-full") || task_save_mode("during-write")) {
   if (task_save_calls++ == 0) return write(fd,data,length<2?length:2);
   if (task_save_mode("during-write")) task_save_checkpoint();
   errno=ENOSPC; return -1;
  }
 }
 return write(fd,data,length);
}
static int task_save_sync(int fd) {
 if (fd == task_save_descriptor && task_save_mode("sync-failure")) { errno=EIO; return -1; }
 int result=fsync(fd);
 if (result==0 && fd==task_save_descriptor && task_save_mode("after-sync")) task_save_checkpoint();
 return result;
}
static int task_save_close(int fd) {
 int fail=fd==task_save_descriptor && task_save_mode("close-failure");
 int result=close(fd);
 if (fd==task_save_descriptor) task_save_descriptor=-1;
 if (fail) { errno=EIO; return -1; }
 return result;
}
static int task_save_rename(const char *from,const char *to) {
 if (task_save_is_temp(from)) {
  if (task_save_mode("before")) task_save_checkpoint();
  if (task_save_mode("replace-full")) { errno=ENOSPC; return -1; }
 }
 int result=rename(from,to);
 if (result==0 && task_save_is_temp(from) && task_save_mode("after")) task_save_checkpoint();
 return result;
}
`
