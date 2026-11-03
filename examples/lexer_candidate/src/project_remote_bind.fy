module project_remote_bind;
import project_remote_plan as plan;
import project_manifest as manifest;
import project_path as work;
import std::bytes as bytes;

fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
// Both inputs are already validated portable manifest paths (ASCII). Like
// Go Bind, compare case-insensitively and only at directory boundaries.
fn bool overlaps(ref u8[] left,ref u8[] right){
    var usize i=0;while(i<left.len && i<right.len){if(folded(left[i])!=folded(right[i])){return false;}i+=1;}
    if(left.len==right.len){return true;}
    return if(left.len<right.len){right[i]==47}else{left[i]==47};
}
// Private status: 0 bound, 1 member/identity/version/overlap rejection,
// 2 shared work bound. This validates metadata only; it never reads a member.
pub fn u32 validate(ref plan::Plan sources,ref manifest::Manifest root,ref mut usize meter){
    let ref u8[] cache=b"build/package-sources";
    let ref u8[] source=manifest::package_value(root,b"source");
    let ref u8[] uid=manifest::package_value(root,b"uid");
    if(!work::charge(meter,source.len+uid.len+cache.len+1)){return 2;}
    if(overlaps(source,cache)){return 1;}
    var usize index=0;while(index<plan::count(sources)){
        let ref u8[] path=plan::value(sources,index,0);let ref u8[] version=plan::value(sources,index,2);
        if(!work::charge(meter,path.len+source.len+uid.len+plan::value(sources,index,1).len+1)){return 2;}
        if(bytes::equal(uid,plan::value(sources,index,1)) || overlaps(path,source) || overlaps(path,cache)){return 1;}
        var bool found=false;var usize member=manifest::member_first(root);
        while(member!=0){let ref u8[] value=manifest::value(root,member);
            if(!work::charge(meter,path.len+value.len+1)){return 2;}
            if(bytes::equal(value,path)){found=true;}else if(overlaps(value,path)){return 1;}
            member=manifest::next_value(root,member);
        }
        if(!found){return 1;}
        var usize dependency=0;while(dependency<manifest::dependency_count(root)){
            let ref u8[] value=manifest::dependency_value(root,dependency,b"path");
            let ref u8[] expected=manifest::dependency_value(root,dependency,b"version");
            if(!work::charge(meter,path.len+value.len+version.len+expected.len+1)){return 2;}
            if(bytes::equal(path,value) && !bytes::equal(version,expected)){return 1;}dependency+=1;
        }index+=1;
    }
    var usize member=manifest::member_first(root);while(member!=0){let ref u8[] value=manifest::value(root,member);
        if(!work::charge(meter,value.len+cache.len+1)){return 2;}if(overlaps(value,cache)){return 1;}member=manifest::next_value(root,member);
    }return 0;
}
