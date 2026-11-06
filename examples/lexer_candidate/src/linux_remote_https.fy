module linux_remote_https;
import project_source_url as url;
import linux_command as command;
import std::bytes as bytes;

// Private pull ABI. Native code owns its request and one overall timeout, and
// consumes close even on error. No operation retains output pointers on return.
@effects(network,clock) @link_name("fairyc_private_linux_https_open_v1") unsafe extern C fn u32 open(ptr u8 address,usize size,usize maximum,usize timeout,ptr mut usize output);
@effects(network,clock) @link_name("fairyc_private_linux_https_headers_v1") unsafe extern C fn u32 headers(usize raw,ptr mut u32 status,ptr mut u8 encoding,usize capacity,ptr mut usize length,ptr mut usize declared);
@effects(network,clock) @link_name("fairyc_private_linux_https_read_v1") unsafe extern C fn u32 read(usize raw,ptr mut u8 data,usize capacity,ptr mut usize copied);
@effects(network) @link_name("fairyc_private_linux_https_close_v1") unsafe extern C fn u32 close(usize raw);
@effects(network) resource Request{usize raw;cleanup{if(this.raw!=0){unsafe{discard close(this.raw);};}}}
// 1 URL/budget, 2 local allocation, 3 native/ABI/timeout, 4 response/encoding,
// 5 byte limit. Preserve the native failure and separate consuming-close error.
pub struct Problem{u32 code;u32 native_code;u32 cleanup_code;}
fn Problem fail(u32 code,u32 native_code){return Problem{.code,.native_code,.cleanup_code=0};}
@effects(network) fn u32 finish(ref mut Request request){let raw=request.raw;request.raw=0;return unsafe{close(raw)};}

@effects(network,clock) fn Result<bytes::Buffer,Problem> collect(ref Request request,usize maximum,ref mut bytes::Buffer chunk,ref mut bytes::Buffer encoding){
    var u32 status=0;var usize length=0;var usize declared=0;
    let code=unsafe{headers(request.raw,ref mut status as ptr mut u32,bytes::view_mut(encoding) as ptr mut u8,128,ref mut length as ptr mut usize,ref mut declared as ptr mut usize)};
    if(code!=0 || length>128){return .err(fail(3,code));}
    let coded=bytes::view(encoding);
    if(status!=200 || length!=0 && !bytes::equal(coded[..length],b"identity")){return .err(fail(4,0));}
    // usize::MAX is the sole absent Content-Length sentinel. Actual bytes still
    // use a bounded one-byte overread, including an EOF probe at the exact limit.
    if(declared!=18446744073709551615 && declared>maximum){return .err(fail(5,0));}
    var data=bytes::empty();
    while(true){
        var usize copied=0;let room=maximum-bytes::len(ref data);let usize count=if(room>=65536){65536}else{room+1};
        let code=unsafe{read(request.raw,bytes::view_mut(chunk) as ptr mut u8,count,ref mut copied as ptr mut usize)};
        if(code!=0 || copied>count){return .err(fail(3,code));}
        if(copied==0){break;}if(copied>room){return .err(fail(5,0));}
        let block=bytes::view(chunk);if(!bytes::append(ref mut data,block[..copied]).is_ok){return .err(fail(2,0));}
    }
    if(declared!=18446744073709551615 && declared!=bytes::len(ref data)){return .err(fail(4,0));}
    return .ok(data);
}
// Caller supplies min(30 seconds, remaining operation time). No retries, ambient
// transport settings, filesystem access or publication occur in this facade.
@effects(network,clock) pub fn Result<bytes::Buffer,Problem> download(ref u8[] address,usize maximum,usize timeout){
    if(!url::valid(address) || maximum==0 || maximum>16777216 || timeout==0 || timeout>30000){return .err(fail(1,0));}
    let encoded=switch(command::path(address)){case .ok(p):{p}case .err(p):{return .err(fail(if(p.kind==3){2}else{1},0));}};
    // Scratch storage outlives the request and its cancellation/consuming close.
    var chunk=bytes::empty();if(!bytes::resize_zeroed(ref mut chunk,65536).is_ok){return .err(fail(2,0));}
    var encoding=bytes::empty();if(!bytes::resize_zeroed(ref mut encoding,128).is_ok){return .err(fail(2,0));}
    var usize raw=0;let code=unsafe{open(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),maximum,timeout,ref mut raw as ptr mut usize)};
    if(raw==0){return .err(fail(3,code));}var request=Request{.raw};
    if(code!=0){let cleanup_code=finish(ref mut request);return .err(Problem{.code=3,.native_code=code,.cleanup_code});}
    let result=collect(ref request,maximum,ref mut chunk,ref mut encoding);let cleanup_code=finish(ref mut request);
    return switch(result){
        case .ok(data):{if(cleanup_code!=0){return .err(Problem{.code=3,.native_code=0,.cleanup_code});}return .ok(data);}
        case .err(p):{.err(Problem{.code=p.code,.native_code=p.native_code,.cleanup_code})}
    };
}
