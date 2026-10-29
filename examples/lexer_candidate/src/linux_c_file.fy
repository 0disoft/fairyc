module linux_c_file;
import module_unit_c_file as base;
import module_unit_body_ir_model as ir;
import sdk_file_rules as rules;
import c_text as w;
import std::bytes as bytes;
pub fn bool admitted(usize code){return base::admitted(code);}
pub fn usize fields(usize record){return base::fields(record);}
fn ref u8[] slot(ref u8[] pool,usize index){var usize start=0;var usize i=0;while(i<index){while(pool[start]!=0){start+=1;}start+=1;i+=1;}var usize end=start;while(pool[end]!=0){end+=1;}return pool[start..end];}
pub fn ref u8[] wire_name(usize record){let pool=b"fy_std_file_result_v0\0fy_std_file_position_result_v0\0fy_std_file_size_result_v0\0fy_std_file_buffer_result_v0\0fy_std_file_open_options_v0\0fy_std_directory_entry_result_v0\0fy_std_windows_file_kind_result_v0\0fy_std_windows_file_id_v0\0fy_std_windows_file_attribute_result_v0\0";return slot(pool,if(record==6){0}else if(record==7){1}else if(record==8){2}else if(record==11){3}else if(record==13){4}else if(record==14){5}else if(record==9){6}else if(record==10){8}else{7});}
pub fn ref u8[] field_name(usize record,usize index){let pool=b"read\0write\0create\0truncate\0exclusive\0append\0status\0file_status\0os_code\0allocation_status\0position\0size\0count\0name_length\0kind\0reparse\0volume_low\0volume_high\0file_low_0\0file_low_1\0file_high_0\0file_high_1\0enabled\0error_number\0";let usize at=if(record==6 && index==1){23}else if(record==12){16+index}else if(record==14){if(index==0){6}else if(index==1){8}else if(index==2){13}else if(index==3){14}else if(index==4){11}else{15}}else if(record==13){index}else if(index==0){if(record==11){7}else{6}}else if(index==1){8}else if(record==11 && index==2){9}else if(record==7){10}else if(record==8){11}else if(record==9){14}else if(record==10){22}else{12};return slot(pool,at);}
pub fn u32 field_type(usize record,usize index){return if(record==6 && index==1){1}else{rules::field_type(record,index)};}
pub fn bool definitions(ref ir::Program p,usize type,usize layout,ref mut bytes::Buffer out,ref mut w::State s){return base::definitions(p,type,layout,out,s);}
