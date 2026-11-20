module project_atlas_options;
import project_manifest_values as manifest;
import project_atlas_wire as wire;
import std::bytes as bytes;
import std::process as process;

pub resource Options{bytes::Buffer uid;bytes::Buffer symbol;usize packages;usize calls;usize depth;usize symbols;usize output;cleanup{}}
pub fn ref u8[] uid(ref Options p){return bytes::view(ref p.uid);}
pub fn ref u8[] symbol(ref Options p){return bytes::view(ref p.symbol);}
pub fn usize packages(ref Options p){return p.packages;}
pub fn usize calls(ref Options p){return p.calls;}
pub fn usize depth(ref Options p){return p.depth;}
pub fn usize symbols(ref Options p){return p.symbols;}
pub fn usize output(ref Options p){return p.output;}
// Return the declaration ID's end. For a bare ID this is the selector length.
pub fn Result<usize,u32> selector_end(ref u8[] target){
 let full_prefix=b"fysym1-sha256-256-";
 let specialization_prefix=b"g0_";
 let usize digest_digits=64;
 if(target.len>full_prefix.len+digest_digits+1+specialization_prefix.len+digest_digits){return .err(2);}
 var usize end=0;
 while(end<target.len && target[end]!=35){end+=1;}
 let id=target[..end];
 if(!wire::hex(id,b"fysym1-sha256-128-",32,false) && !wire::hex(id,full_prefix,digest_digits,false)){return .err(2);}
 if(end<target.len && !wire::hex(target[end+1..],specialization_prefix,digest_digits,false)){return .err(2);}
 return .ok(end);
}
// Owned request selection for a host-supplied policy. No request can widen the
// invocation limits, and invalid selectors fail before workspace access.
pub fn Result<Options,u32> selected(ref u8[] package_uid,ref u8[] target,usize packages,usize calls,usize depth,usize symbols,usize output){if(!manifest::uid(package_uid) || !selector_end(target).is_ok || packages==0 || packages>65536 || calls==0 || calls>65536 || depth>64 || symbols==0 || symbols>4096 || output==0 || output>4194304){return .err(2);}var bytes::Buffer uid=bytes::empty();var bytes::Buffer symbol=bytes::empty();if(!bytes::append(ref mut uid,package_uid).is_ok || !bytes::append(ref mut symbol,target).is_ok){return .err(2);}return .ok(Options{.uid,.symbol,.packages,.calls,.depth,.symbols,.output});}
fn usize flag(ref u8[] s){if(bytes::equal(s,b"--package-uid")){return 0;}if(bytes::equal(s,b"--symbol")){return 1;}if(bytes::equal(s,b"--max-packages")){return 2;}if(bytes::equal(s,b"--max-calls")){return 3;}if(bytes::equal(s,b"--max-depth")){return 4;}if(bytes::equal(s,b"--max-symbols")){return 5;}if(bytes::equal(s,b"--max-output-bytes")){return 6;}if(bytes::equal(s,b"--locked")){return 7;}return 8;}
fn Result<usize,u32> number(ref u8[] s,usize maximum,bool zero){if(s.len==0 || s.len>1 && s[0]==48){return .err(2);}var usize n=0;var usize i=0;while(i<s.len){if(s[i]<48 || s[i]>57){return .err(2);}let digit=(s[i]-48) as usize;if(n>maximum/10 || n==maximum/10 && digit>maximum%10){return .err(2);}n=n*10+digit;i+=1;}if(n==0 && !zero){return .err(2);}return .ok(n);}
// Syntax is completely checked before opening any workspace or reading stdin.
pub fn Result<Options,u32> arguments(){var bytes::Buffer uid=bytes::empty();var bytes::Buffer symbol=bytes::empty();var usize[5] limits=[0;5];var bool[8] seen=[false;8];var usize at=3;let argc=process::arg_count();if(argc>18){return .err(2);}
 while(at<argc){var u8[4096] key=[0;4096];let n=switch(process::arg(at,ref mut key)){case .ok(p):{p}case .err(_):{return .err(2);}};let ref u8[] raw=ref key;let index=flag(raw[..n]);if(index==8 || seen[index]){return .err(2);}seen[index]=true;at+=1;if(index==7){continue;}if(at==argc){return .err(2);}var u8[4096] value=[0;4096];let size=switch(process::arg(at,ref mut value)){case .ok(p):{p}case .err(_):{return .err(2);}};let ref u8[] input=ref value;let s=input[..size];
 if(index==0){if(!manifest::uid(s) || !bytes::append(ref mut uid,s).is_ok){return .err(2);}}
 else if(index==1){if(!selector_end(s).is_ok || !bytes::append(ref mut symbol,s).is_ok){return .err(2);}}
 else{limits[index-2]=number(s,if(index==4){64}else if(index==5){4096}else if(index==6){4194304}else{65536},index==4)?;}at+=1;
 }
 var usize i=0;while(i<7){if(!seen[i]){return .err(2);}i+=1;}return .ok(Options{.uid,.symbol,.packages=limits[0],.calls=limits[1],.depth=limits[2],.symbols=limits[3],.output=limits[4]});
}
