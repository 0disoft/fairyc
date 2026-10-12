module tests::module_constant_arithmetic;
import module_constant_arithmetic as arithmetic;

fn void value(u32 scalar,ref u8[] op,usize left,usize right,u32 result_scalar,usize bits){let p=arithmetic::binary(scalar,op,left,right);assert(p.code==0 && p.scalar==result_scalar && p.bits==bits);}
fn void trap(u32 scalar,ref u8[] op,usize left,usize right){assert(arithmetic::binary(scalar,op,left,right).code==4303);}
@test
fn void declared_width_integer_limits_trap_before_host_arithmetic(){
    value(1,b"+",2147483647,0,1,2147483647);trap(1,b"+",2147483647,1);trap(1,b"-",2147483648,1);
    value(1,b"*",2147483648,1,1,2147483648);trap(1,b"*",2147483648,4294967295);trap(1,b"/",2147483648,4294967295);trap(1,b"%",2147483648,4294967295);
    value(3,b"*",15,17,3,255);trap(3,b"+",255,1);trap(3,b"-",0,1);trap(3,b"*",16,16);
    value(4,b"*",65535,65537,4,4294967295);trap(4,b"*",65536,65536);trap(4,b"+",4294967295,1);trap(4,b"/",42,0);trap(4,b"%",42,0);
    value(2,b"+",9223372036854775807,0,2,9223372036854775807);trap(2,b"+",9223372036854775807,1);trap(2,b"-",9223372036854775808,1);
    value(2,b"-",9223372036854775808,18446744073709551615,2,9223372036854775809);
    value(2,b"*",9223372036854775808,1,2,9223372036854775808);trap(2,b"*",9223372036854775808,18446744073709551615);
    trap(2,b"/",9223372036854775808,18446744073709551615);trap(2,b"%",9223372036854775808,18446744073709551615);
    value(2,b"*",18446744073709551613,7,2,18446744073709551595);value(2,b"*",18446744073709551613,18446744073709551609,2,21);
}
@test
fn void shifts_prefixes_and_comparisons_preserve_signed_bits(){
    value(1,b">>",2147483648,31,1,4294967295);value(2,b">>",9223372036854775808,63,2,18446744073709551615);
    value(1,b"<<",4294967295,31,1,2147483648);value(2,b"<<",18446744073709551615,63,2,9223372036854775808);
    trap(1,b"<<",1,31);trap(2,b"<<",1,63);trap(3,b"<<",2,7);trap(4,b"<<",4294967295,1);trap(2,b">>",1,64);trap(1,b">>",1,4294967295);
    value(1,b"<",4294967295,0,6,1);value(2,b"<",9223372036854775808,0,6,1);value(4,b">",4294967295,0,6,1);
    value(6,b"==",1,0,6,0);value(6,b"&&",1,1,6,1);value(6,b"||",0,1,6,1);
    value(2,b"^",9223372036854775808,18446744073709551615,2,9223372036854775807);
    assert(arithmetic::prefix(1,b"-",2147483648).code==4303 && arithmetic::prefix(2,b"-",9223372036854775808).code==4303);
    assert(arithmetic::prefix(2,b"-",18446744073709551615).bits==1 && arithmetic::prefix(2,b"~",0).bits==18446744073709551615);
    assert(arithmetic::prefix(3,b"~",0).bits==255 && arithmetic::prefix(6,b"!",1).bits==0);
}
@test
fn void forged_scalar_bits_and_excluded_operators_fail_without_execution(){
    assert(arithmetic::binary(3,b"+",256,0).code==12 && arithmetic::binary(6,b"==",2,1).code==12);
    assert(arithmetic::binary(5,b"+",1,1).code==12 && arithmetic::prefix(0,b"~",0).code==12);
    assert(arithmetic::binary(1,b"**",1,1).code==4302 && arithmetic::binary(6,b"+",0,1).code==1006);
    assert(arithmetic::binary(1,b"&&",1,1).code==1007 && arithmetic::prefix(1,b"!",1).code==1007);
}
