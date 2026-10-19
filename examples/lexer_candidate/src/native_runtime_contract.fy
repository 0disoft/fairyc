module native_runtime_contract;
import std::bytes as bytes;

// Private Windows runtime selection. IDs follow the Go contract link order.
// 101 target-specific std symbol, 102 unknown std symbol, 103 foreign symbol.
pub fn u32 classify(ref u8[] name){
    if(bytes::equal(name,b"fy_std_stderr_write_all_v0") ||
       bytes::equal(name,b"fy_std_stderr_write_i32_v0") ||
       bytes::equal(name,b"fy_std_stderr_write_u32_v0") ||
       bytes::equal(name,b"fy_std_stderr_write_usize_v0") ||
       bytes::equal(name,b"fy_std_stdin_read_v0") ||
       bytes::equal(name,b"fy_std_stdout_write_all_v0") ||
       bytes::equal(name,b"fy_std_stdout_write_i32_v0") ||
       bytes::equal(name,b"fy_std_stdout_write_u32_v0") ||
       bytes::equal(name,b"fy_std_stdout_write_usize_v0") ||
       bytes::equal(name,b"fy_std_windows_stdout_write_all_v0")){return 3;}
    if(bytes::equal(name,b"fy_std_file_close_v0") ||
       bytes::equal(name,b"fy_std_file_drop_v0") ||
       bytes::equal(name,b"fy_std_file_open_utf8_v0") ||
       bytes::equal(name,b"fy_std_file_open_v0") ||
       bytes::equal(name,b"fy_std_file_position_v0") ||
       bytes::equal(name,b"fy_std_file_read_exact_v0") ||
       bytes::equal(name,b"fy_std_file_read_v0") ||
       bytes::equal(name,b"fy_std_file_remove_v0") ||
       bytes::equal(name,b"fy_std_file_replace_v0") ||
       bytes::equal(name,b"fy_std_file_seek_current_v0") ||
       bytes::equal(name,b"fy_std_file_seek_end_v0") ||
       bytes::equal(name,b"fy_std_file_seek_start_v0") ||
       bytes::equal(name,b"fy_std_file_size_v0") ||
       bytes::equal(name,b"fy_std_file_sync_v0") ||
       bytes::equal(name,b"fy_std_file_write_all_v0") ||
       bytes::equal(name,b"fy_std_file_write_v0") ||
       bytes::equal(name,b"fy_std_windows_dir_remove_all_v0") ||
       bytes::equal(name,b"fy_std_windows_directory_create_v0") ||
       bytes::equal(name,b"fy_std_windows_directory_ensure_v0") ||
       bytes::equal(name,b"fy_std_windows_directory_remove_v0") ||
       bytes::equal(name,b"fy_std_windows_directory_rename_v0") ||
       bytes::equal(name,b"fy_std_windows_file_alloc_size_v0") ||
       bytes::equal(name,b"fy_std_windows_file_close_v0") ||
       bytes::equal(name,b"fy_std_windows_file_copy_bytes_v0") ||
       bytes::equal(name,b"fy_std_windows_file_drop_v0") ||
       bytes::equal(name,b"fy_std_windows_file_hard_link_v0") ||
       bytes::equal(name,b"fy_std_windows_file_id_v0") ||
       bytes::equal(name,b"fy_std_windows_file_kind_v0") ||
       bytes::equal(name,b"fy_std_windows_file_link_count_v0") ||
       bytes::equal(name,b"fy_std_windows_file_open_v0") ||
       bytes::equal(name,b"fy_std_windows_file_path_kind_v0") ||
       bytes::equal(name,b"fy_std_windows_file_position_v0") ||
       bytes::equal(name,b"fy_std_windows_file_read_exact_v0") ||
       bytes::equal(name,b"fy_std_windows_file_read_only_v0") ||
       bytes::equal(name,b"fy_std_windows_file_read_v0") ||
       bytes::equal(name,b"fy_std_windows_file_remove_v0") ||
       bytes::equal(name,b"fy_std_windows_file_rename_v0") ||
       bytes::equal(name,b"fy_std_windows_file_replace_v0") ||
       bytes::equal(name,b"fy_std_windows_file_seek_current_v0") ||
       bytes::equal(name,b"fy_std_windows_file_seek_end_v0") ||
       bytes::equal(name,b"fy_std_windows_file_seek_start_v0") ||
       bytes::equal(name,b"fy_std_windows_file_set_len_v0") ||
       bytes::equal(name,b"fy_std_windows_file_set_readonly_v0") ||
       bytes::equal(name,b"fy_std_windows_file_size_v0") ||
       bytes::equal(name,b"fy_std_windows_file_sync_v0") ||
       bytes::equal(name,b"fy_std_windows_file_write_all_v0") ||
       bytes::equal(name,b"fy_std_windows_file_write_v0")){return 4;}
    if(bytes::equal(name,b"fy_std_windows_read_dir_close_v0") ||
       bytes::equal(name,b"fy_std_windows_read_dir_drop_v0") ||
       bytes::equal(name,b"fy_std_windows_read_dir_next_v0") ||
       bytes::equal(name,b"fy_std_windows_read_dir_open_v0")){return 5;}
    if(bytes::equal(name,b"fy_std_process_arg_count_v0") ||
       bytes::equal(name,b"fy_std_process_arg_v0")){return 6;}
    if(bytes::equal(name,b"fy_std_system_heap_allocate_v0") ||
       bytes::equal(name,b"fy_std_system_heap_deallocate_v0")){return 7;}
    if(bytes::equal(name,b"fy_std_bytes_match_at_v0")){return 8;}
    if(bytes::equal(name,b"fy_std_buffer_append_v0") ||
       bytes::equal(name,b"fy_std_buffer_clear_v0") ||
       bytes::equal(name,b"fy_std_buffer_drop_v0") ||
       bytes::equal(name,b"fy_std_buffer_initialize_v0") ||
       bytes::equal(name,b"fy_std_buffer_reserve_v0") ||
       bytes::equal(name,b"fy_std_buffer_resize_zeroed_v0") ||
       bytes::equal(name,b"fy_std_buffer_truncate_v0")){return 9;}
    if(bytes::equal(name,b"fy_std_windows_file_read_append_v0")){return 10;}
    if(bytes::equal(name,b"fy_std_arena_drop_v0") ||
       bytes::equal(name,b"fy_std_arena_initialize_v0") ||
       bytes::equal(name,b"fy_std_region_allocate_zeroed_v0") ||
       bytes::equal(name,b"fy_std_region_begin_v0") ||
       bytes::equal(name,b"fy_std_region_copy_bytes_v0") ||
       bytes::equal(name,b"fy_std_region_reset_v0")){return 11;}
    if(bytes::equal(name,b"fy_std_text_utf8_valid_v0")){return 12;}
    if(bytes::equal(name,b"fy_std_cstr_validate_v0") ||
       bytes::equal(name,b"fy_std_cstring_as_view_v0") ||
       bytes::equal(name,b"fy_std_cstring_drop_storage_v0") ||
       bytes::equal(name,b"fy_std_cstring_drop_v0") ||
       bytes::equal(name,b"fy_std_cstring_from_text_storage_v0") ||
       bytes::equal(name,b"fy_std_cstring_from_text_v0") ||
       bytes::equal(name,b"fy_std_string_append_storage_v0") ||
       bytes::equal(name,b"fy_std_string_append_v0") ||
       bytes::equal(name,b"fy_std_string_as_view_v0") ||
       bytes::equal(name,b"fy_std_string_copy_storage_v0") ||
       bytes::equal(name,b"fy_std_string_copy_v0") ||
       bytes::equal(name,b"fy_std_string_drop_storage_v0") ||
       bytes::equal(name,b"fy_std_string_drop_v0") ||
       bytes::equal(name,b"fy_std_string_from_buffer_storage_v0") ||
       bytes::equal(name,b"fy_std_string_into_buffer_storage_v0")){return 14;}
    if(bytes::equal(name,b"fy_std_linux_file_close_v0")){return 101;}
    if(bytes::equal(name,b"fy_std_linux_file_open_v0")){return 101;}
    if(bytes::equal(name,b"fy_std_linux_file_read_v0")){return 101;}
    if(bytes::equal(name,b"fy_std_linux_file_sync_v0")){return 101;}
    if(bytes::equal(name,b"fy_std_linux_file_write_all_v0")){return 101;}
    if(bytes::equal(name,b"fy_std_text_validate_utf8_v0")){return 101;}
    if(name.len>=7 && bytes::equal(name[..7],b"fy_std_")){return 102;}return 103;
}
// 0 unknown, 1 process host, 2 process plus rooted-file host.
pub fn u32 host(ref u8[] name){
    if(bytes::equal(name,b"fairyc_private_windows_assign_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_close_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_compare_name_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_create_env_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_create_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_current_path_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_directory_open_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_environment_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_environment_word_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_error_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_exit_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_free_environment_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_handle_count_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_handle_value_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_image_path_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_inherit_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_input_file_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_job_info_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_job_size_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_job_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_kill_job_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_kill_process_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_open_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_peek_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_pipe_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_read_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_resume_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_root_directory_query_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_close_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_delete_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_info_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_open_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_publish_new_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_read_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_rename_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_size_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_sync_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_time_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_root_file_write_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_windows_startup_size_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_std_handle_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_temp_path_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_ticks_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_windows_wait_v1")){return 1;}
    return 0;
}
pub fn ref u8[] component_name(u32 id){
    if(id==1){return b"linux-console";}
    if(id==2){return b"linux-file";}
    if(id==3){return b"windows-console";}
    if(id==4){return b"windows-file";}
    if(id==5){return b"windows-directory";}
    if(id==6){return b"windows-process-args";}
    if(id==7){return b"system-heap";}
    if(id==8){return b"byte-compare";}
    if(id==9){return b"buffer";}
    if(id==10){return b"windows-file-buffer";}
    if(id==11){return b"arena";}
    if(id==12){return b"text-view";}
    if(id==13){return b"linux-process-args";}
    if(id==14){return b"owned-text";}
    return b"";
}
// Include dependencies exactly once. Bits are IDs minus one.
pub fn u32 closure(u32 id){
    let u32 own=1<<(id-1);
    if(id==9 || id==11){return own | 64;}
    if(id==10){return own | 8 | 64 | 256;}
    if(id==14){return own | 64 | 256 | 2048;}
    if(id==13){return own | 2048;}
    return own;
}
