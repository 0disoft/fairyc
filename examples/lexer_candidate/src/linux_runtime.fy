module linux_runtime;
import linux_runtime_contract as contract;
import std::bytes as bytes;

// Pure GNU companion runtime admission. IDs are Linux-local; runtime bytes and
// process launch are separate. Source owners: candidatebuild/linux.go,
// nativesupport/source.go and candidatehost/linux_paths.go.
pub struct Plan {u32 components;u32 hosts;bool vectors;bool arguments;}
// 1 target marker, 2 stdlib symbol, 3 vector symbol, 4 Linux host symbol,
// 5 Windows dependency. Spans borrow the supplied packet header.
pub struct Problem {u32 kind;usize start;usize end;}
fn bool at(ref u8[] data,usize start,ref u8[] prefix){return start<=data.len && prefix.len<=data.len-start && bytes::equal(data[start..start+prefix.len],prefix);}
fn bool word(u8 c){return c>=48 && c<=57 || c>=65 && c<=90 || c>=97 && c<=122 || c==95;}
fn bool space(u8 c){return c==9 || c==10 || c==12 || c==13 || c==32;}
pub fn bool contains(ref Plan p,u32 id){return id>=1 && id<=contract::component_count() && (p.components & (1<<(id-1)))!=0;}
// Host mask order for assembly: rooted, process, capture, input, paths,
// directory, retention, workspace, build scratch, starter, GNU; then package
// directory, package file, clock, HTTPS protocol, worker, reader, parent.
// Edit-file is bit 18; its asset is assembled after package-file.
pub fn u32 host(ref u8[] name){
    if(bytes::equal(name,b"fairyc_private_linux_edit_file_prepare_v1") || bytes::equal(name,b"fairyc_private_linux_edit_file_publish_v1")){return 1|(1<<18);}
    if(bytes::equal(name,b"fairyc_private_linux_https_open_v1") || bytes::equal(name,b"fairyc_private_linux_https_headers_v1") || bytes::equal(name,b"fairyc_private_linux_https_read_v1") || bytes::equal(name,b"fairyc_private_linux_https_close_v1")){return (1<<13)|(1<<14)|(1<<15)|(1<<16)|(1<<17);}
    if(bytes::equal(name,b"fairyc_private_linux_monotonic_ms_v1")){return 1<<13;}
    if(bytes::equal(name,b"fairyc_private_linux_package_directory_create_v1") || bytes::equal(name,b"fairyc_private_linux_package_directory_edit_v1") || bytes::equal(name,b"fairyc_private_linux_package_directory_publish_v1") || bytes::equal(name,b"fairyc_private_linux_package_directory_erase_v1") || bytes::equal(name,b"fairyc_private_linux_package_directory_clone_v1")){return 1|(1<<11);}
    if(bytes::equal(name,b"fairyc_private_linux_package_file_info_v1")){return 1|(1<<12);}
    if(bytes::equal(name,b"fairyc_private_linux_gnu_stage_v1") || bytes::equal(name,b"fairyc_private_linux_gnu_write_v1") || bytes::equal(name,b"fairyc_private_linux_gnu_read_v1") || bytes::equal(name,b"fairyc_private_linux_gnu_find_v1") || bytes::equal(name,b"fairyc_private_linux_gnu_run_v1") || bytes::equal(name,b"fairyc_private_linux_gnu_publish_v1")){return 1031;}
    if(bytes::equal(name,b"fairyc_private_linux_host_path_v1")){return 16;}
    if(bytes::equal(name,b"fairyc_private_linux_process_start_v1") || bytes::equal(name,b"fairyc_private_linux_process_wait_v1") || bytes::equal(name,b"fairyc_private_linux_process_discard_v1") || bytes::equal(name,b"fairyc_private_linux_process_wait_for_v1")){return 2;}
    if(bytes::equal(name,b"fairyc_private_linux_capture_open_v1") || bytes::equal(name,b"fairyc_private_linux_run_capture_open_v1") || bytes::equal(name,b"fairyc_private_linux_run_capture_receipt_v1") || bytes::equal(name,b"fairyc_private_linux_capture_drop_v1") || bytes::equal(name,b"fairyc_private_linux_capture_collect_v1")){return 6;}
    if(bytes::equal(name,b"fairyc_private_linux_test_capture_open_v1")){return 14;}
    if(bytes::equal(name,b"fairyc_private_linux_directory_open_v1") || bytes::equal(name,b"fairyc_private_linux_root_directory_open_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_open_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_info_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_read_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_close_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_create_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_write_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_sync_v1") || bytes::equal(name,b"fairyc_private_linux_root_file_publish_new_v1")){return 1;}
    if(bytes::equal(name,b"fairyc_private_linux_directory_scan_open_v1") || bytes::equal(name,b"fairyc_private_linux_directory_scan_next_v1") || bytes::equal(name,b"fairyc_private_linux_directory_scan_close_v1")){return 33;}
    if(bytes::equal(name,b"fairyc_private_linux_root_file_erase_if_v1")){return 65;}
    if(bytes::equal(name,b"fairyc_private_linux_workspace_path_kind_v1")){return 129;}
    if(bytes::equal(name,b"fairyc_private_linux_build_scratch_open_v1") || bytes::equal(name,b"fairyc_private_linux_build_scratch_drop_v1")){return 257;}
    if(bytes::equal(name,b"fairyc_private_linux_starter_create_v1")){return 513;}return 0;
}
fn bool vector(ref u8[] name){
    return bytes::equal(name,b"fy_private_vector_initialize_v0") || bytes::equal(name,b"fy_private_vector_release_empty_v0") || bytes::equal(name,b"fy_private_vector_reserve_v0") || bytes::equal(name,b"fy_private_vector_prepare_push_v0") || bytes::equal(name,b"fy_private_vector_commit_push_v0") || bytes::equal(name,b"fy_private_vector_pop_slot_v0") || bytes::equal(name,b"fy_private_vector_element_slot_v0") || bytes::equal(name,b"fy_private_vector_element_slot_mut_v0");
}
pub fn Result<Plan,Problem> select(ref u8[] header){
    let ref u8[] marker=b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */";var bool marked=false;var usize i=0;
    while(i<header.len){if(at(header,i,marker)){marked=true;break;}i+=1;}if(!marked){return .err(Problem{.kind=1,.start=0,.end=0});}
    var Plan plan=Plan{.components=0,.hosts=0,.vectors=false,.arguments=false};i=0;
    while(i<header.len){
        if((i==0 || !word(header[i-1])) && (at(header,i,b"fy_std_") || at(header,i,b"fy_private_vector_") || at(header,i,b"fairyc_private_"))){
            var usize end=i;while(end<header.len && word(header[end])){end+=1;}var usize call=end;while(call<header.len && space(header[call])){call+=1;}
            if(call<header.len && header[call]==40){let name=header[i..end];var u32 problem=0;
                if(at(name,0,b"fairyc_private_windows_")){problem=5;}
                else if(at(name,0,b"fy_std_")){
                    // Portable std::fs still uses some legacy Windows ABI names.
                    let id=contract::component(name);
                    if(id==0){problem=if(at(name,0,b"fy_std_windows_")){5}else{2};}
                    else{plan.components|=contract::closure(id);if(id==contract::process_args_id()){plan.arguments=true;}}
                }
                else if(at(name,0,b"fy_private_vector_")){if(!vector(name)){problem=3;}else{plan.vectors=true;plan.components|=contract::closure(contract::system_heap_id());}}
                else if(at(name,0,b"fairyc_private_linux_")){let bits=host(name);if(bits==0){problem=4;}else{plan.hosts|=bits;}}
                if(problem!=0){return .err(Problem{.kind=problem,.start=i,.end});}
            }i=end;
        }else{i+=1;}
    }return .ok(plan);
}
