module project_toml_context;
import project_toml_model as m;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Decoder metadata belongs to logical component paths, shared by all physical
// array-table entries. Root context is 0; an empty component has its own ID.
pub struct Entry{usize parent;usize start;usize end;usize next;usize hash;usize depth;bool implicit;u32 kind;}
pub struct Flags{bool implicit;u32 kind;}
// Mutable columns stay separate so ownership and exclusive borrows remain explicit.
fn m::Problem fail(ref m::State meta,u32 code,usize start,usize end){let usize a=if(start<=meta.source_bytes){start}else{meta.source_bytes};let usize b=if(end<=meta.source_bytes){end}else{meta.source_bytes};return m::Problem{.code,.start=a,.end=if(b>=a){b}else{a}};}
fn bool charge(ref mut m::State meta,usize cost){if(meta.work>16777216 || cost>16777216-meta.work){return false;}meta.work+=cost;return true;}
fn usize hash(usize parent,ref u8[] name){var usize h=parent%16777213;var usize i=0;while(i<name.len){h=(h*33+(name[i] as usize))%16777213;i+=1;}return h;}
fn void number(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
fn u32 capacity(ref mut bytes::Buffer names,ref mut v::Vec<Entry> entries,ref mut v::Vec<usize> buckets,ref mut m::State meta){
    let usize initial=v::len<usize>(buckets);let usize count=v::len<Entry>(entries);var usize size=if(initial==0){64}else{initial};while(count+1>size/2){size*=2;}
    if(size==initial){return 0;}if(!charge(meta,size+count)){return 4;}
    if(!v::reserve<usize>(buckets,size).is_ok){return 3;}while(v::len<usize>(buckets)<size){if(!v::push<usize>(buckets,0).is_ok){discard v::truncate<usize>(buckets,initial);return 3;}}
    var usize i=0;while(i<size){number(buckets,i,0);i+=1;}i=0;while(i<count){var Entry e=*v::at<Entry>(entries,i);let usize bucket=e.hash%size;e.next=*v::at<usize>(buckets,bucket);let ref mut Entry slot=v::at_mut<Entry>(entries,i);*slot=e;number(buckets,bucket,i+1);i+=1;}return 0;
}
pub fn Result<usize,m::Problem> intern(ref mut bytes::Buffer names,ref mut v::Vec<Entry> entries,ref mut v::Vec<usize> buckets,ref mut m::State meta,usize parent,ref u8[] key,usize start,usize end){
    let usize count=v::len<Entry>(entries);if(meta.source_bytes>1048576 || parent>count || start>end || end>meta.source_bytes){return .err(fail(meta,1,start,end));}
    let usize depth=if(parent==0){1}else{v::at<Entry>(entries,parent-1).depth+1};if(depth>512 || key.len>1048576 || !charge(meta,key.len+1)){return .err(fail(meta,4,start,end));}
    if(!text::validate_utf8(key).is_ok){return .err(fail(meta,2,start,end));}let usize h=hash(parent,key);
    if(v::len<usize>(buckets)!=0){var usize id=*v::at<usize>(buckets,h%v::len<usize>(buckets));var usize steps=0;while(id!=0){if(id>count || steps>=count){return .err(fail(meta,1,start,end));}if(!charge(meta,1)){return .err(fail(meta,4,start,end));}steps+=1;let Entry e=*v::at<Entry>(entries,id-1);
        if(e.hash==h && e.parent==parent && e.end-e.start==key.len){if(!charge(meta,key.len)){return .err(fail(meta,4,start,end));}if(bytes::equal(bytes::view(names)[e.start..e.end],key)){return .ok(id);}}id=e.next;
    }}
    if(count>=meta.source_bytes+1 || bytes::len(names)>1048576 || key.len>1048576-bytes::len(names)){return .err(fail(meta,4,start,end));}
    let u32 code=capacity(names,entries,buckets,meta);if(code!=0){return .err(fail(meta,code,start,end));}if(!v::reserve<Entry>(entries,count+1).is_ok){return .err(fail(meta,3,start,end));}
    let usize offset=bytes::len(names);if(!bytes::append(names,key).is_ok){return .err(fail(meta,3,start,end));}let usize bucket=h%v::len<usize>(buckets);
    let Entry e=Entry{.parent,.start=offset,.end=bytes::len(names),.next=*v::at<usize>(buckets,bucket),.hash=h,.depth,.implicit=false,.kind=0};if(!v::push<Entry>(entries,e).is_ok){discard bytes::truncate(names,offset);return .err(fail(meta,3,start,end));}number(buckets,bucket,count+1);return .ok(count+1);
}
pub fn Flags flags(ref v::Vec<Entry> entries,usize id){return if(id>0 && id<=v::len<Entry>(entries)){let Entry e=*v::at<Entry>(entries,id-1);Flags{.implicit=e.implicit,.kind=e.kind}}else{Flags{.implicit=false,.kind=0}};}
pub fn Result<usize,m::Problem> mark_implicit(ref mut v::Vec<Entry> entries,ref m::State meta,usize id){if(id==0 || id>v::len<Entry>(entries)){return .err(fail(meta,1,0,0));}let ref mut Entry e=v::at_mut<Entry>(entries,id-1);e.implicit=true;return .ok(id);}
// setValue: 0 assigns a new mapping, 1 keeps an implicit table, 2 replaces an
// implicit Array-typed mapping. A second concrete definition is rejected.
pub fn Result<u32,m::Problem> define(ref mut v::Vec<Entry> entries,ref m::State meta,usize id,bool exists){
    if(id==0 || id>v::len<Entry>(entries)){return .err(fail(meta,1,0,0));}let Flags prior=flags(entries,id);if(exists && !prior.implicit){return .err(fail(meta,1,0,0));}
    // Missing physical values leave global implicit metadata alone, as Go does.
    if(!exists){return .ok(0);}let ref mut Entry e=v::at_mut<Entry>(entries,id-1);e.implicit=false;return .ok(if(prior.kind==3){2}else{1});
}
pub fn Result<usize,m::Problem> set_kind(ref mut v::Vec<Entry> entries,ref m::State meta,usize id,u32 kind){if(id==0 || id>v::len<Entry>(entries) || kind<1 || kind>8){return .err(fail(meta,1,0,0));}let ref mut Entry e=v::at_mut<Entry>(entries,id-1);e.kind=kind;return .ok(id);}
// Go setType("", kind) targets the current context except at the document root.
// Empty assignment keys share that special case; their mapping ID stays distinct.
pub fn Result<usize,m::Problem> record_type(ref mut v::Vec<Entry> entries,ref m::State meta,usize id,u32 kind){
    if(id==0 || id>v::len<Entry>(entries)){return .err(fail(meta,1,0,0));}let Entry e=*v::at<Entry>(entries,id-1);return set_kind(entries,meta,if(e.start==e.end && e.parent!=0){e.parent}else{id},kind);
}
pub fn Result<usize,m::Problem> physical(ref mut bytes::Buffer names,ref mut v::Vec<Entry> entries,ref mut v::Vec<usize> buckets,ref mut m::State meta,ref bytes::Buffer source_names,ref v::Vec<m::Node> nodes,usize id){
    if(id==0 || id>v::len<m::Node>(nodes)){return .err(fail(meta,1,0,0));}var usize[512] path=[0;512];var usize size=0;var usize cursor=id;var usize steps=0;
    while(cursor!=1){if(cursor<=1 || cursor>v::len<m::Node>(nodes) || steps>=1024){return .err(fail(meta,1,0,0));}if(!charge(meta,1)){return .err(fail(meta,4,0,0));}steps+=1;let m::Node n=m::entry(nodes,cursor);
        if(n.kind==9 || n.parent==0 || n.parent>=cursor || n.name_start>n.name_end || n.name_end>bytes::len(source_names)){return .err(fail(meta,1,n.start,n.end));}let m::Node parent=m::entry(nodes,n.parent);
        if(parent.kind!=3 && parent.kind!=4){if(parent.kind!=1){return .err(fail(meta,1,n.start,n.end));}if(size==512){return .err(fail(meta,4,n.start,n.end));}path[size]=cursor;size+=1;}cursor=n.parent;
    }
    let m::Node root=m::entry(nodes,1);if(root.kind!=1 || root.parent!=0){return .err(fail(meta,1,0,0));}var usize logical=0;while(size>0){size-=1;let m::Node n=m::entry(nodes,path[size]);logical=switch(intern(names,entries,buckets,meta,logical,bytes::view(source_names)[n.name_start..n.name_end],n.start,n.end)){case .ok(key):{key}case .err(e):{return .err(e);}};}return .ok(logical);
}
