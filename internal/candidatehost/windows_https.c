//go:build ignore

/* Private WinHTTP ABI. Fairy owns URL admission, byte/deadline budgets,
 * archive verification and publication. No credentials or proxy discovery. */
#include <windows.h>
#include <winhttp.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>

typedef struct {
    HMODULE module;
    HINTERNET session, connection, request;
    HANDLE ready, closing;
    ULONGLONG deadline;
    DWORD status, copied;
    HINTERNET (WINAPI *open)(LPCWSTR,DWORD,LPCWSTR,LPCWSTR,DWORD);
    BOOL (WINAPI *crack)(LPCWSTR,DWORD,DWORD,LPURL_COMPONENTS);
    HINTERNET (WINAPI *connect)(HINTERNET,LPCWSTR,INTERNET_PORT,DWORD);
    HINTERNET (WINAPI *request_open)(HINTERNET,LPCWSTR,LPCWSTR,LPCWSTR,LPCWSTR,LPCWSTR const *,DWORD);
    BOOL (WINAPI *option)(HINTERNET,DWORD,LPVOID,DWORD);
    WINHTTP_STATUS_CALLBACK (WINAPI *callback)(HINTERNET,WINHTTP_STATUS_CALLBACK,DWORD,DWORD_PTR);
    BOOL (WINAPI *send)(HINTERNET,LPCWSTR,DWORD,LPVOID,DWORD,DWORD,DWORD_PTR);
    BOOL (WINAPI *receive)(HINTERNET,LPVOID);
    BOOL (WINAPI *headers)(HINTERNET,DWORD,LPCWSTR,LPVOID,LPDWORD,LPDWORD);
    BOOL (WINAPI *read)(HINTERNET,LPVOID,DWORD,LPDWORD);
    BOOL (WINAPI *close)(HINTERNET);
} fy_https;

static void CALLBACK fy_https_callback(HINTERNET handle, DWORD_PTR context, DWORD status, LPVOID info, DWORD length) {
    (void)handle; (void)info;
    fy_https *p = (fy_https *)context;
    if (p == NULL) { return; }
    if (status == WINHTTP_CALLBACK_STATUS_HANDLE_CLOSING) { SetEvent(p->closing); return; }
    if (status == WINHTTP_CALLBACK_STATUS_SENDREQUEST_COMPLETE || status == WINHTTP_CALLBACK_STATUS_HEADERS_AVAILABLE ||
        status == WINHTTP_CALLBACK_STATUS_READ_COMPLETE || status == WINHTTP_CALLBACK_STATUS_REQUEST_ERROR) {
        p->copied = status == WINHTTP_CALLBACK_STATUS_READ_COMPLETE ? length : 0;
        p->status = status;
        SetEvent(p->ready);
    }
}
static int fy_https_wait(fy_https *p, DWORD wanted) {
    ULONGLONG now = GetTickCount64();
    if (now >= p->deadline || WaitForSingleObject(p->ready, (DWORD)(p->deadline - now)) != WAIT_OBJECT_0) { return 0; }
    return p->status == wanted && GetTickCount64() < p->deadline;
}
int32_t fairyc_private_windows_https_close_v1(size_t raw) {
    fy_https *p = (fy_https *)(uintptr_t)raw;
    if (p == NULL) { return 0; }
    BOOL ok = TRUE;
    if (p->request != NULL) {
        ok = p->close(p->request);
        /* HANDLE_CLOSING is the last callback. Keep the callback context and
         * any caller read buffer alive until asynchronous cancellation ends. */
        if (ok) { WaitForSingleObject(p->closing, INFINITE); }
        else { return 0; } /* retain context on an unexpected invalid close */
    }
    if (p->connection != NULL) { ok = p->close(p->connection) && ok; }
    if (p->session != NULL) { ok = p->close(p->session) && ok; }
    if (p->ready != NULL) { CloseHandle(p->ready); }
    if (p->closing != NULL) { CloseHandle(p->closing); }
    if (p->module != NULL) { FreeLibrary(p->module); }
    HeapFree(GetProcessHeap(), 0, p);
    return ok;
}
size_t fairyc_private_windows_https_open_v1(const uint8_t *url, size_t bytes, size_t timeout) {
    if (url == NULL || bytes < 2 || bytes > 8192 || bytes % 2 != 0 || timeout == 0 || timeout > 30000 ||
        ((const WCHAR *)url)[bytes / 2 - 1] != 0) { return 0; }
    fy_https *p = (fy_https *)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(*p));
    if (p == NULL) { return 0; }
    p->deadline = GetTickCount64() + timeout;
    p->module = LoadLibraryExW(L"winhttp.dll", NULL, LOAD_LIBRARY_SEARCH_SYSTEM32);
    if (p->module == NULL) { goto fail; }
