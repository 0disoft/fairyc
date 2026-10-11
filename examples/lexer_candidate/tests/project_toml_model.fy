module tests::project_toml_model;
import project_toml_model as model;
import std::bytes as bytes;
import std::vec as v;

fn usize key(ref mut model::State state,ref mut bytes::Buffer names,ref mut v::Vec<model::Node> nodes,ref mut v::Vec<usize> buckets,usize parent,ref u8[] name){
    return switch(model::reserve_key(state,names,nodes,buckets,parent,name,0,1)){case .ok(id):{id}case .err(_):{assert(false);0}};
}
fn usize element(ref mut model::State state,ref mut bytes::Buffer names,ref mut v::Vec<model::Node> nodes,ref mut v::Vec<usize> buckets,usize parent){
    return switch(model::element(state,names,nodes,buckets,parent,0,1)){case .ok(id):{id}case .err(_):{assert(false);0}};
}

@test
fn void preserves_key_components_arrays_and_closed_tables(){
    var model::State state=model::state(4096);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();
    var v::Vec<model::Node> nodes=v::empty<model::Node>();var v::Vec<usize> buckets=v::empty<usize>();
    assert(model::initialize(ref state,ref mut nodes)==0);
    let usize quoted=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"a.b");
    assert(model::string(ref mut state,ref mut values,ref mut nodes,quoted,b"quoted",6).is_ok);
    let usize table=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"a");
    assert(model::container(ref state,ref mut nodes,table,1,0,1).is_ok);
    let usize dotted=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,table,b"b");
    assert(model::string(ref mut state,ref mut values,ref mut nodes,dotted,b"nested",6).is_ok);
    assert(model::declare(ref state,ref mut nodes,table,10,13).is_ok);
    assert(!model::declare(ref state,ref mut nodes,table,10,13).is_ok);
    let usize array=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"members");
    assert(model::container(ref state,ref mut nodes,array,3,0,1).is_ok);
    let usize item=element(ref mut state,ref mut names,ref mut nodes,ref mut buckets,array);
    assert(model::string(ref mut state,ref mut values,ref mut nodes,item,b"libs/net",10).is_ok);
    let usize tables=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"tables");
    assert(model::container(ref state,ref mut nodes,tables,4,0,1).is_ok);
    let usize first=element(ref mut state,ref mut names,ref mut nodes,ref mut buckets,tables);
    assert(model::container(ref state,ref mut nodes,first,1,2,1).is_ok);
    let usize second=element(ref mut state,ref mut names,ref mut nodes,ref mut buckets,tables);
    assert(model::container(ref state,ref mut nodes,second,1,2,1).is_ok);
    let usize nested=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,second,b"inline");
    assert(model::container(ref state,ref mut nodes,nested,1,1,1).is_ok);
    let usize child=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,nested,b"");
    assert(model::string(ref mut state,ref mut values,ref mut nodes,child,b"\xc3\xbf\x00",3).is_ok);
    assert(model::close(ref mut state,ref mut nodes,second,20).is_ok);
    assert(!model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,nested,b"late",0,1).is_ok);
    assert(!model::close(ref mut state,ref mut nodes,second,20).is_ok);
    assert(!model::close(ref mut state,ref mut nodes,1,20).is_ok);
    // Force multiple index rebuilds; the dot above remains a single component.
    var usize i=0;var u8[4] spelling=[0;4];spelling[0]=107;spelling[1]=48;spelling[2]=48;spelling[3]=48;
    while(i<150){
        let usize id=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,ref spelling);
        assert(model::string(ref mut state,ref mut values,ref mut nodes,id,ref spelling,4).is_ok);i+=1;spelling[3]+=1;if(spelling[3]==58){spelling[3]=48;spelling[2]+=1;if(spelling[2]==58){spelling[2]=48;spelling[1]+=1;}}
    }
    switch(model::publish(state,names,values,nodes,buckets)){case .ok(tree):{
    assert(model::source_bytes(ref tree)==4096 && model::count(ref tree)==161);
    assert(model::lookup(ref tree,1,b"a.b")==quoted && model::lookup(ref tree,table,b"b")==dotted);
    assert(model::lookup(ref tree,1,b"b")==0 && model::lookup(ref tree,nested,b"")==child);
    assert(bytes::equal(model::value(ref tree,quoted),b"quoted") && bytes::equal(model::value(ref tree,dotted),b"nested"));
    assert(model::node(ref tree,array).first==item && model::node(ref tree,tables).children==2 && model::node(ref tree,first).next==second);
    assert(model::node(ref tree,nested).state==3 && model::node(ref tree,child).state==3);
    assert(bytes::equal(model::value(ref tree,child),b"\xc3\xbf\x00"));
    i=0;spelling[1]=48;spelling[2]=48;spelling[3]=48;while(i<150){
        let usize id=model::lookup(ref tree,1,ref spelling);assert(id==12+i && bytes::equal(model::name(ref tree,id),ref spelling) && bytes::equal(model::value(ref tree,id),ref spelling));i+=1;spelling[3]+=1;if(spelling[3]==58){spelling[3]=48;spelling[2]+=1;if(spelling[2]==58){spelling[2]=48;spelling[1]+=1;}}
    }
    }case .err(_):{assert(false);}};
}

