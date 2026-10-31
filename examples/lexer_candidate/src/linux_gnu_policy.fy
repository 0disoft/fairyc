module linux_gnu_policy;
import std::bytes as bytes;

// Native GNU companion admission. Mirrors internal/cdriver/linux_toolchain.go;
// process launch and filesystem access belong to the existing Linux adapters.
pub struct Compiler { bool clang; usize major; }
fn bool space(u8 c){return c==9 || c==10 || c==11 || c==12 || c==13 || c==32;}
fn ref u8[] trim(ref u8[] value){var usize first=0;var usize end=value.len;while(first<end && space(value[first])){first+=1;}while(end>first && space(value[end-1])){end-=1;}return value[first..end];}
fn bool at(ref u8[] value,usize offset,ref u8[] text){return offset<=value.len && text.len<=value.len-offset && bytes::equal(value[offset..offset+text.len],text);}
fn Result<usize,u32> number(ref u8[] value,ref mut usize cursor){
    let first=*cursor;var usize n=0;
    while(*cursor<value.len && value[*cursor]>=48 && value[*cursor]<=57){
        let digit=(value[*cursor]-48) as usize;if(n>214748364 || n==214748364 && digit>7){return .err(1);}n=n*10+digit;*cursor+=1;
    }return if(*cursor==first){.err(1)}else{.ok(n)};
}
pub fn bool libc(ref u8[] output){
    let value=trim(output);if(!at(value,0,b"glibc ")){return false;}var usize cursor=6;
    let major=switch(number(value,ref mut cursor)){case .ok(n):{n}case .err(_):{return false;}};
    if(cursor==value.len || value[cursor]!=46){return false;}cursor+=1;
    let minor=switch(number(value,ref mut cursor)){case .ok(n):{n}case .err(_):{return false;}};
    return cursor==value.len && (major>2 || major==2 && minor>=39);
}
pub fn bool target(ref u8[] output){
    let value=trim(output);if(value.len<16 || !at(value,0,b"x86_64-") || !at(value,value.len-10,b"-linux-gnu")){return false;}
    var usize i=0;while(i<value.len){if(space(value[i]) || value[i]==0){return false;}i+=1;}return true;
}
pub fn Result<usize,u32> version(ref u8[] output){
    let value=trim(output);var usize cursor=0;let major=number(value,ref mut cursor)?;var usize components=1;
    while(cursor<value.len){if(value[cursor]!=46 || components==3){return .err(2);}cursor+=1;let usize ignored=number(value,ref mut cursor)?;components+=1;}
    return .ok(major);
}
fn bool mentions_clang(ref u8[] line){
    var usize i=0;while(i+5<=line.len){if((line[i]==99 || line[i]==67) && (line[i+1]==108 || line[i+1]==76) && (line[i+2]==97 || line[i+2]==65) && (line[i+3]==110 || line[i+3]==78) && (line[i+4]==103 || line[i+4]==71)){return true;}i+=1;}return false;
}
// 1 libc floor/identity, 2 compiler floor/identity, 3 non-native target.
pub fn Result<Compiler,u32> admit(bool clang,ref u8[] libc_output,ref u8[] banner,ref u8[] gcc_numeric,ref u8[] triple){
    if(!libc(libc_output)){return .err(1);}let all=trim(banner);var usize end=0;while(end<all.len && all[end]!=10){end+=1;}let line=trim(all[..end]);
    if(line.len==0){return .err(2);}var usize major=0;
    if(clang){var usize i=0;var bool found=false;while(i<line.len){if((i==0 || !(line[i-1]>=48 && line[i-1]<=57 || line[i-1]>=65 && line[i-1]<=90 || line[i-1]>=97 && line[i-1]<=122 || line[i-1]==95)) && at(line,i,b"clang version ")){
        let first=i+14;var usize stop=first;while(stop<line.len && !space(line[stop])){stop+=1;}major=switch(version(line[first..stop])){case .ok(n):{n}case .err(_):{return .err(2);}};found=true;break;}i+=1;}if(!found || major<18){return .err(2);}
    }else{if(mentions_clang(line)){return .err(2);}major=switch(version(gcc_numeric)){case .ok(n):{n}case .err(_):{return .err(2);}};if(major<13){return .err(2);}}
    if(!target(triple)){return .err(3);}return .ok(Compiler{.clang,.major});
}
pub fn bool timeout(u32 milliseconds){return milliseconds>0 && milliseconds<=600000;}
pub fn ref u8[] optimization(bool release){if(release){return b"-O2";}return b"-O0";}