#define FY_HTTP_LOAD(field, name) do { FARPROC proc = GetProcAddress(p->module, name); if (proc == NULL) { goto fail; } memcpy(&p->field, &proc, sizeof(proc)); } while (0)
    FY_HTTP_LOAD(open,"WinHttpOpen"); FY_HTTP_LOAD(crack,"WinHttpCrackUrl"); FY_HTTP_LOAD(connect,"WinHttpConnect");
    FY_HTTP_LOAD(request_open,"WinHttpOpenRequest"); FY_HTTP_LOAD(option,"WinHttpSetOption"); FY_HTTP_LOAD(callback,"WinHttpSetStatusCallback");
    FY_HTTP_LOAD(send,"WinHttpSendRequest"); FY_HTTP_LOAD(receive,"WinHttpReceiveResponse"); FY_HTTP_LOAD(headers,"WinHttpQueryHeaders");
    FY_HTTP_LOAD(read,"WinHttpReadData"); FY_HTTP_LOAD(close,"WinHttpCloseHandle");
#undef FY_HTTP_LOAD
    URL_COMPONENTS parts;
    memset(&parts, 0, sizeof(parts)); parts.dwStructSize = sizeof(parts);
    WCHAR host[4096], path[4096], user[4096], password[4096], extra[4096];
    parts.lpszHostName=host; parts.dwHostNameLength=4096; parts.lpszUrlPath=path; parts.dwUrlPathLength=4096;
    parts.lpszUserName=user; parts.dwUserNameLength=4096; parts.lpszPassword=password; parts.dwPasswordLength=4096;
    parts.lpszExtraInfo=extra; parts.dwExtraInfoLength=4096;
    if (!p->crack((LPCWSTR)url, (DWORD)(bytes/2-1), 0, &parts) || parts.nScheme != INTERNET_SCHEME_HTTPS ||
        parts.dwHostNameLength == 0 || parts.dwUrlPathLength == 0 || parts.dwUserNameLength != 0 ||
        parts.dwPasswordLength != 0 || parts.dwExtraInfoLength != 0) { goto fail; }
    p->ready=CreateEventW(NULL,FALSE,FALSE,NULL); p->closing=CreateEventW(NULL,TRUE,FALSE,NULL);
    if (p->ready == NULL || p->closing == NULL) { goto fail; }
    p->session=p->open(L"fairyc",WINHTTP_ACCESS_TYPE_NO_PROXY,WINHTTP_NO_PROXY_NAME,WINHTTP_NO_PROXY_BYPASS,WINHTTP_FLAG_ASYNC);
    if (p->session == NULL) { goto fail; }
    p->connection=p->connect(p->session,host,parts.nPort,0);
    if (p->connection == NULL) { goto fail; }
    p->request=p->request_open(p->connection,L"GET",path,NULL,WINHTTP_NO_REFERER,WINHTTP_DEFAULT_ACCEPT_TYPES,
        WINHTTP_FLAG_SECURE | WINHTTP_FLAG_ESCAPE_DISABLE);
    if (p->request == NULL) { goto fail; }
    /* Until a callback and context are both installed, close without waiting. */
    HINTERNET request=p->request; p->request=NULL;
    DWORD disabled=WINHTTP_DISABLE_REDIRECTS | WINHTTP_DISABLE_COOKIES | WINHTTP_DISABLE_AUTHENTICATION;
    DWORD logon=WINHTTP_AUTOLOGON_SECURITY_LEVEL_HIGH;
    if (!p->option(request,WINHTTP_OPTION_DISABLE_FEATURE,&disabled,sizeof(disabled)) ||
        !p->option(request,WINHTTP_OPTION_AUTOLOGON_POLICY,&logon,sizeof(logon))) { p->close(request); goto fail; }
    if (p->callback(request,fy_https_callback,WINHTTP_CALLBACK_FLAG_ALL_COMPLETIONS | WINHTTP_CALLBACK_FLAG_HANDLES,0) == WINHTTP_INVALID_STATUS_CALLBACK) { p->close(request); goto fail; }
    DWORD_PTR context=(DWORD_PTR)p;
    if (!p->option(request,WINHTTP_OPTION_CONTEXT_VALUE,&context,sizeof(context))) { p->close(request); goto fail; }
    p->request=request;
    if (!p->send(request,L"Accept: application/zip\r\n",(DWORD)-1L,WINHTTP_NO_REQUEST_DATA,0,0,context) && GetLastError()!=ERROR_IO_PENDING) { goto fail; }
    if (!fy_https_wait(p,WINHTTP_CALLBACK_STATUS_SENDREQUEST_COMPLETE)) { goto fail; }
    if (!p->receive(request,NULL) && GetLastError()!=ERROR_IO_PENDING) { goto fail; }
    if (!fy_https_wait(p,WINHTTP_CALLBACK_STATUS_HEADERS_AVAILABLE)) { goto fail; }
    return (size_t)(uintptr_t)p;
