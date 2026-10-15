module main;
import std::fs as fs;

@effects(filesystem) fn void inspect(ref fs::File file,usize links,bool readonly) {
    assert(switch(fs::is_read_only(file)){case .ok(v):{v==readonly}case .err(_):{false}});
    assert(switch(fs::hard_link_count(file)){case .ok(n):{n==links}case .err(_):{false}});
    assert(switch(fs::allocated_size(file)){case .ok(n):{n>=3}case .err(_):{false}});
    assert(switch(fs::size(file)){case .ok(n):{n==3}case .err(_):{false}});
}

@effects(filesystem) fn void readonly(ref mut fs::File file,bool enabled) {
    assert(switch(fs::set_read_only(file,enabled)){case .ok(v):{v==enabled}case .err(_):{false}});
    inspect(file,1,enabled);
    assert(switch(fs::position(file)){case .ok(n):{n==3}case .err(_):{false}});
}

@effects(filesystem) fn i32 main() {
    var file=switch(fs::create_new(c"metadata-λ.bin")){case .ok(v):{v}case .err(_):{return 1;}};
    assert(fs::write_all(ref mut file,b"A\0Z").is_ok);
    inspect(ref file,1,false);
    readonly(ref mut file,true);
    readonly(ref mut file,true);
    readonly(ref mut file,false);
    readonly(ref mut file,false);
    assert(fs::close(file).is_ok);
    assert(fs::hard_link(c"metadata-λ.bin",c"alias-λ.bin").is_ok);
    var alias=switch(fs::open_read(c"alias-λ.bin")){case .ok(v):{v}case .err(_):{return 1;}};
    inspect(ref alias,2,false);
    switch(fs::set_read_only(ref mut alias,true)) {
        case .ok(_): { assert(false); }
        case .err(e): { assert(fs::error_status(e)==1 && fs::error_os_code(e)==5 && fs::error_completed(e)==0); }
    };
    inspect(ref alias,2,false);
    assert(switch(fs::position(ref mut alias)){case .ok(n):{n==0}case .err(_):{false}});
    assert(fs::close(alias).is_ok);
    assert(fs::remove_file(c"metadata-λ.bin").is_ok);
    var survivor=switch(fs::open_read(c"alias-λ.bin")){case .ok(v):{v}case .err(_):{return 1;}};
    inspect(ref survivor,1,false);
    var u8[3] data=[0;3];
    assert(fs::read_exact(ref mut survivor,ref mut data).is_ok && data[0]==65 && data[1]==0 && data[2]==90);
    inspect(ref survivor,1,false);
    assert(switch(fs::position(ref mut survivor)){case .ok(n):{n==3}case .err(_):{false}});
    assert(fs::close(survivor).is_ok);
    assert(fs::remove_file(c"alias-λ.bin").is_ok);
    return 0;
}
