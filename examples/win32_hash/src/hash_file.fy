module hash_file;
import sha256_stream as sha;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;

pub struct Digest { usize[8] words; usize length; fs::FileId identity; }

@effects(filesystem)
pub fn Result<fs::File,u32> open(ref u8[] path) {
    let name = switch(text::validate_utf8(path)) { case .ok(v): { v } case .err(_): { return .err(1); } };
    var file = switch(fs::open_read_utf8(name)) { case .ok(v): { v } case .err(_): { return .err(2); } };
    let kind = switch(fs::kind(ref file)) { case .ok(v): { v } case .err(_): { return .err(2); } };
    if (!fs::kind_is_disk(kind)) { return .err(3); }
    return .ok(file);
}

@effects(filesystem)
pub fn Result<usize,u32> read(ref mut fs::File file, ref mut sha::State state, ref mut bytes::Buffer buffer) {
    let count = switch(fs::read(file,bytes::view_mut(buffer))) { case .ok(v): { v } case .err(_): { return .err(2); } };
    let ref u8[] input = bytes::view(buffer);
    if (!sha::update(state,input[..count])) { return .err(4); }
    return .ok(count);
}

@effects(filesystem)
pub fn Result<Digest,u32> finish(ref fs::File file,ref sha::State state) {
    let size = switch(fs::size(file)) { case .ok(v): { v } case .err(_): { return .err(2); } };
    if (size != state.length) { return .err(3); }
    let identity = switch(fs::file_id(file)) { case .ok(v): { v } case .err(_): { return .err(2); } };
    return .ok(Digest{.words=sha::finish(state),.length=state.length,.identity});
}

// CLI orchestration has no UI/clock branches. Both paths share read/finish and
// the same SHA implementation, rather than duplicating the hash algorithm.
@effects(filesystem)
pub fn Result<Digest,u32> compute(ref u8[] path) {
    var file=switch(open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    let total=switch(fs::size(ref file)){case .ok(v):{v}case .err(_):{return .err(2);}};
    var sha::State state=sha::empty();var bytes::Buffer buffer=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    while(true) {
        let count=switch(read(ref mut file,ref mut state,ref mut buffer)){case .ok(v):{v}case .err(e):{return .err(e);}};
        if(count==0){break;}
    }
    let value=switch(finish(ref file,ref state)){case .ok(v):{v}case .err(e):{return .err(e);}};
    if(value.length!=total){return .err(3);}
    if(!fs::close(file).is_ok){return .err(2);}return .ok(value);
}

pub fn bool append_hex(ref Digest value,ref mut bytes::Buffer out) {
    let ref u8[] digits=b"0123456789abcdef";
    var usize i=0; var usize shift=28;
    while(i<64) {
        if(!bytes::append_byte(out,digits[(value.words[i/8]>>shift)&15]).is_ok){return false;}
        i+=1; if(i%8==0){shift=28;}else{shift-=4;}
    }
    return true;
}

pub fn bool report(ref u8[] path,ref Digest value,ref mut bytes::Buffer out) {
    if(!bytes::clear(out).is_ok || !bytes::append(out,b"{\"algorithm\":\"SHA-256\",\"file\":\"").is_ok){return false;}
    let ref u8[] digits=b"0123456789abcdef";
    var usize i=0;
    while(i<path.len) {
        let u8 c=path[i];
        if(c==34 || c==92) { if(!bytes::append_byte(out,92).is_ok){return false;} }
        if(c<32) {
            if(!bytes::append(out,b"\\u00").is_ok || !bytes::append_byte(out,digits[(c/16) as usize]).is_ok || !bytes::append_byte(out,digits[(c%16) as usize]).is_ok){return false;}
        }else{if(!bytes::append_byte(out,c).is_ok){return false;}}
        i+=1;
    }
    return bytes::append(out,b"\",\"bytes\":").is_ok && bytes::append_usize(out,value.length).is_ok &&
        bytes::append(out,b",\"sha256\":\"").is_ok && append_hex(value,out) && bytes::append(out,b"\"}\n").is_ok;
}

fn bool same_path(ref u8[] a,ref u8[] b) {
    if(a.len!=b.len){return false;} var usize i=0;
    while(i<a.len){var u8 x=a[i];var u8 y=b[i];if(x>=65 && x<=90){x+=32;}if(y>=65 && y<=90){y+=32;}if(x==47){x=92;}if(y==47){y=92;}if(x!=y){return false;}i+=1;}return true;
}

@effects(filesystem)
fn bool prepared(fs::File output,ref u8[] data) {
    var file=output;
    let bool written=fs::write_all(ref mut file,data).is_ok && fs::sync(ref mut file).is_ok;
    let bool closed=fs::close(file).is_ok;
    return written && closed;
}

// A sibling is created exclusively, written/synced/closed, then published.
// Never truncate a source file, hard-link alias, or an existing temporary name.
@effects(filesystem)
pub fn Result<i32,u32> save(ref u8[] destination,ref u8[] source,ref Digest digest,ref u8[] data) {
    if(same_path(source,destination)){return .err(7);}
    let name=switch(text::validate_utf8(destination)){case .ok(v):{v}case .err(_):{return .err(1);}};
    switch(fs::open_read_utf8(name)) {
        case .ok(existing): {
            let id=switch(fs::file_id(ref existing)){case .ok(v):{v}case .err(_):{return .err(6);}};
            if(fs::file_id_equal(ref id,ref digest.identity)){return .err(7);}
        }
        case .err(_): {}
    };
    let target=switch(text::cstring_from_text(name)){case .ok(v):{v}case .err(_):{return .err(1);}};
    var bytes::Buffer temporary=bytes::empty(); var usize index=0;
    while(index<32) {
        if(!bytes::clear(ref mut temporary).is_ok || !bytes::append(ref mut temporary,destination).is_ok ||
            !bytes::append(ref mut temporary,b".fairy-hash-").is_ok || !bytes::append_usize(ref mut temporary,index).is_ok ||
            !bytes::append(ref mut temporary,b".tmp").is_ok){return .err(5);}
        let temp_text=switch(text::validate_utf8(bytes::view(ref temporary))){case .ok(v):{v}case .err(_):{return .err(1);}};
        let temp=switch(text::cstring_from_text(temp_text)){case .ok(v):{v}case .err(_):{return .err(5);}};
        switch(fs::create_new(text::cstring_as_cstr(ref temp))) {
            case .ok(file): {
                let bool written=prepared(file,data);
                if(written && fs::replace_file(text::cstring_as_cstr(ref temp),text::cstring_as_cstr(ref target)).is_ok){return .ok(0);}
                let bool removed=fs::remove_file(text::cstring_as_cstr(ref temp)).is_ok;
                return .err(6);
            }
            case .err(e): {if(fs::error_os_code(e)!=80 && fs::error_os_code(e)!=183){return .err(6);}}
        };
        index+=1;
    }
    return .err(6);
}
