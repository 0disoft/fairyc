module module_unit_origin_sets;
import module_unit_value_model as m;
import std::vec as v;

// Analysis-only handles: 0 unreachable/no result, 1 Unknown, 2 Static,
// 3..16386 lexical owners 1..16384, 32768+ canonical mixed sets.
// Unknown absorbs joins. Static remains a member when joined with an owner.
pub struct Entry {usize start;usize count;}
pub fn bool valid(ref v::Vec<Entry> entries,ref v::Vec<usize> members,usize origin){return origin<=16386 || origin>=32768 && origin-32768<v::len<Entry>(entries);}
pub fn Result<usize,u32> owner(usize id){return if(id==0 || id>16384){.err(12)}else{.ok(id+2)};}
pub fn usize owner_id(usize origin){return if(origin>=3 && origin<=16386){origin-2}else{0};}
pub fn usize count(ref v::Vec<Entry> entries,ref v::Vec<usize> members,usize origin){assert(valid(entries,members,origin));return if(origin<=1){0}else if(origin>=32768){v::at<Entry>(entries,origin-32768).count}else{1};}
pub fn usize member(ref v::Vec<Entry> entries,ref v::Vec<usize> members,usize origin,usize index){assert(index<count(entries,members,origin));return if(origin>=32768){*v::at<usize>(members,v::at<Entry>(entries,origin-32768).start+index)}else{origin};}
pub fn usize set_count(ref v::Vec<Entry> entries,ref v::Vec<usize> members){return v::len<Entry>(entries);}
pub fn usize stored_members(ref v::Vec<Entry> entries,ref v::Vec<usize> members){return v::len<usize>(members);}
fn Result<usize,u32> intern(ref mut v::Vec<Entry> entries,ref mut v::Vec<usize> members,ref v::Vec<usize> values,ref mut usize work){
    let usize size=v::len<usize>(values);if(size==0){return .ok(0);}if(size==1){return .ok(*v::at<usize>(values,0));}var usize i=0;
    while(i<v::len<Entry>(entries)){if(!m::charge(work,1)){return .err(10);}let e=*v::at<Entry>(entries,i);if(e.count==size){var usize j=0;var bool same=true;while(j<size && same){if(!m::charge(work,1)){return .err(10);}same=*v::at<usize>(members,e.start+j)==*v::at<usize>(values,j);j+=1;}if(same){return .ok(32768+i);}}i+=1;}
    if(v::len<Entry>(entries)>=4096 || size>32768-v::len<usize>(members)){return .err(10);}if(!m::charge(work,size+1)){return .err(10);}let usize start=v::len<usize>(members);i=0;
    while(i<size){if(!v::push<usize>(members,*v::at<usize>(values,i)).is_ok){return .err(11);}i+=1;}if(!v::push<Entry>(entries,Entry{.start,.count=size}).is_ok){return .err(11);}return .ok(32768+v::len<Entry>(entries)-1);
}
pub fn Result<usize,u32> join(ref mut v::Vec<Entry> entries,ref mut v::Vec<usize> members,usize a,usize b,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(!valid(entries,members,a) || !valid(entries,members,b)){return .err(12);}if(a==1 || b==1){return .ok(1);}if(a==b || b==0){return .ok(a);}if(a==0){return .ok(b);}
    let usize na=count(entries,members,a);let usize nb=count(entries,members,b);var v::Vec<usize> merged=v::empty<usize>();var usize ia=0;var usize ib=0;
    while(ia<na || ib<nb){if(!m::charge(work,1) || v::len<usize>(ref merged)>=16385){return .err(10);}let usize left=if(ia<na){member(entries,members,a,ia)}else{0};let usize right=if(ib<nb){member(entries,members,b,ib)}else{0};let usize item=if(ia==na){right}else if(ib==nb || left<right){left}else{right};if(ia<na && left==item){ia+=1;}if(ib<nb && right==item){ib+=1;}if(!v::push<usize>(ref mut merged,item).is_ok){return .err(11);}}
    return intern(entries,members,ref merged,work);
}
pub fn Result<bool,u32> contains(ref v::Vec<Entry> entries,ref v::Vec<usize> members,usize origin,usize leaf,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(!valid(entries,members,origin) || leaf<2 || leaf>16386){return .err(12);}if(origin<=1){return .ok(false);}var usize low=0;var usize high=count(entries,members,origin);while(low<high){if(!m::charge(work,1)){return .err(10);}let usize mid=low+(high-low)/2;let value=member(entries,members,origin,mid);if(value==leaf){return .ok(true);}if(value<leaf){low=mid+1;}else{high=mid;}}return .ok(false);
}
