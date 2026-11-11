module main;
import hash_file as hash;
import sha256_stream as sha;
import ui as ui;
import std::bytes as bytes;
import std::fs as fs;
import std::process as process;
import std::io as io;

@effects(clock) unsafe extern C fn usize hash_profile_counter();
@effects(clock) unsafe extern C fn usize hash_profile_frequency();
@effects(clock) fn usize counter(){return unsafe{hash_profile_counter()};}
@effects(console) fn void field(ref u8[] name,usize value){discard io::write_bytes(name);discard io::write_usize(value);}

// A profiling entry point over the production read/SHA/UI primitives. This is
// not shipped and does not claim to measure dialogs or full app startup.
@effects(console,filesystem,clock)
fn i32 main(){
    if(process::arg_count()!=3){return 1;}
    var u8[32] mode=[0;32];
    let m=switch(process::arg(1,ref mut mode)){case .ok(v):{v}case .err(_):{return 1;}};
    let ref u8[] modes=ref mode;
    let reading=bytes::equal(modes[..m],b"read");
    let memory=bytes::equal(modes[..m],b"memory");
    let gui=bytes::equal(modes[..m],b"gui");
    if(!reading && !memory && !gui && !bytes::equal(modes[..m],b"cli")){return 1;}
    var bytes::Buffer path=bytes::empty();
    if(!bytes::resize_zeroed(ref mut path,131072).is_ok){return 1;}
    let n=switch(process::arg(2,bytes::view_mut(ref mut path))){case .ok(v):{v}case .err(_):{return 1;}};
    let ref u8[] paths=bytes::view(ref path);
    var file=switch(hash::open(paths[..n])){case .ok(v):{v}case .err(_):{return 1;}};
    let total=switch(fs::size(ref file)){case .ok(v):{v}case .err(_):{return 1;}};
    if(total==0 || total%65536!=0){return 1;}
    var bytes::Buffer buffer=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return 1;}
    if(memory){
        let count=switch(fs::read(ref mut file,bytes::view_mut(ref mut buffer))){case .ok(v):{v}case .err(_):{return 1;}};
        if(count!=65536){return 1;}
    }
    var ptr mut void window=null;
    if(gui){
        window=ui::open();if(window==null){return 1;}
        if(!ui::text(window,207,b"SHA-256") || !ui::layout(window,0) || !ui::progress(window,0,total,0)){ui::destroy(window);return 1;}
        ui::show(window);
    }
    let frequency=unsafe{hash_profile_frequency()};if(frequency==0){if(gui){ui::destroy(window);}return 1;}
    var sha::State state=sha::empty();
    var usize read_ticks=0;var usize hash_ticks=0;var usize poll_ticks=0;var usize paint_ticks=0;
    var usize processed=0;var usize polls=0;var usize updates=0;
    let started=ui::ticks();var usize displayed=started;let start=counter();
    while(processed<total){
        var usize count=65536;
        if(!memory){
            let at=counter();
            count=switch(fs::read(ref mut file,bytes::view_mut(ref mut buffer))){case .ok(v):{v}case .err(_):{if(gui){ui::destroy(window);}return 1;}};
            read_ticks+=counter()-at;
            if(count==0){if(gui){ui::destroy(window);}return 1;}
        }
        if(!reading){
            let at=counter();let ref u8[] input=bytes::view(ref buffer);
            if(!sha::update(ref mut state,input[..count])){if(gui){ui::destroy(window);}return 1;}
            hash_ticks+=counter()-at;
        }
        processed+=count;
        if(gui){
            let at=counter();let event=ui::poll(window,0);
            if(event==3 || event==7){ui::destroy(window);return 1;}
            if((event==4 || event==5) && !ui::layout(window,event)){ui::destroy(window);return 1;}
            poll_ticks+=counter()-at;polls+=1;
            let now=ui::ticks();
            if(now-displayed>=100 || processed==total){
                let paint=counter();
                if(!ui::progress(window,processed,total,now-started)){ui::destroy(window);return 1;}
                paint_ticks+=counter()-paint;updates+=1;displayed=now;
            }
        }
    }
    let final_started=counter();var bytes::Buffer hex=bytes::empty();
    if(!reading && !sha::append_hex(ref state,ref mut hex)){if(gui){ui::destroy(window);}return 1;}
    let final_ticks=counter()-final_started;let elapsed=counter()-start;
    if(!fs::close(file).is_ok){if(gui){ui::destroy(window);}return 1;}
    if(gui){ui::destroy(window);}
    discard io::write_bytes(b"{\"ok\":true");field(b",\"frequency\":",frequency);field(b",\"bytes\":",processed);
    field(b",\"totalTicks\":",elapsed);field(b",\"readTicks\":",read_ticks);field(b",\"hashTicks\":",hash_ticks);
    field(b",\"pollTicks\":",poll_ticks);field(b",\"progressTicks\":",paint_ticks);field(b",\"finalTicks\":",final_ticks);
    field(b",\"polls\":",polls);field(b",\"updates\":",updates);
    discard io::write_bytes(b",\"sha256\":\"");discard io::write_bytes(bytes::view(ref hex));discard io::write_bytes(b"\"}\n");return 0;
}
