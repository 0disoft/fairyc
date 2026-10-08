module tagged_encoding;
import reference_types as refs;

// Canonical source-independent nested identities. Thirty payload bits encode
// prefix nodes: 01 + three leaf bits, 10 Option, 11 Result. Leaf 6 escapes
// to a nine-bit struct identity; leaf 7 escapes to a nine-bit resource identity.
// Tag 00 retains views/references and reserves the closed seven-bit zero prefix
// plus nine declaration bits for a sixteen-bit enum escape (512 identities).
// Existing scalar, nominal, resource and borrow encodings stay unchanged.
// Scalar Option/Result keep their earlier IDs; the marker is nested-only.
fn u32 size(u32 raw,usize depth) {
    if(depth>=16){return 0;}let u32 kind=raw&3;
    if(kind==0){let u32 category=raw>>2&3;return if(category==0){if(raw&127==0){16}else if(raw&127==64){7}else if(raw&63==16 || raw&63==32 || raw&63==48){6}else{0}}else if(category==1 && (raw>>5&511)>5){0}else{14};}
    if(kind==1){return if((raw>>2&7)<=5){5}else{14};}
    if(kind==2){let u32 n=size(raw>>2,depth+1);return if(n!=0 && n<=28){n+2}else{0};}
    if(kind==3){let u32 left=size(raw>>2,depth+1);if(left==0 || left>23){return 0;}
        let u32 right=size(raw>>(left+2),depth+1);return if(right!=0 && left+right<=28){left+right+2}else{0};}
    return 0;
}
pub fn bool valid(u32 code) {
    if(code<1073741824 || code>2147483647){return false;}let u32 raw=code-1073741824;let u32 n=size(raw,0);
    if(n==0 || raw>>n!=0){return false;}
    if(raw&3==2){return size(raw>>2,1)!=5;}
    if(raw&3==3){let u32 left=size(raw>>2,1);return left!=5 || size(raw>>(left+2),1)!=5;}
    return false;
}
pub fn bool option(u32 code){return code>=196609 && code<=196614 || valid(code) && code&3==2;}
pub fn bool result(u32 code){return code>=262145 && code<=262180 || valid(code) && code&3==3;}
fn u32 bits(u32 code) {
    if(code==12){return 64;}if(code==9){return 16;}if(code==10){return 32;}if(code==11){return 48;}
    if(refs::valid(code)){let u32 t=refs::inner(code);if(t>98815){return 0;}let u32 category=if(t<=6){1}else if(t<98304){2}else{3};let u32 index=t-if(category==1){1}else if(category==2){256}else{98304};return category<<2|if(refs::mutable(code)){16}else{0}|index<<5;}
    if(code>=1 && code<=6){return (code-1)<<2|1;}
    if(code>=256 && code<=767){return 25|(code-256)<<5;}
    if(code>=98304 && code<=98815){return 29|(code-98304)<<5;}
    if(code>=131072 && code<=131583){return (code-131072)<<7;}
    if(code>=196609 && code<=196614){return 2|bits(code-196608)<<2;}
    if(code>=262145 && code<=262180){return 3|bits((code-262145)/6+1)<<2|bits((code-262145)%6+1)<<7;}
    return if(valid(code)){code-1073741824}else{0};
}
fn u32 decoded(u32 raw) {
    let u32 n=size(raw,0);assert(n!=0);let u32 cut=raw&((1<<n)-1);let u32 kind=cut&3;
    if(kind==0){let u32 category=cut>>2&3;if(category==0){return if(cut&127==0){131072+(cut>>7&511)}else if(cut==16){9}else if(cut==32){10}else if(cut==64){12}else{11};}return refs::make((cut>>5&511)+if(category==1){1}else if(category==2){256}else{98304},cut&16!=0);}
    if(kind==1){return if((cut>>2&7)==7){98304+(cut>>5&511)}else if((cut>>2&7)==6){256+(cut>>5&511)}else{(cut>>2&7)+1};}
    if(kind==2){let u32 child=decoded(cut>>2);return if(child<=6){196608+child}else{1073741824|cut};}
    let u32 left_size=size(cut>>2,1);let u32 left=decoded(cut>>2);let u32 right=decoded(cut>>(left_size+2));
    return if(left<=6 && right<=6){262144+(left-1)*6+right}else{1073741824|cut};
}
pub fn u32 make_option(u32 child) {
    if(child>=1 && child<=6){return 196608+child;}let u32 raw=bits(child);let u32 n=size(raw,0);
    return if((raw!=0 || child==131072) && n!=0 && n<=28){1073741824|2|raw<<2}else{0};
}
pub fn u32 make_result(u32 left,u32 right) {
    if(left>=1 && left<=6 && right>=1 && right<=6){return 262144+(left-1)*6+right;}
    let u32 a=bits(left);let u32 b=bits(right);let u32 na=size(a,0);let u32 nb=size(b,0);
    return if((a!=0 || left==131072) && (b!=0 || right==131072) && na!=0 && nb!=0 && na+nb<=28){1073741824|3|a<<2|b<<(na+2)}else{0};
}
pub fn u32 inner(u32 code){assert(option(code));return if(code<=196614){code-196608}else{decoded((code-1073741824)>>2)};}
pub fn u32 success(u32 code){assert(result(code));return if(code<=262180){(code-262145)/6+1}else{decoded((code-1073741824)>>2)};}
pub fn u32 failure(u32 code){assert(result(code));let u32 raw=bits(code);let u32 n=size(raw>>2,1);return decoded(raw>>(n+2));}
