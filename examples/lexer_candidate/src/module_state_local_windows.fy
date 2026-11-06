module module_state_local_windows;
import module_state_local_codec as codec;
import module_state_local_schema as schema;
import module_state_layout as layout;
import windows_rooted_file as host;
import std::bytes as bytes;

// Every operation owns its independent, nonblocking OS lease. There is no
// transaction body, user callback or native execution while that lease is held.
pub resource Store{host::Directory root;host::StateLock authority;codec::Limits limits;cleanup{}}
pub struct Receipt{u32[8] layout;usize version;usize writes;}
// 1 pure admission, 2 open, 3 lease, 4 read, 5 record, 6 compare,
// 7 staging/write/sync, 8 atomic rename. operation retains rooted host stage.
pub struct Failure{u32 stage;u32 code;u32 os_code;u32 operation;bool observed;Receipt current;}
fn Receipt empty_receipt(){return Receipt{.layout=[0;8],.version=0,.writes=0};}
fn Failure fail(u32 stage,u32 code){return Failure{.stage,.code,.os_code=0,.operation=0,.observed=false,.current=empty_receipt()};}
fn Failure native(u32 stage,host::Problem cause){return Failure{.stage,.code=0,.os_code=cause.os_code,.operation=cause.stage,.observed=false,.current=empty_receipt()};}
fn Receipt receipt(ref codec::State state,usize writes){return Receipt{.layout=codec::layout_hash(state),.version=codec::version(state),.writes};}
fn Failure compared(u32 code,ref codec::State current){return Failure{.stage=6,.code,.os_code=0,.operation=0,.observed=true,.current=receipt(current,0)};}
fn Failure observed(Failure error,ref codec::State current){var out=error;out.observed=true;out.current=receipt(current,0);return out;}
fn bool bytes_charge(ref mut codec::Work work,usize n){if(work.bytes>1073741824 || n>1073741824-work.bytes){return false;}work.bytes+=n;return true;}
fn bool structure_charge(ref mut codec::Work work,usize n){if(work.structure>16777216 || n>16777216-work.structure){return false;}work.structure+=n;return true;}
fn u32 ready(codec::Limits limits,ref mut codec::Work work){if(!codec::valid(limits)){return 1001;}return if(structure_charge(work,1) && bytes_charge(work,0)){0}else{1002};}

