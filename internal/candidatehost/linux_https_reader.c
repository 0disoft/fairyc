//go:build ignore

/* Append after linux_https_protocol.c. The owner supplies deadline-bounded I/O;
 * no caller pointer or whole response body is retained between calls. */
typedef uint32_t (*fy_linux_https_input)(void *, void *, size_t, size_t *);
typedef struct {
    fy_linux_https_input input;
    void *context;
    fy_linux_https_head head;
    size_t maximum, received, remaining;
    uint32_t error;
    int headed, ended;
} fy_linux_https_reader;

static uint32_t fy_linux_https_exact(fy_linux_https_reader *p, void *data, size_t size) {
    size_t used = 0;
    while (used < size) {
        size_t copied = 0;
        uint32_t code = p->input(p->context, (uint8_t *)data+used, size-used, &copied);
        if (code != 0) { return code; }
        if (copied == 0 || copied > size-used) { return EPROTO; }
        used += copied;
    }
    return 0;
}

static uint32_t fy_linux_https_next(fy_linux_https_reader *p) {
    fy_linux_https_frame frame;
    uint32_t code = fy_linux_https_exact(p, &frame, sizeof frame);
    if (code != 0) { return code; }
    if (frame.kind == 3) {
        if (frame.length != 0 || (!p->headed && frame.code == 0)) { return EPROTO; }
        /* An end frame alone is not EOF: reject trailing bytes, and let the
         * parent's absolute deadline terminate a child which keeps its pipe. */
        uint8_t extra = 0;
        size_t copied = 0;
        code = p->input(p->context, &extra, 1, &copied);
        if (code != 0) { return code; }
        if (copied != 0) { return EPROTO; }
        p->ended = 1;
        return frame.code;
    }
    if (frame.code != 0) { return EPROTO; }
    if (frame.kind == 1) {
        if (p->headed || frame.length != sizeof p->head) { return EPROTO; }
        code = fy_linux_https_exact(p, &p->head, sizeof p->head);
        if (code != 0) { return code; }
        if (p->head.status < 100 || p->head.status > 599 ||
            p->head.encoding_length > sizeof p->head.encoding) { return EPROTO; }
        for (size_t i = 0; i < p->head.encoding_length; ++i) {
            if (p->head.encoding[i] < 32 || p->head.encoding[i] == 127) { return EPROTO; }
        }
        p->headed = 1;
        return 0;
    }
    if (frame.kind != 2 || !p->headed || frame.length == 0 || frame.length > 65536 ||
        p->received > p->maximum || (size_t)frame.length > p->maximum+1-p->received) { return EPROTO; }
    p->remaining = frame.length;
    p->received += frame.length;
    return 0;
}

static uint32_t fy_linux_https_reader_headers(fy_linux_https_reader *p) {
    if (p->error != 0) { return p->error; }
    if (!p->headed) { p->error = fy_linux_https_next(p); }
    return p->error;
}

static uint32_t fy_linux_https_reader_read(fy_linux_https_reader *p, void *data,
        size_t capacity, size_t *copied) {
    *copied = 0;
    if (p->error != 0) { return p->error; }
    if (!p->headed) { p->error = fy_linux_https_reader_headers(p); }
    if (p->error != 0) { return p->error; }
    size_t used = 0;
    while (used < capacity && !p->ended) {
        if (p->remaining == 0) {
            p->error = fy_linux_https_next(p);
            if (p->error != 0) { return p->error; }
            if (p->ended) { break; }
        }
        size_t chunk = capacity-used < p->remaining ? capacity-used : p->remaining;
        p->error = fy_linux_https_exact(p, (uint8_t *)data+used, chunk);
        if (p->error != 0) { return p->error; }
        p->remaining -= chunk;
        used += chunk;
    }
    *copied = used;
    return 0;
}
