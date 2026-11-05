module project_calls_options;
import project_manifest_values as manifest;
import project_atlas_wire as wire;
import std::bytes as bytes;
import std::process as process;

// Argument selection only: no workspace, map/cache, network or stdin access.
pub resource Options{bytes::Buffer uid;bytes::Buffer symbol;u32 direction;usize packages;usize calls;usize output;u32 version;bool locked;cleanup{}}
pub fn ref u8[] uid(ref Options p){return bytes::view(ref p.uid);}
pub fn ref u8[] symbol(ref Options p){return bytes::view(ref p.symbol);}
pub fn ref u8[] direction(ref Options p){let ref u8[] data=b"outgoingincomingboth";return if(p.direction==0){data[..8]}else if(p.direction==1){data[8..16]}else{data[16..20]};}
pub fn bool outgoing(ref Options p){return p.direction!=1;}
pub fn bool incoming(ref Options p){return p.direction!=0;}
pub fn usize packages(ref Options p){return p.packages;}
pub fn usize calls(ref Options p){return p.calls;}
pub fn usize output(ref Options p){return p.output;}
pub fn u32 version(ref Options p){return p.version;}
pub fn u32 map_version(ref Options p){return p.version+1;}
pub fn bool locked(ref Options p){return p.locked;}
fn usize flag(ref u8[] s){
    if(bytes::equal(s,b"--catalog-version")){return 0;}if(bytes::equal(s,b"--package-uid")){return 1;}
    if(bytes::equal(s,b"--symbol")){return 2;}if(bytes::equal(s,b"--direction")){return 3;}
    if(bytes::equal(s,b"--max-packages")){return 4;}if(bytes::equal(s,b"--max-calls")){return 5;}
    if(bytes::equal(s,b"--max-output-bytes")){return 6;}if(bytes::equal(s,b"--locked")){return 7;}return 8;
}
fn Result<usize,u32> positive(ref u8[] s,usize maximum){
    if(s.len==0 || s.len>1 && s[0]==48){return .err(2);}var usize n=0;var usize i=0;
    while(i<s.len){if(s[i]<48 || s[i]>57){return .err(2);}let digit=(s[i]-48) as usize;
        if(n>maximum/10 || n==maximum/10 && digit>maximum%10){return .err(2);}n=n*10+digit;i+=1;
    }return if(n==0){.err(2)}else{.ok(n)};
}
pub fn Result<Options,u32> arguments(){
    // Four command words, seven scalar flags and the optional locked switch.
    let argc=process::arg_count();if(argc<4 || argc>19){return .err(2);}
    var u8[4096] raw=[0;4096];let n=switch(process::arg(3,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
    let ref u8[] action=ref raw;if(!bytes::equal(action[..n],b"list")){return .err(2);}
    var uid=bytes::empty();var symbol=bytes::empty();var u32 direction=0;
    var usize packages=0;var usize calls=0;var usize output=0;var u32 version=0;var bool locked=false;
    var bool[8] seen=[false;8];var usize at=4;
    while(at<argc){let n=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
        let ref u8[] key=ref raw;let index=flag(key[..n]);if(index==8 || seen[index]){return .err(2);}seen[index]=true;at+=1;
        if(index==7){locked=true;continue;}if(at==argc){return .err(2);}
        let size=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};let ref u8[] value=ref raw;let s=value[..size];
        if(s.len>=2 && s[0]==45 && s[1]==45){return .err(2);}
        if(index==0){if(bytes::equal(s,b"0")){version=0;}else if(bytes::equal(s,b"1")){version=1;}else if(bytes::equal(s,b"2")){version=2;}else{return .err(2);}}
        else if(index==1){if(s.len>256 || !manifest::uid(s) || !bytes::append(ref mut uid,s).is_ok){return .err(2);}}
        else if(index==2){if(!(wire::hex(s,b"fysym1-sha256-128-",32,false) || wire::hex(s,b"fysym1-sha256-256-",64,false)) || !bytes::append(ref mut symbol,s).is_ok){return .err(2);}}
        else if(index==3){if(bytes::equal(s,b"outgoing")){direction=0;}else if(bytes::equal(s,b"incoming")){direction=1;}else if(bytes::equal(s,b"both")){direction=2;}else{return .err(2);}}
        else{let limit=positive(s,if(index==6){1073741824}else{65536})?;if(index==4){packages=limit;}else if(index==5){calls=limit;}else{output=limit;}}
        at+=1;
    }
    if(!seen[1] || !seen[2] || !seen[3] || !seen[4] || !seen[5] || !seen[6]){return .err(2);}
    return .ok(Options{.uid,.symbol,.direction,.packages,.calls,.output,.version,.locked});
}
