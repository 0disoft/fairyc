module module_unit_interpreter_math;
import module_constant_arithmetic as constants;
import std::bytes as bytes;

// Same exact-width bits as the constant evaluator. Runtime failures retain
// language trap categories; usize arithmetic never converts through signed i64.
pub struct Value{usize bits;u32 scalar;u32 trap;bool supported;}
fn Value value(u32 scalar,usize bits){return Value{.bits,.scalar,.trap=0,.supported=true};}
fn Value trap(u32 category){return Value{.bits=0,.scalar=0,.trap=category,.supported=true};}
pub fn Value unsupported(){return Value{.bits=0,.scalar=0,.trap=0,.supported=false};}
fn bool negative(u32 scalar,usize bits){return scalar==1 && bits>2147483647 || scalar==2 && bits>9223372036854775807;}
pub fn Value prefix(u32 scalar,ref u8[] op,usize bits){
    if(scalar==5){return if(bytes::equal(op,b"~")){value(5,bits^18446744073709551615)}else{unsupported()};}
    let r=constants::prefix(scalar,op,bits);return if(r.code==0){value(r.scalar,r.bits)}else if(r.code==4303){trap(2)}else{unsupported()};
}
pub fn Value binary(u32 scalar,ref u8[] op,usize left,usize right){
    if(bytes::equal(op,b"/") || bytes::equal(op,b"%")){
        if(right==0){return trap(3);}if((scalar==1 && left==2147483648 || scalar==2 && left==9223372036854775808) && right==if(scalar==1){4294967295}else{18446744073709551615}){return trap(4);}
    }
    if(bytes::equal(op,b"<<") || bytes::equal(op,b">>")){
        let usize width=if(scalar==3){8}else if(scalar==1 || scalar==4){32}else{64};if(negative(scalar,right) || right>=width){return trap(5);}
    }
    if(scalar==5){
        let usize maximum=18446744073709551615;
        if(bytes::equal(op,b"+")){return if(left>maximum-right){trap(2)}else{value(5,left+right)};}
        if(bytes::equal(op,b"-")){return if(left<right){trap(2)}else{value(5,left-right)};}
        if(bytes::equal(op,b"*")){return if(right!=0 && left>maximum/right){trap(2)}else{value(5,left*right)};}
        if(bytes::equal(op,b"/")){return value(5,left/right);}if(bytes::equal(op,b"%")){return value(5,left%right);}
        if(bytes::equal(op,b"<<")){return if(left>(maximum>>right)){trap(2)}else{value(5,left<<right)};}
        if(bytes::equal(op,b">>")){return value(5,left>>right);}
        if(bytes::equal(op,b"&")){return value(5,left&right);}if(bytes::equal(op,b"|")){return value(5,left|right);}if(bytes::equal(op,b"^")){return value(5,left^right);}
        let yes=bytes::equal(op,b"==") && left==right || bytes::equal(op,b"!=") && left!=right || bytes::equal(op,b"<") && left<right || bytes::equal(op,b"<=") && left<=right || bytes::equal(op,b">") && left>right || bytes::equal(op,b">=") && left>=right;
        if(bytes::equal(op,b"==") || bytes::equal(op,b"!=") || bytes::equal(op,b"<") || bytes::equal(op,b"<=") || bytes::equal(op,b">") || bytes::equal(op,b">=")){return value(6,if(yes){1}else{0});}return unsupported();
    }
    let r=constants::binary(scalar,op,left,right);return if(r.code==0){value(r.scalar,r.bits)}else if(r.code==4303){trap(2)}else{unsupported()};
}
pub fn Value cast(u32 source,u32 target,usize bits){
    if(target==5 && (source==3 || source==4)){return value(5,bits);}
    let r=constants::cast(source,target,bits);return if(r.code==0){value(r.scalar,r.bits)}else{unsupported()};
}
