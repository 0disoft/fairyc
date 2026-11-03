module project_remote_https;
import project_source_url as url;
import windows_command as command;
import std::bytes as bytes;

@effects(network,clock) @link_name("fairyc_private_windows_https_open_v1") unsafe extern C fn usize open(ptr u8 address,usize size,usize timeout);
@effects(network) @link_name("fairyc_private_windows_https_headers_v1") unsafe extern C fn i32 headers(usize raw,ptr mut u32 status,ptr mut u8 encoding,usize capacity,ptr mut usize length,ptr mut usize declared);
@effects(network,clock) @link_name("fairyc_private_windows_https_read_v1") unsafe extern C fn i32 read(usize raw,ptr mut u8 data,usize capacity,ptr mut usize copied);
@effects(network) @link_name("fairyc_private_windows_https_close_v1") unsafe extern C fn i32 close(usize raw);
@effects(network) resource Request{usize raw;cleanup{if(this.raw!=0){unsafe{discard close(this.raw);};}}}
// 1 URL/budget, 2 allocation, 3 HTTPS/timeout, 4 response/encoding, 5 byte limit.
// The caller gives min(30s, remaining operation time), never an ambient setting.
@effects(network,clock) pub fn Result<bytes::Buffer,u32> download(ref u8[] address,usize maximum,usize timeout){
    if(!url::valid(address) || maximum==0 || maximum>16777216 || timeout==0 || timeout>30000){return .err(1);}
    let wide=switch(command::path(address)){case .ok(p):{p}case .err(_):{return .err(2);}};
    // Keep native read storage alive through Request's cancellation/close.
    var bytes::Buffer chunk=bytes::empty();if(!bytes::resize_zeroed(ref mut chunk,65536).is_ok){return .err(2);}var bytes::Buffer encoding=bytes::empty();if(!bytes::resize_zeroed(ref mut encoding,128).is_ok){return .err(2);}var bytes::Buffer data=bytes::empty();
    let usize raw=unsafe{open(bytes::view(ref wide) as ptr u8,bytes::len(ref wide),timeout)};if(raw==0){return .err(3);}let request=Request{.raw};
    var u32 status=0;var usize length=0;var usize declared=0;
    if(unsafe{headers(request.raw,ref mut status as ptr mut u32,bytes::view_mut(ref mut encoding) as ptr mut u8,128,ref mut length as ptr mut usize,ref mut declared as ptr mut usize)}==0 || length>128){return .err(3);}
    let ref u8[] coded=bytes::view(ref encoding);
    if(status!=200 || length!=0 && !bytes::equal(coded[..length],b"i\0d\0e\0n\0t\0i\0t\0y\0")){return .err(4);}
    // usize::MAX means absent length. Check actual bytes even when absent.
    if(declared!=18446744073709551615 && declared>maximum){return .err(5);}
    while(true){var usize copied=0;let usize room=maximum-bytes::len(ref data);let usize count=if(room>=65536){65536}else{room+1};
        if(unsafe{read(request.raw,bytes::view_mut(ref mut chunk) as ptr mut u8,count,ref mut copied as ptr mut usize)}==0 || copied>count){return .err(3);}
        if(copied==0){break;}if(copied>room){return .err(5);}let ref u8[] block=bytes::view(ref chunk);if(!bytes::append(ref mut data,block[..copied]).is_ok){return .err(2);}
    }
    if(declared!=18446744073709551615 && declared!=bytes::len(ref data)){return .err(4);}
    return .ok(data);
}
