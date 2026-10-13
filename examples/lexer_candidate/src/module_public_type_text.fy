module module_public_type_text;
import module_unit_interfaces as typed;
import module_public_type_ids as identities;
import module_signatures as signatures;
import module_public_interface_write as write;
import module_registry as registry;
import module_state_layout as budget;
import scalar_views as views;
import pointer_types as pointers;
import c_abi_types as c;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Public namespace coordinates exclude source content. Instance IDs are never
// accepted here: dependency implementation changes must preserve type spelling.
pub struct Coordinate{usize owner;usize start;usize end;}
pub resource Coordinates{bytes::Buffer names;v::Vec<Coordinate> rows;cleanup{}}
pub fn Coordinates assemble(bytes::Buffer names,v::Vec<Coordinate> rows){return Coordinates{.names,.rows};}
pub fn usize coordinate_count(ref Coordinates p){return v::len<Coordinate>(ref p.rows);}
pub fn Coordinate coordinate_at(ref Coordinates p,usize at){assert(at<coordinate_count(p));return *v::at<Coordinate>(ref p.rows,at);}
pub fn ref u8[] coordinate_text(ref Coordinates p,usize at){let row=coordinate_at(p,at);assert(row.start<=row.end && row.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[row.start..row.end];}
pub fn bool valid_coordinate(ref u8[] text){let prefix=b"fypc1-sha256-";if(text.len!=prefix.len+64 || !bytes::equal(text[..prefix.len],prefix)){return false;}var usize i=prefix.len;while(i<text.len){let b=text[i];if(!(b>=48 && b<=57 || b>=97 && b<=102)){return false;}i+=1;}return true;}
fn Result<usize,u32> coordinate(ref Coordinates p,usize owner,ref mut usize work){if(v::len<Coordinate>(ref p.rows)>512 || bytes::len(ref p.names)>1048576){return .err(9);}var usize found=v::len<Coordinate>(ref p.rows);var usize i=0;while(i<v::len<Coordinate>(ref p.rows)){if(!budget::charge(work,1)){return .err(10);}let row=*v::at<Coordinate>(ref p.rows,i);if(row.owner==owner){if(found!=v::len<Coordinate>(ref p.rows) || row.start>row.end || row.end>bytes::len(ref p.names)){return .err(9);}if(!budget::charge(work,row.end-row.start+1)){return .err(10);}if(!valid_coordinate(bytes::view(ref p.names)[row.start..row.end])){return .err(9);}found=i;}i+=1;}return if(found==v::len<Coordinate>(ref p.rows)){.err(9)}else{.ok(found)};}
fn ref u8[] scalar(u32 id){let ref u8[] names=b"|i32|i64|u8|u32|usize|bool|void|";var u32 index=0;var usize start=0;var usize end=0;while(end<names.len){if(names[end]==124){if(index==id && id<=7){break;}index+=1;start=end+1;}end+=1;}return if(id>=1 && id<=7){names[start..end]}else{names[..0]};}
fn ref u8[] c_scalar(u32 id){let ref u8[] names=b"bool|char|double|float|int|long|longlong|schar|short|size_t|ssize_t|uchar|uint|ulong|ulonglong|ushort|";var u32 index=128;var usize start=0;var usize end=0;while(end<names.len){if(names[end]==124){if(index==id){break;}index+=1;start=end+1;}end+=1;}return names[start..end];}
fn u32 nominal(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref Coordinates coordinates,usize owner,usize module_index,usize declaration,u32 kind,ref mut bytes::Buffer out,ref mut usize work){
    if(module_index>=v::len<syntax::Parsed>(sources) || declaration==0){return 9;}let source=v::at<syntax::Parsed>(sources,module_index);if(!budget::charge(work,syntax::source_view(source).len+1)){return 10;}if(!signatures::source_matches(metadata,module_index,source)){return 12;}if(declaration>syntax::declaration_count(source)){return 9;}let d=syntax::declaration_at(source,declaration-1);let m=syntax::declaration_at(source,0);if(d.kind!=if(kind==2){4}else if(kind==3){6}else{11} || !d.public || d.generic_parameters!=0 || m.kind!=1){return 9;}
    let raw=syntax::source_view(source);if(d.name_start>d.name_end || d.name_end>raw.len || m.name_start>m.name_end || m.name_end>raw.len){return 9;}if(!budget::charge(work,m.name_end-m.name_start+1)){return 10;}var bytes::Buffer module_name=bytes::empty();let normalized=registry::normalize(raw[m.name_start..m.name_end],ref mut module_name);if(normalized!=0){return normalized;}if(bytes::len(ref module_name)==0 || d.name_start==d.name_end){return 9;}
    let target=signatures::module_owner(metadata,module_index);var u32 code=0;
    if(target==owner){code=write::raw(out,b"self::",work);}
    else{let id=switch(coordinate(coordinates,target,work)){case .ok(id):{id}case .err(code):{return code;}};let row=*v::at<Coordinate>(ref coordinates.rows,id);code=write::raw(out,b"dep[",work);if(code!=0){return code;}code=write::raw(out,bytes::view(ref coordinates.names)[row.start..row.end],work);if(code!=0){return code;}code=write::raw(out,b"]::",work);}
    if(code!=0){return code;}code=write::raw(out,bytes::view(ref module_name),work);if(code!=0){return code;}code=write::raw(out,b"::",work);return if(code!=0){code}else{write::raw(out,raw[d.name_start..d.name_end],work)};
}
fn u32 walk(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref Coordinates coordinates,ref identities::Table named,usize owner,usize id,usize depth,ref mut bytes::Buffer out,ref mut usize work){
    if(depth>=64){return 10;}if(id==0 || id>typed::type_count(graph)){return 9;}if(!budget::charge(work,1)){return 10;}let t=typed::type_at(graph,id);var u32 code=0;
    if(t.kind==1){if(c::valid(t.scalar)){code=write::raw(out,b"c::",work);return if(code!=0){code}else{write::raw(out,c_scalar(t.scalar),work)};}let text=scalar(t.scalar);return if(text.len==0){9}else{write::raw(out,text,work)};}
    if(t.kind>=2 && t.kind<=4){return nominal(sources,metadata,coordinates,owner,t.module_index,t.declaration,t.kind,out,work);}
    if(t.kind==9 || t.kind==14 || t.kind==19){return write::raw(out,if(t.kind==9){b"ref str"}else if(t.kind==14){b"ref CStr"}else{b"Region"},work);}
    if(t.kind==10 || t.kind==11 || t.kind==15){let u32 tag=if(t.kind==10){10}else if(t.kind==11){11}else{t.scalar};if(!views::valid(tag)){return 9;}code=write::raw(out,if(views::mutable(tag)){b"ref mut "}else{b"ref "},work);if(code!=0){return code;}code=write::raw(out,scalar(views::element(tag)),work);return if(code!=0){code}else{write::raw(out,b"[]",work)};}
    if(t.kind==12){let mutable=if(t.first==0){pointers::mutable(t.scalar)}else{t.scalar==1};if(t.first==0 && !pointers::valid(t.scalar)){return 9;}code=write::raw(out,if(mutable){b"ptr mut "}else{b"ptr "},work);if(code!=0){return code;}if(t.first!=0){return walk(sources,metadata,graph,coordinates,named,owner,t.first,depth+1,out,work);}let inner=pointers::inner(t.scalar);return write::raw(out,if(inner==1048591){b"ptr mut void"}else{scalar(inner)},work);}
    if(t.kind==5 || t.kind==6 || t.kind==7 || t.kind==8 || t.kind==16 || t.kind==17 || t.kind==18){
        if(t.kind==5 || t.kind==6 || t.kind==7 || t.kind==8 || t.kind==17 || t.kind==18){code=write::raw(out,if(t.kind==5){b"Option<"}else if(t.kind==6){b"Result<"}else if(t.kind==8 || t.kind==18){b"ref mut "}else{b"ref "},work);if(code!=0){return code;}}
        code=walk(sources,metadata,graph,coordinates,named,owner,t.first,depth+1,out,work);if(code!=0){return code;}if(t.kind==6){code=write::raw(out,b", ",work);if(code!=0){return code;}code=walk(sources,metadata,graph,coordinates,named,owner,t.second,depth+1,out,work);if(code!=0){return code;}}
        if(t.kind==5 || t.kind==6){return write::raw(out,b">",work);}if(t.kind==17 || t.kind==18){return write::raw(out,b"[]",work);}if(t.kind==16){if(t.second==0 || t.second>4096){return 9;}code=write::raw(out,b"[",work);if(code!=0){return code;}var bytes::Buffer number=bytes::empty();if(!bytes::append_usize(ref mut number,t.second).is_ok){return 11;}code=write::raw(out,bytes::view(ref number),work);return if(code!=0){code}else{write::raw(out,b"]",work)};}return 0;
    }
    if(t.kind==13){let target=switch(identities::owner(named,id,t)){case .ok(n):{n}case .err(code):{return code;}};if(target==owner){code=write::raw(out,b"self::",work);}else{let index=switch(coordinate(coordinates,target,work)){case .ok(n):{n}case .err(code):{return code;}};code=write::raw(out,b"dep[",work);if(code!=0){return code;}code=write::raw(out,coordinate_text(coordinates,index),work);if(code!=0){return code;}code=write::raw(out,b"]::",work);}return if(code!=0){code}else{write::raw(out,identities::name(named,id),work)};}
    return if(t.kind==13){4503}else{9};
}
pub fn u32 validate_owner(ref Coordinates coordinates,usize owner,ref mut usize work){return switch(coordinate(coordinates,owner,work)){case .ok(_):{0}case .err(code):{code}};}
pub fn Result<bytes::Buffer,u32> render(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref Coordinates coordinates,usize owner,usize id,ref mut usize work){if(owner>=512 || v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata)){return .err(9);}let named=identities::empty();var bytes::Buffer out=bytes::empty();let code=walk(sources,metadata,graph,coordinates,ref named,owner,id,0,ref mut out,work);if(code!=0){return .err(code);}return .ok(out);}

pub fn Result<bytes::Buffer,u32> render_identified(ref identities::Table named,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref Coordinates coordinates,usize owner,usize id,ref mut usize work){if(owner>=512){return .err(9);}let checked=identities::current(named,sources,metadata,graph,work);if(checked!=0){return .err(checked);}var bytes::Buffer out=bytes::empty();let code=walk(sources,metadata,graph,coordinates,named,owner,id,0,ref mut out,work);if(code!=0){return .err(code);}return .ok(out);}
