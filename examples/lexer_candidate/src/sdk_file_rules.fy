module sdk_file_rules;
import std::bytes as bytes;
import pointer_types as pointers;

// Closed transport of the source-authenticated bundled file SDK. These facts
// never grant authority to an unverified source or a caller-supplied type ID.
fn ref u8[] item(ref u8[] pool,usize index){
    var usize start=0;var usize i=0;
    while(i<index){while(start<pool.len && pool[start]!=0){start+=1;}if(start==pool.len){return pool[..0];}start+=1;i+=1;}
    var usize end=start;while(end<pool.len && pool[end]!=0){end+=1;}return pool[start..end];
}
pub fn usize rule(ref u8[] symbol){var usize i=0;while(i<37){if(bytes::equal(symbol,item(b"fy_std_file_open_v0\0fy_std_file_open_utf8_v0\0fy_std_file_remove_v0\0fy_std_windows_directory_create_v0\0fy_std_windows_directory_ensure_v0\0fy_std_windows_directory_remove_v0\0fy_std_windows_dir_remove_all_v0\0fy_std_windows_directory_rename_v0\0fy_std_windows_file_rename_v0\0fy_std_file_replace_v0\0fy_std_windows_file_hard_link_v0\0fy_std_windows_file_copy_bytes_v0\0fy_std_file_read_v0\0fy_std_file_read_exact_v0\0fy_std_file_read_append_v0\0fy_std_file_position_v0\0fy_std_file_size_v0\0fy_std_windows_file_kind_v0\0fy_std_windows_file_read_only_v0\0fy_std_windows_file_set_readonly_v0\0fy_std_windows_file_link_count_v0\0fy_std_windows_file_alloc_size_v0\0fy_std_windows_file_path_kind_v0\0fy_std_windows_file_id_v0\0fy_std_windows_file_set_len_v0\0fy_std_file_seek_start_v0\0fy_std_file_seek_current_v0\0fy_std_file_seek_end_v0\0fy_std_file_write_v0\0fy_std_file_write_all_v0\0fy_std_file_sync_v0\0fy_std_file_close_v0\0fy_std_file_drop_v0\0fy_std_windows_read_dir_open_v0\0fy_std_windows_read_dir_next_v0\0fy_std_windows_read_dir_close_v0\0fy_std_windows_read_dir_drop_v0\0",i))){return i+33;}i+=1;}return 0;}
pub fn bool valid(usize code){return code>=33 && code<=69;}
pub fn ref u8[] function_name(usize code){return item(b"file_open_raw\0file_open_utf8_raw\0file_remove_raw\0directory_create_raw\0directory_create_all_raw\0directory_remove_raw\0directory_remove_all_raw\0directory_rename_raw\0file_rename_raw\0file_replace_raw\0file_hard_link_raw\0file_copy_bytes_raw\0file_read_raw\0file_read_exact_raw\0file_read_append_raw\0file_position_raw\0file_size_raw\0file_kind_raw\0file_read_only_raw\0file_set_read_only_raw\0file_hard_link_count_raw\0file_allocated_size_raw\0file_path_kind_raw\0file_id_raw\0file_set_len_raw\0file_seek_start_raw\0file_seek_current_raw\0file_seek_end_raw\0file_write_raw\0file_write_all_raw\0file_sync_raw\0file_close_raw\0file_drop_raw\0read_dir_open_raw\0read_dir_next_raw\0read_dir_close_raw\0read_dir_drop_raw\0",if(valid(code)){code-33}else{37});}
pub fn ref u8[] record_name(usize code){return item(b"raw_file_result\0raw_file_position_result\0raw_file_size_result\0raw_file_kind_result\0raw_file_attribute_result\0raw_file_buffer_result\0raw_file_id\0raw_open_options\0raw_directory_entry_result\0",if(code>=6 && code<=14){code-6}else{9});}
pub fn usize result_record(usize code){if(!valid(code)){return 0;}let data=b"\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x0b\x07\x08\x09\x0a\x0a\x08\x08\x09\x06\x08\x07\x07\x07\x06\x06\x06\x06\x00\x06\x0e\x06\x00";return data[code-33] as usize;}
pub fn ref u8[] result_name(usize code){let usize record=result_record(code);return item(b"raw_file_result\0raw_file_position_result\0raw_file_size_result\0raw_file_kind_result\0raw_file_attribute_result\0raw_file_buffer_result\0raw_file_id\0raw_open_options\0raw_directory_entry_result\0u32\0",if(record!=0){record-6}else if(code==65 || code==69){9}else{10});}
pub fn usize parameter_count(usize code){if(!valid(code)){return 0;}let data=b"\x03\x04\x01\x01\x01\x01\x01\x02\x02\x02\x02\x02\x03\x03\x03\x01\x01\x01\x01\x02\x01\x01\x01\x02\x02\x02\x02\x02\x03\x03\x01\x01\x01\x03\x03\x01\x01";return data[code-33] as usize;}
pub fn usize parameter_record(usize code,usize index){return if(code==33 && index==1){13} else if(code==34 && index==2){13} else if(code==56 && index==1){12}else{0};}
pub fn bool parameter_mutable(usize code,usize index){return code==56 && index==1;}
pub fn u32 parameter(usize code,usize index){
    if(code==33){return if(index==0){pointers::make(3,false)} else if(index==1){0} else if(index==2){pointers::make(7,true)}else{0};}
    if(code==34){return if(index==0){pointers::make(3,false)} else if(index==1){5} else if(index==2){0} else if(index==3){pointers::make(7,true)}else{0};}
    if(code==35){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==36){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==37){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==38){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==39){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==40){return if(index==0){pointers::make(3,false)} else if(index==1){pointers::make(3,false)}else{0};}
    if(code==41){return if(index==0){pointers::make(3,false)} else if(index==1){pointers::make(3,false)}else{0};}
    if(code==42){return if(index==0){pointers::make(3,false)} else if(index==1){pointers::make(3,false)}else{0};}
    if(code==43){return if(index==0){pointers::make(3,false)} else if(index==1){pointers::make(3,false)}else{0};}
    if(code==44){return if(index==0){pointers::make(3,false)} else if(index==1){pointers::make(3,false)}else{0};}
    if(code==45){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(3,true)} else if(index==2){5}else{0};}
    if(code==46){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(3,true)} else if(index==2){5}else{0};}
    if(code==47){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(7,true)} else if(index==2){5}else{0};}
    if(code==48){return if(index==0){pointers::make(7,true)}else{0};}
    if(code==49){return if(index==0){pointers::make(7,false)}else{0};}
    if(code==50){return if(index==0){pointers::make(7,false)}else{0};}
    if(code==51){return if(index==0){pointers::make(7,false)}else{0};}
    if(code==52){return if(index==0){pointers::make(7,true)} else if(index==1){4}else{0};}
    if(code==53){return if(index==0){pointers::make(7,false)}else{0};}
    if(code==54){return if(index==0){pointers::make(7,false)}else{0};}
    if(code==55){return if(index==0){pointers::make(3,false)}else{0};}
    if(code==56){return if(index==0){pointers::make(7,false)} else if(index==1){0}else{0};}
    if(code==57){return if(index==0){pointers::make(7,true)} else if(index==1){5}else{0};}
    if(code==58){return if(index==0){pointers::make(7,true)} else if(index==1){5}else{0};}
    if(code==59){return if(index==0){pointers::make(7,true)} else if(index==1){1}else{0};}
    if(code==60){return if(index==0){pointers::make(7,true)} else if(index==1){1}else{0};}
    if(code==61){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(3,false)} else if(index==2){5}else{0};}
    if(code==62){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(3,false)} else if(index==2){5}else{0};}
    if(code==63){return if(index==0){pointers::make(7,true)}else{0};}
    if(code==64){return if(index==0){pointers::make(7,true)}else{0};}
    if(code==65){return if(index==0){pointers::make(7,true)}else{0};}
    if(code==66){return if(index==0){pointers::make(3,false)} else if(index==1){5} else if(index==2){pointers::make(7,true)}else{0};}
    if(code==67){return if(index==0){pointers::make(7,true)} else if(index==1){pointers::make(3,true)} else if(index==2){5}else{0};}
    if(code==68){return if(index==0){pointers::make(7,true)}else{0};}
    if(code==69){return if(index==0){pointers::make(7,true)}else{0};}
    return 0;
}
pub fn usize field_count(usize record){let usize index=if(record>=6 && record<=14){record-6}else{9};return if(index==0){3} else if(index==1){3} else if(index==2){3} else if(index==3){3} else if(index==4){3} else if(index==5){4} else if(index==6){6} else if(index==7){6} else if(index==8){6}else{0};}
pub fn u32 field_type(usize record,usize index){
    if(record==6){return if(index==0){4} else if(index==1){4} else if(index==2){5}else{0};}
    if(record==7){return if(index==0){4} else if(index==1){4} else if(index==2){5}else{0};}
    if(record==8){return if(index==0){4} else if(index==1){4} else if(index==2){5}else{0};}
    if(record==9){return if(index==0){4} else if(index==1){4} else if(index==2){4}else{0};}
    if(record==10){return if(index==0){4} else if(index==1){4} else if(index==2){4}else{0};}
    if(record==11){return if(index==0){4} else if(index==1){4} else if(index==2){4} else if(index==3){5}else{0};}
    if(record==12){return if(index==0){4} else if(index==1){4} else if(index==2){4} else if(index==3){4} else if(index==4){4} else if(index==5){4}else{0};}
    if(record==13){return if(index==0){3} else if(index==1){3} else if(index==2){3} else if(index==3){3} else if(index==4){3} else if(index==5){3}else{0};}
    if(record==14){return if(index==0){4} else if(index==1){4} else if(index==2){5} else if(index==3){4} else if(index==4){5} else if(index==5){4}else{0};}
    return 0;
}
// Locate the extern declaration rather than an earlier cleanup call.
pub fn usize declaration(ref u8[] raw,usize code){
    let prefix=b"unsafe extern C fn ";let result=result_name(code);let name=function_name(code);if(result.len==0 || name.len==0){return 0;}
    var usize i=0;while(i<=raw.len && prefix.len+result.len+1+name.len<raw.len-i){
        let usize type=i+prefix.len;let usize start=type+result.len+1;
        if(bytes::equal(raw[i..type],prefix) && bytes::equal(raw[type..type+result.len],result) && raw[type+result.len]==32 && bytes::equal(raw[start..start+name.len],name) && raw[start+name.len]==40){return start+1;}
        i+=1;
    }return 0;
}
