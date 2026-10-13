module generic_source_root_location;
import generic_source_plan_input as input;
import generic_source_dependency_model as roots;
import generic_source_dependency_walk as walk;
import generic_source_catalog as catalog;
import generic_source_arguments as arguments;
import generic_package_bindings as bindings;
import project_manifest_values as lexical;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Location{u32 code;usize module_index;usize start;usize end;}
fn Location fail(u32 code){return Location{.code,.module_index=0,.start=0,.end=0};}
// Original declarations are source ordered. Ignore unused generic bodies and
// parser nodes outside the same ordinary-function/aggregate/cleanup projection.
fn Result<bool,u32> ordinary(ref syntax::Parsed p,usize start,usize end,ref mut usize work){
    var usize first=1;var usize stop=syntax::declaration_count(p);while(first<stop){if(!budget::charge(work,1)){return .err(10);}let at=first+(stop-first)/2;if(syntax::declaration_at(p,at).end<=start){first=at+1;}else{stop=at;}}
    if(first>=syntax::declaration_count(p)){return .ok(false);}let d=syntax::declaration_at(p,first);return .ok(d.start<=start && end<=d.end && d.generic_parameters==0 && (d.kind==3 || d.kind==4 || d.kind==6));
}
fn Result<bool,u32> matches(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize module_index,usize name_start,usize name_end,usize tail,u32 kind,ref u8[] key,ref mut usize work){
    let p=v::at<syntax::Parsed>(sources,module_index);let name=switch(arguments::path(p,name_start,name_end,work)){case .ok(value):{value}case .err(code):{return .err(code);}};let candidate=switch(walk::request_key(c,sources,module_index,bytes::view(ref name),tail,kind,work)){case .ok(value):{value}case .err(code):{if(code==10 || code==11){return .err(code);}return .ok(false);}};if(!budget::charge(work,key.len+bytes::len(ref candidate)+1)){return .err(10);}return .ok(bytes::equal(bytes::view(ref candidate),key));
}
fn Result<bool,u32> before(ref bindings::Bindings packages,Location a,Location b,ref mut usize work){
    let left=bindings::module_at(packages,a.module_index);let right=bindings::module_at(packages,b.module_index);let an=bindings::text(packages,left.namespace);let bn=bindings::text(packages,right.namespace);let ap=bindings::text(packages,left.path);let bp=bindings::text(packages,right.path);if(!budget::charge(work,an.len+bn.len+ap.len+bp.len+1)){return .err(10);}
    return .ok(if(!bytes::equal(an,bn)){lexical::before(an,bn)}else if(!bytes::equal(ap,bp)){lexical::before(ap,bp)}else{a.start<b.start || a.start==b.start && a.end<b.end});
}
fn Result<Location,u32> select(ref bindings::Bindings packages,Location current,usize module_index,usize start,usize end,ref mut usize work){let candidate=Location{.code=0,.module_index,.start,.end};if(current.code!=0){return .ok(candidate);}return switch(before(packages,candidate,current,work)){case .ok(earlier):{.ok(if(earlier){candidate}else{current})}case .err(code):{.err(code)}};}
// Called only after planning fails. Read the retained exact request key and
// original parsed source; do not rebuild templates, bodies, typing or the plan.
// Equivalent aliases select canonical owner/module and then earliest byte span.
pub fn Location find(ref input::Input original,ref v::Vec<syntax::Parsed> sources,usize root,ref mut usize work){
    if(!budget::charge(work,0)){return fail(10);}let data=input::roots(original);if(root==0 || root>roots::edge_count(data)){return fail(9);}let edge=roots::edge_at(data,root-1);let key=roots::text(data,edge.key);let c=input::templates(original);let packages=catalog::packages(c);if(bindings::count(packages)!=v::len<syntax::Parsed>(sources)){return fail(9);}var Location found=fail(9);var usize module_index=0;
    while(module_index<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,module_index);let current=catalog::current(packages,module_index,p,work);if(current!=0){return fail(current);}var usize i=1;
        while(i<=syntax::expression_count(p)){if(!budget::charge(work,1)){return fail(10);}let n=syntax::expression_at(p,i);if((n.kind==6 || n.kind==10) && n.left!=0 && n.left<i){let applied=syntax::expression_at(p,n.left);if(applied.kind==20 && applied.left!=0 && applied.left<n.left){let name=syntax::expression_at(p,applied.left);let admitted=switch(ordinary(p,n.start,n.end,work)){case .ok(value):{value}case .err(code):{return fail(code);}};
                    if(name.kind==2 && admitted){let same=switch(matches(c,sources,module_index,name.start,name.end,applied.right,if(n.kind==6){3}else{0},key,work)){case .ok(value):{value}case .err(code):{return fail(code);}};if(same){found=switch(select(packages,found,module_index,applied.start,applied.end,work)){case .ok(value):{value}case .err(code):{return fail(code);}};}}
                }}i+=1;
        }i=1;while(i<=syntax::type_count(p)){if(!budget::charge(work,1)){return fail(10);}let n=syntax::type_at(p,i);if(n.kind==6 && n.left!=0 && n.left<i){let name=syntax::type_at(p,n.left);let admitted=switch(ordinary(p,n.start,n.end,work)){case .ok(value):{value}case .err(code):{return fail(code);}};if(name.kind==1 && admitted){let same=switch(matches(c,sources,module_index,name.start,name.end,n.right,0,key,work)){case .ok(value):{value}case .err(code):{return fail(code);}};if(same){found=switch(select(packages,found,module_index,n.start,n.end,work)){case .ok(value):{value}case .err(code):{return fail(code);}};}}}i+=1;}
        module_index+=1;
    }return found;
}
