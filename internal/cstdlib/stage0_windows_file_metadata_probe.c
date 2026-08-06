struct fy_std_windows_file_result_v0 fy_std_windows_file_open_v0(
    const uint8_t *,
    const struct fy_std_windows_open_options_v0 *,
    void *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_write_all_v0(
    void *,
    const uint8_t *,
    size_t
);
struct fy_std_windows_file_position_result_v0 fy_std_windows_file_position_v0(void *);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_size_v0(const void *);
struct fy_std_windows_file_attribute_result_v0 fy_std_windows_file_read_only_v0(
    const void *
);
struct fy_std_windows_file_attribute_result_v0 fy_std_windows_file_set_readonly_v0(
    void *,
    uint32_t
);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_link_count_v0(
    const void *
);
struct fy_std_windows_file_size_result_v0 fy_std_windows_file_alloc_size_v0(
    const void *
);
struct fy_std_windows_file_result_v0 fy_std_windows_file_close_v0(void *);
struct fy_std_windows_file_result_v0 fy_std_windows_file_remove_v0(const uint8_t *);

int fy_std_windows_file_metadata_probe_v0(void) {
    static const uint8_t path[] = "fairyc-windows-file-metadata-probe.tmp";
    static const uint8_t link_path[] = "fairyc-windows-file-metadata-probe-link.tmp";
    static const uint8_t payload[] = {UINT8_C(70), UINT8_C(121)};
    SetFileAttributesW(L"fairyc-windows-file-metadata-probe.tmp", FILE_ATTRIBUTE_NORMAL);
    DeleteFileW(L"fairyc-windows-file-metadata-probe-link.tmp");
    DeleteFileW(L"fairyc-windows-file-metadata-probe.tmp");
    struct fy_std_windows_open_options_v0 options = {
        UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(1), UINT8_C(0), UINT8_C(0)
    };
    struct fy_std_windows_file_v0 file = {NULL, UINT8_C(0)};
    struct fy_std_windows_file_result_v0 result =
        fy_std_windows_file_open_v0(path, &options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 219;
    result = fy_std_windows_file_write_all_v0(&file, payload, sizeof payload);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0 || result.count != sizeof payload) return 220;

    struct fy_std_windows_file_size_result_v0 links =
        fy_std_windows_file_link_count_v0(&file);
    if (links.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        links.os_code != (uint32_t)ERROR_SUCCESS || links.size != UINT64_C(1)) return 231;
    if (!CreateHardLinkW(
            L"fairyc-windows-file-metadata-probe-link.tmp",
            L"fairyc-windows-file-metadata-probe.tmp",
            NULL
        )) return 232;
    links = fy_std_windows_file_link_count_v0(&file);
    if (links.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        links.os_code != (uint32_t)ERROR_SUCCESS || links.size != UINT64_C(2)) return 233;

    struct fy_std_windows_file_attribute_result_v0 attribute =
        fy_std_windows_file_read_only_v0(&file);
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 221;
    attribute = fy_std_windows_file_set_readonly_v0(&file, UINT32_C(1));
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(1)) return 265;
    attribute = fy_std_windows_file_read_only_v0(&file);
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(1)) return 223;
    struct fy_std_windows_file_position_result_v0 position =
        fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != sizeof payload) return 224;
    struct fy_std_windows_file_size_result_v0 size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 || size.size != sizeof payload) return 225;
    links = fy_std_windows_file_link_count_v0(&file);
    if (links.status != FY_STD_WINDOWS_FILE_OK_V0 || links.size != UINT64_C(2)) return 234;
    struct fy_std_windows_file_size_result_v0 allocated =
        fy_std_windows_file_alloc_size_v0(&file);
    if (allocated.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        allocated.os_code != (uint32_t)ERROR_SUCCESS || allocated.size < sizeof payload) return 260;
    position = fy_std_windows_file_position_v0(&file);
    if (position.status != FY_STD_WINDOWS_FILE_OK_V0 || position.position != sizeof payload) return 261;
    size = fy_std_windows_file_size_v0(&file);
    if (size.status != FY_STD_WINDOWS_FILE_OK_V0 || size.size != sizeof payload) return 262;
    attribute = fy_std_windows_file_set_readonly_v0(&file, UINT32_C(0));
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 266;
    attribute = fy_std_windows_file_read_only_v0(&file);
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 267;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 227;
    attribute = fy_std_windows_file_read_only_v0(&file);
    if (attribute.status != FY_STD_WINDOWS_FILE_CLOSED_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 228;
    attribute = fy_std_windows_file_read_only_v0(NULL);
    if (attribute.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 229;
    attribute = fy_std_windows_file_set_readonly_v0(&file, UINT32_C(1));
    if (attribute.status != FY_STD_WINDOWS_FILE_CLOSED_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 268;
    attribute = fy_std_windows_file_set_readonly_v0(NULL, UINT32_C(1));
    if (attribute.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 269;
    attribute = fy_std_windows_file_set_readonly_v0(&file, UINT32_C(2));
    if (attribute.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        attribute.os_code != (uint32_t)ERROR_SUCCESS || attribute.enabled != UINT32_C(0)) return 270;
    links = fy_std_windows_file_link_count_v0(&file);
    if (links.status != FY_STD_WINDOWS_FILE_CLOSED_V0 ||
        links.os_code != (uint32_t)ERROR_SUCCESS || links.size != UINT64_C(0)) return 235;
    links = fy_std_windows_file_link_count_v0(NULL);
    if (links.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        links.os_code != (uint32_t)ERROR_SUCCESS || links.size != UINT64_C(0)) return 236;
    allocated = fy_std_windows_file_alloc_size_v0(&file);
    if (allocated.status != FY_STD_WINDOWS_FILE_CLOSED_V0 ||
        allocated.os_code != (uint32_t)ERROR_SUCCESS || allocated.size != UINT64_C(0)) return 263;
    allocated = fy_std_windows_file_alloc_size_v0(NULL);
    if (allocated.status != FY_STD_WINDOWS_FILE_INVALID_ARGUMENT_V0 ||
        allocated.os_code != (uint32_t)ERROR_SUCCESS || allocated.size != UINT64_C(0)) return 264;
    struct fy_std_windows_open_options_v0 read_options = {
        UINT8_C(1), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0), UINT8_C(0)
    };
    result = fy_std_windows_file_open_v0(path, &read_options, &file);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 271;
    attribute = fy_std_windows_file_set_readonly_v0(&file, UINT32_C(1));
    if (attribute.status != FY_STD_WINDOWS_FILE_IO_ERROR_V0 ||
        attribute.os_code != (uint32_t)ERROR_ACCESS_DENIED || attribute.enabled != UINT32_C(0)) {
        return 272;
    }
    attribute = fy_std_windows_file_read_only_v0(&file);
    if (attribute.status != FY_STD_WINDOWS_FILE_OK_V0 || attribute.enabled != UINT32_C(0)) return 273;
    if (fy_std_windows_file_close_v0(&file).status != FY_STD_WINDOWS_FILE_OK_V0) return 274;
    result = fy_std_windows_file_remove_v0(link_path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 237;
    result = fy_std_windows_file_remove_v0(path);
    if (result.status != FY_STD_WINDOWS_FILE_OK_V0) return 230;
    return 0;
}
