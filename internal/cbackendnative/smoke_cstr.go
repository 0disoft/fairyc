package cbackendnative

const cStrValidationSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_cstr_view_v0 {
    ptr u8 data;
    c::size_t length;
}

@link_name("fairyc_fixture_cstr_valid_storage")
unsafe extern C fn ptr u8 valid_storage() ;
@link_name("fairyc_fixture_cstr_valid_storage_length")
unsafe extern C fn c::size_t valid_storage_length() ;
@link_name("fairyc_fixture_cstr_interior_nul_storage")
unsafe extern C fn ptr u8 interior_nul_storage() ;
@link_name("fairyc_fixture_cstr_interior_nul_storage_length")
unsafe extern C fn c::size_t interior_nul_storage_length() ;
@link_name("fairyc_fixture_cstr_unterminated_storage")
unsafe extern C fn ptr u8 unterminated_storage() ;
@link_name("fairyc_fixture_cstr_unterminated_storage_length")
unsafe extern C fn c::size_t unterminated_storage_length() ;
@link_name("fy_std_cstr_validate_v0")
unsafe extern C fn u32 validate_cstr(ptr u8 storage, c::size_t storage_length, ptr mut fy_std_cstr_view_v0 output) ;
@link_name("fairyc_fixture_cstr_view_is_valid")
unsafe extern C fn u32 view_is_valid(ptr fy_std_cstr_view_v0 view) ;

fn i32 main() {
    let ptr u8 valid = unsafe { valid_storage() };
    let c::size_t valid_length = unsafe { valid_storage_length() };
    var fy_std_cstr_view_v0 output = fy_std_cstr_view_v0 {
        .data = null,
        .length = valid_length,
    };
    let u32 accepted = unsafe {
        validate_cstr(valid, valid_length, ref mut output as ptr mut fy_std_cstr_view_v0)
    };
    let u32 valid_observed = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    let ptr u8 interior_nul = unsafe { interior_nul_storage() };
    let c::size_t interior_nul_length = unsafe { interior_nul_storage_length() };
    let u32 interior_nul_status = unsafe {
        validate_cstr(
            interior_nul,
            interior_nul_length,
            ref mut output as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 interior_failure_preserved_output = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    let ptr u8 unterminated = unsafe { unterminated_storage() };
    let c::size_t unterminated_length = unsafe { unterminated_storage_length() };
    let u32 unterminated_status = unsafe {
        validate_cstr(
            unterminated,
            unterminated_length,
            ref mut output as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 unterminated_failure_preserved_output = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    return if (accepted == 0 && valid_observed == 1 &&
        interior_nul_status == 4 && interior_failure_preserved_output == 1 &&
        unterminated_status == 3 && unterminated_failure_preserved_output == 1) {
        42
    } else {
        1
    };
}
`

const cStrValidationSourceCallBridge = `
static const uint8_t fairyc_fixture_cstr_valid_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)
};
static const uint8_t fairyc_fixture_cstr_interior_nul_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(0), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)
};
static const uint8_t fairyc_fixture_cstr_unterminated_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)
};

const uint8_t *fairyc_fixture_cstr_valid_storage(void) {
    return fairyc_fixture_cstr_valid_bytes;
}

size_t fairyc_fixture_cstr_valid_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_valid_bytes);
}

const uint8_t *fairyc_fixture_cstr_interior_nul_storage(void) {
    return fairyc_fixture_cstr_interior_nul_bytes;
}

size_t fairyc_fixture_cstr_interior_nul_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_interior_nul_bytes);
}

const uint8_t *fairyc_fixture_cstr_unterminated_storage(void) {
    return fairyc_fixture_cstr_unterminated_bytes;
}

size_t fairyc_fixture_cstr_unterminated_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_unterminated_bytes);
}

uint32_t fairyc_fixture_cstr_view_is_valid(const struct fy_std_cstr_view_v0 *view) {
    return view != NULL && view->data == fairyc_fixture_cstr_valid_bytes &&
           view->length == sizeof(fairyc_fixture_cstr_valid_bytes) - (size_t)1 &&
           view->data[0] == UINT8_C(70) && view->data[1] == UINT8_C(97) &&
           view->data[2] == UINT8_C(105) && view->data[3] == UINT8_C(114) &&
           view->data[4] == UINT8_C(121) ? UINT32_C(1) : UINT32_C(0);
}
`

var cStrValidationSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_cstr_view_v0 {",
	"const uint8_t * data;",
	"size_t length;",
	"_Static_assert(sizeof(struct fy_std_cstr_view_v0) == UINT64_C(16)",
	"offsetof(struct fy_std_cstr_view_v0, data) == UINT64_C(0)",
	"offsetof(struct fy_std_cstr_view_v0, length) == UINT64_C(8)",
	"extern uint32_t fy_std_cstr_validate_v0(const uint8_t * fy_a_0, size_t fy_a_1, struct fy_std_cstr_view_v0 * fy_a_2);",
	"extern uint32_t fairyc_fixture_cstr_view_is_valid(const struct fy_std_cstr_view_v0 * fy_a_0);",
	"fy_std_cstr_validate_v0(fy_l_",
	" = &fy_l_",
}
