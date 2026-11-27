#ifndef FAIRYC_EXAMPLE_CPP_WRAPPER_H
#define FAIRYC_EXAMPLE_CPP_WRAPPER_H

#include "legacy.h"

#ifdef __cplusplus
extern "C" {
#endif

/* Returns the accepted count, -1 for invalid input, -2 for the requested
 * exception, or -3 for an unexpected C++ failure. releases points to a
 * separate writable counter and records this call's resource releases. */
int32_t legacy_cpp_filter(int32_t force_exception, int32_t *releases,
                          legacy_predicate predicate, void *data);

#ifdef __cplusplus
}
#endif

#endif
