module borrow_origins;
import std::vec as v;

// Analysis-local canonical sets. Leaves name HIR bindings or MIR locals/input
// parameters. Static and Unknown keep their old, distinct representations.
// IDs above the parameter namespace never enter published HIR/MIR schemas.
pub struct Entry{usize start;usize count;}
pub fn bool mixed(usize origin){return origin>=262144 && origin<266240;}
pub fn usize count(ref v::Vec<Entry> sets,usize origin){return if(origin==0 || origin==65538){0}else if(mixed(origin)){v::at<Entry>(sets,origin-262144).count}else{1};}
pub fn usize at(ref v::Vec<Entry> sets,ref v::Vec<usize> members,usize origin,usize index){return if(mixed(origin)){*v::at<usize>(members,v::at<Entry>(sets,origin-262144).start+index)}else{origin};}
fn bool charge(ref mut usize work){*work=*work+1;return *work<=1048576;}
pub fn Result<usize,u32> join(ref mut v::Vec<Entry> sets,ref mut v::Vec<usize> members,usize a,usize b,ref mut usize work){
    if(a==65538){return .ok(b);}if(b==65538 || a==b || b==0){return .ok(a);}if(a==0){return .ok(b);}
    let usize na=count(sets,a);let usize nb=count(sets,b);
    var usize[1152] merged=[0;1152];var usize size=0;var usize ia=0;var usize ib=0;
    while(ia<na || ib<nb){
        if(!charge(work)){return .err(2);}
        let usize left=if(ia<na){at(sets,members,a,ia)}else{0};let usize right=if(ib<nb){at(sets,members,b,ib)}else{0};
        let usize item=if(ia==na){right}else if(ib==nb || left<right){left}else{right};
        if(ia<na && left==item){ia+=1;}if(ib<nb && right==item){ib+=1;}
        if(size>=1152){return .err(2);}merged[size]=item;size+=1;
    }
    if(size==1){return .ok(merged[0]);}
    var usize i=0;
    while(i<v::len<Entry>(sets)){
        if(!charge(work)){return .err(2);}let entry=*v::at<Entry>(sets,i);
        if(entry.count==size){var bool same=true;var usize j=0;
            while(j<size && same){if(!charge(work)){return .err(2);}same=*v::at<usize>(members,entry.start+j)==merged[j];j+=1;}
            if(same){return .ok(262144+i);}
        }i+=1;
    }
    if(v::len<Entry>(sets)>=4096 || size>32768-v::len<usize>(members)){return .err(2);}
    let usize start=v::len<usize>(members);i=0;
    while(i<size){if(!v::push<usize>(members,merged[i]).is_ok){return .err(3);}i+=1;}
    if(!v::push<Entry>(sets,Entry{.start,.count=size}).is_ok){return .err(3);}
    return .ok(262144+v::len<Entry>(sets)-1);
}
pub fn bool contains(ref v::Vec<Entry> sets,ref v::Vec<usize> members,usize origin,usize leaf,ref mut usize work){
    if(leaf==0 || leaf==65538){return false;}if(!mixed(origin)){return origin==leaf;}var usize i=0;let usize size=count(sets,origin);
    while(i<size){if(!charge(work)){return false;}let usize item=at(sets,members,origin,i);if(item==leaf){return true;}if(item>leaf){return false;}i+=1;}return false;
}
pub fn bool overlap(ref v::Vec<Entry> sets,ref v::Vec<usize> members,usize a,usize b,ref mut usize work){
    if(!mixed(a) && !mixed(b)){return a!=0 && a!=65538 && a==b;}
    let usize na=count(sets,a);let usize nb=count(sets,b);var usize ia=0;var usize ib=0;
    while(ia<na && ib<nb){if(!charge(work)){return false;}let usize left=at(sets,members,a,ia);let usize right=at(sets,members,b,ib);if(left==right){return true;}if(left<right){ia+=1;}else{ib+=1;}}return false;
}
