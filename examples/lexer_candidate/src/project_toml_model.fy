module project_toml_model;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;
import project_toml_scalars as atoms;

// Separate mutable columns are verified and moved into an immutable owner.
// Kind: 0 unassigned, 1 table, 2 string, 3 array, 4 table array, 5 boolean,
// 6 integer, 7 float, 8 date/time. Non-string atoms support exact rejection.
// State: 0 implicit, 1 dotted, 2 declared, 3 closed. Syntax chooses the policy.
pub struct Node{
    usize parent;usize name_start;usize name_end;usize value_start;usize value_end;
    usize start;usize end;usize first;usize last;usize next;usize children;
    usize hash;usize hash_next;usize depth;u32 kind;u32 state;
}
pub struct State{usize source_bytes;usize work;usize named;}
pub struct Problem{u32 code;usize start;usize end;}
pub resource Tree{bytes::Buffer names;bytes::Buffer values;v::Vec<Node> nodes;v::Vec<usize> buckets;State meta;cleanup{}}
struct Expected{usize first;usize last;usize children;bool closed;}
fn Node blank(){return Node{.parent=0,.name_start=0,.name_end=0,.value_start=0,.value_end=0,.start=0,.end=0,.first=0,.last=0,.next=0,.children=0,.hash=0,.hash_next=0,.depth=0,.kind=0,.state=0};}
fn Problem problem(ref State meta,u32 code,usize start,usize end){let usize a=if(start<=meta.source_bytes){start}else{meta.source_bytes};let usize b=if(end<=meta.source_bytes){end}else{meta.source_bytes};return Problem{.code,.start=a,.end=if(b>=a){b}else{a}};}
fn bool span(ref State meta,usize start,usize end){return start<=end && end<=meta.source_bytes;}
fn bool charge(ref mut State meta,usize amount){if(meta.work>16777216 || amount>16777216-meta.work){return false;}meta.work+=amount;return true;}
pub fn Node entry(ref v::Vec<Node> nodes,usize id){return if(id>0 && id<=v::len<Node>(nodes)){*v::at<Node>(nodes,id-1)}else{blank()};}
fn void set(ref mut v::Vec<Node> nodes,usize id,Node value){let ref mut Node slot=v::at_mut<Node>(nodes,id-1);*slot=value;}
fn void number(ref mut v::Vec<usize> values,usize id,usize value){let ref mut usize slot=v::at_mut<usize>(values,id);*slot=value;}
pub fn usize count(ref Tree tree){return v::len<Node>(ref tree.nodes);}
pub fn Node node(ref Tree tree,usize id){return entry(ref tree.nodes,id);}
pub fn ref u8[] name(ref Tree tree,usize id){let Node n=node(tree,id);return bytes::view(ref tree.names)[n.name_start..n.name_end];}
pub fn ref u8[] value(ref Tree tree,usize id){let Node n=node(tree,id);return bytes::view(ref tree.values)[n.value_start..n.value_end];}
pub fn usize source_bytes(ref Tree tree){return tree.meta.source_bytes;}
pub fn State state(usize source_bytes){return State{.source_bytes,.work=0,.named=0};}
pub fn u32 initialize(ref State meta,ref mut v::Vec<Node> nodes){
    if(meta.source_bytes>1048576){return 4;}if(v::len<Node>(nodes)!=0 || meta.work!=0 || meta.named!=0){return 1;}
    var Node root=blank();root.kind=1;root.state=2;root.end=meta.source_bytes;return if(v::push<Node>(nodes,root).is_ok){0}else{3};
}
fn usize fingerprint(usize parent,ref u8[] key){var usize hash=parent%16777213;var usize i=0;while(i<key.len){hash=(hash*33+(key[i] as usize))%16777213;i+=1;}return hash;}
fn usize find(ref bytes::Buffer names,ref v::Vec<Node> nodes,ref v::Vec<usize> buckets,usize parent,ref u8[] key){
    if(entry(nodes,parent).kind!=1 || v::len<usize>(buckets)==0){return 0;}
    let usize hash=fingerprint(parent,key);var usize id=*v::at<usize>(buckets,hash%v::len<usize>(buckets));var usize steps=0;
    while(id!=0 && steps<v::len<Node>(nodes)){let Node n=entry(nodes,id);if(n.kind!=9 && n.hash==hash && n.parent==parent && bytes::equal(bytes::view(names)[n.name_start..n.name_end],key)){return id;}id=n.hash_next;steps+=1;}return 0;
}
pub fn usize lookup(ref Tree tree,usize parent,ref u8[] key){return find(ref tree.names,ref tree.nodes,ref tree.buckets,parent,key);}
pub fn Result<usize,Problem> resolve(ref Tree tree,usize parent,ref u8[] key,ref mut usize work){
    var State meta=tree.meta;meta.work=*work;let result=locate(ref mut meta,ref tree.names,ref tree.nodes,ref tree.buckets,parent,key,0,0);*work=meta.work;return result;
}
// Syntax uses the metered lookup while building. Immutable readers operate on
// the already verified index and traverse at most the published node count.
pub fn Result<usize,Problem> locate(ref mut State meta,ref bytes::Buffer names,ref v::Vec<Node> nodes,ref v::Vec<usize> buckets,usize parent,ref u8[] key,usize start,usize end){
    if(!span(meta,start,end) || entry(nodes,parent).kind!=1){return .err(problem(meta,1,start,end));}
    if(key.len>1048576 || !charge(meta,key.len+1)){return .err(problem(meta,4,start,end));}
    if(v::len<usize>(buckets)==0){return .ok(0);}
    let usize hash=fingerprint(parent,key);var usize id=*v::at<usize>(buckets,hash%v::len<usize>(buckets));var usize steps=0;
    while(id!=0){
        if(id>v::len<Node>(nodes) || steps>=v::len<Node>(nodes)){return .err(problem(meta,1,start,end));}
        if(!charge(meta,1)){return .err(problem(meta,4,start,end));}steps+=1;let Node n=entry(nodes,id);
        if(n.name_start>n.name_end || n.name_end>bytes::len(names)){return .err(problem(meta,1,start,end));}
        if(n.kind!=9 && n.hash==hash && n.parent==parent && n.name_end-n.name_start==key.len){
            if(!charge(meta,key.len)){return .err(problem(meta,4,start,end));}
            if(bytes::equal(bytes::view(names)[n.name_start..n.name_end],key)){return .ok(id);}
        }id=n.hash_next;
    }return .ok(0);
}
fn u32 index_capacity(ref mut State meta,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets){
    let usize initial=v::len<usize>(buckets);var usize target=if(initial==0){64}else{initial};while(meta.named+1>target/2){target*=2;}if(target==initial){return 0;}
    if(!charge(meta,target+v::len<Node>(nodes))){return 4;}if(!v::reserve<usize>(buckets,target).is_ok){return 3;}
    while(v::len<usize>(buckets)<target){if(!v::push<usize>(buckets,0).is_ok){discard v::truncate<usize>(buckets,initial);return 3;}}
    var usize i=0;while(i<target){number(buckets,i,0);i+=1;}
    i=2;while(i<=v::len<Node>(nodes)){var Node n=entry(nodes,i);if(entry(nodes,n.parent).kind==1){let usize bucket=n.hash%target;n.hash_next=*v::at<usize>(buckets,bucket);set(nodes,i,n);number(buckets,bucket,i);}i+=1;}return 0;
}
fn Result<usize,Problem> append_node(ref mut State meta,ref mut bytes::Buffer names,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets,usize parent,ref u8[] key,bool named,usize start,usize end){
    let Node owner=entry(nodes,parent);
    if(!span(meta,start,end) || parent==0 || owner.depth>=512 || owner.state==3 || (named && owner.kind!=1) || (!named && owner.kind!=3 && owner.kind!=4)){return .err(problem(meta,if(owner.depth>=512){4}else{1},start,end));}
    if(v::len<Node>(nodes)>=meta.source_bytes+1 || bytes::len(names)>1048576 || key.len>1048576-bytes::len(names) || !charge(meta,key.len+1)){return .err(problem(meta,4,start,end));}
    if(!text::validate_utf8(key).is_ok){return .err(problem(meta,2,start,end));}
    if(named){let u32 code=index_capacity(meta,nodes,buckets);if(code!=0){return .err(problem(meta,code,start,end));}let usize previous=switch(locate(meta,names,nodes,buckets,parent,key,start,end)){case .ok(id):{id}case .err(e):{return .err(e);}};if(previous!=0){return .err(problem(meta,1,start,end));}}
    let usize minimum=v::len<Node>(nodes)+1;if(!v::reserve<Node>(nodes,minimum).is_ok){return .err(problem(meta,3,start,end));}
    let usize initial=bytes::len(names);if(!bytes::append(names,key).is_ok){return .err(problem(meta,3,start,end));}
    var Node n=blank();n.parent=parent;n.name_start=initial;n.name_end=bytes::len(names);n.start=start;n.end=end;n.depth=owner.depth+1;
    if(named){n.hash=fingerprint(parent,key);n.hash_next=*v::at<usize>(buckets,n.hash%v::len<usize>(buckets));}
    if(!v::push<Node>(nodes,n).is_ok){discard bytes::truncate(names,initial);return .err(problem(meta,3,start,end));}
    let usize id=v::len<Node>(nodes);if(owner.last!=0){var Node previous=entry(nodes,owner.last);previous.next=id;set(nodes,owner.last,previous);}
    var Node updated=owner;if(updated.first==0){updated.first=id;}updated.last=id;updated.children+=1;set(nodes,parent,updated);
    if(named){let usize bucket=n.hash%v::len<usize>(buckets);number(buckets,bucket,id);meta.named+=1;}return .ok(id);
}
pub fn Result<usize,Problem> reserve_key(ref mut State meta,ref mut bytes::Buffer names,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets,usize parent,ref u8[] key,usize start,usize end){return append_node(meta,names,nodes,buckets,parent,key,true,start,end);}
pub fn Result<usize,Problem> element(ref mut State meta,ref mut bytes::Buffer names,ref mut v::Vec<Node> nodes,ref mut v::Vec<usize> buckets,usize parent,usize start,usize end){return append_node(meta,names,nodes,buckets,parent,b"",false,start,end);}
pub fn Result<usize,Problem> container(ref State meta,ref mut v::Vec<Node> nodes,usize id,u32 kind,u32 table_state,usize end){
    var Node n=entry(nodes,id);if(id==0 || id>v::len<Node>(nodes) || n.kind!=0 || n.state==3 || kind<1 || kind>4 || kind==2 || table_state>2 || (kind!=1 && table_state!=0) || !span(meta,n.start,end)){return .err(problem(meta,1,n.start,end));}n.kind=kind;n.state=table_state;n.end=end;set(nodes,id,n);return .ok(id);
}
fn Result<usize,Problem> leaf(ref mut State meta,ref mut bytes::Buffer values,ref mut v::Vec<Node> nodes,usize id,ref u8[] decoded,usize end,u32 kind){
    var Node n=entry(nodes,id);if(id==0 || id!=v::len<Node>(nodes) || n.kind!=0 || n.state==3 || !span(meta,n.start,end)){return .err(problem(meta,1,n.start,end));}
    if(!text::validate_utf8(decoded).is_ok){return .err(problem(meta,2,n.start,end));}
    if(bytes::len(values)>1048576 || decoded.len>1048576-bytes::len(values) || !charge(meta,decoded.len+1)){return .err(problem(meta,4,n.start,end));}
    let usize initial=bytes::len(values);if(!bytes::append(values,decoded).is_ok){return .err(problem(meta,3,n.start,end));}n.kind=kind;n.value_start=initial;n.value_end=bytes::len(values);n.end=end;set(nodes,id,n);return .ok(id);
}
pub fn Result<usize,Problem> string(ref mut State meta,ref mut bytes::Buffer values,ref mut v::Vec<Node> nodes,usize id,ref u8[] decoded,usize end){return leaf(meta,values,nodes,id,decoded,end,2);}
pub fn Result<usize,Problem> boolean(ref mut State meta,ref mut bytes::Buffer values,ref mut v::Vec<Node> nodes,usize id,bool value,usize end){return leaf(meta,values,nodes,id,if(value){b"true"}else{b"false"},end,5);}
pub fn Result<usize,Problem> atom(ref mut State meta,ref mut bytes::Buffer values,ref mut v::Vec<Node> nodes,usize id,ref u8[] raw,usize end,u32 kind){if(kind<6 || kind>8 || atoms::kind(raw)!=kind){return .err(problem(meta,1,entry(nodes,id).start,end));}return leaf(meta,values,nodes,id,raw,end,kind);}
pub fn Result<usize,Problem> declare(ref State meta,ref mut v::Vec<Node> nodes,usize id,usize start,usize end){
    var Node n=entry(nodes,id);if(id==0 || id>v::len<Node>(nodes) || n.kind!=1 || n.state!=0 || !span(meta,start,end)){return .err(problem(meta,1,start,end));}n.state=2;n.start=start;n.end=end;set(nodes,id,n);return .ok(id);
}
pub fn Result<usize,Problem> finish_container(ref State meta,ref mut v::Vec<Node> nodes,usize id,usize end){
    var Node n=entry(nodes,id);if(id<=1 || id>v::len<Node>(nodes) || (n.kind!=1 && n.kind!=3 && n.kind!=4) || n.state==3 || !span(meta,n.start,end)){return .err(problem(meta,1,n.start,end));}n.end=end;set(nodes,id,n);return .ok(id);
}
pub fn Result<usize,Problem> close(ref mut State meta,ref mut v::Vec<Node> nodes,usize id,usize end){
    let Node root=entry(nodes,id);if(id<=1 || id>v::len<Node>(nodes) || root.kind!=1 || root.state==3 || !span(meta,root.start,end)){return .err(problem(meta,1,root.start,end));}
    if(!charge(meta,v::len<Node>(nodes)*3)){return .err(problem(meta,4,root.start,end));}
    var usize cursor=id;while(true){var Node n=entry(nodes,cursor);n.state=3;if(cursor==id){n.end=end;}set(nodes,cursor,n);
        if(n.first!=0){cursor=n.first;continue;}while(cursor!=id && entry(nodes,cursor).next==0){cursor=entry(nodes,cursor).parent;}if(cursor==id){break;}cursor=entry(nodes,cursor).next;
    }return .ok(id);
}
// Decoder-compatible replacement keeps arena IDs stable until parsing ends.
// Kind 9 is a discarded scratch node; ordinary publish always rejects it.
pub fn Result<usize,Problem> replace_table(ref mut State meta,ref mut v::Vec<Node> nodes,usize id){
    var Node owner=entry(nodes,id);if(id<=1 || id>v::len<Node>(nodes) || owner.kind==0 || owner.kind==9){return .err(problem(meta,1,owner.start,owner.end));}
    if(!charge(meta,v::len<Node>(nodes)*3)){return .err(problem(meta,4,owner.start,owner.end));}
    var usize cursor=owner.first;var usize steps=0;
    while(cursor!=0){if(cursor<=id || cursor>v::len<Node>(nodes) || steps>=v::len<Node>(nodes)*2){return .err(problem(meta,1,owner.start,owner.end));}steps+=1;var Node n=entry(nodes,cursor);n.kind=9;set(nodes,cursor,n);
        if(n.first!=0){cursor=n.first;continue;}while(entry(nodes,cursor).next==0 && entry(nodes,cursor).parent!=id){cursor=entry(nodes,cursor).parent;if(cursor<=id || cursor>v::len<Node>(nodes) || steps>=v::len<Node>(nodes)*2){return .err(problem(meta,1,owner.start,owner.end));}steps+=1;}
        cursor=entry(nodes,cursor).next;
    }
    owner.kind=1;owner.state=2;owner.first=0;owner.last=0;owner.children=0;owner.value_start=0;owner.value_end=0;set(nodes,id,owner);return .ok(id);
}
pub fn Result<usize,Problem> define_table(ref State meta,ref mut v::Vec<Node> nodes,usize id,usize start,usize end){
    var Node n=entry(nodes,id);if(id<=1 || id>v::len<Node>(nodes) || n.kind!=1 || !span(meta,start,end)){return .err(problem(meta,1,start,end));}n.state=2;n.start=start;n.end=end;set(nodes,id,n);return .ok(id);
}
fn bool power_of_two(usize size){var usize n=size;while(n>1 && n%2==0){n/=2;}return n==1;}
fn u32 verify(ref Tree tree){
    let usize total=count(tree);let usize size=v::len<usize>(ref tree.buckets);
    if(tree.meta.source_bytes>1048576 || tree.meta.work>16777216 || total==0 || total>tree.meta.source_bytes+1 || bytes::len(ref tree.names)>1048576 || bytes::len(ref tree.values)>1048576 || (size!=0 && (size<64 || !power_of_two(size) || size>4194304))){return 1;}
    var v::Vec<Expected> expected=v::empty<Expected>();var v::Vec<usize> heads=v::empty<usize>();
    if(!v::reserve<Expected>(ref mut expected,total).is_ok || !v::reserve<usize>(ref mut heads,size).is_ok){return 3;}
    var usize i=0;while(i<total){if(!v::push<Expected>(ref mut expected,Expected{.first=0,.last=0,.children=0,.closed=false}).is_ok){return 3;}i+=1;}
    i=0;while(i<size){if(!v::push<usize>(ref mut heads,0).is_ok){return 3;}i+=1;}
    var usize names_end=0;var usize values_end=0;var usize named=0;var usize probes=0;i=1;
    while(i<=total){
        let Node n=node(tree,i);
        if(n.kind<1 || n.kind>8 || n.state>3 || (n.kind!=1 && n.state!=0 && n.state!=3) || !span(ref tree.meta,n.start,n.end) || n.name_start!=names_end || n.name_end<n.name_start || n.name_end>bytes::len(ref tree.names) || n.value_end<n.value_start || n.value_end>bytes::len(ref tree.values) || !text::validate_utf8(name(tree,i)).is_ok){return 1;}names_end=n.name_end;
        if(n.kind==2 || n.kind>=5){if(n.value_start!=values_end || !text::validate_utf8(value(tree,i)).is_ok){return 1;}if(n.kind==5 && !bytes::equal(value(tree,i),b"true") && !bytes::equal(value(tree,i),b"false")){return 1;}if(n.kind>=6 && atoms::kind(value(tree,i))!=n.kind){return 1;}values_end=n.value_end;}else if(n.value_start!=0 || n.value_end!=0){return 1;}
        if(i==1){if(n.kind!=1 || n.state!=2 || n.parent!=0 || n.depth!=0 || n.name_end!=0 || n.next!=0 || n.hash!=0 || n.hash_next!=0){return 1;}}
        else{
            if(n.parent==0 || n.parent>=i){return 1;}let Node p=node(tree,n.parent);if((p.kind!=1 && p.kind!=3 && p.kind!=4) || (p.kind==4 && n.kind!=1) || n.depth!=p.depth+1 || n.depth>512){return 1;}
            var Expected e=*v::at<Expected>(ref expected,n.parent-1);let bool closed=e.closed || p.state==3;if(closed && n.state!=3){return 1;}
            if(e.last!=0 && node(tree,e.last).next!=i){return 1;}if(e.first==0){e.first=i;}e.last=i;e.children+=1;
            {let ref mut Expected slot=v::at_mut<Expected>(ref mut expected,n.parent-1);*slot=e;}{let ref mut Expected slot=v::at_mut<Expected>(ref mut expected,i-1);slot.closed=closed;}
            if(p.kind==1){
                if(size==0 || n.hash!=fingerprint(n.parent,name(tree,i))){return 1;}let usize bucket=n.hash%size;
                if(n.hash_next!=*v::at<usize>(ref heads,bucket)){return 1;}
                var usize previous=n.hash_next;while(previous!=0){
                    if(probes>=16777216){return 4;}probes+=1;let Node old=node(tree,previous);
                    if(old.parent==n.parent && old.hash==n.hash && old.name_end-old.name_start==n.name_end-n.name_start){
                        let usize length=n.name_end-n.name_start;if(length>16777216-probes){return 4;}probes+=length;
                        if(bytes::equal(name(tree,previous),name(tree,i))){return 1;}
                    }previous=old.hash_next;
                }
                number(ref mut heads,bucket,i);named+=1;
            }else if(n.name_start!=n.name_end || n.hash!=0 || n.hash_next!=0){return 1;}
        }i+=1;
    }
    if(names_end!=bytes::len(ref tree.names) || values_end!=bytes::len(ref tree.values) || named!=tree.meta.named){return 1;}
    i=1;while(i<=total){let Node n=node(tree,i);let Expected e=*v::at<Expected>(ref expected,i-1);if(n.first!=e.first || n.last!=e.last || n.children!=e.children || (n.last!=0 && node(tree,n.last).next!=0) || ((n.kind==2 || n.kind>=5) && n.children!=0)){return 1;}i+=1;}
    i=0;while(i<size){if(*v::at<usize>(ref tree.buckets,i)!=*v::at<usize>(ref heads,i)){return 1;}i+=1;}return 0;
}
pub fn Result<Tree,Problem> publish(State meta,bytes::Buffer names,bytes::Buffer values,v::Vec<Node> nodes,v::Vec<usize> buckets){
    let Tree tree=Tree{.names,.values,.nodes,.buckets,.meta};let u32 code=verify(ref tree);if(code!=0){return .err(problem(ref tree.meta,code,0,0));}return .ok(tree);
}
// Rebuild the live graph from trusted parser scratch after replacement. All
// pools/indexes are rebuilt, then the normal publication verifier runs. The
// strict factory above still rejects forged/orphaned/tombstoned normal input.
pub fn Result<Tree,Problem> publish_compacted(State raw,bytes::Buffer names,bytes::Buffer values,v::Vec<Node> nodes,v::Vec<usize> buckets){
    let usize total=v::len<Node>(ref nodes);let Node root=entry(ref nodes,1);
    if(raw.source_bytes>1048576 || total==0 || total>raw.source_bytes+1 || root.kind!=1 || root.parent!=0 || root.state!=2 || bytes::len(ref names)>1048576 || bytes::len(ref values)>1048576){return .err(problem(ref raw,1,0,0));}
    var State meta=state(raw.source_bytes);var bytes::Buffer out_names=bytes::empty();var bytes::Buffer out_values=bytes::empty();var v::Vec<Node> out_nodes=v::empty<Node>();var v::Vec<usize> out_buckets=v::empty<usize>();var v::Vec<usize> map=v::empty<usize>();
    let u32 initialized=initialize(ref meta,ref mut out_nodes);if(initialized!=0){return .err(problem(ref meta,initialized,0,0));}meta.work=raw.work;if(!charge(ref mut meta,total)){return .err(problem(ref meta,4,0,0));}
    if(!v::reserve<usize>(ref mut map,total).is_ok){return .err(problem(ref meta,3,0,0));}var usize at=0;while(at<total){if(!v::push<usize>(ref mut map,0).is_ok){return .err(problem(ref meta,3,0,0));}at+=1;}number(ref mut map,0,1);
    at=2;while(at<=total){let Node n=entry(ref nodes,at);if(n.kind==9){at+=1;continue;}
        if(n.parent==0 || n.parent>=at || n.kind<1 || n.kind>8 || n.state>3 || n.name_start>n.name_end || n.name_end>bytes::len(ref names) || n.value_start>n.value_end || n.value_end>bytes::len(ref values) || !span(ref meta,n.start,n.end)){return .err(problem(ref meta,1,n.start,n.end));}
        let usize parent=*v::at<usize>(ref map,n.parent-1);if(parent==0){return .err(problem(ref meta,1,n.start,n.end));}
        let usize id=if(entry(ref out_nodes,parent).kind==1){switch(reserve_key(ref mut meta,ref mut out_names,ref mut out_nodes,ref mut out_buckets,parent,bytes::view(ref names)[n.name_start..n.name_end],n.start,n.end)){case .ok(x):{x}case .err(e):{return .err(e);}}}else{switch(element(ref mut meta,ref mut out_names,ref mut out_nodes,ref mut out_buckets,parent,n.start,n.end)){case .ok(x):{x}case .err(e):{return .err(e);}}};
        let result=if(n.kind==1 || n.kind==3 || n.kind==4){container(ref meta,ref mut out_nodes,id,n.kind,if(n.kind==1){if(n.state==3){2}else{n.state}}else{0},n.end)}else{if(n.kind==2){string(ref mut meta,ref mut out_values,ref mut out_nodes,id,bytes::view(ref values)[n.value_start..n.value_end],n.end)}else{if(n.kind==5){
            let ref u8[] value=bytes::view(ref values)[n.value_start..n.value_end];if(!bytes::equal(value,b"true") && !bytes::equal(value,b"false")){return .err(problem(ref meta,1,n.start,n.end));}boolean(ref mut meta,ref mut out_values,ref mut out_nodes,id,bytes::equal(value,b"true"),n.end)
        }else{atom(ref mut meta,ref mut out_values,ref mut out_nodes,id,bytes::view(ref values)[n.value_start..n.value_end],n.end,n.kind)}}};
        switch(result){case .ok(_):{}case .err(e):{return .err(e);}};number(ref mut map,at-1,id);at+=1;
    }
    at=2;while(at<=total){let Node n=entry(ref nodes,at);if(n.kind!=9 && n.state==3){let usize id=*v::at<usize>(ref map,at-1);var Node out=entry(ref out_nodes,id);out.state=3;set(ref mut out_nodes,id,out);}at+=1;}
    return publish(meta,out_names,out_values,out_nodes,out_buckets);
}
