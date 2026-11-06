module tests::linux_remote_https;
import linux_remote_https as https;
import std::bytes as bytes;
@link_name("linux_https_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("linux_https_probe_count") unsafe extern C fn u32 count(u32 kind);
@link_name("linux_https_probe_budget") unsafe extern C fn usize budget(u32 kind);
@effects(network,clock) fn https::Problem problem(ref u8[] address,usize maximum,usize timeout){return switch(https::download(address,maximum,timeout)){case .ok(_):{assert(false);return https::Problem{.code=0,.native_code=0,.cleanup_code=0};}case .err(p):{p}};}
@effects(network,clock) fn void fault(u32 value,u32 code,u32 native_code,u32 cleanup_code,u32 closes){
    unsafe{mode(value);}let p=problem(b"https://packages.invalid/library.zip",3,30000);
    assert(p.code==code && p.native_code==native_code && p.cleanup_code==cleanup_code);
    assert(unsafe{count(0)}==1 && unsafe{count(3)}==closes && unsafe{count(4)}==0);
}
@test @effects(network,clock) fn void rejects_invalid_inputs_without_starting_requests(){
    unsafe{mode(0);}assert(problem(b"http://packages.invalid/a",3,30000).code==1 && problem(b"https://user:pass@packages.invalid/a",3,30000).code==1);
    assert(problem(b"https://packages.invalid/a?b",3,30000).code==1 && problem(b"https://packages.invalid/a#b",3,30000).code==1);
    assert(problem(b"https://packages.invalid/\xff",3,30000).code==1 && problem(b"https://packages.invalid/a\0",3,30000).code==1);
    assert(problem(b"https://packages.invalid/a",0,30000).code==1 && problem(b"https://packages.invalid/a",16777217,30000).code==1);
    assert(problem(b"https://packages.invalid/a",3,0).code==1 && problem(b"https://packages.invalid/a",3,30001).code==1);
    var path=bytes::empty();assert(bytes::append(ref mut path,b"https://packages.invalid/").is_ok);while(bytes::len(ref path)<4096){assert(bytes::append_byte(ref mut path,97).is_ok);}
    assert(problem(bytes::view(ref path),3,30000).code==1 && bytes::len(ref path)==4096 && unsafe{count(0)}==0 && unsafe{count(3)}==0);
}
@test @effects(network,clock) fn void preserves_native_errors_and_closes_partial_outputs_once(){
    fault(1,3,5,0,0);fault(2,3,5,0,1);fault(3,3,0,0,0);fault(4,3,61,0,1);fault(9,3,62,0,1);fault(10,3,0,0,1);
    fault(13,3,0,63,1);fault(14,3,62,63,1);fault(16,3,5,63,1);fault(23,4,0,63,1);fault(24,3,12,0,0);fault(25,3,110,0,0);
}
@test @effects(network,clock) fn void checks_headers_body_and_exact_byte_boundaries(){
    fault(5,3,0,0,1);assert(unsafe{count(2)}==0);fault(6,4,0,0,1);assert(unsafe{count(2)}==0);
    fault(7,4,0,0,1);assert(unsafe{count(2)}==0);fault(8,5,0,0,1);assert(unsafe{count(2)}==0);
    fault(11,5,0,0,1);assert(unsafe{count(2)}==1);fault(12,4,0,0,1);fault(19,4,0,0,1);fault(22,4,0,0,1);
    unsafe{mode(17);}let empty=switch(https::download(b"https://packages.invalid/a",16777216,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(bytes::len(ref empty)==0 && unsafe{budget(0)}==16777216 && unsafe{budget(1)}==1 && unsafe{count(3)}==1 && unsafe{count(4)}==0);
}
@test @effects(network,clock) fn void preserves_binary_chunks_and_exact_url_bytes(){
    var u32 value=0;while(value<=27){if(value==0 || value==15 || value==18 || value==26 || value==27){
        unsafe{mode(value);}let address=if(value==26){b"https://packages.invalid/a%2Fb.zip"}else if(value==27){b"https://packages.invalid/한글.zip"}else{b"https://packages.invalid/a"};
        let data=switch(https::download(address,3,30000)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(bytes::equal(bytes::view(ref data),b"a\0\xff") && unsafe{count(2)}==2 && unsafe{count(3)}==1 && unsafe{count(4)}==0);
    }value+=1;}
    value=20;while(value<=21){unsafe{mode(value);}let usize maximum=if(value==20){65536}else{65537};
        let data=switch(https::download(b"https://packages.invalid/a",maximum,30000)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(bytes::len(ref data)==maximum && unsafe{count(2)}==if(value==20){2}else{3} && unsafe{count(3)}==1 && unsafe{count(4)}==0);
        let view=bytes::view(ref data);var usize i=0;while(i<maximum){assert((view[i] as usize)==i%251);i+=1;}value+=1;
    }
}
