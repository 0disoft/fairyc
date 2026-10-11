module module_unit_c_file;
import module_unit_body_ir_model as ir;
import module_unit_c_aggregate as aggregate;
import sdk_file_rules as rules;
import c_text as w;
import std::bytes as bytes;

// Windows storage functions transport the File descriptor with memcpy. The
// private carrier keeps its integer handle and never aliases a HANDLE lvalue.
pub fn bool admitted(usize code){return code==33 || code==34 || code==35 || code==42 || code>=45 && code<=49 || code>=57 && code<=69;}
pub fn usize fields(usize record){return if(record==6 || record==7 || record==8 || record==11 || record==13 || record==14){rules::field_count(record)}else{0};}
fn ref u8[] slot(ref u8[] pool,usize index){var usize start=0;var usize i=0;while(i<index){while(pool[start]!=0){start+=1;}start+=1;i+=1;}var usize end=start;while(pool[end]!=0){end+=1;}return pool[start..end];}
pub fn ref u8[] wire_name(usize record){let pool=b"fy_std_windows_file_result_v0\0fy_std_windows_file_position_result_v0\0fy_std_windows_file_size_result_v0\0fy_std_windows_file_buffer_result_v0\0fy_std_windows_open_options_v0\0fy_std_directory_entry_result_v0\0";return slot(pool,if(record==6){0}else if(record==7){1}else if(record==8){2}else if(record==11){3}else if(record==13){4}else{5});}
pub fn ref u8[] field_name(usize record,usize index){let pool=b"read\0write\0create\0truncate\0exclusive\0append\0status\0file_status\0os_code\0allocation_status\0position\0size\0count\0name_length\0kind\0reparse\0";let usize at=if(record==14){if(index==0){6}else if(index==1){8}else if(index==2){13}else if(index==3){14}else if(index==4){11}else{15}}else if(record==13){index}else if(index==0){if(record==11){7}else{6}}else if(index==1){8}else if(record==11 && index==2){9}else if(record==7){10}else if(record==8){11}else{12};return slot(pool,at);}
pub fn bool definitions(ref ir::Program p,usize type,usize layout,ref mut bytes::Buffer out,ref mut w::State s){if(type==0){return true;}let shape=ir::layout_at(p,layout-1).value;
    if(!w::text(out,s,b"_Static_assert(sizeof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==16&&_Alignof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==8,\"SDK File storage layout\");\n")){return false;}
    var usize i=0;while(i<2){if(!w::text(out,s,b"_Static_assert(offsetof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b",") || !aggregate::field(out,s,shape.field_start+i+1) || !w::text(out,s,b")==") || !w::number(out,s,i*8) || !w::text(out,s,b",\"SDK File storage offset\");\n")){return false;}i+=1;}return true;
}
