/* Private GNU/Linux ABI adapters. Fairy owns start/wait/result sequencing. */
#ifndef _GNU_SOURCE
#define _GNU_SOURCE 1
#endif
#include <errno.h>
#include <limits.h>
#include <signal.h>
#include <spawn.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/wait.h>
#include <unistd.h>

extern char **environ;

uint32_t fairyc_private_linux_process_start_v1(const uint8_t *data, size_t size,
        size_t argc, const uint8_t *cwd, size_t cwd_size, uint32_t *output) {
    if (output == NULL) { return EINVAL; }
    *output = 0;
    if (data == NULL || cwd == NULL || argc == 0 || argc > 131071 ||
        size > 1048576 - 8 * (argc + 1) || argc > size ||
        cwd_size < 2 || cwd_size > 4096 || cwd[cwd_size-1] != 0 ||
        memchr(cwd, 0, cwd_size-1) != NULL) { return EINVAL; }
    size_t at = 0;
    for (size_t i = 0; i < argc; ++i) {
        const uint8_t *end = memchr(data + at, 0, size - at);
        if (end == NULL || (i == 0 && (end == data || end - data > 4095))) { return EINVAL; }
        at = (size_t)(end - data) + 1;
    }
    if (at != size) { return EINVAL; }
    char **argv = calloc(argc + 1, sizeof *argv);
    if (argv == NULL) { return ENOMEM; }
    at = 0;
    for (size_t i = 0; i < argc; ++i) {
        argv[i] = (char *)(data + at);
        at += strlen(argv[i]) + 1;
    }
    posix_spawn_file_actions_t actions;
    int code = posix_spawn_file_actions_init(&actions);
    if (code != 0) { free(argv); return (uint32_t)code; }
    code = posix_spawn_file_actions_addchdir_np(&actions, (const char *)cwd);
    if (code == 0) { code = posix_spawn_file_actions_addclosefrom_np(&actions, 3); }
    pid_t pid = 0;
    if (code == 0) { code = posix_spawn(&pid, argv[0], &actions, NULL, argv, environ); }
    posix_spawn_file_actions_destroy(&actions);
    free(argv);
    if (code == 0) { *output = (uint32_t)pid; }
    return (uint32_t)code;
}

uint32_t fairyc_private_linux_process_wait_v1(uint32_t pid, uint32_t *kind, uint32_t *value) {
    if (kind == NULL || value == NULL) { return EINVAL; }
    *kind = 0; *value = 0;
    if (pid == 0 || pid > INT_MAX) { return EINVAL; }
    int status = 0;
    pid_t result;
    do { result = waitpid((pid_t)pid, &status, 0); } while (result < 0 && errno == EINTR);
    if (result < 0) { return (uint32_t)errno; }
    if (WIFEXITED(status)) { *kind = 1; *value = (uint32_t)WEXITSTATUS(status); }
    else if (WIFSIGNALED(status)) { *kind = 2; *value = (uint32_t)WTERMSIG(status); }
    else { return EIO; }
    return 0;
}

/* Only the owning Fairy call may reap this child. ECHILD never authorizes kill. */
uint32_t fairyc_private_linux_process_discard_v1(uint32_t pid) {
    if (pid == 0) { return 0; }
    if (pid > INT_MAX) { return EINVAL; }
    int status = 0;
    pid_t result;
    do { result = waitpid((pid_t)pid, &status, WNOHANG); } while (result < 0 && errno == EINTR);
    if (result == (pid_t)pid || (result < 0 && errno == ECHILD)) { return 0; }
    if (result < 0) { return (uint32_t)errno; }
    if (kill((pid_t)pid, SIGKILL) != 0 && errno != ESRCH) { return (uint32_t)errno; }
    do { result = waitpid((pid_t)pid, &status, 0); } while (result < 0 && errno == EINTR);
    return result < 0 && errno != ECHILD ? (uint32_t)errno : 0;
}
