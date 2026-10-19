module tests::windows_msvc_toolchain;
import windows_msvc_toolchain as msvc;
import windows_run_control as env;
import windows_command as command;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer copied(ref u8[] value){return switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] raw=[0;4096];let usize count=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref raw;return copied(view[..count]);}
fn ref u8[] line(ref u8[] data,usize index){var usize i=0;var usize start=0;var usize row=0;while(i<data.len){if(data[i]==10){if(row==index){return data[start..i];}row+=1;start=i+1;}i+=1;}return data[..0];}
fn bool same_path(ref u8[] a,ref u8[] b){if(a.len!=b.len){return false;}var usize i=0;while(i<a.len){if(a[i]!=b[i] && !(a[i]==47 && b[i]==92 || a[i]==92 && b[i]==47)){return false;}i+=1;}return true;}
@effects(console) fn void require(bool ok,u32 checkpoint){if(!ok){discard io::write_stderr("MSVC checkpoint ");discard io::write_stderr_u32(checkpoint);discard io::write_stderr("\n");assert(false);}}
@test @effects(filesystem,console,environment,clock) fn void matches_real_go_toolchain_and_child_environment(){
    let path=fixture();let data=switch(files::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let t=switch(msvc::resolve()){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.status);discard io::write_stderr("/");discard io::write_stderr_u32(e.os_code);require(false,1);return;}};let expected=bytes::view(ref data);
    require(same_path(msvc::compiler(ref t),line(expected,0)) && bytes::equal(msvc::version(ref t),line(expected,1)),2);
    let include=switch(msvc::value(msvc::block(ref t),b"include")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let library=switch(msvc::value(msvc::block(ref t),b"LIB")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let search=switch(msvc::value(msvc::block(ref t),b"Path")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    require(same_path(bytes::view(ref include),line(expected,2)),3);require(same_path(bytes::view(ref library),line(expected,3)),4);
    let prefix=line(expected,4);require(prefix.len<=bytes::len(ref search) && same_path(bytes::view(ref search)[..prefix.len],prefix),5);
    let inherited=switch(env::snapshot()){case .ok(p):{p}case .err(_):{assert(false);return;}};let prior=switch(msvc::value(bytes::view(ref inherited),b"PATH")){case .ok(p):{p}case .err(_):{assert(false);return;}};require(bytes::equal(bytes::view(ref search)[prefix.len..],bytes::view(ref prior)),6);
}
@test @effects(filesystem) fn void selects_latest_complete_sdk_and_rejects_missing_sdk(){
    let path=fixture();var bytes::Buffer root=copied(bytes::view(ref path));assert(bytes::append(ref mut root,b".sdk").is_ok);
    let version=switch(msvc::latest_sdk(bytes::view(ref root))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref version),b"10.2.0"));
    assert(bytes::append(ref mut root,b"/missing").is_ok);assert(!msvc::latest_sdk(bytes::view(ref root)).is_ok);
}
@test fn void validates_components_and_replaces_environment_without_mutation(){
    assert(msvc::component(b"14.51.36231") && !msvc::component(b"..") && !msvc::component(b"bad/version") && !msvc::component(b"bad\\version") && !msvc::component(b"bad\nversion"));
    assert(msvc::absolute(b"C:\\tools") && msvc::absolute(b"\\\\server\\tools") && !msvc::absolute(b"C:tools") && !msvc::absolute(b"tools"));
    let original=b"A\0=\0o\0l\0d\0\0\0Z\0=\0e\0n\0d\0\0\0\0\0";
    let changed=switch(env::with_value(original,b"a",b"new")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let before=switch(msvc::value(original,b"a")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let after=switch(msvc::value(bytes::view(ref changed),b"A")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref before),b"old") && bytes::equal(bytes::view(ref after),b"new"));
    assert(!env::with_value(original,b"bad=key",b"value").is_ok && !msvc::value(b"bad",b"A").is_ok);
}
