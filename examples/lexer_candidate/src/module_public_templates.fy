module module_public_templates;
import generic_source_dependency_model as graph;
import generic_source_catalog as catalog;
import generic_source_closure as closure;
import generic_package_bindings as bindings;
import module_generic_interface as generic;
import module_public_interface_model as model;
import project_path as budget;
import project_manifest_values as lexical;
import std::bytes as bytes;
import std::vec as v;

fn Result<model::Text,u32> retain(ref mut bytes::Buffer names,ref u8[] value,ref mut usize work){if(value.len>1048576 || bytes::len(names)>1048576-value.len || !budget::charge(work,value.len+1)){return .err(10);}let start=bytes::len(names);if(!bytes::append(names,value).is_ok){return .err(11);}return .ok(model::Text{.start,.end=bytes::len(names)});}
fn Result<i32,u32> compare(ref bytes::Buffer names,generic::Template a,generic::Template b,ref mut usize work){let data=bytes::view(names);let am=data[a.module_path.start..a.module_path.end];let bm=data[b.module_path.start..b.module_path.end];let an=data[a.name.start..a.name.end];let bn=data[b.name.start..b.name.end];if(!budget::charge(work,am.len+bm.len+an.len+bn.len+1)){return .err(10);}return .ok(if(!bytes::equal(am,bm)){if(lexical::before(am,bm)){-1}else{1}}else if(bytes::equal(an,bn)){0}else if(lexical::before(an,bn)){-1}else{1});}
fn void swap(ref mut v::Vec<generic::Template> rows,usize a,usize b){let left=*v::at<generic::Template>(rows,a);let right=*v::at<generic::Template>(rows,b);{let slot=v::at_mut<generic::Template>(rows,a);*slot=right;}{let slot=v::at_mut<generic::Template>(rows,b);*slot=left;}}
fn u32 sift(ref bytes::Buffer names,ref mut v::Vec<generic::Template> rows,usize root,usize count,ref mut usize work){var usize at=root;while(at<count/2){var usize child=at*2+1;if(child+1<count){let order=switch(compare(names,*v::at<generic::Template>(rows,child),*v::at<generic::Template>(rows,child+1),work)){case .ok(n):{n}case .err(code):{return code;}};if(order<0){child+=1;}}let order=switch(compare(names,*v::at<generic::Template>(rows,at),*v::at<generic::Template>(rows,child),work)){case .ok(n):{n}case .err(code):{return code;}};if(order>=0){return 0;}swap(rows,at,child);at=child;}return 0;}
fn u32 sort(ref bytes::Buffer names,ref mut v::Vec<generic::Template> rows,ref mut usize work){let count=v::len<generic::Template>(rows);var usize i=count/2;while(i>0){i-=1;let code=sift(names,rows,i,count,work);if(code!=0){return code;}}i=count;while(i>1){i-=1;swap(rows,0,i);let code=sift(names,rows,0,i,work);if(code!=0){return code;}}return 0;}
// Project package-owned public templates from original declarations. Each
// closure retains its independent 1MiB ledger, folded once into project work.
// Uninstantiated public templates count; private dependencies stay in closures.
pub fn Result<generic::Templates,u32> collect(ref graph::Graph input,usize owner,ref mut usize work){
    let templates=graph::templates(input);let packages=catalog::packages(templates);if(owner>=512){return .err(9);}var bool found=false;var usize i=0;
    while(i<bindings::count(packages)){if(!budget::charge(work,1)){return .err(10);}if(bindings::module_at(packages,i).owner==owner){found=true;}i+=1;}if(!found){return .err(9);}
    var bytes::Buffer names=bytes::empty();var v::Vec<generic::Template> rows=v::empty<generic::Template>();i=0;
    while(i<catalog::count(templates)){if(!budget::charge(work,1)){return .err(10);}let row=catalog::template_at(templates,i);let owned_module=bindings::module_at(packages,row.module_index);
        if(row.public && owned_module.owner==owner){var usize measured=0;let digest=switch(closure::digest(input,i,ref mut measured)){case .ok(value):{value}case .err(code):{discard budget::charge(work,measured);return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}
            let module_path=switch(retain(ref mut names,bindings::text(packages,owned_module.path),work)){case .ok(value):{value}case .err(code):{return .err(code);}};let name=switch(retain(ref mut names,catalog::text(templates,row.name),work)){case .ok(value):{value}case .err(code):{return .err(code);}};let closure=switch(retain(ref mut names,bytes::view(ref digest),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            if(!v::push<generic::Template>(ref mut rows,generic::Template{.module_path,.name,.closure}).is_ok){return .err(11);}
        }i+=1;
    }let code=sort(ref names,ref mut rows,work);if(code!=0){return .err(code);}return .ok(generic::assemble(names,rows));
}
