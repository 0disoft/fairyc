module tests::project_lock_write;
import project_lock_write as store;
import project_source_file as source;
import file_source as files;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
fn text::CString cpath(ref u8[] root,ref u8[] leaf){let owned=join(root,leaf);let value=switch(text::validate_utf8(bytes::view(ref owned))){case .ok(value):{value}case .err(_):{assert(false);return cpath(root,leaf);}};return switch(text::cstring_from_text(value)){case .ok(value):{value}case .err(_):{assert(false);return cpath(root,leaf);}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;return switch(files::manifest(path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(source::read(root,leaf,8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn void seed(ref u8[] root,ref u8[] data){let path=cpath(root,b"fairy.lock");var fs::File file=switch(fs::create_truncate(text::cstring_as_cstr(ref path))){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(fs::write_all(ref mut file,data).is_ok);assert(fs::close(file).is_ok);}
@effects(filesystem)
fn void equal(ref u8[] root,ref u8[] leaf,ref u8[] expected){let actual=read(root,leaf);assert(bytes::equal(bytes::view(ref actual),expected));}
@effects(filesystem)
fn void no_new_temporary(ref u8[] root,bool collisions,bool occupied){let value=switch(text::validate_utf8(root)){case .ok(value):{value}case .err(_):{assert(false);return;}};var fs::ReadDir stream=switch(fs::read_dir_utf8(value)){case .ok(value):{value}case .err(_):{assert(false);return;}};var u8[4096] name=[0;4096];var usize temps=0;
    while(true){let entry=switch(fs::next_dir(ref mut stream,ref mut name)){case .ok(value):{value}case .err(_):{assert(false);return;}};if(!entry.has_entry){break;}let ref u8[] input=ref name;let view=input[..entry.name_length];if(view.len>=12 && bytes::equal(view[..12],b".fairy-lock-")){temps+=1;if(collisions){assert(bytes::equal(view,b".fairy-lock-0.tmp") || bytes::equal(view,b".fairy-lock-1.tmp"));}}}
    assert(fs::close_dir(stream).is_ok && temps==if(occupied){4096}else if(collisions){2}else{0});
}
@effects(filesystem)
fn void save(ref u8[] root,ref u8[] payload,bool missing,bool collisions){if(missing){let path=cpath(root,b"fairy.lock");discard fs::remove_file(text::cstring_as_cstr(ref path));}else{seed(root,b"OLD");}var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,payload).is_ok);var usize meter=0;assert(store::write(root,input,ref mut meter).is_ok);equal(root,b"fairy.lock",payload);no_new_temporary(root,collisions,false);}
@test
@effects(filesystem)
fn void canonical_save_creates_replaces_unicode_and_preserves_collisions(){let owned=root();let base=bytes::view(ref owned);let ordinary=read(base,b"None.json");let sdk=read(base,b"Core.json");let existing=join(base,b"Existing");save(bytes::view(ref existing),bytes::view(ref ordinary),false,false);save(bytes::view(ref existing),bytes::view(ref sdk),false,false);let missing=join(base,b"Missing");save(bytes::view(ref missing),bytes::view(ref ordinary),true,false);let unicode=join(base,b"한글-λ");save(bytes::view(ref unicode),bytes::view(ref sdk),false,false);let collisions=join(base,b"Collision");save(bytes::view(ref collisions),bytes::view(ref ordinary),false,true);equal(bytes::view(ref collisions),b".fairy-lock-0.tmp",b"COLLISION");equal(bytes::view(ref collisions),b".fairy-lock-1.tmp/sentinel",b"DIRECTORY");}
@effects(filesystem)
fn void rejects(ref u8[] root,bytes::Buffer input,u32 stage,u32 code){seed(root,b"OLD");var usize meter=0;assert(switch(store::write(root,input,ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==stage && p.code==code && p.cleanup_code==0}});equal(root,b"fairy.lock",b"OLD");no_new_temporary(root,false,false);}
@test
@effects(filesystem)
fn void rejected_validation_publication_and_bounds_preserve_live_names(){let owned=root();let base=bytes::view(ref owned);let invalid=join(base,b"Invalid");rejects(bytes::view(ref invalid),read(base,b"invalid.json"),1,3);let noncanonical=join(base,b"Noncanonical");rejects(bytes::view(ref noncanonical),read(base,b"noncanonical.json"),3,1);
    let bounded=join(base,b"Bounds");seed(bytes::view(ref bounded),b"OLD");var usize meter=16777216;assert(switch(store::write(bytes::view(ref bounded),read(base,b"None.json"),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==3}});equal(bytes::view(ref bounded),b"fairy.lock",b"OLD");no_new_temporary(bytes::view(ref bounded),false,false);
    let directory=join(base,b"Directory");meter=0;assert(switch(store::write(bytes::view(ref directory),read(base,b"None.json"),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==9 && p.code==1 && p.os_code!=0 && p.cleanup_code==0}});equal(bytes::view(ref directory),b"fairy.lock/sentinel",b"DIRECTORY");no_new_temporary(bytes::view(ref directory),false,false);
    let sharing=join(base,b"Sharing");seed(bytes::view(ref sharing),b"OLD");let target=cpath(bytes::view(ref sharing),b"fairy.lock");let blocker=switch(fs::open_read(text::cstring_as_cstr(ref target))){case .ok(value):{value}case .err(_):{assert(false);return;}};meter=0;assert(switch(store::write(bytes::view(ref sharing),read(base,b"None.json"),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==9 && p.code==1 && (p.os_code==32 || p.os_code==5) && p.cleanup_code==0}});assert(fs::close(blocker).is_ok);equal(bytes::view(ref sharing),b"fairy.lock",b"OLD");no_new_temporary(bytes::view(ref sharing),false,false);
    let full=join(base,b"Namespace");seed(bytes::view(ref full),b"OLD");meter=0;assert(switch(store::write(bytes::view(ref full),read(base,b"None.json"),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==5 && p.code==3}});equal(bytes::view(ref full),b"fairy.lock",b"OLD");equal(bytes::view(ref full),b".fairy-lock-0.tmp",b"OCCUPIED");equal(bytes::view(ref full),b".fairy-lock-4095.tmp",b"OCCUPIED");no_new_temporary(bytes::view(ref full),false,true);
}
