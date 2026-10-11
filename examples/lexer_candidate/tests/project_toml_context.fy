module tests::project_toml_context;
import project_toml_context as c;
import project_toml_model as m;
import std::bytes as bytes;
import std::vec as v;

fn usize intern(ref mut bytes::Buffer ctx_names,ref mut v::Vec<c::Entry> ctx_entries,ref mut v::Vec<usize> ctx_buckets,ref mut m::State meta,usize parent,ref u8[] key){return switch(c::intern(ctx_names,ctx_entries,ctx_buckets,meta,parent,key,0,1)){case .ok(id):{id}case .err(_):{assert(false);0}};}
fn usize key(ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize parent,ref u8[] name){return switch(m::reserve_key(meta,names,nodes,buckets,parent,name,0,1)){case .ok(id):{id}case .err(_):{assert(false);0}};}
fn usize physical(ref mut bytes::Buffer ctx_names,ref mut v::Vec<c::Entry> ctx_entries,ref mut v::Vec<usize> ctx_buckets,ref mut m::State meta,ref bytes::Buffer names,ref v::Vec<m::Node> nodes,usize id){return switch(c::physical(ctx_names,ctx_entries,ctx_buckets,meta,names,nodes,id)){case .ok(n):{n}case .err(_):{assert(false);0}};}

@test
fn void tracks_decoder_implicit_and_type_transitions_by_component(){
    var bytes::Buffer ctx_names=bytes::empty();var v::Vec<c::Entry> ctx_entries=v::empty<c::Entry>();var v::Vec<usize> ctx_buckets=v::empty<usize>();var m::State meta=m::state(4096);let usize a=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,0,b"a");let usize b=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,a,b"b");let usize quoted=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,0,b"a.b");assert(a!=b && b!=quoted);
    assert(c::mark_implicit(ref mut ctx_entries,ref meta,a).is_ok);assert(switch(c::define(ref mut ctx_entries,ref meta,a,false)){case .ok(action):{action==0}case .err(_):{false}});assert(c::flags(ref ctx_entries,a).implicit);
    assert(switch(c::define(ref mut ctx_entries,ref meta,a,true)){case .ok(action):{action==1}case .err(_):{false}});assert(!c::define(ref mut ctx_entries,ref meta,a,true).is_ok);
    assert(c::set_kind(ref mut ctx_entries,ref meta,a,3).is_ok && c::mark_implicit(ref mut ctx_entries,ref meta,a).is_ok);assert(switch(c::define(ref mut ctx_entries,ref meta,a,true)){case .ok(action):{action==2}case .err(_):{false}});
    assert(!c::flags(ref ctx_entries,a).implicit && c::flags(ref ctx_entries,a).kind==3);assert(c::mark_implicit(ref mut ctx_entries,ref meta,a).is_ok);assert(switch(c::define(ref mut ctx_entries,ref meta,a,true)){case .ok(action):{action==2}case .err(_):{false}});
    assert(c::set_kind(ref mut ctx_entries,ref meta,a,4).is_ok && c::mark_implicit(ref mut ctx_entries,ref meta,a).is_ok);assert(switch(c::define(ref mut ctx_entries,ref meta,a,true)){case .ok(action):{action==1}case .err(_):{false}});assert(c::flags(ref ctx_entries,b).kind==0 && c::flags(ref ctx_entries,quoted).kind==0);
    let usize root_empty=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,0,b"");let usize nested_empty=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,a,b"");assert(root_empty!=nested_empty && root_empty!=0);
    assert(c::record_type(ref mut ctx_entries,ref meta,root_empty,2).is_ok && c::record_type(ref mut ctx_entries,ref meta,nested_empty,3).is_ok);assert(c::flags(ref ctx_entries,root_empty).kind==2 && c::flags(ref ctx_entries,a).kind==3 && c::flags(ref ctx_entries,nested_empty).kind==0);
    assert(intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,a,b"b")==b && intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,0,b"a.b")==quoted);assert(!c::mark_implicit(ref mut ctx_entries,ref meta,0).is_ok && !c::set_kind(ref mut ctx_entries,ref meta,a,9).is_ok);
}

