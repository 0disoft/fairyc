#include "cpp_wrapper.h"

#include <memory>

namespace {
struct RequestedException {};

struct ReleaseResult {
    int32_t *releases;

    void operator()(int32_t *result) const noexcept {
        delete result;
        ++*releases;
    }
};
}

extern "C" int32_t legacy_cpp_filter(int32_t force_exception, int32_t *releases,
                                      legacy_predicate predicate, void *data) {
    if (releases == nullptr) {
        return -1;
    }
    *releases = 0;
    if (predicate == nullptr || data == nullptr) {
        return -1;
    }
    try {
        std::unique_ptr<int32_t, ReleaseResult> result(new int32_t{}, ReleaseResult{releases});
        *result = legacy_filter(predicate, data);
        // The callback and C filter have returned before the C++ throw.
        if (force_exception != 0) {
            throw RequestedException{};
        }
        return *result;
    } catch (const RequestedException &) {
        return -2;
    } catch (...) {
        return -3;
    }
}
