module generic_source_arguments;
import generic_source_catalog as catalog;
import generic_package_bindings as bindings;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub fn bool scalar(ref u8[] name){return bytes::equal(name,b"bool") || bytes::equal(name,b"i8") || bytes::equal(name,b"i16") || bytes::equal(name,b"i32") || bytes::equal(name,b"i64") || bytes::equal(name,b"isize") || bytes::equal(name,b"u8") || bytes::equal(name,b"u16") || bytes::equal(name,b"u32") || bytes::equal(name,b"u64") || bytes::equal(name,b"usize") || bytes::equal(name,b"f32") || bytes::equal(name,b"f64");}
pub fn Result<bool,u32> generated(ref u8[] name,ref mut usize work){var usize i=0;while(i+12<=name.len){if(!budget::charge(work,1)){return .err(10);}if(bytes::equal(name[i..i+12],b"_fy_generic_")){return .ok(true);}i+=1;}return .ok(false);}
// Normalize original path tokens, including comments between alias separators.
pub fn Result<bytes::Buffer,u32> path(ref syntax::Parsed p,usize start,usize end,ref mut usize work){
    if(start>=end || end>syntax::source_view(p).len){return .err(9);}var usize low=0;var usize high=syntax::token_count(p);
    while(low<high){if(!budget::charge(work,1)){return .err(10);}let mid=low+(high-low)/2;if(syntax::token_info(p,mid).start<start){low=mid+1;}else{high=mid;}}
    var bytes::Buffer out=bytes::empty();while(low<syntax::token_count(p)){if(!budget::charge(work,1)){return .err(10);}let t=syntax::token_info(p,low);if(t.start>=end){break;}if(t.end>end){return .err(9);}if(t.kind!=2){if(!budget::charge(work,t.end-t.start+1)){return .err(10);}if(!bytes::append(ref mut out,syntax::source_view(p)[t.start..t.end]).is_ok){return .err(11);}}low+=1;}
    return .ok(out);
}
// Go source_nominal_v0 IDs exclude disposable owner numbers and source text.
pub fn Result<bytes::Buffer,u32> nominal(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize module_index,ref u8[] name,ref mut usize work){
    let resolved=catalog::packages(c);if(module_index>=bindings::count(resolved)){return .err(9);}var usize split=name.len;var usize i=0;while(i<name.len){if(name[i]==58){if(split!=name.len || i+1>=name.len || name[i+1]!=58){return .err(9);}split=i;i+=1;}i+=1;}
    var usize target=module_index;if(split!=name.len){let m=bindings::module_at(resolved,module_index);target=bindings::count(resolved);i=0;while(i<m.count){if(!budget::charge(work,split+1)){return .err(10);}let row=bindings::import_at(resolved,m.first+i);if(bytes::equal(bindings::text(resolved,row.alias),name[..split])){target=row.target;break;}i+=1;}if(target==bindings::count(resolved)){return .err(9);}}
    if(target>=v::len<syntax::Parsed>(sources)){return .err(9);}let leaf=if(split==name.len){name}else{name[split+2..]};let p=v::at<syntax::Parsed>(sources,target);i=1;
    while(i<syntax::declaration_count(p)){if(!budget::charge(work,1)){return .err(10);}let d=syntax::declaration_at(p,i);if((d.kind==4 || d.kind==6) && d.generic_parameters==0){if(!budget::charge(work,leaf.len+1)){return .err(10);}if(bytes::equal(leaf,syntax::source_view(p)[d.name_start..d.name_end])){if(target!=module_index && !d.public){return .err(9);}let m=bindings::module_at(resolved,target);var bytes::Buffer out=bytes::empty();let owner=bindings::text(resolved,m.namespace);let module_path=bindings::text(resolved,m.path);if(!budget::charge(work,owner.len+module_path.len+leaf.len+24)){return .err(10);}if(!bytes::append(ref mut out,b"source_nominal_v0::").is_ok || !bytes::append(ref mut out,owner).is_ok || !bytes::append(ref mut out,b"::").is_ok || !bytes::append(ref mut out,module_path).is_ok || !bytes::append(ref mut out,b"::").is_ok || !bytes::append(ref mut out,leaf).is_ok){return .err(11);}return .ok(out);}}i+=1;}
    return .err(9);
}
pub fn Result<bytes::Buffer,u32> concrete(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize module_index,ref u8[] name,ref mut usize work){if(!budget::charge(work,name.len+1)){return .err(10);}if(scalar(name)){return switch(bytes::copy(name)){case .ok(value):{.ok(value)}case .err(_):{.err(11)}};}return nominal(c,sources,module_index,name,work);}
pub fn Result<usize,u32> parameter(ref catalog::Catalog c,usize template_index,ref u8[] name,ref mut usize work){let row=catalog::template_at(c,template_index);var usize i=0;while(i<row.count){if(!budget::charge(work,name.len+1)){return .err(10);}let p=catalog::parameter_at(c,row.first+i);if(bytes::equal(catalog::text(c,p.name),name)){return .ok(i+1);}i+=1;}return .ok(0);}
