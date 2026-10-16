module module_constant_arithmetic;
import std::bytes as bytes;

// Values use exact declared-width two's-complement bits on the candidate's
// AMD64 hosts. Every potentially trapping host operation is guarded first.
pub struct Computed{u32 code;u32 scalar;usize bits;}
fn Computed result(u32 scalar,usize bits){return Computed{.code=0,.scalar,.bits};}
fn Computed fail(u32 code){return Computed{.code,.scalar=0,.bits=0};}
fn bool signed(u32 scalar){return scalar==1 || scalar==2;}
fn usize mask(u32 scalar){return if(scalar==3){255}else if(scalar==2){18446744073709551615}else{4294967295};}
fn bool valid(u32 scalar,usize bits){return (scalar==1 || scalar==2 || scalar==3 || scalar==4) && bits<=mask(scalar) || scalar==6 && bits<=1;}
fn i64 positive(usize bits){var usize remaining=bits;var i64 value=0;var i64 factor=1;while(remaining!=0){if((remaining&1)!=0){value+=factor;}remaining>>=1;if(remaining!=0){factor*=2;}}return value;}
fn i64 number(u32 scalar,usize bits){
    if(signed(scalar) && bits>if(scalar==1){2147483647}else{9223372036854775807}){return -1-positive(mask(scalar)-bits);}
    return positive(bits);
}
fn usize natural(i64 n){var i64 remaining=n;var usize value=0;var usize factor=1;while(remaining!=0){if(remaining%2!=0){value+=factor;}remaining/=2;if(remaining!=0){factor*=2;}}return value;}
fn usize magnitude(i64 n){return if(n<0){natural(-(n+1))+1}else{natural(n)};}
fn usize encoded(u32 scalar,bool negative,usize magnitude){return if(!negative || magnitude==0){magnitude}else{mask(scalar)-(magnitude-1)};}
fn usize bits(u32 scalar,i64 n){return encoded(scalar,n<0,magnitude(n));}
pub fn Computed prefix(u32 scalar,ref u8[] op,usize value){
    if(!valid(scalar,value)){return fail(12);}
    if(bytes::equal(op,b"!")){return if(scalar==6){result(6,1-value)}else{fail(1007)};}
    if(bytes::equal(op,b"~")){return if(scalar!=6){result(scalar,value^mask(scalar))}else{fail(1006)};}
    if(bytes::equal(op,b"+")){return if(signed(scalar)){result(scalar,value)}else{fail(1006)};}
    if(bytes::equal(op,b"-")){
        if(!signed(scalar)){return fail(1006);}let n=number(scalar,value);
        let i64 minimum=if(scalar==1){-2147483648}else{-9223372036854775808};if(n==minimum){return fail(4303);}
        return result(scalar,bits(scalar,-n));
    }return fail(4302);
}
pub fn Computed binary(u32 scalar,ref u8[] op,usize left,usize right){
    if(!valid(scalar,left) || !valid(scalar,right)){return fail(12);}
    if(bytes::equal(op,b"&&") || bytes::equal(op,b"||")){
        if(scalar!=6){return fail(1007);}return result(6,if(bytes::equal(op,b"&&")){if(left==1 && right==1){1}else{0}}else{if(left==1 || right==1){1}else{0}});
    }
    let equality=bytes::equal(op,b"==") || bytes::equal(op,b"!=");let comparison=equality || bytes::equal(op,b"<") || bytes::equal(op,b"<=") || bytes::equal(op,b">") || bytes::equal(op,b">=");
    if(comparison){
        if(scalar==6 && !equality){return fail(1006);}let l=number(scalar,left);let r=number(scalar,right);
        let yes=bytes::equal(op,b"==") && l==r || bytes::equal(op,b"!=") && l!=r || bytes::equal(op,b"<") && l<r || bytes::equal(op,b"<=") && l<=r || bytes::equal(op,b">") && l>r || bytes::equal(op,b">=") && l>=r;
        return result(6,if(yes){1}else{0});
    }
    if(scalar==6){return fail(1006);}
    if(bytes::equal(op,b"&")){return result(scalar,left&right);}
    if(bytes::equal(op,b"|")){return result(scalar,left|right);}
    if(bytes::equal(op,b"^")){return result(scalar,left^right);}
    let l=number(scalar,left);let r=number(scalar,right);
    if(bytes::equal(op,b"<<") || bytes::equal(op,b">>")){
        let i64 width=if(scalar==3){8}else if(scalar==2){64}else{32};
        if(r<0 || r>=width){return fail(4303);}
        if(bytes::equal(op,b">>")){return if(signed(scalar)){result(scalar,bits(scalar,l>>r))}else{result(scalar,left>>natural(r))};}
        let usize factor=1<<natural(r);let usize size=magnitude(l);let usize limit=if(signed(scalar)){if(l<0){if(scalar==1){2147483648}else{9223372036854775808}}else{if(scalar==1){2147483647}else{9223372036854775807}}}else{mask(scalar)};
        if(size>limit/factor){return fail(4303);}return result(scalar,encoded(scalar,l<0,size*factor));
    }
    if(bytes::equal(op,b"/") || bytes::equal(op,b"%")){
        let i64 minimum=if(scalar==1){-2147483648}else{-9223372036854775808};if(r==0 || signed(scalar) && l==minimum && r== -1){return fail(4303);}
        let n=if(bytes::equal(op,b"/")){l/r}else{l%r};return result(scalar,bits(scalar,n));
    }
    if(!signed(scalar)){
        let maximum=mask(scalar);
        if(bytes::equal(op,b"+")){return if(left>maximum-right){fail(4303)}else{result(scalar,left+right)};}
        if(bytes::equal(op,b"-")){return if(left<right){fail(4303)}else{result(scalar,left-right)};}
        if(bytes::equal(op,b"*")){return if(right!=0 && left>maximum/right){fail(4303)}else{result(scalar,left*right)};}
        return fail(4302);
    }
    let i64 maximum=if(scalar==1){2147483647}else{9223372036854775807};let i64 minimum=if(scalar==1){-2147483648}else{-9223372036854775808};
    if(bytes::equal(op,b"+")){if(r>0 && l>maximum-r || r<0 && l<minimum-r){return fail(4303);}return result(scalar,bits(scalar,l+r));}
    if(bytes::equal(op,b"-")){if(r<0 && l>maximum+r || r>0 && l<minimum+r){return fail(4303);}return result(scalar,bits(scalar,l-r));}
    if(bytes::equal(op,b"*")){
        let negative=(l<0)!=(r<0);let a=magnitude(l);let b=magnitude(r);let usize limit=if(negative){if(scalar==1){2147483648}else{9223372036854775808}}else{natural(maximum)};
        if(b!=0 && a>limit/b){return fail(4303);}return result(scalar,encoded(scalar,negative,a*b));
    }return fail(4302);
}

