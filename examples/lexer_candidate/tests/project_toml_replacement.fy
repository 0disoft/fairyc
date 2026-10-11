module tests::project_toml_replacement;
import project_toml_model as m;
import std::bytes as bytes;
import std::vec as v;

fn usize key(ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize parent,ref u8[] name){return switch(m::reserve_key(meta,names,nodes,buckets,parent,name,0,1)){case .ok(id):{id}case .err(_):{assert(false);0}};}

@test
fn void rebuilds_replaced_arrays_and_discards_stale_key_value_storage(){
    var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();var v::Vec<m::Node> nodes=v::empty<m::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(m::initialize(ref meta,ref mut nodes)==0);
    let usize array=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,1,b"array");assert(m::container(ref meta,ref mut nodes,array,3,0,1).is_ok);
    let usize first=switch(m::element(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(m::string(ref mut meta,ref mut values,ref mut nodes,first,b"old",3).is_ok);
    let usize table=switch(m::element(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(m::container(ref meta,ref mut nodes,table,1,2,1).is_ok);
    let usize nested=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,table,b"nested");assert(m::boolean(ref mut meta,ref mut values,ref mut nodes,nested,true,4).is_ok);
    assert(m::replace_table(ref mut meta,ref mut nodes,array).is_ok);
    let usize stale=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,b"keep");assert(m::string(ref mut meta,ref mut values,ref mut nodes,stale,b"discard",7).is_ok);
    assert(m::replace_table(ref mut meta,ref mut nodes,array).is_ok);
    assert(switch(m::locate(ref mut meta,ref names,ref nodes,ref buckets,array,b"keep",0,1)){case .ok(id):{id==0}case .err(_):{false}});
    let usize keep=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,b"keep");assert(m::string(ref mut meta,ref mut values,ref mut nodes,keep,b"new",3).is_ok);
    assert(m::entry(ref nodes,first).kind==9 && m::entry(ref nodes,table).kind==9 && m::entry(ref nodes,nested).kind==9 && m::entry(ref nodes,stale).kind==9);
    switch(m::publish_compacted(meta,names,values,nodes,buckets)){case .ok(tree):{assert(m::count(ref tree)==3 && m::lookup(ref tree,1,b"array")==2 && m::lookup(ref tree,2,b"keep")==3 && m::node(ref tree,2).children==1);assert(bytes::equal(m::value(ref tree,3),b"new") && m::lookup(ref tree,2,b"nested")==0);}case .err(_):{assert(false);}};
}

@test
fn void ordinary_publication_rejects_discarded_scratch_nodes(){
    var m::State meta=m::state(64);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();var v::Vec<m::Node> nodes=v::empty<m::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(m::initialize(ref meta,ref mut nodes)==0);
    let usize table=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,1,b"table");assert(m::container(ref meta,ref mut nodes,table,1,2,1).is_ok);
    let usize child=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,table,b"child");assert(m::string(ref mut meta,ref mut values,ref mut nodes,child,b"x",1).is_ok);assert(m::replace_table(ref mut meta,ref mut nodes,table).is_ok);
    assert(!m::publish(meta,names,values,nodes,buckets).is_ok);
}

@test
fn void replaces_a_deep_branch_before_its_later_sibling(){
    var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();var v::Vec<m::Node> nodes=v::empty<m::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(m::initialize(ref meta,ref mut nodes)==0);
    let usize owner=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,1,b"owner");assert(m::container(ref meta,ref mut nodes,owner,1,2,1).is_ok);var usize parent=owner;var usize i=0;
    while(i<50){let usize child=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,parent,b"child");assert(m::container(ref meta,ref mut nodes,child,1,2,1).is_ok);parent=child;i+=1;}
    let usize deep=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,parent,b"leaf");assert(m::string(ref mut meta,ref mut values,ref mut nodes,deep,b"deep",4).is_ok);
    let usize sibling=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,owner,b"sibling");assert(m::string(ref mut meta,ref mut values,ref mut nodes,sibling,b"later",5).is_ok);assert(m::replace_table(ref mut meta,ref mut nodes,owner).is_ok);
    assert(m::entry(ref nodes,deep).kind==9 && m::entry(ref nodes,sibling).kind==9);
    switch(m::publish_compacted(meta,names,values,nodes,buckets)){case .ok(tree):{assert(m::count(ref tree)==2 && m::lookup(ref tree,1,b"owner")==2 && m::node(ref tree,2).children==0);}case .err(_):{assert(false);}};
}
