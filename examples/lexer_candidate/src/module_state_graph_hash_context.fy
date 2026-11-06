module module_state_graph_hash_context;
import project_cache_encoding as budget;
import std::bytes as bytes;

// A validated portable descriptor is identity data, never source authority.
pub resource Context{bytes::Buffer coordinate;bytes::Buffer descriptor;u32 effects;cleanup{}}
pub fn ref u8[] coordinate(ref Context p){return bytes::view(ref p.coordinate);}
pub fn ref u8[] descriptor(ref Context p){return bytes::view(ref p.descriptor);}
pub fn u32 effects(ref Context p){return p.effects;}
fn bool first(u8 c){return c==95 || c>=65 && c<=90 || c>=97 && c<=122;}
fn bool logical(ref u8[] name){var usize i=0;var usize segments=0;while(i<name.len){if(!first(name[i])){return false;}i+=1;while(i<name.len && (first(name[i]) || name[i]>=48 && name[i]<=57)){i+=1;}segments+=1;if(i==name.len){return segments>=2;}if(name.len-i<3 || name[i]!=58 || name[i+1]!=58){return false;}i+=2;}return false;}
fn bool begins(ref u8[] value,ref u8[] prefix){return value.len>=prefix.len && bytes::equal(value[..prefix.len],prefix);}
fn bool atom(ref u8[] value,ref u8[] choices){var usize start=0;var usize i=0;while(i<choices.len){if(choices[i]==124){if(bytes::equal(value,choices[start..i])){return true;}start=i+1;}i+=1;}return false;}
fn bool scalar(ref u8[] value){return atom(value,b"void|bool|u8|i32|u32|usize|str|CStr|Region|[bool]|[u8]|[i32]|[i64]|[u32]|[usize]|c::bool|c::char|c::double|c::float|c::int|c::long|c::longlong|c::schar|c::short|c::size_t|c::ssize_t|c::uchar|c::uint|c::ulong|c::ulonglong|c::ushort|");}
// Exactly Go's stable type grammar, with its 256-level recursion bound.
fn u32 type_check(ref u8[] value,usize depth,ref mut usize work){
    if(value.len>=16777216 || !budget::charge(work,value.len+1)){return 1002;}if(depth>256 || value.len==0){return 1001;}
    if(begins(value,b"ref mut ")){return type_check(value[8..],depth+1,work);}if(begins(value,b"ref ")){return type_check(value[4..],depth+1,work);}
    if(begins(value,b"ptr mut ")){return type_check(value[8..],depth+1,work);}if(begins(value,b"ptr ")){return type_check(value[4..],depth+1,work);}if(scalar(value)){return 0;}
    if(value.len>=2 && bytes::equal(value[value.len-2..],b"[]")){let element=value[..value.len-2];if(!begins(element,b"pkg[") || element[element.len-1]==93){return 1001;}return type_check(element,depth+1,work);}
    if(begins(value,b"Option<") && value[value.len-1]==62){return type_check(value[7..value.len-1],depth+1,work);}
    if(begins(value,b"Result<") && value[value.len-1]==62){var usize nesting=0;var usize i=7;while(i<value.len-1){let c=value[i];if(c==60){nesting+=1;}else if(c==62){if(nesting==0){return 1001;}nesting-=1;}else if(c==44 && nesting==0){if(i+2>=value.len || value[i+1]!=32){return 1001;}let code=type_check(value[7..i],depth+1,work);return if(code!=0){code}else{type_check(value[i+2..value.len-1],depth+1,work)};}i+=1;}return 1001;}
    if(value[value.len-1]==93){var usize at=value.len-1;while(at>0 && value[at]!=91){at-=1;}if(at==0 || at+1==value.len-1 || value[at+1]==48){return 1001;}var usize i=at+1;var usize n=0;while(i<value.len-1){let c=value[i];if(c<48 || c>57){return 1001;}let digit=(c-48) as usize;if(n>1844674407370955161 || n==1844674407370955161 && digit>5){return 1001;}n=n*10+digit;i+=1;}return type_check(value[..at],depth+1,work);}
    if(value.len<85 || !begins(value,b"pkg[fypc1-sha256-") || !bytes::equal(value[81..84],b"]::")){return 1001;}var usize i=17;while(i<81){let c=value[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return 1001;}i+=1;}return if(logical(value[84..])){0}else{1001};
}
struct Reader{usize offset;u32 code;}
fn usize number(ref u8[] raw,ref mut Reader c,ref mut usize work){if(c.code!=0){return 0;}if(!budget::charge(work,9)){c.code=1002;return 0;}if(c.offset>raw.len || raw.len-c.offset<8){c.code=1001;return 0;}var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[c.offset+i] as usize);i+=1;}c.offset+=8;return n;}
fn ref u8[] text(ref u8[] raw,ref mut Reader c,ref mut usize work){let n=number(raw,c,work);if(c.code!=0){return raw[..0];}if(c.offset>raw.len || n>raw.len-c.offset){c.code=1001;return raw[..0];}if(!budget::charge(work,n+1)){c.code=1002;return raw[..0];}let start=c.offset;c.offset+=n;return raw[start..c.offset];}
pub fn Result<Context,u32> validate(ref u8[] coordinate,ref u8[] descriptor,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(coordinate.len!=32 || descriptor.len==0){return .err(1001);}if(descriptor.len>=16777216 || !budget::charge(work,33)){return .err(1002);}var bool present=false;var usize i=0;while(i<32){present=present || coordinate[i]!=0;i+=1;}if(!present){return .err(1001);}
    var Reader c=Reader{.offset=0,.code=0};let name=text(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}if(!logical(name)){return .err(1001);}let count=number(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}if(count>(descriptor.len-c.offset)/8){return .err(1001);}i=0;
    while(i<count){let value=text(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}let code=type_check(value,0,work);if(code!=0){return .err(code);}i+=1;}
    let result=text(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}let code=type_check(result,0,work);if(code!=0){return .err(code);}let effects=number(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}if(effects>2){return .err(1001);}var u32 mask=0;i=0;
    while(i<effects){let effect=text(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}let u32 bit=if(bytes::equal(effect,b"storage_read")){1}else if(bytes::equal(effect,b"storage_write")){2}else{0};if(bit==0 || bit<=mask){return .err(1001);}mask|=bit;i+=1;}
    let drop=text(descriptor,ref mut c,work);if(c.code!=0){return .err(c.code);}if(drop.len!=0){let code=type_check(drop,0,work);if(code!=0){return .err(code);}}if(c.offset!=descriptor.len){return .err(1001);}if(!budget::charge(work,descriptor.len+33)){return .err(1002);}
    let owned=switch(bytes::copy(descriptor)){case .ok(p):{p}case .err(_):{return .err(1003);}};let coord=switch(bytes::copy(coordinate)){case .ok(p):{p}case .err(_):{return .err(1003);}};return .ok(Context{.coordinate=coord,.descriptor=owned,.effects=mask});
}
