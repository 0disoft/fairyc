module hir_shapes;
import c_abi_types as c_types;
import pointer_types as pointers;
import borrow_shapes as borrows;
import reference_types as refs;
import array_types as arrays;
import enum_types as enums;
import tagged_types as tagged;
import literal_types as literals;
import nominal_identity as nominal;
import resource_identity as resources;
import resource_properties as properties;
import ffi_symbols as ffi;
import std::bytes as bytes;
import std::vec as v;
import sdk_scopes as sdk;

// Source-independent nominal layout facts. Field IDs are one-based; shape type
// IDs preserve 256 + declaration index. Every name/span indexes owned HIR bytes.
pub struct Shape { u32 type; usize binding; usize field_start; usize field_count; bool public; }
pub struct Field { u32 owner; u32 type; usize start; usize end; usize payload_start; usize payload_count; }
pub struct Abi {u32 type;usize declaration;usize name_start;usize name_end;usize marker_start;usize marker_end;}
pub resource Table { v::Vec<Shape> shapes; v::Vec<Field> fields;v::Vec<Abi> abi;v::Vec<sdk::Scope> sdk_scopes; cleanup { } }
pub fn Table empty() { return Table { .shapes = v::empty<Shape>(), .fields = v::empty<Field>(),.abi=v::empty<Abi>(),.sdk_scopes=v::empty<sdk::Scope>() }; }
pub fn usize sdk_count(ref Table p){return v::len<sdk::Scope>(ref p.sdk_scopes);}
pub fn sdk::Scope sdk_at(ref Table p,usize index){return *v::at<sdk::Scope>(ref p.sdk_scopes,index);}
pub fn bool storage_rule(ref Table p,ref u8[] source,usize start,usize end,u32 code){return sdk::storage_owned(source,ref p.sdk_scopes,start,end,code);}
// A safe storage alias must be backed by the authenticated SDK Vec layout.
// A user resource with the same field types cannot claim that ownership role.
pub fn bool storage_owner(ref Table p,u32 type){
    if(!refs::valid(type) || !resources::valid(refs::inner(type))){return false;}
    let usize id=find(p,refs::inner(type));if(id==0){return false;}let s=at(p,id-1);if(s.field_count!=6){return false;}
    var usize i=0;while(i<6){let f=field_at(p,s.field_start+i+1);if(f.type!=if(i==0){pointers::make(7,true)}else{5}){return false;}
        var bool owned=false;var usize j=0;while(j<sdk_count(p)){let scope=sdk_at(p,j);owned=owned || scope.sdk_id==8 && f.start>=scope.start && f.end<=scope.end;j+=1;}if(!owned){return false;}i+=1;
    }return true;
}
pub fn usize sdk_rule(ref Table p,ref u8[] source,usize start,usize end,usize link_start,usize link_end){return sdk::foreign_owned(source,ref p.sdk_scopes,start,end,link_start,link_end);}
pub fn bool sdk_result(ref Table p,ref u8[] source,usize rule,u32 type){
    if(rule==1 || rule>=3 && rule<=8){return type==sdk::scalar_result(rule);}if(rule!=2){return false;}let usize a=abi_find(p,type);let usize s=find(p,type);if(a==0 || s==0){return false;}
    let abi=abi_at(p,a-1);if(!sdk::result_owned(source,ref p.sdk_scopes,abi.name_start,abi.name_end)){return false;}
    let shape=at(p,s-1);return shape.field_count==2 && field_at(p,shape.field_start+1).type==4 && field_at(p,shape.field_start+2).type==5;
}
pub fn usize abi_count(ref Table p){return v::len<Abi>(ref p.abi);}
pub fn Abi abi_at(ref Table p,usize index){return *v::at<Abi>(ref p.abi,index);}
pub fn usize abi_find(ref Table p,u32 type){var usize i=0;while(i<abi_count(p)){if(abi_at(p,i).type==type){return i+1;}i+=1;}return 0;}
pub fn bool abi_field(u32 type){return type>=1 && type<=6 || c_types::valid(type) || pointers::valid(type);}
pub fn bool foreign_result(ref Table p,u32 type){return ffi::scalar(type,true) || abi_find(p,type)!=0;}
fn usize abi_trivia(ref u8[] input,usize index){var usize i=index;while(i<input.len){
    if(input[i]==9 || input[i]==10 || input[i]==13 || input[i]==32){i+=1;continue;}
    if(i+1<input.len && input[i]==47 && input[i+1]==47){i+=2;while(i<input.len && input[i]!=10){i+=1;}continue;}
    if(i+1<input.len && input[i]==47 && input[i+1]==42){i+=2;while(i+1<input.len && !(input[i]==42 && input[i+1]==47)){i+=1;}if(i+1>=input.len){return input.len+1;}i+=2;continue;}
    break;
}return i;}
fn bool abi_marker(ref u8[] input){var usize i=abi_trivia(input,0);var usize part=0;while(part<5){
    let token=if(part==0){b"@"}else if(part==1){b"repr"}else if(part==2){b"("}else if(part==3){b"C"}else{b")"};
    if(i>input.len || token.len>input.len-i || !bytes::equal(input[i..i+token.len],token)){return false;}i=abi_trivia(input,i+token.len);part+=1;
}return i==input.len;}
// ABI permission is tied to owned declaration/name/attribute bytes and a
// rechecked closed field layout, never a caller-supplied boolean by itself.
pub fn bool abi_valid(ref Table p,ref u8[] source){
    if(!sdk::valid(source,ref p.sdk_scopes)){return false;}
    if(abi_count(p)>count(p)){return false;}var usize i=0;
    while(i<abi_count(p)){let a=abi_at(p,i);let usize slot=find(p,a.type);
        if(!nominal::valid(a.type) || slot==0 || a.declaration!=(a.type-255) as usize || a.name_start>=a.name_end || a.name_end>source.len || a.marker_start>=a.marker_end || a.marker_end>a.name_start){return false;}
        if(!ffi::valid(source[a.name_start..a.name_end]) || !abi_marker(source[a.marker_start..a.marker_end])){return false;}
        let s=at(p,slot-1);if(s.field_count==0 || s.field_start>field_count(p) || s.field_count>field_count(p)-s.field_start){return false;}
        var usize j=0;while(j<i){let old=abi_at(p,j);if(old.type==a.type || bytes::equal(source[old.name_start..old.name_end],source[a.name_start..a.name_end])){return false;}j+=1;}
        j=0;while(j<s.field_count){let f=field_at(p,s.field_start+j+1);
            if(f.owner!=a.type || f.start>=f.end || f.end>source.len || !abi_field(f.type) || !ffi::valid(source[f.start..f.end])){return false;}
            var usize k=0;while(k<j){let old=field_at(p,s.field_start+k+1);if(bytes::equal(source[old.start..old.end],source[f.start..f.end])){return false;}k+=1;}j+=1;
        }i+=1;
    }return true;
}
pub fn usize count(ref Table p) { return v::len<Shape>(ref p.shapes); }
pub fn bool has_enums(ref Table p) { var usize i=0;while(i<count(p)){if(enums::valid(at(p,i).type)){return true;}i+=1;}return false; }
pub fn bool has_payloads(ref Table p) { var usize i=1;while(i<=field_count(p)){if(field_at(p,i).payload_count!=0){return true;}i+=1;}return false; }
pub fn bool has_options(ref Table p) { var usize i=0;while(i<count(p)){if(tagged::option(at(p,i).type)){return true;}i+=1;}return false; }
pub fn bool has_results(ref Table p) { var usize i=0;while(i<count(p)){if(tagged::result(at(p,i).type)){return true;}i+=1;}return false; }
pub fn bool has_resources(ref Table p) { var usize i=0;while(i<count(p)){if(resources::valid(at(p,i).type)){return true;}i+=1;}return false; }
// Properties are derived from canonical category/active-payload layouts; they
// cannot be forged as caller-supplied Copyable or NeedsDrop flags. Inert structs
// and arrays are separately verified not to contain resource-bearing fields.
pub fn bool needs_drop(u32 type) { return properties::needs_drop(type); }
pub fn bool copyable(ref Table p,u32 type) { return valid_type(p,type,false) && type!=11 && !needs_drop(type); }
pub fn bool field_owns_resource(ref Table p,usize field) { return needs_drop(field_at(p,field).type); }
pub fn usize member_count(ref Table p,u32 type) { let s=at(p,find(p,type)-1);var usize n=0;var usize i=0;while(i<s.field_count){if(field_at(p,s.field_start+i+1).type==0){n+=1;}i+=1;}return n; }
pub fn usize member_at(ref Table p,u32 type,usize tag) {
    let usize found=find(p,type);if(found==0){return 0;}let s=at(p,found-1);var usize i=0;var usize ordinal=0;
    while(i<s.field_count){let usize id=s.field_start+i+1;if(field_at(p,id).type==0){if(ordinal==tag){return id;}ordinal+=1;}i+=1;}return 0;
}
pub fn usize member_tag(ref Table p,u32 type,usize member) {
    var usize tag=0;while(tag<member_count(p,type)){if(member_at(p,type,tag)==member){return tag;}tag+=1;}return tag;
}
pub fn usize payload_member(ref Table p,u32 type,usize field) {
    let usize found=find(p,type);if(found==0){return 0;}let s=at(p,found-1);var usize i=0;
    while(i<s.field_count){let usize id=s.field_start+i+1;let f=field_at(p,id);if(f.type==0 && field>f.payload_start && field<=f.payload_start+f.payload_count){return id;}i+=1;}return 0;
}
pub fn usize field_count(ref Table p) { return v::len<Field>(ref p.fields); }
pub fn Shape at(ref Table p, usize i) { assert(i < count(p)); return *v::at<Shape>(ref p.shapes,i); }
pub fn Field field_at(ref Table p, usize id) { assert(id > 0 && id <= field_count(p)); return *v::at<Field>(ref p.fields,id - 1); }
pub fn usize find(ref Table p, u32 type) { var usize i = 0; while (i < count(p)) { if (at(p,i).type == type) { return i + 1; } i += 1; } return 0; }
pub fn u32 array_element(ref Table p,u32 type) {
    if(!arrays::valid(type)){return 0;}if(!arrays::owned(type)){return arrays::element(type);}
    let usize id=find(p,type);return if(id==0){0}else{field_at(p,at(p,id-1).field_start+1).type};
}
pub fn bool has_owned_arrays(ref Table p) { var usize i=0;while(i<count(p)){if(arrays::owned(at(p,i).type)){return true;}i+=1;}return false; }
pub fn bool valid_type(ref Table p, u32 type, bool allow_void) { if(c_types::valid(type) || pointers::valid(type)){return true;}if(refs::valid(type)){return valid_type(p,refs::inner(type),false);}return (type >= 1 && type <= if (allow_void) { 7 } else { 6 }) || literals::valid(type) || type == 11 || arrays::valid(type) && (!arrays::owned(type) || find(p,type)!=0) || ((nominal::valid(type) || resources::valid(type) || enums::valid(type)) && find(p,type) != 0); }
fn bool visit(ref Table p, usize i, ref mut u8[] colors, usize depth) {
    if (colors[i] == 2) { return true; } if (colors[i] == 1 || depth >= 64) { return false; }
    colors[i] = 1; let Shape s = at(p,i); var usize j = 0;
    while (j < s.field_count) {
        let Field f = field_at(p,s.field_start + j + 1);
        if ((!enums::valid(s.type) || tagged::builtin(s.type)) && f.type >= 256 && !pointers::valid(f.type) && !refs::valid(f.type) && (!arrays::valid(f.type) || arrays::owned(f.type)) && !visit(p,find(p,f.type) - 1,colors,depth + 1)) { return false; } j += 1;
    }
    colors[i] = 2; return true;
}
// Validation precedes DFS and all subtraction/indexing. A malformed/cyclic
// table is never published, including tables assembled by mutation tests.
pub fn bool valid(ref Table p, usize source_bytes, usize bindings) {
    if (count(p) > 512 || field_count(p) > 1024) { return false; }
    var usize offset = 0; var usize i = 0;
    while (i < count(p)) {
        let Shape s = at(p,i);
        if(tagged::builtin(s.type) && borrows::count(s.type)>1){return false;}
        if (!nominal::valid(s.type) && !resources::valid(s.type) && !enums::valid(s.type) && !arrays::owned(s.type) || (!tagged::builtin(s.type) && !arrays::owned(s.type) && s.binding == 0) || s.binding > bindings || s.field_start != offset || s.field_count > field_count(p) - offset) { return false; }
        if(arrays::owned(s.type)){
            if(s.binding!=0 || s.public || s.field_count!=1){return false;}
            let u32 child=field_at(p,offset+1).type;let u32 encoded=arrays::encoded_element(s.type);let u32 key=arrays::key(s.type);
            if(encoded!=0 && child!=encoded || !arrays::rank_two(s.type) && key>=2048 && !(child>=1073741824 && tagged::builtin(child)) || arrays::rank_two(s.type) && (!arrays::owned(child) || arrays::rank_two(child))){return false;}
        }
        if(tagged::option(s.type) && (s.binding!=0 || s.public || s.field_count!=3)){return false;}
        if(tagged::option(s.type)){
            let a=field_at(p,offset+1);let b=field_at(p,offset+2);let c=field_at(p,offset+3);
            if(a.type!=tagged::inner(s.type) || b.type!=0 || b.payload_start!=offset || b.payload_count!=1 || c.type!=0 || c.payload_count!=0){return false;}
        }
        if(tagged::result(s.type)){
            if(s.binding!=0 || s.public || s.field_count!=4){return false;}
            let a=field_at(p,offset+1);let b=field_at(p,offset+2);let c=field_at(p,offset+3);let d=field_at(p,offset+4);
            if(a.type!=tagged::success(s.type) || c.type!=tagged::failure(s.type) || b.type!=0 || b.payload_start!=offset || b.payload_count!=1 || d.type!=0 || d.payload_start!=offset+2 || d.payload_count!=1){return false;}
        }
        if(enums::valid(s.type) && (member_count(p,s.type)==0 || member_count(p,s.type)>128)){return false;}
        var usize j = 0; while (j < i) { if (at(p,j).type == s.type || s.binding!=0 && at(p,j).binding == s.binding) { return false; } j += 1; }
        j = 0; while (j < s.field_count) {
            let Field f = field_at(p,offset + j + 1);
            if(enums::valid(s.type)){
                if(f.type==0){
                    if(f.payload_count==0 && f.payload_start!=0 || f.payload_count>16 || f.payload_count>j || f.payload_count!=0 && f.payload_start!=offset+j-f.payload_count){return false;}
                    var usize k=0;while(k<f.payload_count){if(field_at(p,f.payload_start+k+1).type==0){return false;}k+=1;}
                    if(j>f.payload_count && field_at(p,offset+j-f.payload_count).type!=0){return false;}
                }else if(f.payload_count!=0 || f.payload_start!=0 || j+1==s.field_count){return false;}
            }else if(f.payload_count!=0 || f.payload_start!=0){return false;}
            if (f.owner != s.type || f.start >= f.end || f.end > source_bytes || (if(enums::valid(s.type)){f.type>6 && !(tagged::builtin(s.type) && (enums::valid(f.type) || nominal::valid(f.type) || resources::valid(f.type) || borrows::count(f.type)==1) && valid_type(p,f.type,false))}else{borrows::count(f.type)!=0 || !valid_type(p,f.type,false)})) { return false; }
            if(!resources::valid(s.type) && !tagged::builtin(s.type) && needs_drop(f.type)){return false;}j += 1;
        }
        offset += s.field_count; i += 1;
    }
    if (offset != field_count(p)) { return false; }
    var u8[512] colors = [0;512]; i = 0;
    while (i < count(p)) { if (!visit(p,i,ref mut colors[..],0)) { return false; } i += 1; } return true;
}
pub fn Table assemble(v::Vec<Shape> shapes, v::Vec<Field> fields) { return Table { .shapes, .fields,.abi=v::empty<Abi>(),.sdk_scopes=v::empty<sdk::Scope>() }; }
pub fn Table assemble_abi(v::Vec<Shape> shapes,v::Vec<Field> fields,v::Vec<Abi> abi){return Table{.shapes,.fields,.abi,.sdk_scopes=v::empty<sdk::Scope>()};}
pub fn Table assemble_sdk(v::Vec<Shape> shapes,v::Vec<Field> fields,v::Vec<Abi> abi,v::Vec<sdk::Scope> sdk_scopes){return Table{.shapes,.fields,.abi,.sdk_scopes};}
pub fn Result<Table,u32> copy(ref Table p) {
    var v::Vec<Shape> records = v::empty<Shape>(); var v::Vec<Field> fields = v::empty<Field>();
    var usize i = 0;
    while (i < count(p)) { if (!v::push<Shape>(ref mut records,at(p,i)).is_ok) { return .err(3); } i += 1; }
    i = 1; while (i <= field_count(p)) { if (!v::push<Field>(ref mut fields,field_at(p,i)).is_ok) { return .err(3); } i += 1; }
    var v::Vec<Abi> abi=v::empty<Abi>();i=0;while(i<abi_count(p)){if(!v::push<Abi>(ref mut abi,abi_at(p,i)).is_ok){return .err(3);}i+=1;}
    var v::Vec<sdk::Scope> scopes=v::empty<sdk::Scope>();i=0;while(i<sdk_count(p)){if(!v::push<sdk::Scope>(ref mut scopes,sdk_at(p,i)).is_ok){return .err(3);}i+=1;}
    return .ok(assemble_sdk(records,fields,abi,scopes));
}
