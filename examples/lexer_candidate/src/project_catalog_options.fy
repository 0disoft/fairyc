module project_catalog_options;
import project_manifest_values as manifest;
import project_atlas_model as model;
import project_atlas_wire as wire;
import std::bytes as bytes;
import std::process as process;

// Exact CLI selection only. Parsing performs no workspace, cache or stdin I/O.
// Kind positions use the same lexical order as the Go catalog request.
pub resource Options{
    bytes::Buffer uid;bytes::Buffer module_name;bytes::Buffer name;bytes::Buffer after;
    usize kinds;usize symbols;usize selection;usize output;u32 version;bool locked;
    cleanup{}
}
pub fn ref u8[] uid(ref Options p){return bytes::view(ref p.uid);}
pub fn ref u8[] module_name(ref Options p){return bytes::view(ref p.module_name);}
pub fn ref u8[] name(ref Options p){return bytes::view(ref p.name);}
pub fn ref u8[] after(ref Options p){return bytes::view(ref p.after);}
pub fn usize symbols(ref Options p){return p.symbols;}
pub fn usize selection(ref Options p){return p.selection;}
pub fn usize output(ref Options p){return p.output;}
pub fn u32 version(ref Options p){return p.version;}
pub fn bool locked(ref Options p){return p.locked;}
pub fn bool has_kind(ref Options p,usize index){return index<5 && (p.kinds & (1<<index))!=0;}
pub fn ref u8[] kind(usize index){let ref u8[] data=b"enumforeign-functionfunctionresourcestruct";return if(index==0){data[..4]}else if(index==1){data[4..20]}else if(index==2){data[20..28]}else if(index==3){data[28..36]}else if(index==4){data[36..42]}else{data[..0]};}
fn usize kind_index(ref u8[] value){var usize i=0;while(i<5){if(bytes::equal(value,kind(i))){return i;}i+=1;}return 5;}
fn usize flag(ref u8[] s){
    if(bytes::equal(s,b"--catalog-version")){return 0;}if(bytes::equal(s,b"--package-uid")){return 1;}
    if(bytes::equal(s,b"--module")){return 2;}if(bytes::equal(s,b"--name")){return 3;}
    if(bytes::equal(s,b"--kind")){return 4;}if(bytes::equal(s,b"--after")){return 5;}
    if(bytes::equal(s,b"--max-symbols")){return 6;}if(bytes::equal(s,b"--max-selection-bytes")){return 7;}
    if(bytes::equal(s,b"--max-output-bytes")){return 8;}if(bytes::equal(s,b"--locked")){return 9;}return 10;
}
fn Result<usize,u32> positive(ref u8[] s,usize maximum){
    if(s.len==0 || s.len>1 && s[0]==48){return .err(2);}var usize n=0;var usize i=0;
    while(i<s.len){if(s[i]<48 || s[i]>57){return .err(2);}let digit=(s[i]-48) as usize;
        if(n>maximum/10 || n==maximum/10 && digit>maximum%10){return .err(2);}n=n*10+digit;i+=1;
    }return if(n==0){.err(2)}else{.ok(n)};
}
pub fn Result<Options,u32> arguments(){
    // Four command words, eight scalar flags, five distinct kinds and a switch.
    let argc=process::arg_count();if(argc<4 || argc>31){return .err(2);}
    var u8[4096] raw=[0;4096];let n=switch(process::arg(3,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
    let ref u8[] action=ref raw;if(!bytes::equal(action[..n],b"list")){return .err(2);}
    var bytes::Buffer uid=bytes::empty();var bytes::Buffer module_name=bytes::empty();
    var bytes::Buffer name=bytes::empty();var bytes::Buffer after=bytes::empty();
    var usize kinds=0;var usize symbols=0;var usize selection=0;var usize output=0;var u32 version=1;var bool locked=false;
    var bool[10] seen=[false;10];var usize at=4;
    while(at<argc){
        let size=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};
        let ref u8[] key=ref raw;let index=flag(key[..size]);if(index==10 || seen[index] && index!=4){return .err(2);}seen[index]=true;at+=1;
        if(index==9){locked=true;continue;}if(at==argc){return .err(2);}
        let value_size=switch(process::arg(at,ref mut raw)){case .ok(p):{p}case .err(_):{return .err(2);}};let ref u8[] value=ref raw;let s=value[..value_size];
        if(s.len>=2 && s[0]==45 && s[1]==45){return .err(2);}
        if(index==0){if(bytes::equal(s,b"1")){version=1;}else if(bytes::equal(s,b"2")){version=2;}else if(bytes::equal(s,b"3")){version=3;}else{return .err(2);}}
        else if(index==1){if(s.len>256 || !manifest::uid(s) || !bytes::append(ref mut uid,s).is_ok){return .err(2);}}
        else if(index==2){if(s.len==0 || s.len>512 || !bytes::append(ref mut module_name,s).is_ok){return .err(2);}}
        else if(index==3){if(s.len>512 || !model::qualified(s) || !bytes::append(ref mut name,s).is_ok){return .err(2);}}
        else if(index==4){let k=kind_index(s);if(k==5){return .err(2);}let usize bit=1<<k;if((kinds & bit)!=0){return .err(2);}kinds|=bit;}
        else if(index==5){if(!wire::hex(s,b"fysym1-sha256-256-",64,false) || !bytes::append(ref mut after,s).is_ok){return .err(2);}}
        else{let limit=positive(s,if(index==6){65536}else{1073741824})?;if(index==6){symbols=limit;}else if(index==7){selection=limit;}else{output=limit;}}
        at+=1;
    }
    if(!seen[1] || !seen[4] || !seen[6] || !seen[7] || !seen[8]){return .err(2);}return .ok(Options{.uid,.module_name,.name,.after,.kinds,.symbols,.selection,.output,.version,.locked});
}
