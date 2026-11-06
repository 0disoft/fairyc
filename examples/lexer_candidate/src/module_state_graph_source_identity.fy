module module_state_graph_source_identity;
import module_state_graph_source as source;
import module_state_graph_model as graph;
import module_state_graph_hash_context as ctx;
import module_state_graph_body_hash as hash;
import module_state_graph_hash_encoding as enc;
import module_public_type_text as coordinates;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import module_registry as paths;
import syntax as syntax;
import std::bytes as bytes;

// Current-source prerequisite only: no Complete, cache or execution authority.
// Both nested resources own their bytes; neither input is retained as a borrow.
pub resource Bound{ctx::Context context;hash::Identity identity;cleanup{}}
pub fn ref ctx::Context context(ref Bound p){return ref p.context;}
pub fn ref hash::Identity identity(ref Bound p){return ref p.identity;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}

pub fn Result<Bound,u32> derive(ref source::Produced produced,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(owner>=512){return .err(1001);}
    // validate_owner deliberately ignores malformed rows for unrelated owners.
    // Its private 1MiB meter is measured separately from this 16MiB phase.
    var usize measured=0;let checked=coordinates::validate_owner(table,owner,ref mut measured);
    if(!budget::charge(work,measured)){return .err(1002);}if(checked!=0){return .err(mapped(checked));}
    var usize at=0;while(at<coordinates::coordinate_count(table)){
        if(!budget::charge(work,1)){return .err(1002);}if(coordinates::coordinate_at(table,at).owner==owner){break;}at+=1;
    }
    if(at==coordinates::coordinate_count(table)){return .err(1001);}
    let coordinate=switch(binary::decode(coordinates::coordinate_text(table,at),b"fypc1-sha256-",true,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};

    // Read only the Parsed retained by the sealed producer, never caller text.
    let parsed=source::parsed(produced);let target=source::owner(produced);
    if(target.module_index!=0 || target.declaration==0 || target.declaration>syntax::declaration_count(parsed)){return .err(1001);}
    let module_decl=syntax::declaration_at(parsed,0);let root=syntax::declaration_at(parsed,target.declaration-1);let raw=syntax::source_view(parsed);
    if(module_decl.kind!=1 || module_decl.name_start>=module_decl.name_end || module_decl.name_end>raw.len || root.kind!=3 || root.public || root.unsafe_end!=0 || root.owner_end!=0 || root.generic_parameters!=0 || root.body==0 || root.result_type==0){return .err(1001);}
    if(!bytes::equal(syntax::declaration_name(parsed,target.declaration),b"transaction") || syntax::type_at(parsed,root.result_type).kind!=1 || !bytes::equal(syntax::type_text(parsed,root.result_type),b"void")){return .err(1001);}
    var usize i=0;while(i<root.entry_count){if(!budget::charge(work,1)){return .err(1002);}if(syntax::entry_at(parsed,root.entry_start+i).kind==1){return .err(1001);}i+=1;}
    if(!budget::charge(work,module_decl.name_end-module_decl.name_start+1)){return .err(1002);}
    var bytes::Buffer name=bytes::empty();let normalized=paths::normalize(raw[module_decl.name_start..module_decl.name_end],ref mut name);if(normalized!=0){return .err(mapped(normalized));}
    let appended=budget::raw(ref mut name,b"::transaction",work);if(appended!=0){return .err(if(appended==4){1003}else{1002});}
    let mask=graph::header(source::graph(produced)).effects;if(mask>3){return .err(1001);}
    var bytes::Buffer descriptor=bytes::empty();var enc::Status status=enc::Status{.code=0};
    enc::text(ref mut descriptor,bytes::view(ref name),ref mut status,work);
    enc::number(ref mut descriptor,0,ref mut status,work);enc::text(ref mut descriptor,b"void",ref mut status,work);
    enc::number(ref mut descriptor,if(mask==3){2}else if(mask==0){0}else{1},ref mut status,work);
    if((mask&1)!=0){enc::text(ref mut descriptor,b"storage_read",ref mut status,work);}if((mask&2)!=0){enc::text(ref mut descriptor,b"storage_write",ref mut status,work);}
    enc::text(ref mut descriptor,b"",ref mut status,work);if(status.code!=0){return .err(status.code);}
    let context=switch(ctx::validate(bytes::view(ref coordinate),bytes::view(ref descriptor),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let identity=switch(hash::derive(source::program(produced),ref context,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    return .ok(Bound{.context,.identity});
}
