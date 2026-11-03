module project_remote_archive;
import project_inflate as inflate;
import project_manifest as manifest;
import project_manifest_values as rules;
import std::bytes as bytes;
import std::vec as v;

pub resource Entry{bytes::Buffer name;bytes::Buffer contents;usize hash;bool directory;bool explicit;cleanup{}}
pub resource Bundle{v::Vec<Entry> entries;usize expanded;usize files;cleanup{}}
pub fn usize count(ref Bundle b){return v::len<Entry>(ref b.entries);}
pub fn ref u8[] name(ref Bundle b,usize i){return bytes::view(ref v::at<Entry>(ref b.entries,i).name);}
pub fn ref u8[] contents(ref Bundle b,usize i){return bytes::view(ref v::at<Entry>(ref b.entries,i).contents);}
pub fn bool directory(ref Bundle b,usize i){return v::at<Entry>(ref b.entries,i).directory;}
pub fn usize expanded(ref Bundle b){return b.expanded;}
pub fn usize files(ref Bundle b){return b.files;}
fn usize little(ref u8[] data,usize at,usize n){var usize value=0;var usize i=n;while(i>0){i-=1;value=value*256+(data[at+i] as usize);}return value;}
fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
fn bool equal_fold(ref u8[] a,ref u8[] b){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(folded(a[i])!=folded(b[i])){return false;}i+=1;}return true;}
fn usize hash(ref u8[] name){var usize value=0;var usize i=0;while(i<name.len){value=(value*33+(folded(name[i]) as usize))%16777213;i+=1;}return value;}
pub fn bool portable(ref u8[] name){
    if(name.len==0 || name.len>1024){return false;}var usize start=0;var usize i=0;var usize depth=0;
    while(i<=name.len){if(i==name.len || name[i]==47){let ref u8[] part=name[start..i];depth+=1;if(part.len==0 || depth>32 || bytes::equal(part,b".") || bytes::equal(part,b"..") || equal_fold(part,b".git") || equal_fold(part,b".hg") || equal_fold(part,b".svn")){return false;}
        if(part[0]==46){var bytes::Buffer adjusted=bytes::empty();if(!bytes::append(ref mut adjusted,b"x").is_ok || !bytes::append(ref mut adjusted,part).is_ok || !rules::segment(bytes::view(ref adjusted))){return false;}}
        else if(!rules::segment(part)){return false;}start=i+1;
    }i+=1;}return true;
}
fn Result<usize,u32> add(ref mut v::Vec<Entry> entries,ref u8[] name,bool directory,bool explicit,bytes::Buffer contents){
    let usize h=hash(name);var usize i=0;while(i<v::len<Entry>(entries)){let ref Entry item=v::at<Entry>(entries,i);
        if(item.hash==h && equal_fold(bytes::view(ref item.name),name)){
            if(!bytes::equal(bytes::view(ref item.name),name) || item.directory!=directory || (explicit && item.explicit)){return .err(1);}
            if(explicit){let ref mut Entry changed=v::at_mut<Entry>(entries,i);changed.explicit=true;}return .ok(i);
        }i+=1;
    }
    if(v::len<Entry>(entries)>=8192){return .err(3);}var bytes::Buffer path=bytes::empty();if(!bytes::append(ref mut path,name).is_ok){return .err(2);}if(!v::push<Entry>(entries,Entry{.name=path,.contents,.hash=h,.directory,.explicit}).is_ok){return .err(2);}return .ok(v::len<Entry>(entries)-1);
}
fn u32 crc32(ref u8[] data){var u32 crc=4294967295;var usize i=0;while(i<data.len){crc^=data[i] as u32;var usize bit=0;while(bit<8){crc=if((crc&1)!=0){(crc>>1)^3988292384}else{crc>>1};bit+=1;}i+=1;}return crc^4294967295;}
// Private errors: 1 ZIP/path/type/CRC/identity, 2 allocation, 3 byte/file bound.
pub fn Result<Bundle,u32> unpack(ref u8[] data,ref u8[] uid,ref u8[] version,ref u8[] language,usize expanded_limit,usize file_limit){
    if(data.len<22 || data.len>16777216 || expanded_limit>67108864 || file_limit>4096){return .err(3);}
    var usize end=data.len-22;var bool found=false;let usize earliest=if(end>65535){end-65535}else{0};
    while(true){if(little(data,end,4)==101010256 && little(data,end+20,2)==data.len-end-22){found=true;break;}if(end==earliest){break;}end-=1;}
    if(!found || little(data,end+4,2)!=0 || little(data,end+6,2)!=0){return .err(1);}
    let usize entries=little(data,end+10,2);let usize size=little(data,end+12,4);let usize offset=little(data,end+16,4);
    if(entries==0 || entries>8192 || little(data,end+8,2)!=entries || size>end || offset>end-size){return .err(1);}
    let usize base=end-size-offset;var usize at=base+offset;var usize index=0;var v::Vec<Entry> nodes=v::empty<Entry>();var usize total_expanded=0;var usize total_files=0;
    while(index<entries){
        if(at>end || end-at<46 || little(data,at,4)!=33639248){return .err(1);}let usize flags=little(data,at+8,2);let usize method=little(data,at+10,2);let usize crc=little(data,at+16,4);let usize packed=little(data,at+20,4);let usize raw=little(data,at+24,4);
        let usize length=little(data,at+28,2);let usize extra=little(data,at+30,2);let usize comment=little(data,at+32,2);let usize local=little(data,at+42,4);
        if(length+extra+comment>end-at-46 || length==0 || (flags&1)!=0 || little(data,at+34,2)!=0 || (method!=0 && method!=8)){return .err(1);}
        let ref u8[] full=data[at+46..at+46+length];let bool dir=full[full.len-1]==47;let ref u8[] path=if(dir){full[..full.len-1]}else{full};
        if(!portable(path)){return .err(1);}let usize creator=little(data,at+5,1);let usize attributes=little(data,at+38,4);let usize kind=(attributes>>16)&61440;
        if((creator==3 || creator==19) && kind!=0 && kind!=32768 && kind!=16384){return .err(1);}let bool attr_dir=if(creator==3 || creator==19){kind==16384}else{(attributes&16)!=0};if(attr_dir && !dir){return .err(1);}
        var usize part=0;while(part<path.len){if(path[part]==47){switch(add(ref mut nodes,path[..part],true,false,bytes::empty())){case .ok(_):{}case .err(e):{return .err(e);}};}part+=1;}
        var bytes::Buffer decoded=bytes::empty();
        if(!dir){
            if(total_files>=file_limit || raw>8388608 || raw>expanded_limit-total_expanded || local>data.len-base){return .err(3);}let usize header=base+local;
            if(header>at || at-header<30 || little(data,header,4)!=67324752){return .err(1);}let usize body=header+30+little(data,header+26,2)+little(data,header+28,2);
            if(body>at || packed>at-body){return .err(1);}let ref u8[] compressed=data[body..body+packed];
            if(method==0){if(packed!=raw){return .err(1);}if(!bytes::append(ref mut decoded,compressed).is_ok){return .err(2);}}
            else{decoded=switch(inflate::decode(compressed,raw)){case .ok(value):{value}case .err(e):{return .err(e);}};}
            if((crc32(bytes::view(ref decoded)) as usize)!=crc){return .err(1);}total_files+=1;total_expanded+=raw;
        }
        switch(add(ref mut nodes,path,dir,true,decoded)){case .ok(_):{}case .err(e):{return .err(e);}};at+=46+length+extra+comment;index+=1;
    }
    if(at!=end){return .err(1);}let Bundle bundle=Bundle{.entries=nodes,.expanded=total_expanded,.files=total_files};var usize manifest_index=count(ref bundle);index=0;while(index<count(ref bundle)){if(bytes::equal(name(ref bundle,index),b"fairy.toml") && !directory(ref bundle,index)){manifest_index=index;}index+=1;}
    if(manifest_index==count(ref bundle)){return .err(1);}let root=switch(manifest::parse(contents(ref bundle,manifest_index))){case .ok(value):{value}case .err(_):{return .err(1);}};
    if(manifest::workspace_present(ref root) || !bytes::equal(manifest::package_value(ref root,b"uid"),uid) || !bytes::equal(manifest::package_value(ref root,b"version"),version) || !bytes::equal(manifest::package_value(ref root,b"language"),language)){return .err(1);}
    let ref u8[] source=manifest::package_value(ref root,b"source");index=0;while(index<count(ref bundle)){let ref u8[] path=name(ref bundle,index);if(!directory(ref bundle,index) && path.len>source.len+4 && bytes::equal(path[..source.len],source) && path[source.len]==47 && bytes::equal(path[path.len-3..],b".fy")){return .ok(bundle);}index+=1;}
    return .err(1);
}
