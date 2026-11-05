module project_pack_relation_order;
import project_pack_relations as relations;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

fn Result<i32,u32> text(ref u8[] a,ref u8[] b,ref mut usize meter){if(!work::charge(meter,a.len+b.len+1)){return .err(3);}return .ok(if(bytes::equal(a,b)){0}else if(model::before(a,b)){-1}else{1});}
pub fn Result<i32,u32> compare(ref validate::Partition p,relations::Relation a,relations::Relation b,ref mut usize meter){
    let t=validate::tree(p);var c=text(relations::source(p,a),relations::source(p,b),meter)?;if(c!=0){return .ok(c);}
    c=text(relations::specialization(p,a),relations::specialization(p,b),meter)?;if(c!=0){return .ok(c);}
    if(a.resource_body!=b.resource_body){return .ok(if(a.resource_body){1}else{-1});}
    if(a.kind!=b.kind){return .ok(if(a.kind<b.kind){-1}else{1});}
    let has_a=a.span<json::count(t);let has_b=b.span<json::count(t);if(has_a!=has_b){return .ok(if(has_a){1}else{-1});}
    if(has_a){let order=model::spans(t,a.span,b.span,meter);if(order==2){return .err(3);}if(order!=0){return .ok(order);}}
    if(a.kind==3 || a.kind==4){
        let order=model::keys(t,wire::field(t,a.item,b"target"),wire::field(t,b.item,b"target"),meter);if(order==2){return .err(3);}if(order!=0){return .ok(order);}
        c=text(wire::string(t,a.target,b"visible"),wire::string(t,b.target,b"visible"),meter)?;if(c!=0){return .ok(c);}
        c=text(wire::string(t,a.item,b"specialization"),wire::string(t,b.item,b"specialization"),meter)?;if(c!=0){return .ok(c);}
    }
    if(a.kind==6){c=text(json::value(t,a.item),json::value(t,b.item),meter)?;if(c!=0){return .ok(c);}}
    if(a.kind==3 || a.kind==4){let names=b"certainty|abi|link_name";var usize start=0;while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}c=text(wire::string(t,a.item,names[start..end]),wire::string(t,b.item,names[start..end]),meter)?;if(c!=0){return .ok(c);}start=end+1;}}
    let escape_a=(a.kind==1 || a.kind==2) && wire::boolean(t,a.item,b"escapes");let escape_b=(b.kind==1 || b.kind==2) && wire::boolean(t,b.item,b"escapes");
    return .ok(if(escape_a==escape_b){0}else if(escape_a){1}else{-1});
}
// Stable O(R log R) mergesort of owned indices; equal observations are retained.
pub fn Result<v::Vec<usize>,u32> sort(ref validate::Partition p,ref v::Vec<relations::Relation> rows,ref mut usize meter){
    let n=v::len<relations::Relation>(rows);var order=v::empty<usize>();var spare=v::empty<usize>();var usize i=0;
    while(i<n){if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut order,i).is_ok || !v::push<usize>(ref mut spare,0).is_ok){return .err(2);}i+=1;}
    var usize width=1;while(width<n){var usize at=0;while(at<n){let middle=if(at+width<n){at+width}else{n};let end=if(middle+width<n){middle+width}else{n};var usize left=at;var usize right=middle;var usize out=at;
        while(out<end){if(!work::charge(meter,1)){return .err(3);}var usize selected=0;
            if(left<middle && (right==end || compare(p,*v::at<relations::Relation>(rows,*v::at<usize>(ref order,left)),*v::at<relations::Relation>(rows,*v::at<usize>(ref order,right)),meter)?<=0)){selected=*v::at<usize>(ref order,left);left+=1;}else{selected=*v::at<usize>(ref order,right);right+=1;}
            {let slot=v::at_mut<usize>(ref mut spare,out);*slot=selected;}out+=1;
        }at=end;
    }i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}let value=*v::at<usize>(ref spare,i);{let slot=v::at_mut<usize>(ref mut order,i);*slot=value;}i+=1;}width*=2;}
    return .ok(order);
}
