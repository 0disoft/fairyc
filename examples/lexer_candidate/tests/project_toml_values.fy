module tests::project_toml_values;
import project_toml_values as g;
import project_toml_model as m;
import std::bytes as bytes;
import std::vec as v;
fn usize make(ref mut m::State meta,ref mut v::Vec<g::Value> values,u32 kind){return switch(g::make(meta,values,kind,if(kind==1){2}else{0},0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};}
fn usize text(ref mut m::State meta,ref mut bytes::Buffer data,ref mut v::Vec<g::Value> values,ref u8[] value){return switch(g::atom(meta,data,values,2,value,0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};}
@test
fn void projects_shared_values_and_only_the_final_global_mapping(){
    var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer data=bytes::empty();var v::Vec<g::Value> values=v::empty<g::Value>();var v::Vec<g::Slot> slots=v::empty<g::Slot>();var v::Vec<usize> buckets=v::empty<usize>();assert(make(ref mut meta,ref mut values,1)==1);
    let usize old=make(ref mut meta,ref mut values,1);let usize stale=text(ref mut meta,ref mut data,ref mut values,b"discarded");assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,old,b"stale",stale,0,1).is_ok);assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"a",old,0,1).is_ok);
    let usize returned=make(ref mut meta,ref mut values,1);let usize item=text(ref mut meta,ref mut data,ref mut values,b"shared");assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,old,b"b",item,0,1).is_ok && g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,returned,b"b",item,0,1).is_ok);
    assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"a",returned,0,1).is_ok && g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"copy",returned,0,1).is_ok);
    switch(g::publish(meta,ref names,ref data,ref values,ref slots)){case .ok(tree):{assert(m::count(ref tree)==5);let usize a=m::lookup(ref tree,1,b"a");let usize copy=m::lookup(ref tree,1,b"copy");assert(a!=copy && m::lookup(ref tree,a,b"stale")==0 && m::lookup(ref tree,copy,b"stale")==0 && bytes::equal(m::value(ref tree,m::lookup(ref tree,a,b"b")),b"shared") && bytes::equal(m::value(ref tree,m::lookup(ref tree,copy,b"b")),b"shared"));}case .err(_):{assert(false);}};
}
@test
fn void preserves_arrays_empty_names_and_reindexed_mapping_identity(){
    var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer data=bytes::empty();var v::Vec<g::Value> values=v::empty<g::Value>();var v::Vec<g::Slot> slots=v::empty<g::Slot>();var v::Vec<usize> buckets=v::empty<usize>();assert(make(ref mut meta,ref mut values,1)==1);let usize array=make(ref mut meta,ref mut values,4);assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"a.b",array,0,1).is_ok);
    let usize first=make(ref mut meta,ref mut values,1);let usize second=make(ref mut meta,ref mut values,1);assert(g::element(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,array,first,0,1).is_ok && g::element(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,array,second,0,1).is_ok);let usize item=text(ref mut meta,ref mut data,ref mut values,b"value");assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,first,b"",item,0,1).is_ok);
    var u8[2] key=[65,65];var usize i=0;while(i<50){assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,second,ref key,item,0,1).is_ok);key[1]+=1;if(key[1]==91){key[1]=65;key[0]+=1;}i+=1;}
    let usize found=switch(g::lookup(ref mut meta,ref names,ref values,ref slots,ref buckets,second,b"AA",0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(found!=0 && g::slot(ref slots,found).value==item);
    switch(g::publish(meta,ref names,ref data,ref values,ref slots)){case .ok(tree):{let usize a=m::lookup(ref tree,1,b"a.b");let m::Node n=m::node(ref tree,a);assert(n.kind==4 && n.children==2 && bytes::equal(m::value(ref tree,m::lookup(ref tree,n.first,b"")),b"value") && m::node(ref tree,n.last).children==50);}case .err(_):{assert(false);}};
}
@test
fn void rejects_invalid_references_and_bounds_a_cyclic_scratch_value(){
    var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer data=bytes::empty();var v::Vec<g::Value> values=v::empty<g::Value>();var v::Vec<g::Slot> slots=v::empty<g::Slot>();var v::Vec<usize> buckets=v::empty<usize>();assert(make(ref mut meta,ref mut values,1)==1);assert(!g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"bad",99,0,1).is_ok);let usize array=make(ref mut meta,ref mut values,4);let usize scalar=text(ref mut meta,ref mut data,ref mut values,b"x");assert(!g::element(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,array,scalar,0,1).is_ok);
    assert(g::put(ref mut meta,ref mut names,ref mut values,ref mut slots,ref mut buckets,1,b"cycle",1,0,1).is_ok);switch(g::publish(meta,ref names,ref data,ref values,ref slots)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==4);}};
    meta.work=16777216;assert(!g::make(ref mut meta,ref mut values,1,0,0,1).is_ok);
}
