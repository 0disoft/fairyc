module progress;
import std::bytes as bytes;

// Compare against exact percentage thresholds without overflowing done*100.
pub fn usize percent(usize done,usize total) {
    if(total==0){return 0;}if(done>=total){return 100;}
    var usize value=0;
    while(value<100) {
        let next=value+1;
        let threshold=(total/100)*next+((total%100)*next+99)/100;
        if(done<threshold){break;}value=next;
    }
    return value;
}

fn bool pair(ref mut bytes::Buffer out,usize value) {
    return (value>=10 || bytes::append_byte(out,48).is_ok) && bytes::append_usize(out,value).is_ok;
}

pub fn bool elapsed(ref mut bytes::Buffer out,usize milliseconds) {
    let seconds=milliseconds/1000;
    return pair(out,seconds/3600) && bytes::append_byte(out,58).is_ok &&
        pair(out,(seconds/60)%60) && bytes::append_byte(out,58).is_ok && pair(out,seconds%60);
}
