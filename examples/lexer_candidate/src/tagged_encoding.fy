module tagged_encoding;

// Canonical source-independent nested identities. Thirty payload bits encode
// prefix nodes: 01 + three scalar bits, 10 Option, 11 Result. Tag 00 is closed.
// Scalar Option/Result keep their earlier IDs; the marker is nested-only.
fn u32 size(u32 raw,usize depth) {
    if(depth>=16){return 0;}let u32 kind=raw&3;
    if(kind==1){return if((raw>>2&7)<=5){5}else{0};}
    if(kind==2){let u32 n=size(raw>>2,depth+1);return if(n!=0 && n<=28){n+2}else{0};}
    if(kind==3){let u32 left=size(raw>>2,depth+1);if(left==0 || left>23){return 0;}
        let u32 right=size(raw>>(left+2),depth+1);return if(right!=0 && left+right<=28){left+right+2}else{0};}
    return 0;
}
pub fn bool valid(u32 code) {
    if(code<1073741824 || code>2147483647){return false;}let u32 raw=code-1073741824;let u32 n=size(raw,0);
    if(n==0 || raw>>n!=0){return false;}
    if(raw&3==2){return raw>>2&3!=1;}
    if(raw&3==3){let u32 left=size(raw>>2,1);return raw>>2&3!=1 || raw>>(left+2)&3!=1;}
    return false;
}
pub fn bool option(u32 code){return code>=196609 && code<=196614 || valid(code) && code&3==2;}
pub fn bool result(u32 code){return code>=262145 && code<=262180 || valid(code) && code&3==3;}
fn u32 bits(u32 code) {
    if(code>=1 && code<=6){return (code-1)<<2|1;}
    if(code>=196609 && code<=196614){return 2|bits(code-196608)<<2;}
    if(code>=262145 && code<=262180){return 3|bits((code-262145)/6+1)<<2|bits((code-262145)%6+1)<<7;}
    return if(valid(code)){code-1073741824}else{0};
}
fn u32 decoded(u32 raw) {
    let u32 n=size(raw,0);assert(n!=0);let u32 cut=raw&((1<<n)-1);let u32 kind=cut&3;
    if(kind==1){return (cut>>2&7)+1;}
    if(kind==2){let u32 child=decoded(cut>>2);return if(child<=6){196608+child}else{1073741824|cut};}
    let u32 left_size=size(cut>>2,1);let u32 left=decoded(cut>>2);let u32 right=decoded(cut>>(left_size+2));
    return if(left<=6 && right<=6){262144+(left-1)*6+right}else{1073741824|cut};
}
pub fn u32 make_option(u32 child) {
    if(child>=1 && child<=6){return 196608+child;}let u32 raw=bits(child);let u32 n=size(raw,0);
    return if(raw!=0 && n!=0 && n<=28){1073741824|2|raw<<2}else{0};
}
pub fn u32 make_result(u32 left,u32 right) {
    if(left>=1 && left<=6 && right>=1 && right<=6){return 262144+(left-1)*6+right;}
    let u32 a=bits(left);let u32 b=bits(right);let u32 na=size(a,0);let u32 nb=size(b,0);
    return if(a!=0 && b!=0 && na!=0 && nb!=0 && na+nb<=28){1073741824|3|a<<2|b<<(na+2)}else{0};
}
pub fn u32 inner(u32 code){assert(option(code));return if(code<=196614){code-196608}else{decoded((code-1073741824)>>2)};}
pub fn u32 success(u32 code){assert(result(code));return if(code<=262180){(code-262145)/6+1}else{decoded((code-1073741824)>>2)};}
pub fn u32 failure(u32 code){assert(result(code));let u32 raw=bits(code);let u32 n=size(raw>>2,1);return decoded(raw>>(n+2));}
