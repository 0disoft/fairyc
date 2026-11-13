module tests::comparison;
import comparison as comparison;
@test fn void expected_hash_validation_and_states(){
    let ref u8[] actual=b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad";
    assert(comparison::classify(b"",actual,true)==0);
    assert(comparison::classify(b" \r\n",actual,true)==0);
    assert(comparison::classify(b"abc",actual,true)==1);
    assert(comparison::classify(actual,actual,false)==2);
    assert(comparison::classify(actual,actual,true)==3);
    assert(comparison::classify(b"BA7816BF8F01CFEA414140DE5DAE2223B00361A396177A9CB410FF61F20015AD",actual,true)==3);
    assert(comparison::classify(b" \tba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad\r\n",actual,true)==3);
    assert(comparison::classify(b"0000000000000000000000000000000000000000000000000000000000000000",actual,true)==4);
    assert(comparison::classify(b"za7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",actual,true)==1);
}