fail:
    fairyc_private_windows_https_close_v1((size_t)(uintptr_t)p); return 0;
}
/* Return response status; encoding/length admission remains in Fairy. */
int32_t fairyc_private_windows_https_headers_v1(size_t raw, uint32_t *status, uint8_t *encoding, size_t capacity, size_t *length, size_t *declared) {
    fy_https *p=(fy_https *)(uintptr_t)raw;
    if (p==NULL || status==NULL || encoding==NULL || capacity<2 || capacity>65536 || length==NULL || declared==NULL) { return 0; }
    DWORD code=0, size=sizeof(code);
    if (!p->headers(p->request,WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,WINHTTP_HEADER_NAME_BY_INDEX,&code,&size,WINHTTP_NO_HEADER_INDEX)) { return 0; }
    *status=code;
    size=(DWORD)capacity;
    if (!p->headers(p->request,WINHTTP_QUERY_CONTENT_ENCODING,WINHTTP_HEADER_NAME_BY_INDEX,encoding,&size,WINHTTP_NO_HEADER_INDEX)) {
        if (GetLastError()!=ERROR_WINHTTP_HEADER_NOT_FOUND) { return 0; } size=0;
    }
    *length=size;
    WCHAR count[32]; size=sizeof(count); *declared=SIZE_MAX;
    if (p->headers(p->request,WINHTTP_QUERY_CONTENT_LENGTH,WINHTTP_HEADER_NAME_BY_INDEX,count,&size,WINHTTP_NO_HEADER_INDEX)) {
        size_t value=0;
        if (size==0 || size%2!=0) { return 0; }
        for (DWORD i=0;i<size/2;i++) {
            if (count[i]<L'0' || count[i]>L'9' || value>(SIZE_MAX-(size_t)(count[i]-L'0'))/10) { return 0; }
            value=value*10+(size_t)(count[i]-L'0');
        }
        if (value==SIZE_MAX) { return 0; }
        *declared=value;
    } else if (GetLastError()!=ERROR_WINHTTP_HEADER_NOT_FOUND) { return 0; }
    return GetTickCount64()<p->deadline;
}
int32_t fairyc_private_windows_https_read_v1(size_t raw, uint8_t *data, size_t capacity, size_t *copied) {
    fy_https *p=(fy_https *)(uintptr_t)raw;
    if (p==NULL || data==NULL || capacity==0 || capacity>65536 || copied==NULL || GetTickCount64()>=p->deadline) { return 0; }
    if (!p->read(p->request,data,(DWORD)capacity,NULL) && GetLastError()!=ERROR_IO_PENDING) { return 0; }
    if (!fy_https_wait(p,WINHTTP_CALLBACK_STATUS_READ_COMPLETE)) { return 0; }
    *copied=p->copied;
    return p->copied<=capacity;
}