// The current language only admits these value-preserving scalar widenings.
pub fn Computed cast(u32 source,u32 target,usize value){
    if(!valid(source,value)){return fail(12);}
    if(source==3 && (target==1 || target==2 || target==4) || source==4 && target==2){return result(target,value);}
    if(source==1 && target==2){return result(target,if(value<=2147483647){value}else{18446744073709551615-(4294967295-value)});}
    return fail(4303);
}

// Same exact-width bits as the constant evaluator. Runtime failures retain
// language trap categories; usize arithmetic never converts through signed i64.
pub struct RuntimeValue{usize bits;u32 scalar;u32 trap;bool supported;}
fn RuntimeValue runtime_value(u32 scalar,usize bits){return RuntimeValue{.bits,.scalar,.trap=0,.supported=true};}
fn RuntimeValue runtime_trap(u32 category){return RuntimeValue{.bits=0,.scalar=0,.trap=category,.supported=true};}
pub fn RuntimeValue runtime_unsupported(){return RuntimeValue{.bits=0,.scalar=0,.trap=0,.supported=false};}
fn bool runtime_negative(u32 scalar,usize bits){return scalar==1 && bits>2147483647 || scalar==2 && bits>9223372036854775807;}
pub fn RuntimeValue runtime_prefix(u32 scalar,ref u8[] op,usize bits){
    if(scalar==5){return if(bytes::equal(op,b"~")){runtime_value(5,bits^18446744073709551615)}else{runtime_unsupported()};}
    let r=prefix(scalar,op,bits);return if(r.code==0){runtime_value(r.scalar,r.bits)}else if(r.code==4303){runtime_trap(2)}else{runtime_unsupported()};
}
pub fn RuntimeValue runtime_binary(u32 scalar,ref u8[] op,usize left,usize right){
    if(bytes::equal(op,b"/") || bytes::equal(op,b"%")){
        if(right==0){return runtime_trap(3);}if((scalar==1 && left==2147483648 || scalar==2 && left==9223372036854775808) && right==if(scalar==1){4294967295}else{18446744073709551615}){return runtime_trap(4);}
    }
    if(bytes::equal(op,b"<<") || bytes::equal(op,b">>")){
        let usize width=if(scalar==3){8}else if(scalar==1 || scalar==4){32}else{64};if(runtime_negative(scalar,right) || right>=width){return runtime_trap(5);}
    }
    if(scalar==5){
        let usize maximum=18446744073709551615;
        if(bytes::equal(op,b"+")){return if(left>maximum-right){runtime_trap(2)}else{runtime_value(5,left+right)};}
        if(bytes::equal(op,b"-")){return if(left<right){runtime_trap(2)}else{runtime_value(5,left-right)};}
        if(bytes::equal(op,b"*")){return if(right!=0 && left>maximum/right){runtime_trap(2)}else{runtime_value(5,left*right)};}
        if(bytes::equal(op,b"/")){return runtime_value(5,left/right);}if(bytes::equal(op,b"%")){return runtime_value(5,left%right);}
        if(bytes::equal(op,b"<<")){return if(left>(maximum>>right)){runtime_trap(2)}else{runtime_value(5,left<<right)};}
        if(bytes::equal(op,b">>")){return runtime_value(5,left>>right);}
        if(bytes::equal(op,b"&")){return runtime_value(5,left&right);}if(bytes::equal(op,b"|")){return runtime_value(5,left|right);}if(bytes::equal(op,b"^")){return runtime_value(5,left^right);}
        let yes=bytes::equal(op,b"==") && left==right || bytes::equal(op,b"!=") && left!=right || bytes::equal(op,b"<") && left<right || bytes::equal(op,b"<=") && left<=right || bytes::equal(op,b">") && left>right || bytes::equal(op,b">=") && left>=right;
        if(bytes::equal(op,b"==") || bytes::equal(op,b"!=") || bytes::equal(op,b"<") || bytes::equal(op,b"<=") || bytes::equal(op,b">") || bytes::equal(op,b">=")){return runtime_value(6,if(yes){1}else{0});}return runtime_unsupported();
    }
    let r=binary(scalar,op,left,right);return if(r.code==0){runtime_value(r.scalar,r.bits)}else if(r.code==4303){runtime_trap(2)}else{runtime_unsupported()};
}
pub fn RuntimeValue runtime_cast(u32 source,u32 target,usize bits){
    if(target==5 && (source==3 || source==4)){return runtime_value(5,bits);}
    let r=cast(source,target,bits);return if(r.code==0){runtime_value(r.scalar,r.bits)}else{runtime_unsupported()};
}
