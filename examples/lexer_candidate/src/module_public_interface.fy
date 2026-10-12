module module_public_interface;
import module_public_interface_model as model;
import module_public_interface_write as write;
import module_state_interface as state_interface;
import module_state_layout as state;
import sha256_stream as sha;
import std::bytes as bytes;

fn Result<i32,u32> order(ref model::Descriptor p,model::Text a,model::Text b,ref mut usize work){if(!model::valid_text(p,a) || !model::valid_text(p,b)){return .err(9);}let left=model::text(p,a);let right=model::text(p,b);var usize i=0;while(i<left.len && i<right.len){if(!state::charge(work,1)){return .err(10);}if(left[i]!=right[i]){return .ok(if(left[i]<right[i]){-1}else{1});}i+=1;}return .ok(if(left.len<right.len){-1}else if(left.len>right.len){1}else{0});}
fn u32 abilities(ref model::Descriptor p,model::Item item,ref mut usize work){if(!model::valid_range(item.abilities,model::list_count(p)) || !model::valid_text(p,item.kind)){return 9;}if(!state::charge(work,item.kind.end-item.kind.start+1)){return 10;}if(!bytes::equal(model::text(p,item.kind),b"resource")){return if(item.abilities.count==0){0}else{9};}if(item.abilities.count==0 || item.abilities.count>3){return 9;}var u32 previous=0;var bool store=false;var usize i=0;while(i<item.abilities.count){if(!state::charge(work,1)){return 10;}let t=model::list_at(p,item.abilities.first+i);if(!model::valid_text(p,t)){return 9;}let value=model::text(p,t);if(!state::charge(work,value.len+1)){return 10;}let u32 id=if(bytes::equal(value,b"drop")){1}else if(bytes::equal(value,b"store")){2}else if(bytes::equal(value,b"key")){3}else{0};if(id<=previous || id==3 && !store){return 9;}if(id==2){store=true;}previous=id;i+=1;}return 0;}
fn u32 item(ref model::Descriptor p,model::Item i,ref mut bytes::Buffer out,ref mut usize work){
    let kind=write::text(p,i.kind,out,work);if(kind!=0){return kind;}let name=write::text(p,i.name,out,work);if(name!=0){return name;}
    let repr=write::flag(out,i.repr_c,work);if(repr!=0){return repr;}let unsafe_call=write::flag(out,i.unsafe_call,work);if(unsafe_call!=0){return unsafe_call;}let must=write::flag(out,i.must_use,work);if(must!=0){return must;}
    let abi=write::text(p,i.abi,out,work);if(abi!=0){return abi;}let link=write::text(p,i.link,out,work);if(link!=0){return link;}let owner=write::text(p,i.owner,out,work);if(owner!=0){return owner;}
    let effects=write::strings(p,i.effects,out,work);if(effects!=0){return effects;}let abilities=write::strings(p,i.abilities,out,work);if(abilities!=0){return abilities;}let parameters=write::strings(p,i.parameters,out,work);if(parameters!=0){return parameters;}let result=write::text(p,i.result,out,work);if(result!=0){return result;}
    let fields=write::fields(p,i.fields,out,work);return if(fields!=0){fields}else{write::variants(p,i.variants,out,work)};
}
pub fn Result<bytes::Buffer,u32> encode(ref state::Plan layouts,usize owner,ref model::Descriptor p,ref mut usize work){
    if(model::module_count(p)==0 || model::module_count(p)>512 || model::item_count(p)>16384 || model::list_count(p)>32768 || model::field_count(p)>16384 || model::variant_count(p)>16384 || model::text_bytes(p)>1048576){return .err(9);}
    var bytes::Buffer out=switch(state_interface::prefix(layouts,owner,work)){case .ok(b):{b}case .err(code):{return .err(code);}};let count=write::number(ref mut out,model::module_count(p),work);if(count!=0){return .err(count);}var usize at=0;
    while(at<model::module_count(p)){if(!state::charge(work,1)){return .err(10);}let m=model::module_at(p,at);if(m.path.start==m.path.end || !model::valid_range(m.items,model::item_count(p))){return .err(9);}if(at>0){let compared=switch(order(p,model::module_at(p,at-1).path,m.path,work)){case .ok(n):{n}case .err(code):{return .err(code);}};if(compared>=0){return .err(9);}}
        let path=write::text(p,m.path,ref mut out,work);if(path!=0){return .err(path);}let items=write::number(ref mut out,m.items.count,work);if(items!=0){return .err(items);}var usize index=0;
        while(index<m.items.count){if(!state::charge(work,1)){return .err(10);}let i=model::item_at(p,m.items.first+index);if(i.kind.start==i.kind.end || i.name.start==i.name.end){return .err(9);}let abilities=abilities(p,i,work);if(abilities!=0){return .err(abilities);}if(index>0){let before=model::item_at(p,m.items.first+index-1);let compared=switch(order(p,before.kind,i.kind,work)){case .ok(n):{n}case .err(code):{return .err(code);}};if(compared>0){return .err(9);}if(compared==0){let names=switch(order(p,before.name,i.name,work)){case .ok(n):{n}case .err(code):{return .err(code);}};if(names>=0){return .err(9);}}}
            let code=item(p,i,ref mut out,work);if(code!=0){return .err(code);}index+=1;
        }at+=1;
    }return .ok(out);
}
pub fn Result<bytes::Buffer,u32> hash(ref state::Plan layouts,usize owner,ref model::Descriptor p,ref mut usize work){let encoded=switch(encode(layouts,owner,p,work)){case .ok(b):{b}case .err(code):{return .err(code);}};if(!state::charge(work,bytes::len(ref encoded)+1)){return .err(10);}var sha::State digest=sha::empty();if(!sha::update(ref mut digest,bytes::view(ref encoded))){return .err(9);}var bytes::Buffer out=bytes::empty();let prefix=write::raw(ref mut out,b"fyif5-sha256-",work);if(prefix!=0){return .err(prefix);}if(!state::charge(work,65)){return .err(10);}if(!sha::append_hex(ref digest,ref mut out)){return .err(11);}return .ok(out);}