@effects(filesystem) fn Result<bytes::Buffer,Failure> read(ref host::Directory root,codec::Limits limits,ref mut codec::Work work){
    var file=switch(host::open_read(root,b"state-v1.bin")){case .ok(p):{p}case .err(e):{return .err(native(4,e));}};
    let size=switch(host::size(ref file)){case .ok(n):{n}case .err(e):{return .err(native(4,e));}};
    if(size>limits.bytes || !bytes_charge(work,(size+1)*2)){return .err(fail(4,1002));}
    var output=bytes::empty();if(!bytes::resize_zeroed(ref mut output,size+1).is_ok){return .err(fail(4,1003));}var usize used=0;
    while(used<=size){let usize n=if(size+1-used>65536){65536}else{size+1-used};let copied=switch(host::read(ref mut file,bytes::view_mut(ref mut output)[used..used+n])){case .ok(n):{n}case .err(e):{return .err(native(4,e));}};if(copied==0){break;}used+=copied;}
    if(used!=size){return .err(fail(4,1002));}switch(host::close(file)){case .ok(_):{}case .err(e):{return .err(native(4,e));}};
    discard bytes::truncate(ref mut output,size);return .ok(output);
}
@effects(filesystem) fn Result<codec::State,Failure> load(ref host::Directory root,codec::Limits limits,ref mut codec::Work work){
    let data=read(root,limits,work)?;let current=switch(schema::read(bytes::view(ref data),limits,work)){case .ok(p):{p}case .err(code):{return .err(fail(5,code));}};
    return switch(codec::decode(ref current,0,bytes::view(ref data),limits,work)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(5,code))}};
}
@effects(filesystem,clock) fn Result<host::File,Failure> temporary(ref host::Directory root,ref mut codec::Work work){
    let stamp=host::now();var usize slot=0;while(slot<4096){if(!structure_charge(work,1)){return .err(fail(7,1002));}var name=bytes::empty();
        if(!bytes::append(ref mut name,b".tmp-").is_ok || !bytes::append_usize(ref mut name,stamp).is_ok || !bytes::append_byte(ref mut name,45).is_ok || !bytes::append_usize(ref mut name,slot).is_ok){return .err(fail(7,1003));}
        switch(host::create_new(root,bytes::view(ref name))){case .ok(p):{return .ok(p);}case .err(e):{if(!host::collision(e)){return .err(native(7,e));}}};slot+=1;
    }return .err(fail(7,1002));
}
@effects(filesystem,clock) fn Result<bool,Failure> publish(ref host::Directory root,ref u8[] encoded,bool replace,ref mut codec::Work work){
    if(!bytes_charge(work,encoded.len)){return .err(fail(7,1002));}var file=temporary(root,work)?;
    switch(host::write_all(ref mut file,encoded)){case .ok(n):{if(n!=encoded.len){return .err(fail(7,1001));}}case .err(e):{return .err(native(7,e));}};
    switch(host::sync(ref mut file)){case .ok(_):{}case .err(e):{return .err(native(7,e));}};
    // Match the Go provider's regular-destination guard immediately before its
    // publication decision. Missing is valid only as a rename destination.
    if(replace){switch(host::open_read(root,b"state-v1.bin")){case .ok(_):{}case .err(e):{if(!host::missing(e)){return .err(native(8,e));}}};};
    let result=if(replace){host::publish(root,ref mut file,b"state-v1.bin")}else{host::publish_new(root,ref mut file,b"state-v1.bin")};
    switch(result){case .ok(_):{}case .err(e):{return .err(native(8,e));}};
    // Rename is the decision. Resource cleanup closes the published handle;
    // there is no fallible post-decision operation that hides known success.
    return .ok(true);
}
@effects(filesystem,clock) pub fn Result<Store,Failure> open(ref u8[] path,ref layout::Plan initial,usize selected,codec::Limits limits,bool create,ref mut codec::Work work){
    let code=ready(limits,work);if(code!=0){return .err(fail(1,code));}
    let empty=switch(codec::initial(initial,selected,limits,work)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    let encoded=switch(codec::encode(ref empty,limits,work)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    let root=switch(host::open(path)){case .ok(p):{p}case .err(e):{return .err(native(2,e));}};
    let authority=switch(host::state_lock_open(ref root,create)){case .ok(p):{p}case .err(e):{return .err(native(2,e));}};
    let lease=switch(host::state_lock_acquire(ref root,ref authority)){case .ok(p):{p}case .err(e):{return .err(native(3,e));}};
    if(create){
        switch(host::open_read(ref root,b"state-v1.bin")){case .ok(_):{return .err(fail(2,1008));}case .err(e):{if(!host::missing(e)){return .err(native(2,e));}}};
        discard publish(ref root,bytes::view(ref encoded),false,work)?;
    }else{
        let current=load(ref root,limits,work)?;if(!layout::equal(codec::layout_hash(ref current),codec::layout_hash(ref empty))){return .err(compared(1004,ref current));}
    }
    return .ok(Store{.root,.authority,.limits});
}
@effects(filesystem) pub fn Result<codec::State,Failure> snapshot(ref Store store,u32[8] expected_layout,ref mut codec::Work work){
    let code=ready(store.limits,work);if(code!=0){return .err(fail(1,code));}
    let lease=switch(host::state_lock_acquire(ref store.root,ref store.authority)){case .ok(p):{p}case .err(e):{return .err(native(3,e));}};
    let current=load(ref store.root,store.limits,work)?;if(!layout::equal(codec::layout_hash(ref current),expected_layout)){return .err(compared(1004,ref current));}return .ok(current);
}
@effects(filesystem,clock) pub fn Result<Receipt,Failure> compare_and_publish(ref Store store,u32[8] expected_layout,usize expected_version,ref codec::Mutations writes,ref mut codec::Work work){
    let code=ready(store.limits,work);if(code!=0){return .err(fail(1,code));}
    // All caller mutation validation/sorting happens before lock or state I/O.
    let canonical=switch(codec::canonical(writes,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    let lease=switch(host::state_lock_acquire(ref store.root,ref store.authority)){case .ok(p):{p}case .err(e):{return .err(native(3,e));}};
    let current=load(ref store.root,store.limits,work)?;
    let next=switch(codec::apply(ref current,expected_layout,expected_version,ref canonical,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(compared(code,ref current));}};
    if(codec::mutation_count(ref canonical)==0){return .ok(receipt(ref next,0));}
    let encoded=switch(codec::encode(ref next,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(compared(code,ref current));}};
    switch(publish(ref store.root,bytes::view(ref encoded),true,work)){case .ok(_):{}case .err(e):{return .err(observed(e,ref current));}};return .ok(receipt(ref next,codec::mutation_count(ref canonical)));
}
@effects(filesystem,clock) pub fn Result<Receipt,Failure> publish_layout(ref Store store,u32[8] expected_layout,usize expected_version,ref layout::Plan next,usize selected,ref mut codec::Work work){
    let code=ready(store.limits,work);if(code!=0){return .err(fail(1,code));}
    // Validate the supplied schema before acquiring any storage authority.
    let validated=switch(codec::initial(next,selected,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    let lease=switch(host::state_lock_acquire(ref store.root,ref store.authority)){case .ok(p):{p}case .err(e):{return .err(native(3,e));}};
    let current=load(ref store.root,store.limits,work)?;
    let changed=switch(codec::upgrade(ref current,expected_layout,expected_version,next,selected,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(compared(code,ref current));}};
    if(layout::equal(codec::layout_hash(ref current),codec::layout_hash(ref validated))){return .ok(receipt(ref changed,0));}
    let encoded=switch(codec::encode(ref changed,store.limits,work)){case .ok(p):{p}case .err(code):{return .err(compared(code,ref current));}};
    switch(publish(ref store.root,bytes::view(ref encoded),true,work)){case .ok(_):{}case .err(e):{return .err(observed(e,ref current));}};return .ok(receipt(ref changed,0));
}
