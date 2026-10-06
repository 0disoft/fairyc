module enum_types;
import syntax as syntax;
import tagged_types as tagged;
import type_state as state;
import std::bytes as bytes;

// Private declaration IDs use a disjoint range; never an integer source type.
pub fn bool valid(u32 type) { return type >= 131072 && type <= 131583 || tagged::builtin(type); }
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
pub fn usize member_count(ref syntax::Parsed p,u32 type) {
    let d=syntax::declaration_at(p,(type-131072) as usize);var usize count=0;var usize i=0;
    while(i<d.field_count){if(syntax::field_at(p,d.field_start+i).kind==3){count+=1;}i+=1;}return count;
}
pub fn usize payload_field(ref syntax::Parsed p,usize member,ref u8[] name) {
    let m=syntax::field_at(p,member-1);var usize i=0;
    while(i<m.payload_count){let f=syntax::field_at(p,m.payload_start+i);if(bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],name)){return m.payload_start+i+1;}i+=1;}return 0;
}
pub fn state::Problem validate(ref syntax::Parsed p) {
    var usize i=0;while(i<syntax::declaration_count(p)){
        let d=syntax::declaration_at(p,i);
        if(d.kind==11){
            let usize count=member_count(p,131072+state::small_id(i));
            if(count==0 || count>128){return state::fail(23,d.start,d.end,0,0);}
            var usize j=0;while(j<d.field_count){let f=syntax::field_at(p,d.field_start+j);
                if(f.kind==3){
                    if(f.payload_count>16 || f.payload_open!=0 && f.payload_count==0){return state::fail(30,f.start,f.end,0,0);}
                    var usize k=0;while(k<j){let previous=syntax::field_at(p,d.field_start+k);if(previous.kind==3 && bytes::equal(syntax::source_view(p)[f.name_start..f.name_end],syntax::source_view(p)[previous.name_start..previous.name_end])){return state::fail(22,f.name_start,f.name_end,0,0);}k+=1;}
                    k=0;while(k<f.payload_count){let field=syntax::field_at(p,f.payload_start+k);let u32 type=state::primitive(p,field.type_root);
                        if(field.kind!=4 || type<1 || type>6){return state::fail(30,field.start,field.end,0,type);}
                        var usize before=0;while(before<k){let previous=syntax::field_at(p,f.payload_start+before);if(bytes::equal(syntax::source_view(p)[field.name_start..field.name_end],syntax::source_view(p)[previous.name_start..previous.name_end])){return state::fail(17,field.name_start,field.name_end,0,0);}before+=1;}k+=1;
                    }
                }else if(f.kind!=4){return state::fail(30,f.start,f.end,0,0);}j+=1;
            }
        }i+=1;
    }return state::ok();
}