@test
fn void rejects_duplicates_bad_payloads_and_forged_graphs(){
    var usize mutation=0;
    while(mutation<16){
        var model::State state=model::state(64);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();
        var v::Vec<model::Node> nodes=v::empty<model::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(model::initialize(ref state,ref mut nodes)==0);
        let usize id=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"name");
        assert(!model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"name",0,1).is_ok);
        assert(!model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"\xff",0,1).is_ok);
        assert(!model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"bad-span",65,66).is_ok);
        assert(!model::string(ref mut state,ref mut values,ref mut nodes,id,b"\xff",1).is_ok);
        assert(v::len<model::Node>(ref nodes)==2 && bytes::equal(bytes::view(ref names),b"name") && bytes::len(ref values)==0);
        assert(model::string(ref mut state,ref mut values,ref mut nodes,id,b"value",5).is_ok);
        var model::Node n=model::entry(ref nodes,id);
        if(mutation==0){n.parent=id;}if(mutation==1){n.next=id;}if(mutation==2){n.first=id;n.last=id;n.children=1;}
        if(mutation==3){n.depth=512;}if(mutation==4){n.hash_next=id;}if(mutation==5){n.hash+=1;}
        if(mutation==6){n.name_end=100;}if(mutation==7){n.value_end=100;}if(mutation==8){n.kind=0;}
        if(mutation==9){n.state=2;}if(mutation==10){n.start=6;n.end=5;}
        {let ref mut model::Node slot=v::at_mut<model::Node>(ref mut nodes,id-1);*slot=n;}
        if(mutation==11){let usize bucket=n.hash%v::len<usize>(ref buckets);let ref mut usize head=v::at_mut<usize>(ref mut buckets,bucket);*head=0;}
        if(mutation==12){state.named+=1;}if(mutation==13){assert(bytes::append(ref mut names,b"orphan").is_ok);}
        if(mutation==14){assert(bytes::append(ref mut values,b"orphan").is_ok);}if(mutation==15){let ref mut model::Node root=v::at_mut<model::Node>(ref mut nodes,0);root.children=0;}
        assert(!model::publish(state,names,values,nodes,buckets).is_ok);mutation+=1;
    }
    // An array-of-tables cannot publish an arbitrary string element.
    var model::State state=model::state(64);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();
    var v::Vec<model::Node> nodes=v::empty<model::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(model::initialize(ref state,ref mut nodes)==0);
    let usize array=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"a");assert(model::container(ref state,ref mut nodes,array,4,0,1).is_ok);
    let usize child=element(ref mut state,ref mut names,ref mut nodes,ref mut buckets,array);assert(model::string(ref mut state,ref mut values,ref mut nodes,child,b"x",1).is_ok);
    assert(!model::publish(state,names,values,nodes,buckets).is_ok);
}

@test
fn void bounds_depth_input_work_and_decoded_bytes(){
    var model::State state=model::state(1048577);var v::Vec<model::Node> nodes=v::empty<model::Node>();assert(model::initialize(ref state,ref mut nodes)==4);
    state=model::state(2048);assert(model::initialize(ref state,ref mut nodes)==0);
    var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();var v::Vec<usize> buckets=v::empty<usize>();
    var usize parent=1;var usize i=0;while(i<512){let usize child=key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,parent,b"x");assert(model::container(ref state,ref mut nodes,child,1,0,1).is_ok);parent=child;i+=1;}
    let u32 depth=switch(model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,parent,b"x",0,1)){case .ok(_):{0}case .err(e):{e.code}};assert(depth==4);
    state.work=16777213;let u32 collision=switch(model::locate(ref mut state,ref names,ref nodes,ref buckets,1,b"x",0,1)){case .ok(_):{0}case .err(e):{e.code}};assert(collision==4 && state.work==16777216);
    let u32 work=switch(model::reserve_key(ref mut state,ref mut names,ref mut nodes,ref mut buckets,1,b"late",0,1)){case .ok(_):{0}case .err(e):{e.code}};assert(work==4);
    switch(model::publish(state,names,values,nodes,buckets)){case .ok(tree):{assert(model::count(ref tree)==513);}case .err(_):{assert(false);}};
    var model::State limited=model::state(16);var v::Vec<model::Node> limited_nodes=v::empty<model::Node>();var v::Vec<usize> limited_buckets=v::empty<usize>();var bytes::Buffer limited_names=bytes::empty();var bytes::Buffer limited_values=bytes::empty();assert(model::initialize(ref limited,ref mut limited_nodes)==0);
    let usize id=key(ref mut limited,ref mut limited_names,ref mut limited_nodes,ref mut limited_buckets,1,b"a");
    var bytes::Buffer payload=bytes::empty();assert(bytes::resize_zeroed(ref mut payload,1048577).is_ok);
    let u32 too_big=switch(model::string(ref mut limited,ref mut limited_values,ref mut limited_nodes,id,bytes::view(ref payload),1)){case .ok(_):{0}case .err(e):{e.code}};
    assert(too_big==4 && bytes::len(ref limited_values)==0 && model::entry(ref limited_nodes,id).kind==0);
}
