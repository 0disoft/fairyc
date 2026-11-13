module desktop_package_options;
import std::bytes as bytes;
import std::vec as v;

pub struct Selection{bool locked;bool json;}
pub struct Problem{usize index;bool duplicate;}
pub fn Result<Selection,Problem> parse(ref v::Vec<bytes::Buffer> input){
    var bool locked=false;var bool json=false;var usize i=0;
    while(i<v::len<bytes::Buffer>(input)){
        let value=bytes::view(v::at<bytes::Buffer>(input,i));
        if(bytes::equal(value,b"--locked")){if(locked){return .err(Problem{.index=i,.duplicate=true});}locked=true;}
        else if(bytes::equal(value,b"--json")){if(json){return .err(Problem{.index=i,.duplicate=true});}json=true;}
        else{return .err(Problem{.index=i,.duplicate=false});}i+=1;
    }
    return .ok(Selection{.locked,.json});
}
