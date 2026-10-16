module tests::module_unit_interpreter_math;
import module_constant_arithmetic as math;
import c_abi_types as casts;
import scalar_casts as compatibility;

@test fn void exact_runtime_traps(){
    assert(math::runtime_binary(1,b"+",2147483647,1).trap==2);
    assert(math::runtime_binary(2,b"*",9223372036854775807,2).trap==2);
    assert(math::runtime_binary(1,b"/",1,0).trap==3);
    assert(math::runtime_binary(2,b"%",9223372036854775808,18446744073709551615).trap==4);
    assert(math::runtime_binary(1,b"<<",1,32).trap==5);
    assert(math::runtime_binary(1,b">>",1,4294967295).trap==5);
    assert(math::runtime_prefix(1,b"-",2147483648).trap==2);
}
@test fn void unsigned_host_width_and_signed_boundaries(){
    assert(math::runtime_binary(5,b"+",18446744073709551614,1).bits==18446744073709551615);
    assert(math::runtime_binary(5,b"+",18446744073709551615,1).trap==2);
    assert(math::runtime_binary(5,b"*",18446744073709551615,2).trap==2);
    assert(math::runtime_binary(5,b">",18446744073709551615,9223372036854775807).bits==1);
    assert(math::runtime_binary(5,b"/",18446744073709551615,3).bits==6148914691236517205);
    assert(math::runtime_binary(1,b"<<",4294967295,1).bits==4294967294);
    assert(math::runtime_binary(2,b">>",9223372036854775808,63).bits==18446744073709551615);
    assert(math::runtime_cast(1,2,4294967295).bits==18446744073709551615);
    assert(math::runtime_cast(4,5,4294967295).bits==4294967295);
    assert(!math::runtime_cast(2,1,1).supported);
}
@test fn void cast_admission_keeps_c_transport_boundaries(){
    assert(casts::allowed(1,132) && casts::allowed(132,1));
    assert(casts::allowed(3,5) && casts::allowed(4,5) && casts::allowed(1,2));
    assert(!casts::allowed(138,2) && !casts::allowed(2,138) && !casts::allowed(138,5));
    assert(!casts::allowed(4,140) && !casts::allowed(128,6) && !casts::allowed(2,1));
    assert(compatibility::allowed(1,132) && !compatibility::allowed(138,5));
}
