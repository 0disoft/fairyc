module enum_types;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;

// Private declaration IDs use a disjoint range; never an integer source type.
pub fn bool valid(u32 type) { return type >= 131072 && type <= 131583; }
pub fn u32 named(ref syntax::Parsed p, ref u8[] spelling) {
    var usize i=0; while(i<syntax::declaration_count(p)) {
        let d=syntax::declaration_at(p,i);
        if(d.kind==11 && bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],spelling)){return 131072+state::small_id(i);} i+=1;
    } return 0;
}
pub fn usize member(ref syntax::Parsed p,u32 type,ref u8[] name) {
    if(!valid(type) || (type-131072) as usize>=syntax::declaration_count(p)){return 0;}
    let d=syntax::declaration_at(p,(type-131072) as usize);if(d.kind!=11){return 0;}
    var usize i=0;while(i<d.field_count){let f=syntax::field_at(p,d.field_start+i);if(f.kind==3 && bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],name)){return d.field_start+i+1;}i+=1;}return 0;
}
pub fn state::Problem validate(ref syntax::Parsed p) {
    var usize i=0;while(i<syntax::declaration_count(p)){
        let d=syntax::declaration_at(p,i);
        if(d.kind==11){
            if(d.field_count==0 || d.field_count>128){return state::fail(23,d.start,d.end,0,0);}
            var usize j=0;while(j<d.field_count){let f=syntax::field_at(p,d.field_start+j);
                if(f.kind!=3 || f.payload_open!=0){return state::fail(23,f.start,f.end,0,0);}
                var usize k=0;while(k<j){let previous=syntax::field_at(p,d.field_start+k);if(bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],syntax::source_view(p)[previous.name_start..previous.name_end])){return state::fail(22,f.name_start,f.name_end,0,0);}k+=1;}j+=1;
            }
        }i+=1;
    }return state::ok();
}
