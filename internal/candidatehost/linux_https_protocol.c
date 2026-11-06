//go:build ignore

/* Private same-process-image child/parent pipe protocol, never network input. */
#include <errno.h>
#include <stdint.h>
#include <stddef.h>
#include <string.h>

typedef uint32_t (*fy_linux_https_writer)(void *, const void *, size_t);
typedef struct { uint32_t kind, length, code; } fy_linux_https_frame;
typedef struct {
    uint32_t status, encoding_length;
    size_t declared;
    uint8_t encoding[128];
} fy_linux_https_head;
_Static_assert(sizeof(size_t) == 8 && sizeof(fy_linux_https_frame) == 12 &&
    sizeof(fy_linux_https_head) == 144, "private Linux HTTPS wire layout");

typedef struct {
    fy_linux_https_writer write;
    void *sink;
    fy_linux_https_head head;
    size_t maximum, written, header_bytes, header_lines;
    uint32_t error;
    int status_seen, sent, length_seen, encoding_seen, transfer_seen;
} fy_linux_https_response;

static uint32_t fy_linux_https_send(fy_linux_https_response *p, uint32_t kind,
        uint32_t code, const void *data, size_t size) {
    if (size > 65536) { return EOVERFLOW; }
    fy_linux_https_frame frame = {kind, (uint32_t)size, code};
    uint32_t result = p->write(p->sink, &frame, sizeof frame);
    if (result == 0 && size != 0) { result = p->write(p->sink, data, size); }
    return result;
}
static unsigned char fy_linux_https_fold(unsigned char c) {
    return c >= 'A' && c <= 'Z' ? (unsigned char)(c + ('a' - 'A')) : c;
}
static int fy_linux_https_equal(const char *value, size_t size, const char *name) {
    size_t length = strlen(name);
    if (length != size) { return 0; }
    for (size_t i = 0; i < size; ++i) {
        if (fy_linux_https_fold((unsigned char)value[i]) != (unsigned char)name[i]) { return 0; }
    }
    return 1;
}
static uint32_t fy_linux_https_line(fy_linux_https_response *p, const char *data, size_t size) {
    if (size < 2 || size > 65536 - p->header_bytes || p->header_lines >= 1024 ||
        data[size-2] != '\r' || data[size-1] != '\n') { return EINVAL; }
    p->header_bytes += size; ++p->header_lines; size -= 2;
    for (size_t i = 0; i < size; ++i) {
        unsigned char c = (unsigned char)data[i];
        if ((c < 32 && c != '\t') || c == 127) { return EINVAL; }
    }
    if (size >= 5 && memcmp(data, "HTTP/", 5) == 0) {
        if (p->sent || p->status_seen) { return EINVAL; }
        size_t at = 5;
        while (at < size && data[at] != ' ') {
            if ((data[at] < '0' || data[at] > '9') && data[at] != '.') { return EINVAL; }
            ++at;
        }
        if (at == 5 || at == size || size - at < 4) { return EINVAL; }
        size_t version = at-5;
        if (!((version == 3 && data[5] == '1' && data[6] == '.' && (data[7] == '0' || data[7] == '1')) ||
            (version == 1 && (data[5] == '2' || data[5] == '3')))) { return EINVAL; }
        ++at;
        if (data[at] < '1' || data[at] > '5' || data[at+1] < '0' || data[at+1] > '9' ||
            data[at+2] < '0' || data[at+2] > '9' || (at+3 < size && data[at+3] != ' ')) { return EINVAL; }
        memset(&p->head, 0, sizeof p->head);
        p->head.status = (uint32_t)(data[at]-'0')*100u + (uint32_t)(data[at+1]-'0')*10u + (uint32_t)(data[at+2]-'0');
        p->head.declared = SIZE_MAX;
        p->status_seen = 1; p->length_seen = p->encoding_seen = p->transfer_seen = 0;
        return 0;
    }
    if (size == 0) {
        if (p->sent) { return 0; } /* End of bounded trailers. */
        if (!p->status_seen) { return EINVAL; }
        if (p->length_seen && p->transfer_seen) { return EINVAL; }
        if (p->head.status < 200 && p->head.status != 101) { p->status_seen = 0; return 0; }
        uint32_t code = fy_linux_https_send(p, 1, 0, &p->head, sizeof p->head);
        if (code == 0) { p->sent = 1; }
        return code;
    }
    if (!p->status_seen) { return EINVAL; }
    size_t colon = 0;
    while (colon < size && data[colon] != ':') {
        unsigned char c = (unsigned char)data[colon];
        if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
            c == '!' || c == '#' || c == '$' || c == '%' || c == '&' || c == '\'' ||
            c == '*' || c == '+' || c == '-' || c == '.' || c == '^' || c == '_' || c == '`' || c == '|' || c == '~')) { return EINVAL; }
        ++colon;
    }
    if (colon == 0 || colon == size) { return EINVAL; }
    size_t first = colon+1, last = size;
    while (first < last && (data[first] == ' ' || data[first] == '\t')) { ++first; }
    while (last > first && (data[last-1] == ' ' || data[last-1] == '\t')) { --last; }
    int length = fy_linux_https_equal(data, colon, "content-length");
    int encoding = fy_linux_https_equal(data, colon, "content-encoding");
    int transfer = fy_linux_https_equal(data, colon, "transfer-encoding");
    if (p->sent) { return length || encoding || transfer ? EINVAL : 0; }
    if (length) {
        size_t value = 0;
        if (first == last) { return EINVAL; }
        for (size_t i = first; i < last; ++i) {
            if (data[i] < '0' || data[i] > '9') { return EINVAL; }
            size_t digit = (size_t)(data[i]-'0');
            if (value > (SIZE_MAX-digit)/10) { return EOVERFLOW; }
            value = value*10+digit;
        }
        if (value == SIZE_MAX || (p->length_seen && p->head.declared != value)) { return EINVAL; }
        p->head.declared = value; p->length_seen = 1;
    } else if (encoding) {
        if (p->encoding_seen || first == last || last-first > sizeof p->head.encoding) { return EINVAL; }
        memcpy(p->head.encoding, data+first, last-first);
        p->head.encoding_length = (uint32_t)(last-first); p->encoding_seen = 1;
    } else if (transfer) {
        if (p->transfer_seen || !fy_linux_https_equal(data+first, last-first, "chunked")) { return EINVAL; }
        p->transfer_seen = 1;
    }
    return 0;
}
static size_t fy_linux_https_header(char *data, size_t size, size_t count, void *context) {
    fy_linux_https_response *p = context;
    if (p->error != 0) { return 0; }
    p->error = size != 1 ? EINVAL : fy_linux_https_line(p, data, count);
    return p->error == 0 ? count : 0;
}
static size_t fy_linux_https_body(char *data, size_t size, size_t count, void *context) {
    fy_linux_https_response *p = context;
    if (p->error != 0) { return 0; }
    if (!p->sent || (size != 0 && count > SIZE_MAX/size)) { p->error = EINVAL; return 0; }
    size_t total = size*count;
    if (p->head.status != 200 || (p->head.encoding_length != 0 &&
        (p->head.encoding_length != 8 || memcmp(p->head.encoding, "identity", 8) != 0))) { p->error = EINVAL; return 0; }
    if (p->written > p->maximum+1) { p->error = EOVERFLOW; return 0; }
    size_t available = p->maximum+1-p->written, used = 0, admitted = total < available ? total : available;
    while (used < admitted) {
        size_t chunk = admitted-used > 65536 ? 65536 : admitted-used;
        p->error = fy_linux_https_send(p, 2, 0, data+used, chunk);
        if (p->error != 0) { return 0; }
        used += chunk; p->written += chunk;
    }
    if (admitted != total) { p->error = EOVERFLOW; return 0; }
    return total;
}
