module project_calls_order;
import project_atlas_join as join;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

fn ref u8[] field(ref join::Graph g,join::Edge e,usize index){
    if(index==0){return join::instance(g,e.source.package);}if(index==1){return join::visible(g,e.source);}
    if(index==2){return join::source_specialization(g,e);}if(index==3){return join::instance(g,e.target.package);}
    if(index==4){return join::visible(g,e.target);}if(index==5){return join::target_specialization(g,e);}
    let p=join::partition(g,e.source.package);let t=validate::tree(p);let span=wire::field(t,e.call,b"span");
    return wire::string(t,span,if(index==6){b"package_instance"}else{b"source"});
}
fn Result<i32,u32> text(ref u8[] left,ref u8[] right,ref mut usize meter){
    if(!work::charge(meter,left.len+right.len+1)){return .err(3);}
    return .ok(if(bytes::equal(left,right)){0}else if(model::before(left,right)){-1}else{1});
}
// Exact Go catalog order, distinct from the join's source/span traversal order.
pub fn Result<i32,u32> compare(ref join::Graph g,join::Edge left,join::Edge right,ref mut usize meter){
    var usize i=0;while(i<8){
        if(i==2 && left.resource_body!=right.resource_body){return .ok(if(left.resource_body){1}else{-1});}
        let c=text(field(g,left,i),field(g,right,i),meter)?;if(c!=0){return .ok(c);}i+=1;
    }
    let a=join::partition(g,left.source.package);let at=validate::tree(a);let x=wire::field(at,left.call,b"span");
    let b=join::partition(g,right.source.package);let bt=validate::tree(b);let y=wire::field(bt,right.call,b"span");
    if(!work::charge(meter,1)){return .err(3);}let start=wire::integer(at,x,b"start_byte");let other=wire::integer(bt,y,b"start_byte");
    if(start!=other){return .ok(if(start<other){-1}else{1});}
    let end=wire::integer(at,x,b"end_byte");let last=wire::integer(bt,y,b"end_byte");return .ok(if(end==last){0}else if(end<last){-1}else{1});
}
// Sort only owned edge indices. Graph and caller-owned arrays remain immutable.
// Bottom-up merge sort costs O(N log N) comparisons and O(N) scratch storage.
pub fn Result<v::Vec<usize>,u32> sort(ref join::Graph g,v::Vec<usize> input,ref mut usize meter){
    var rows=input;let n=v::len<usize>(ref rows);var spare=v::empty<usize>();var usize i=0;
    while(i<n){if(!work::charge(meter,1)){return .err(3);}if(!v::push<usize>(ref mut spare,0).is_ok){return .err(2);}i+=1;}
    var usize width=1;while(width<n){var usize at=0;
        while(at<n){let middle=if(at+width<n){at+width}else{n};let end=if(middle+width<n){middle+width}else{n};
            var usize left=at;var usize right=middle;var usize out=at;
            while(out<end){if(!work::charge(meter,1)){return .err(3);}var usize selected=0;
                if(left<middle && (right==end || compare(g,join::edge(g,*v::at<usize>(ref rows,left)),join::edge(g,*v::at<usize>(ref rows,right)),meter)?<=0)){selected=*v::at<usize>(ref rows,left);left+=1;}
                else{selected=*v::at<usize>(ref rows,right);right+=1;}
                {let slot=v::at_mut<usize>(ref mut spare,out);*slot=selected;}out+=1;
            }at=end;
        }
        i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}let value=*v::at<usize>(ref spare,i);{let slot=v::at_mut<usize>(ref mut rows,i);*slot=value;}i+=1;}
        width*=2;
    }return .ok(rows);
}