@test
fn void shares_metadata_between_physical_array_table_entries_and_replacements(){
    var bytes::Buffer ctx_names=bytes::empty();var v::Vec<c::Entry> ctx_entries=v::empty<c::Entry>();var v::Vec<usize> ctx_buckets=v::empty<usize>();var m::State meta=m::state(4096);var bytes::Buffer names=bytes::empty();var v::Vec<m::Node> nodes=v::empty<m::Node>();var v::Vec<usize> buckets=v::empty<usize>();assert(m::initialize(ref meta,ref mut nodes)==0);
    let usize array=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,1,b"a");assert(m::container(ref meta,ref mut nodes,array,4,0,1).is_ok);let usize a=physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,array);var usize prior=0;var usize i=0;
    while(i<2){let usize row=switch(m::element(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,0,1)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(m::container(ref meta,ref mut nodes,row,1,2,1).is_ok);
        let usize b=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,row,b"b");assert(m::container(ref meta,ref mut nodes,b,1,0,1).is_ok);let usize logical=physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,b);assert(physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,row)==a);
        if(i==0){prior=logical;assert(c::mark_implicit(ref mut ctx_entries,ref meta,logical).is_ok && c::set_kind(ref mut ctx_entries,ref meta,logical,3).is_ok);}else{assert(logical==prior && c::flags(ref ctx_entries,logical).implicit && c::flags(ref ctx_entries,logical).kind==3);}i+=1;
    }
    assert(m::replace_table(ref mut meta,ref mut nodes,array).is_ok);let usize fresh=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,b"b");assert(m::container(ref meta,ref mut nodes,fresh,1,2,1).is_ok);assert(physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,fresh)==prior && c::flags(ref ctx_entries,prior).kind==3);assert(!c::physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,3).is_ok);
    let usize empty=key(ref mut meta,ref mut names,ref mut nodes,ref mut buckets,array,b"");assert(m::container(ref meta,ref mut nodes,empty,1,2,1).is_ok);assert(physical(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,ref names,ref nodes,empty)!=a);
}

@test
fn void meters_depth_nodes_and_collision_chains(){
    var bytes::Buffer ctx_names=bytes::empty();var v::Vec<c::Entry> ctx_entries=v::empty<c::Entry>();var v::Vec<usize> ctx_buckets=v::empty<usize>();var m::State meta=m::state(4096);var usize id=0;var usize i=0;while(i<512){id=intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,id,b"x");i+=1;}assert(!c::intern(ref mut ctx_names,ref mut ctx_entries,ref mut ctx_buckets,ref mut meta,id,b"x",0,1).is_ok);
    var bytes::Buffer small_names=bytes::empty();var v::Vec<c::Entry> small_entries=v::empty<c::Entry>();var v::Vec<usize> small_buckets=v::empty<usize>();var m::State tiny=m::state(1);assert(c::intern(ref mut small_names,ref mut small_entries,ref mut small_buckets,ref mut tiny,0,b"",0,1).is_ok && c::intern(ref mut small_names,ref mut small_entries,ref mut small_buckets,ref mut tiny,0,b"a",0,1).is_ok);assert(!c::intern(ref mut small_names,ref mut small_entries,ref mut small_buckets,ref mut tiny,0,b"b",0,1).is_ok);
    var bytes::Buffer collisions_names=bytes::empty();var v::Vec<c::Entry> collisions_entries=v::empty<c::Entry>();var v::Vec<usize> collisions_buckets=v::empty<usize>();var m::State work=m::state(4096);var u8[2] name=[65,90];i=0;while(i<26){let usize item=intern(ref mut collisions_names,ref mut collisions_entries,ref mut collisions_buckets,ref mut work,0,ref name);assert(c::set_kind(ref mut collisions_entries,ref work,item,2).is_ok);name[0]+=1;name[1]-=1;i+=1;}
    name[0]=65;name[1]=90;i=0;while(i<26){let usize found=intern(ref mut collisions_names,ref mut collisions_entries,ref mut collisions_buckets,ref mut work,0,ref name);assert(c::flags(ref collisions_entries,found).kind==2);name[0]+=1;name[1]-=1;i+=1;}
    work.work=16777216;assert(!c::intern(ref mut collisions_names,ref mut collisions_entries,ref mut collisions_buckets,ref mut work,0,b"AZ",0,1).is_ok);assert(!c::intern(ref mut collisions_names,ref mut collisions_entries,ref mut collisions_buckets,ref mut meta,0,b"\xff",0,1).is_ok);
}
