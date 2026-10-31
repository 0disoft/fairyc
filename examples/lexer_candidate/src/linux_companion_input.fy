module linux_companion_input;
import linux_workspace_fs as fs;
import std::bytes as bytes;
import std::text as text;

// Reads one regular nofollow input through the retained directory chain. No
// Windows FileKind operation or unbounded SDK read enters the companion.
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read(ref u8[] path,usize maximum){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return .err(1);}};
    var file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(e):{return .err(e.os_code);}};
    let size=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(e.os_code);}};
    if(size==0 || size>maximum || maximum>100663296){return .err(1);}var bytes::Buffer data=bytes::empty();if(!bytes::resize_zeroed(ref mut data,size).is_ok){return .err(1);}
    var usize used=0;while(used<size){let end=used+(if(size-used>65536){65536}else{size-used});
        let n=switch(fs::read(ref mut file,bytes::view_mut(ref mut data)[used..end])){case .ok(n):{n}case .err(e):{return .err(e.os_code);}};
        if(n==0 || n>end-used){return .err(1);}used+=n;
    }
    var u8[1] tail=[0;1];let extra=switch(fs::read(ref mut file,ref mut tail)){case .ok(n):{n}case .err(e):{return .err(e.os_code);}};
    let after=switch(fs::size(ref file)){case .ok(n):{n}case .err(e):{return .err(e.os_code);}};if(extra!=0 || after!=size){return .err(1);}switch(fs::close(file)){case .ok(_):{}case .err(e):{return .err(e.os_code);}};return .ok(data);
}
