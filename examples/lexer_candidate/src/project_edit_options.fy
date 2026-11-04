module project_edit_options;
import std::bytes as bytes;
import std::process as process;

pub resource Options{bytes::Buffer uid;bytes::Buffer symbol;bytes::Buffer body;bytes::Buffer expected;bool apply;cleanup{}}
pub fn ref u8[] uid(ref Options p){return bytes::view(ref p.uid);}
pub fn ref u8[] symbol(ref Options p){return bytes::view(ref p.symbol);}
pub fn ref u8[] body_file(ref Options p){return bytes::view(ref p.body);}
pub fn ref u8[] expected(ref Options p){return bytes::view(ref p.expected);}
pub fn bool apply(ref Options p){return p.apply;}
fn usize flag(ref u8[] value){
    if(bytes::equal(value,b"--package-uid")){return 0;}
    if(bytes::equal(value,b"--symbol")){return 1;}
    if(bytes::equal(value,b"--body-file")){return 2;}
    if(bytes::equal(value,b"--expected-source-sha256")){return 3;}
    if(bytes::equal(value,b"--apply")){return 4;}
    return 5;
}
// Parse invocation syntax without touching a workspace. Selector, path and raw
// digest checks belong to the host in the same order as the Go command.
pub fn Result<Options,u32> arguments(){
    let count=process::arg_count();if(count<12 || count>13){return .err(2);}
    var u8[4096] command=[0;4096];
    let length=switch(process::arg(3,ref mut command)){case .ok(p):{p}case .err(_):{return .err(2);}};
    let ref u8[] spelling=ref command;if(!bytes::equal(spelling[..length],b"replace-body")){return .err(2);}
    var uid=bytes::empty();var symbol=bytes::empty();var body=bytes::empty();var expected=bytes::empty();
    var bool[5] seen=[false;5];var usize at=4;
    while(at<count){
        var u8[4096] raw=[0;4096];let n=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
        let ref u8[] key=ref raw;let which=flag(key[..n]);if(which==5 || seen[which]){return .err(2);}seen[which]=true;at+=1;
        if(which==4){continue;}if(at==count){return .err(2);}
        var u8[4096] value=[0;4096];let size=switch(process::arg(at,ref mut value)){case .ok(p):{p}case .err(_):{return .err(2);}};
        let ref u8[] source=ref value;
        let ok=if(which==0){bytes::append(ref mut uid,source[..size]).is_ok}else if(which==1){bytes::append(ref mut symbol,source[..size]).is_ok}else if(which==2){bytes::append(ref mut body,source[..size]).is_ok}else{bytes::append(ref mut expected,source[..size]).is_ok};
        if(!ok){return .err(2);}at+=1;
    }
    if(!seen[0] || !seen[1] || !seen[2] || !seen[3]){return .err(2);}
    return .ok(Options{.uid,.symbol,.body,.expected,.apply=seen[4]});
}
