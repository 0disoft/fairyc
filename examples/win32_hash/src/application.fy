module application;
import hash_file as hash;
import sha256_stream as sha;
import ui as ui;
import std::fs as fs;
import std::bytes as bytes;

pub fn bool error(ptr mut void window,u32 code){
    if(code==7){return ui::text(window,205,b"Cannot overwrite the input file or its hard-link alias.");}
    if(code==6){return ui::text(window,205,b"Could not save the report. Choose a writable file location.");}
    if(code==5){return ui::text(window,205,b"Not enough memory.");}
    if(code==3){return ui::text(window,205,b"Input size changed or the input is not a regular disk file.");}
    if(code==8){return ui::text(window,205,b"Cancelled.");}
    return ui::text(window,205,b"Could not read the selected file.");
}

@effects(filesystem)
pub fn Result<hash::Digest,u32> compute(ref u8[] path,ptr mut void window) {
    var file=switch(hash::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var sha::State state=sha::empty();var bytes::Buffer buffer=bytes::empty();var bytes::Buffer status=bytes::empty();
    if(!bytes::resize_zeroed(ref mut buffer,65536).is_ok){return .err(5);}
    var usize displayed=0;
    while(true) {
        let count=switch(hash::read(ref mut file,ref mut state,ref mut buffer)){case .ok(v):{v}case .err(e):{return .err(e);}};
        if(window!=null) {
            let event=ui::poll(window,0);if(event==3){return .err(8);}
            if((event==4 || event==5) && !ui::layout(window,event)){return .err(5);}
            if(state.length-displayed>=1048576 || count==0) {
                if(!bytes::clear(ref mut status).is_ok || !bytes::append(ref mut status,b"Bytes hashed: ").is_ok || !bytes::append_usize(ref mut status,state.length).is_ok){return .err(5);}
                if(!ui::text(window,205,bytes::view(ref status))){return .err(5);}displayed=state.length;
            }
        }
        if(count==0){break;}
    }
    let value=switch(hash::finish(ref file,ref state)){case .ok(v):{v}case .err(e):{return .err(e);}};
    if(!fs::close(file).is_ok){return .err(2);}return .ok(value);
}

@effects(filesystem)
pub fn i32 run() {
    var bytes::Buffer path=bytes::empty();var bytes::Buffer selection=bytes::empty();
    var bytes::Buffer report=bytes::empty();var bytes::Buffer hex=bytes::empty();
    if(!bytes::resize_zeroed(ref mut selection,131072).is_ok){return 1;}
    let window=ui::open();if(window==null){return 1;}
    let bool initialized=ui::text(window,206,b"File") && ui::text(window,207,b"SHA-256") &&
        ui::text(window,202,b"&Open file...") && ui::text(window,204,b"&Save report...") &&
        ui::text(window,205,b"Choose a file to calculate its SHA-256.") && ui::enable(window,204,false) && ui::layout(window,0);
    if(!initialized){ui::destroy(window);return 1;}ui::show(window);
    var bool ready=false;var i32 result=0;
    var hash::Digest last=hash::Digest{.words=[0;8],.length=0,.identity=fs::FileId{.volume_low=0,.volume_high=0,.file_low_0=0,.file_low_1=0,.file_high_0=0,.file_high_1=0}};
    while(true) {
        let event=ui::poll(window,1);if(event==3){break;}
        if(event==4 || event==5){if(!ui::layout(window,event)){result=1;break;}}
        if(event==1) {
            let chosen=ui::choose(window,false,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072) {
                ready=false;
                let bool disabled=ui::enable(window,202,false) && ui::enable(window,204,false);
                if(!disabled || !bytes::clear(ref mut path).is_ok || !bytes::append(ref mut path,bytes::view(ref selection)[..chosen]).is_ok){result=1;break;}
                let bool shown=ui::text(window,201,bytes::view(ref path)) && ui::text(window,203,b"") && ui::text(window,205,b"Calculating SHA-256...");
                if(!shown){result=1;break;}
                switch(compute(bytes::view(ref path),window)) {
                    case .ok(value): {
                        last=value;
                        if(!bytes::clear(ref mut hex).is_ok || !hash::append_hex(ref last,ref mut hex) ||
                            !hash::report(bytes::view(ref path),ref last,ref mut report) || !ui::text(window,203,bytes::view(ref hex))){result=1;break;}
                        ready=true;
                    }
                    case .err(code): {let bool shown=error(window,code);}
                };
                let bool enabled=ui::enable(window,202,true) && ui::enable(window,204,ready);
                if(!enabled){result=1;break;}
            }else if(chosen>131072){let bool shown=ui::text(window,205,b"Could not open the file chooser.");}
        }
        if(event==2 && ready) {
            let chosen=ui::choose(window,true,bytes::view_mut(ref mut selection));
            if(chosen>0 && chosen<131072) {
                switch(hash::save(bytes::view(ref selection)[..chosen],bytes::view(ref path),ref last,bytes::view(ref report))) {
                    case .ok(_): {let bool shown=ui::text(window,205,b"Report saved.");}
                    case .err(code): {let bool shown=error(window,code);}
                };
            }else if(chosen>131072){let bool shown=ui::text(window,205,b"Could not open the save dialog.");}
        }
    }
    ui::destroy(window);return result;
}
