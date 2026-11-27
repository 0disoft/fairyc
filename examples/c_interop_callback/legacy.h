#ifndef FAIRYC_EXAMPLE_LEGACY_H
#define FAIRYC_EXAMPLE_LEGACY_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef int32_t (*legacy_predicate)(int32_t value, void *data);

/* Calls finish on this thread before return; neither pointer is retained. */
int32_t legacy_filter(legacy_predicate predicate, void *data);
int32_t legacy_note_visit(void *data);
int32_t legacy_report(int32_t accepted, int32_t visits);

#ifdef __cplusplus
}
#endif

#endif
