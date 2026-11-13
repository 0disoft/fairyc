module checksum_ui;
import checksum as checksum;
import reader as reader;
import ui as ui;
import std::bytes as bytes;

// Reuse the asynchronous reader; even a tiny checksum may live on slow storage.
@effects(filesystem,clock)
pub fn Result<i32,u32> load(ref u8[] path,ptr mut void window,ref mut u8[] expected){
    let input=switch(reader::open(path)){case .ok(v):{v}case .err(e):{return .err(e);}};
    var bytes::Buffer chunk=bytes::empty();var bytes::Buffer content=bytes::empty();
    if(!bytes::resize_zeroed(ref mut chunk,65536).is_ok){return .err(5);}
    var observed=reader::observe(ref input);var first=observed;var bool opened=false;
    while(true){
        let event=ui::poll(window,0);if(event==3 || event==7){return .err(8);}
        if((event==4 || event==5) && !ui::layout(window,event)){return .err(5);}
        observed=reader::observe(ref input);
        if(observed.phase==4){return .err(2);}if(observed.phase==6){return .err(11);}
        if(observed.phase==1){
            if(observed.kind!=1 || observed.directory!=0 || observed.size>65536){return .err(13);}
            first=observed;opened=true;if(!reader::next(ref input)){return .err(2);}
        }
        if(observed.phase==2){
            if(!opened || observed.count>65536 || bytes::view(ref content).len+observed.count>65536){return .err(13);}
            if(!reader::copy(ref input,bytes::view_mut(ref mut chunk))){return .err(2);}
            if(!bytes::append(ref mut content,bytes::view(ref chunk)[..observed.count]).is_ok){return .err(5);}
            if(!reader::next(ref input)){return .err(2);}
        }
        if(observed.phase==3){
            if(!reader::unchanged(ref first,ref observed)){return .err(11);}
            if(!reader::next(ref input)){return .err(2);}
        }
        if(observed.phase==5){break;}
        if(observed.phase==0){reader::wait(ref input);}
    }
    if(!opened || !reader::unchanged(ref first,ref observed) || bytes::view(ref content).len!=first.size){return .err(11);}
    return if(checksum::parse(bytes::view(ref content),expected)){.ok(0)}else{.err(13)};
}
