module project_sdk_embedded;
import project_sdk_bundle as sdk;
import project_source_set as sources;
import project_sdk_data_alloc_0 as alloc;
import project_sdk_data_bytes_0 as byte_data;
import project_sdk_data_fs_0 as fs_first;
import project_sdk_data_fs_1 as fs_second;
import project_sdk_data_fs_2 as fs_last;
import project_sdk_data_io_0 as io;
import project_sdk_data_num_0 as num;
import project_sdk_data_process_0 as process;
import project_sdk_data_text_0 as text;
import project_sdk_data_vec_0 as vector;
import std::bytes as bytes;
import std::vec as v;

fn sdk::Problem fail(u32 code,usize file){return sdk::Problem{.code,.file};}
fn Result<bytes::Buffer,u32> source(usize id,ref mut usize meter){var bytes::Buffer output=bytes::empty();var u32 code=0;
    if(id==1){code=alloc::append(ref mut output,meter);}else if(id==2){code=byte_data::append(ref mut output,meter);}else if(id==3){code=fs_first::append(ref mut output,meter);if(code==0){code=fs_second::append(ref mut output,meter);}if(code==0){code=fs_last::append(ref mut output,meter);}}else if(id==4){code=io::append(ref mut output,meter);}else if(id==5){code=num::append(ref mut output,meter);}else if(id==6){code=process::append(ref mut output,meter);}else if(id==7){code=text::append(ref mut output,meter);}else if(id==8){code=vector::append(ref mut output,meter);}else{return .err(2);}if(code!=0){return .err(code);}return .ok(output);
}
// A compiler-owned copy of the current Go SDK source bytes. Literal data grants
// no authority: the existing Fairy source-pin checker authenticates the bundle.
// 0/unknown profiles are invalid; callers skip this factory when selection is 0.
pub fn Result<sdk::Bundle,sdk::Problem> load(u32 profile,ref mut usize meter){if(profile<1 || profile>3){return .err(fail(2,0));}let usize count=if(profile==1){7}else{8};var v::Vec<sdk::File> input=v::empty<sdk::File>();var usize at=0;
    while(at<count){let bytes::Buffer body=switch(source(at+1,meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,at));}};if(!v::push<sdk::File>(ref mut input,sdk::file(at+1,body)).is_ok){return .err(fail(4,at));}at+=1;}
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};return sdk::assemble(input,profile==3,ref mut budget,meter);
}
