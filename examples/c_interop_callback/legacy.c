#include "legacy.h"
#include <stddef.h>
#include <stdio.h>

int32_t legacy_filter(legacy_predicate predicate, void *data) {
    const int32_t values[] = {2, 12, 7, 20, 1};
    int32_t accepted = 0;
    if (predicate == NULL || data == NULL) {
        return -1;
    }
    for (size_t i = 0; i < sizeof(values) / sizeof(values[0]); ++i) {
        if (predicate(values[i], data) != 0) {
            ++accepted;
        }
    }
    return accepted;
}

int32_t legacy_note_visit(void *data) {
    if (data == NULL) {
        return -1;
    }
    return ++*(int32_t *)data;
}

int32_t legacy_report(int32_t accepted, int32_t visits) {
    return printf("accepted=%d callbacks=%d\n", (int)accepted, (int)visits) < 0 ? 4 : 0;
}
