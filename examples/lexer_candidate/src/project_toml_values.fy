module project_toml_values;
import project_toml_model as m;
import project_toml_scalars as atoms;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Parser scratch uses value handles: the decoder's global mapping and an inline
// table's returned mapping can reference the same literal without copying it.
// Only the mapping reachable from root 1 is projected into the immutable model.
pub struct Value{usize first;usize last;usize count;usize start;usize end;usize data_start;usize data_end;u32 kind;u32 state;}
pub struct Slot{usize parent;usize value;usize start;usize end;usize name_start;usize name_end;usize next;usize hash;usize hash_next;}
fn Value blank(){return Value{.first=0,.last=0,.count=0,.start=0,.end=0,.data_start=0,.data_end=0,.kind=0,.state=0};}
fn Slot vacant(){return Slot{.parent=0,.value=0,.start=0,.end=0,.name_start=0,.name_end=0,.next=0,.hash=0,.hash_next=0};}
fn m::Problem fail(ref m::State meta,u32 code,usize start,usize end){let usize a=if(start<=meta.source_bytes){start}else{meta.source_bytes};let usize b=if(end<=meta.source_bytes){end}else{meta.source_bytes};return m::Problem{.code,.start=a,.end=if(b>=a){b}else{a}};}
fn bool span(ref m::State meta,usize start,usize end){return start<=end && end<=meta.source_bytes;}
pub fn bool charge(ref mut m::State meta,usize cost){if(meta.work>16777216 || cost>16777216-meta.work){return false;}meta.work+=cost;return true;}
pub fn Value node(ref v::Vec<Value> values,usize id){return if(id>0 && id<=v::len<Value>(values)){*v::at<Value>(values,id-1)}else{blank()};}
pub fn Slot slot(ref v::Vec<Slot> slots,usize id){return if(id>0 && id<=v::len<Slot>(slots)){*v::at<Slot>(slots,id-1)}else{vacant()};}
fn void number(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize out=v::at_mut<usize>(values,at);*out=value;}
fn void set_value(ref mut v::Vec<Value> values,usize id,Value value){let ref mut Value out=v::at_mut<Value>(values,id-1);*out=value;}
fn void set_slot(ref mut v::Vec<Slot> slots,usize id,Slot value){let ref mut Slot out=v::at_mut<Slot>(slots,id-1);*out=value;}
fn usize fingerprint(usize parent,ref u8[] key){var usize hash=parent%16777213;var usize i=0;while(i<key.len){hash=(hash*33+(key[i] as usize))%16777213;i+=1;}return hash;}
pub fn Result<usize,m::Problem> make(ref mut m::State meta,ref mut v::Vec<Value> values,u32 kind,u32 state,usize start,usize end){
    if(meta.source_bytes>1048576 || !span(meta,start,end) || kind<1 || kind>8 || state>2 || (kind!=1 && state!=0)){return .err(fail(meta,1,start,end));}
    if(v::len<Value>(values)>=meta.source_bytes+1 || !charge(meta,1)){return .err(fail(meta,4,start,end));}var Value n=blank();n.kind=kind;n.state=state;n.start=start;n.end=end;
    if(!v::push<Value>(values,n).is_ok){return .err(fail(meta,3,start,end));}return .ok(v::len<Value>(values));
}
pub fn Result<usize,m::Problem> atom(ref mut m::State meta,ref mut bytes::Buffer data,ref mut v::Vec<Value> values,u32 kind,ref u8[] bytes_value,usize start,usize end){
    if(kind!=2 && kind<5 || kind>8 || !text::validate_utf8(bytes_value).is_ok || (kind==5 && !bytes::equal(bytes_value,b"true") && !bytes::equal(bytes_value,b"false")) || (kind>=6 && atoms::kind(bytes_value)!=kind)){return .err(fail(meta,1,start,end));}
    if(bytes::len(data)>1048576 || bytes_value.len>1048576-bytes::len(data) || !charge(meta,bytes_value.len+1)){return .err(fail(meta,4,start,end));}
    let usize id=switch(make(meta,values,kind,0,start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};let usize offset=bytes::len(data);if(!bytes::append(data,bytes_value).is_ok){return .err(fail(meta,3,start,end));}var Value n=node(values,id);n.data_start=offset;n.data_end=bytes::len(data);set_value(values,id,n);return .ok(id);
}
pub fn Result<usize,m::Problem> finish(ref m::State meta,ref mut v::Vec<Value> values,usize id,usize end,u32 state){var Value n=node(values,id);if(n.kind!=1 && n.kind!=3 && n.kind!=4 || state>2 || (n.kind!=1 && state!=0) || !span(meta,n.start,end)){return .err(fail(meta,1,n.start,end));}n.end=end;n.state=state;set_value(values,id,n);return .ok(id);}
pub fn Result<usize,m::Problem> lookup(ref mut m::State meta,ref bytes::Buffer names,ref v::Vec<Value> values,ref v::Vec<Slot> slots,ref v::Vec<usize> buckets,usize parent,ref u8[] key,usize start,usize end){
    if(node(values,parent).kind!=1 || !span(meta,start,end)){return .err(fail(meta,1,start,end));}if(key.len>1048576 || !charge(meta,key.len+1)){return .err(fail(meta,4,start,end));}if(v::len<usize>(buckets)==0){return .ok(0);}
    let usize hash=fingerprint(parent,key);var usize id=*v::at<usize>(buckets,hash%v::len<usize>(buckets));var usize steps=0;while(id!=0){if(id>v::len<Slot>(slots) || steps>=v::len<Slot>(slots)){return .err(fail(meta,1,start,end));}if(!charge(meta,1)){return .err(fail(meta,4,start,end));}steps+=1;let Slot s=slot(slots,id);if(s.name_start>s.name_end || s.name_end>bytes::len(names)){return .err(fail(meta,1,start,end));}
        if(s.hash==hash && s.parent==parent && s.name_end-s.name_start==key.len){if(!charge(meta,key.len)){return .err(fail(meta,4,start,end));}if(bytes::equal(bytes::view(names)[s.name_start..s.name_end],key)){return .ok(id);}}id=s.hash_next;
    }return .ok(0);
}
fn u32 capacity(ref mut m::State meta,ref v::Vec<Value> values,ref mut v::Vec<Slot> slots,ref mut v::Vec<usize> buckets){
    let usize initial=v::len<usize>(buckets);var usize size=if(initial==0){64}else{initial};while(v::len<Slot>(slots)+1>size/2){size*=2;}if(size==initial){return 0;}if(!charge(meta,size+v::len<Slot>(slots))){return 4;}
    if(!v::reserve<usize>(buckets,size).is_ok){return 3;}while(v::len<usize>(buckets)<size){if(!v::push<usize>(buckets,0).is_ok){discard v::truncate<usize>(buckets,initial);return 3;}}var usize i=0;while(i<size){number(buckets,i,0);i+=1;}
    i=1;while(i<=v::len<Slot>(slots)){var Slot s=slot(slots,i);if(node(values,s.parent).kind==1){let usize bucket=s.hash%size;s.hash_next=*v::at<usize>(buckets,bucket);set_slot(slots,i,s);number(buckets,bucket,i);}i+=1;}return 0;
}
fn Result<usize,m::Problem> append(ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<Value> values,ref mut v::Vec<Slot> slots,ref mut v::Vec<usize> buckets,usize parent,ref u8[] key,usize value,usize start,usize end){
    var Value owner=node(values,parent);let Value item=node(values,value);let bool named=owner.kind==1;if((!named && owner.kind!=3 && owner.kind!=4) || item.kind==0 || owner.kind==4 && item.kind!=1 || !span(meta,start,end) || !text::validate_utf8(key).is_ok){return .err(fail(meta,1,start,end));}
    if(v::len<Slot>(slots)>=meta.source_bytes+1 || bytes::len(names)>1048576 || key.len>1048576-bytes::len(names) || !charge(meta,key.len+1)){return .err(fail(meta,4,start,end));}
    if(named){let u32 code=capacity(meta,values,slots,buckets);if(code!=0){return .err(fail(meta,code,start,end));}}let usize minimum=v::len<Slot>(slots)+1;if(!v::reserve<Slot>(slots,minimum).is_ok){return .err(fail(meta,3,start,end));}
    let usize initial=bytes::len(names);if(!bytes::append(names,key).is_ok){return .err(fail(meta,3,start,end));}var Slot s=vacant();s.parent=parent;s.value=value;s.start=start;s.end=end;s.name_start=initial;s.name_end=bytes::len(names);if(named){s.hash=fingerprint(parent,key);s.hash_next=*v::at<usize>(buckets,s.hash%v::len<usize>(buckets));}
    if(!v::push<Slot>(slots,s).is_ok){discard bytes::truncate(names,initial);return .err(fail(meta,3,start,end));}let usize id=v::len<Slot>(slots);if(owner.last!=0){var Slot previous=slot(slots,owner.last);previous.next=id;set_slot(slots,owner.last,previous);}if(owner.first==0){owner.first=id;}owner.last=id;owner.count+=1;set_value(values,parent,owner);if(named){let usize bucket=s.hash%v::len<usize>(buckets);number(buckets,bucket,id);}return .ok(id);
}
pub fn Result<usize,m::Problem> put(ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<Value> values,ref mut v::Vec<Slot> slots,ref mut v::Vec<usize> buckets,usize parent,ref u8[] key,usize value,usize start,usize end){
    let usize id=switch(lookup(meta,names,values,slots,buckets,parent,key,start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};if(node(values,value).kind==0){return .err(fail(meta,1,start,end));}if(id==0){return append(meta,names,values,slots,buckets,parent,key,value,start,end);}var Slot s=slot(slots,id);s.value=value;s.start=start;s.end=end;set_slot(slots,id,s);return .ok(id);
}
pub fn Result<usize,m::Problem> element(ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<Value> values,ref mut v::Vec<Slot> slots,ref mut v::Vec<usize> buckets,usize parent,usize value,usize start,usize end){if(node(values,parent).kind!=3 && node(values,parent).kind!=4){return .err(fail(meta,1,start,end));}return append(meta,names,values,slots,buckets,parent,b"",value,start,end);}
// Projection duplicates shared literals, skips unreachable scratch and never
// trusts scratch indexes or parent links as published ownership authority.
pub fn Result<m::Tree,m::Problem> publish(m::State raw,ref bytes::Buffer names,ref bytes::Buffer data,ref v::Vec<Value> values,ref v::Vec<Slot> slots){
    let Value root=node(values,1);if(root.kind!=1 || raw.source_bytes>1048576){return .err(fail(ref raw,1,0,0));}var m::State meta=m::state(raw.source_bytes);var bytes::Buffer out_names=bytes::empty();var bytes::Buffer out_data=bytes::empty();var v::Vec<m::Node> out_nodes=v::empty<m::Node>();var v::Vec<usize> out_buckets=v::empty<usize>();let u32 code=m::initialize(ref meta,ref mut out_nodes);if(code!=0){return .err(fail(ref meta,code,0,0));}meta.work=raw.work;
    var usize[513] parents=[0;513];var usize[513] owners=[0;513];var usize[513] next=[0;513];parents[0]=1;owners[0]=1;next[0]=root.first;var usize depth=1;
    while(depth>0){let usize at=depth-1;let usize current=next[at];if(current==0){depth-=1;continue;}if(!charge(ref mut meta,1)){return .err(fail(ref meta,4,0,0));}if(current>v::len<Slot>(slots)){return .err(fail(ref meta,1,0,0));}let Slot s=slot(slots,current);let Value n=node(values,s.value);
        if(s.parent!=owners[at] || n.kind<1 || n.kind>8 || s.name_start>s.name_end || s.name_end>bytes::len(names) || n.data_start>n.data_end || n.data_end>bytes::len(data) || !span(ref meta,s.start,s.end) || !span(ref meta,n.start,n.end)){return .err(fail(ref meta,1,s.start,s.end));}next[at]=s.next;
        let usize end=if(n.end>=s.start){n.end}else{s.end};let usize id=if(node(values,owners[at]).kind==1){switch(m::reserve_key(ref mut meta,ref mut out_names,ref mut out_nodes,ref mut out_buckets,parents[at],bytes::view(names)[s.name_start..s.name_end],s.start,end)){case .ok(x):{x}case .err(e):{return .err(e);}}}else{switch(m::element(ref mut meta,ref mut out_names,ref mut out_nodes,ref mut out_buckets,parents[at],s.start,end)){case .ok(x):{x}case .err(e):{return .err(e);}}};
        let built=if(n.kind==1 || n.kind==3 || n.kind==4){m::container(ref meta,ref mut out_nodes,id,n.kind,n.state,end)}else{if(n.kind==2){m::string(ref mut meta,ref mut out_data,ref mut out_nodes,id,bytes::view(data)[n.data_start..n.data_end],end)}else{if(n.kind==5){m::boolean(ref mut meta,ref mut out_data,ref mut out_nodes,id,bytes::equal(bytes::view(data)[n.data_start..n.data_end],b"true"),end)}else{m::atom(ref mut meta,ref mut out_data,ref mut out_nodes,id,bytes::view(data)[n.data_start..n.data_end],end,n.kind)}}};switch(built){case .ok(_):{}case .err(e):{return .err(e);}};
        if(n.first!=0){if(n.kind!=1 && n.kind!=3 && n.kind!=4){return .err(fail(ref meta,1,n.start,n.end));}if(depth==513){return .err(fail(ref meta,4,n.start,n.end));}parents[depth]=id;owners[depth]=s.value;next[depth]=n.first;depth+=1;}
    }return m::publish(meta,out_names,out_data,out_nodes,out_buckets);
}
