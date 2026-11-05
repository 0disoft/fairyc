module project_pack_options;
import project_manifest_values as manifest;
import project_atlas_wire as wire;
import std::bytes as bytes;
import std::process as process;

// Exact Atlas pack argument selection. This leaf performs no workspace, cache,
// network, map-store, or stdin I/O.
pub resource Options{
    bytes::Buffer uid;bytes::Buffer symbol;u32 action;u32 direction;usize relation_mask;
    usize records;usize relations;usize depth;usize selection;usize output;
    u32 version;bool locked;cleanup{}
}
pub fn ref u8[] uid(ref Options p){return bytes::view(ref p.uid);}
pub fn ref u8[] symbol(ref Options p){return bytes::view(ref p.symbol);}
pub fn ref u8[] direction(ref Options p){
    if(p.action==0){return b"";}let ref u8[] data=b"outgoingincomingboth";
    return if(p.direction==0){data[..8]}else if(p.direction==1){data[8..16]}else{data[16..20]};
}
pub fn ref u8[] action(ref Options p){
    let ref u8[] data=b"overviewshowaround";
    return if(p.action==0){data[..8]}else if(p.action==1){data[8..12]}else{data[12..18]};
}
pub fn usize records(ref Options p){return p.records;}
pub fn usize relations(ref Options p){return p.relations;}
pub fn usize depth(ref Options p){return p.depth;}
pub fn usize selection(ref Options p){return p.selection;}
pub fn usize output(ref Options p){return p.output;}
pub fn u32 version(ref Options p){return p.version;}
pub fn bool locked(ref Options p){return p.locked;}
pub fn bool has_relation(ref Options p,usize index){
    return index<10 && (p.relation_mask & (1<<index))!=0;
}
pub fn ref u8[] relation(usize index){
    let ref u8[] data=b"allocatesborrows-mutableborrows-sharedcallscalls-foreigndropsmay-trapmovesreturns-erroruses-unsafe";
    return if(index==0){data[..9]}
    else if(index==1){data[9..24]}
    else if(index==2){data[24..38]}
    else if(index==3){data[38..43]}
    else if(index==4){data[43..56]}
    else if(index==5){data[56..61]}
    else if(index==6){data[61..69]}
    else if(index==7){data[69..74]}
    else if(index==8){data[74..87]}
    else if(index==9){data[87..98]}
    else{data[..0]};
}
fn usize relation_index(ref u8[] value){
    var usize i=0;while(i<10){if(bytes::equal(value,relation(i))){return i;}i+=1;}return 10;
}
fn usize flag(ref u8[] value){
    if(bytes::equal(value,b"--pack-version")){return 0;}
    if(bytes::equal(value,b"--package-uid")){return 1;}
    if(bytes::equal(value,b"--symbol")){return 2;}
    if(bytes::equal(value,b"--direction")){return 3;}
    if(bytes::equal(value,b"--relation")){return 4;}
    if(bytes::equal(value,b"--max-records")){return 5;}
    if(bytes::equal(value,b"--max-relations")){return 6;}
    if(bytes::equal(value,b"--max-depth")){return 7;}
    if(bytes::equal(value,b"--max-selection-bytes")){return 8;}
    if(bytes::equal(value,b"--max-output-bytes")){return 9;}
    if(bytes::equal(value,b"--locked")){return 10;}return 11;
}
fn Result<usize,u32> decimal(ref u8[] value,usize maximum,bool zero){
    if(value.len==0 || value.len>1 && value[0]==48){return .err(2);}
    var usize number=0;var usize i=0;
    while(i<value.len){
        if(value[i]<48 || value[i]>57){return .err(2);}
        let digit=(value[i]-48) as usize;
        if(number>maximum/10 || number==maximum/10 && digit>maximum%10){return .err(2);}
        number=number*10+digit;i+=1;
    }
    return if(!zero && number==0){.err(2)}else{.ok(number)};
}
pub fn Result<Options,u32> arguments(){
    // argv includes the executable, experimental, atlas-pack and query words.
    // The largest valid around invocation has all ten relations and is 43.
    let argc=process::arg_count();if(argc<4 || argc>43){return .err(2);}
    var u8[4096] raw=[0;4096];
    let action_size=switch(process::arg(3,ref mut raw)){case .ok(n):{n}case .err(_):{return .err(2);}};
    let ref u8[] action_value=ref raw;
    var u32 action=3;
    if(bytes::equal(action_value[..action_size],b"overview")){action=0;}
    else if(bytes::equal(action_value[..action_size],b"show")){action=1;}
    else if(bytes::equal(action_value[..action_size],b"around")){action=2;}
    if(action==3){return .err(2);}

    var bytes::Buffer uid=bytes::empty();var bytes::Buffer symbol=bytes::empty();
    var u32 direction=0;var usize relation_mask=0;var usize records=0;var usize relations=0;
    var usize depth=0;var usize selection=0;var usize output=0;var u32 version=1;var bool locked=false;
    var bool[11] seen=[false;11];var usize at=4;
    while(at<argc){
        let key_size=switch(process::arg(at,ref mut raw)){case .ok(n):{n}case .err(_):{return .err(2);}};
        let ref u8[] key=ref raw;let index=flag(key[..key_size]);
        if(index==11 || seen[index] && index!=4){return .err(2);}
        if(action==0 && (index==2 || index==3 || index==4 || index==6 || index==7)){return .err(2);}
        if(action==1 && index==7){return .err(2);}seen[index]=true;at+=1;
        if(index==10){locked=true;continue;}if(at==argc){return .err(2);}
        let value_size=switch(process::arg(at,ref mut raw)){case .ok(n):{n}case .err(_):{return .err(2);}};
        let ref u8[] value=ref raw;let s=value[..value_size];
        if(s.len>=2 && s[0]==45 && s[1]==45){return .err(2);}
        if(index==0){
            if(bytes::equal(s,b"1")){version=1;}
            else if(bytes::equal(s,b"2")){version=2;}
            else if(bytes::equal(s,b"3")){version=3;}
            else{return .err(2);}
        }else if(index==1){
            if(s.len>256 || !manifest::uid(s) || !bytes::append(ref mut uid,s).is_ok){return .err(2);}
        }else if(index==2){
            if(!(wire::hex(s,b"fysym1-sha256-128-",32,false) || wire::hex(s,b"fysym1-sha256-256-",64,false)) || !bytes::append(ref mut symbol,s).is_ok){return .err(2);}
        }else if(index==3){
            if(bytes::equal(s,b"outgoing")){direction=0;}
            else if(bytes::equal(s,b"incoming")){direction=1;}
            else if(bytes::equal(s,b"both")){direction=2;}
            else{return .err(2);}
        }else if(index==4){
            let kind=relation_index(s);if(kind==10){return .err(2);}let bit=1<<kind;
            if((relation_mask & bit)!=0){return .err(2);}relation_mask|=bit;
        }else if(index==7){
            depth=decimal(s,64,true)?;
        }else{
            let limit=decimal(s,if(index==5 || index==6){65536}else{1073741824},false)?;
            if(index==5){records=limit;}else if(index==6){relations=limit;}else if(index==8){selection=limit;}else{output=limit;}
        }
        at+=1;
    }
    if(action==0){
        if(!seen[1] || !seen[5] || !seen[8] || !seen[9]){return .err(2);}
    }else if(!seen[1] || !seen[2] || !seen[3] || !seen[4] || !seen[5] || !seen[6] || !seen[8] || !seen[9] || action==2 && !seen[7]){
        return .err(2);
    }
    return .ok(Options{.uid,.symbol,.action,.direction,.relation_mask,.records,.relations,.depth,.selection,.output,.version,.locked});
}
