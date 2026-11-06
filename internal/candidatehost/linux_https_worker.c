//go:build ignore

/* Append after linux_https_protocol.c. Call only in the dedicated, isolated
 * child: parent owns the absolute watchdog, pipe lifetime and kill/reap. */
#include <dlfcn.h>
#include <curl/curl.h>

typedef struct {
    void *module;
    CURL *easy;
    struct curl_slist *headers;
    int initialized;
    CURLcode (*global_init)(long);
    void (*global_cleanup)(void);
    curl_version_info_data *(*version)(CURLversion);
    CURL *(*init)(void);
    CURLcode (*option)(CURL *, CURLoption, ...);
    CURLcode (*perform)(CURL *);
    void (*cleanup)(CURL *);
    struct curl_slist *(*append)(struct curl_slist *, const char *);
    void (*free_headers)(struct curl_slist *);
} fy_linux_https_curl;

static uint32_t fy_linux_https_curl_error(CURLcode code) {
    return code > 0 && (uint32_t)code < 65536u ? 65536u+(uint32_t)code : EIO;
}
static uint32_t fy_linux_https_transfer(const char *address, size_t maximum, size_t timeout,
        fy_linux_https_writer writer, void *sink) {
    fy_linux_https_response response;
    memset(&response, 0, sizeof response);
    response.write = writer; response.sink = sink; response.maximum = maximum;
    fy_linux_https_curl p;
    memset(&p, 0, sizeof p);
    uint32_t result = 0;
    if (address == NULL || writer == NULL || maximum == 0 || maximum > 16777216 || timeout == 0 || timeout > 30000) { return EINVAL; }
    /* Reference Ubuntu/Debian amd64 system package paths. Never load from the
     * working directory, PATH, an environment option or a caller-supplied path. */
    p.module = dlopen("/usr/lib/x86_64-linux-gnu/libcurl.so.4", RTLD_NOW | RTLD_LOCAL);
    if (p.module == NULL) { p.module = dlopen("/lib/x86_64-linux-gnu/libcurl.so.4", RTLD_NOW | RTLD_LOCAL); }
    if (p.module == NULL) { result = ENOENT; goto done; }
#define FY_HTTPS_SYMBOL(field, name) do { void *symbol = dlsym(p.module, name); _Static_assert(sizeof p.field == sizeof symbol, "Linux curl symbol width"); if (symbol == NULL) { result = ENOSYS; goto done; } memcpy(&p.field, &symbol, sizeof symbol); } while (0)
    FY_HTTPS_SYMBOL(global_init,"curl_global_init"); FY_HTTPS_SYMBOL(global_cleanup,"curl_global_cleanup");
    FY_HTTPS_SYMBOL(version,"curl_version_info"); FY_HTTPS_SYMBOL(init,"curl_easy_init");
    FY_HTTPS_SYMBOL(option,"curl_easy_setopt"); FY_HTTPS_SYMBOL(perform,"curl_easy_perform");
    FY_HTTPS_SYMBOL(cleanup,"curl_easy_cleanup"); FY_HTTPS_SYMBOL(append,"curl_slist_append");
    FY_HTTPS_SYMBOL(free_headers,"curl_slist_free_all");
#undef FY_HTTPS_SYMBOL
    CURLcode code = p.global_init(CURL_GLOBAL_DEFAULT);
    if (code != CURLE_OK) { result = fy_linux_https_curl_error(code); goto done; }
    p.initialized = 1;
    curl_version_info_data *version = p.version(CURLVERSION_FIRST);
    if (version == NULL || version->version_num < 0x080000 || (version->features & CURL_VERSION_SSL) == 0) { result = ENOSYS; goto done; }
    p.easy = p.init();
    if (p.easy == NULL) { result = ENOMEM; goto done; }
    p.headers = p.append(NULL, "Accept: application/zip");
    if (p.headers == NULL) { result = ENOMEM; goto done; }
#define FY_HTTPS_OPTION(name, value) do { code = p.option(p.easy, name, value); if (code != CURLE_OK) { result = fy_linux_https_curl_error(code); goto done; } } while (0)
    FY_HTTPS_OPTION(CURLOPT_URL,address);
    FY_HTTPS_OPTION(CURLOPT_PROTOCOLS_STR,"https"); FY_HTTPS_OPTION(CURLOPT_REDIR_PROTOCOLS_STR,"https");
    FY_HTTPS_OPTION(CURLOPT_PROXY,""); FY_HTTPS_OPTION(CURLOPT_NETRC,(long)CURL_NETRC_IGNORED);
    FY_HTTPS_OPTION(CURLOPT_DISALLOW_USERNAME_IN_URL,1L);
    FY_HTTPS_OPTION(CURLOPT_HTTPAUTH,0L); FY_HTTPS_OPTION(CURLOPT_PROXYAUTH,0L);
    FY_HTTPS_OPTION(CURLOPT_UNRESTRICTED_AUTH,0L);
    FY_HTTPS_OPTION(CURLOPT_FOLLOWLOCATION,0L); FY_HTTPS_OPTION(CURLOPT_MAXREDIRS,0L);
    FY_HTTPS_OPTION(CURLOPT_SSL_VERIFYPEER,1L); FY_HTTPS_OPTION(CURLOPT_SSL_VERIFYHOST,2L);
    FY_HTTPS_OPTION(CURLOPT_SSLVERSION,(long)CURL_SSLVERSION_TLSv1_2);
    FY_HTTPS_OPTION(CURLOPT_CAINFO,"/etc/ssl/certs/ca-certificates.crt");
    FY_HTTPS_OPTION(CURLOPT_CAPATH,"/etc/ssl/certs");
    FY_HTTPS_OPTION(CURLOPT_TIMEOUT_MS,(long)timeout); FY_HTTPS_OPTION(CURLOPT_CONNECTTIMEOUT_MS,(long)timeout);
    FY_HTTPS_OPTION(CURLOPT_NOSIGNAL,1L); FY_HTTPS_OPTION(CURLOPT_PATH_AS_IS,1L);
    FY_HTTPS_OPTION(CURLOPT_HTTP_CONTENT_DECODING,0L); FY_HTTPS_OPTION(CURLOPT_HTTP_TRANSFER_DECODING,1L);
    FY_HTTPS_OPTION(CURLOPT_FAILONERROR,0L); FY_HTTPS_OPTION(CURLOPT_HTTPHEADER,p.headers);
    FY_HTTPS_OPTION(CURLOPT_HEADERFUNCTION,fy_linux_https_header); FY_HTTPS_OPTION(CURLOPT_HEADERDATA,&response);
    FY_HTTPS_OPTION(CURLOPT_WRITEFUNCTION,fy_linux_https_body); FY_HTTPS_OPTION(CURLOPT_WRITEDATA,&response);
#undef FY_HTTPS_OPTION
    code = p.perform(p.easy);
    result = response.error != 0 ? response.error : code != CURLE_OK ? fy_linux_https_curl_error(code) : !response.sent ? EIO : 0;
done:
    /* Callbacks/context stay alive through cleanup. No native diagnostic text
     * is sent, and the parent never borrows worker memory. */
    if (p.easy != NULL) { p.cleanup(p.easy); }
    if (p.headers != NULL) { p.free_headers(p.headers); }
    if (p.initialized) { p.global_cleanup(); }
    if (p.module != NULL && dlclose(p.module) != 0 && result == 0) { result = EIO; }
    uint32_t sent = fy_linux_https_send(&response, 3, result, NULL, 0);
    return result != 0 ? result : sent;
}
