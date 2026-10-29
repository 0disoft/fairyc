module tests::native_run_control;
import native_run_control as control;
import windows_run_control as windows;
import std::bytes as bytes;
@test fn void preserves_all_normal_exit_bits_and_requires_exact_trap_receipts(){
    var u32 code=1;while(code<=7){var u8[1] receipt=[0;1];receipt[0]=if(code==1){1}else if(code==2){2}else if(code==3){3}else if(code==4){4}else if(code==5){5}else if(code==6){6}else{7};let ref u8[] view=ref receipt;
        let normal=switch(control::classify(128+code,b"",42)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(normal.status==128+code && normal.trap==0 && normal.pid==42);
        let trapped=switch(control::classify(128+code,view,42)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(trapped.status==5 && trapped.trap==(if(code==6){1}else if(code==7){7}else{code+1}) && trapped.pid==42);code+=1;
    }
    assert(bytes::equal(control::category(1),b"bounds") && bytes::equal(control::category(2),b"integer-overflow") && bytes::equal(control::category(3),b"division-by-zero") && bytes::equal(control::category(4),b"signed-division-overflow") && bytes::equal(control::category(5),b"invalid-shift") && bytes::equal(control::category(6),b"assertion-failed") && bytes::equal(control::category(7),b"unreachable") && control::category(0).len==0 && control::category(8).len==0);
    let full=switch(control::classify(4294967295,b"",7)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(full.status==4294967295 && full.trap==0);
    assert(!control::classify(42,b"\x03",1).is_ok && !control::classify(131,b"\x02",1).is_ok && !control::classify(131,b"\x03\x03",1).is_ok && !control::classify(128,b"\0",1).is_ok && !control::classify(136,b"\x08",1).is_ok);
}
@test fn void validates_extended_trap_location_records(){
    let record=b"\x05\x01\x0a\0\x01\0\0\0\0\0\0\0\x09\0\0\0\0\0\0\0p1/main.fy";
    let accepted=switch(control::classify(133,record,42)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(accepted.trap==6 && accepted.status==5 && accepted.pid==42);
    assert(bytes::equal(control::location_source(record),b"p1/main.fy") && control::location_start(record)==1 && control::location_end(record)==9);
    assert(control::location_source(b"\x05").len==0 && control::location_start(b"\x05")==0 && control::location_end(b"")==0);
    var usize n=2;while(n<record.len){assert(!control::classify(133,record[..n],42).is_ok);n+=1;}
    var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,record[..20]).is_ok);while(bytes::len(ref data)<277){assert(bytes::append_byte(ref mut data,97).is_ok);}
    bytes::view_mut(ref mut data)[2]=0;bytes::view_mut(ref mut data)[3]=1;assert(control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    assert(!control::classify(133,bytes::view(ref data),42).is_ok && !control::classify(132,record,42).is_ok);
    bytes::view_mut(ref mut data)[1]=2;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[1]=1;
    bytes::view_mut(ref mut data)[20]=0;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[20]=255;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);bytes::view_mut(ref mut data)[20]=97;
    bytes::view_mut(ref mut data)[4]=10;assert(!control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    var usize i=4;while(i<20){bytes::view_mut(ref mut data)[i]=255;i+=1;}assert(control::classify(133,bytes::view(ref data)[..276],42).is_ok);
    assert(control::location_start(bytes::view(ref data)[..276])==18446744073709551615 && control::location_end(bytes::view(ref data)[..276])==18446744073709551615);
}
@test fn void formats_snapshot_locations_with_unicode_columns_and_safe_paths(){
    var bytes::Buffer out=bytes::empty();let source=b"first\n\xed\x95\x9c\xea\xb8\x80X";
    assert(control::append_location(b"src/main.fy",source,12,13,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"  at src/main.fy:2:3\n"));
    let before=bytes::len(ref out);
    assert(control::append_location(b"src/main.fy",source,7,13,ref mut out) && control::append_location(b"src/main.fy",source,12,14,ref mut out) && control::append_location(b"src/main.fy",source,13,12,ref mut out));
    assert(control::append_location(b"",source,0,0,ref mut out) && control::append_location(b"src/main.fy",b"\xff",0,1,ref mut out) && bytes::len(ref out)==before);
    assert(bytes::truncate(ref mut out,0).is_ok && control::append_location(b"src/a\nb.fy",b"a\n",2,2,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"  at \"src/a\\nb.fy\":2:1\n"));
}

@test fn void keeps_windows_wrapper_results_and_locations(){
    let record=b"\x05\x01\x0a\0\x01\0\0\0\0\0\0\0\x09\0\0\0\0\0\0\0p1/main.fy";
    let result=switch(windows::classify(133,record,42)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(result.status==5 && result.trap==6 && result.pid==42);
    assert(windows::max_receipt_bytes()==276 && windows::canonical(5)==6 && windows::location_start(record)==1 && windows::location_end(record)==9);
    assert(bytes::equal(windows::category(6),b"assertion-failed") && bytes::equal(windows::location_source(record),b"p1/main.fy"));
    var bytes::Buffer out=bytes::empty();assert(windows::append_location(b"src/main.fy",b"abcd",1,2,ref mut out) && bytes::equal(bytes::view(ref out),b"  at src/main.fy:1:2\n"));
    assert(!windows::classify(132,record,42).is_ok);
}
