module closure_value_source_set;
import closure_value_type_source as closure;
import module_catalog as catalog;
import syntax as syntax;
import std::vec as v;

// Source policy only. The signature owner authenticates the complete source set
// before this pass; no retained table or rendered type text grants authority.
fn catalog::Problem fail(u32 code,usize module_index,usize start,usize end){return catalog::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
pub fn bool needed(ref v::Vec<syntax::Parsed> sources){
    var usize m=0;while(m<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,m);var usize i=1;while(i<=syntax::type_count(p)){if(syntax::type_at(p,i).kind==17){return true;}i+=1;}m+=1;}return false;
}
fn Result<usize,u32> named(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref mut usize work){
    if(root==0 || root>syntax::type_count(p)){return .err(9);}let n=syntax::type_at(p,root);if(n.kind!=1){return .ok(0);}
    var usize probes=0;let found=catalog::lookup_measured(c,module_index,syntax::type_text(p,root),ref mut probes);if(!charge(work,probes+1)){return .err(10);}return .ok(if(found.code==0){found.item}else{0});
}
// Structural children point backwards. Nominal edges read the fixed-point
// table instead of following source recursion, so cycles cannot hide a closure.
fn Result<bool,u32> contains(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref v::Vec<u8> flags,ref mut usize work){
    if(root==0){return .ok(false);}var pending=v::empty<usize>();if(!v::push<usize>(ref mut pending,root).is_ok){return .err(11);}
    while(v::len<usize>(ref pending)!=0){let usize id=switch(v::pop<usize>(ref mut pending)){case .some(n):{n}case .none:{0}};
        if(!charge(work,1)){return .err(10);}if(id==0 || id>syntax::type_count(p)){return .err(9);}let n=syntax::type_at(p,id);if(n.kind==17){return .ok(true);}
        if(n.kind==1){let target=switch(named(c,p,module_index,id,work)){case .ok(n):{n}case .err(code):{return .err(code);}};if(target!=0 && target<v::len<u8>(flags) && *v::at<u8>(flags,target)!=0){return .ok(true);}}
        else{if(n.left>=id || n.right>=id){return .err(9);}if(n.left!=0 && !v::push<usize>(ref mut pending,n.left).is_ok || n.right!=0 && !v::push<usize>(ref mut pending,n.right).is_ok){return .err(11);}}
    }return .ok(false);
}
fn Result<bool,u32> nominal_contains(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,catalog::Declaration row,ref v::Vec<u8> flags,ref mut usize work){
    let p=v::at<syntax::Parsed>(sources,row.module_index);let d=syntax::declaration_at(p,row.declaration-1);var usize i=0;
    while(i<d.field_count){let f=syntax::field_at(p,d.field_start+i);let found=switch(contains(c,p,row.module_index,f.type_root,flags,work)){case .ok(b):{b}case .err(code):{return .err(code);}};if(found){return .ok(true);}i+=1;}return .ok(false);
}
fn Result<bool,u32> direct(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,ref v::Vec<u8> flags,ref mut usize work){
    let n=syntax::type_at(p,root);if(n.kind==17){return .ok(closure::signature(p,root) && closure::root(p,root));}if(n.kind!=1){return .ok(false);}
    let id=switch(named(c,p,module_index,root,work)){case .ok(n):{n}case .err(code):{return .err(code);}};if(id==0 || id>=v::len<u8>(flags) || *v::at<u8>(flags,id)==0){return .ok(false);}let d=catalog::declaration_at(c,id);return .ok(d.kind==4 && d.parameters==0);
}
fn catalog::Problem use(ref catalog::Catalog c,ref syntax::Parsed p,usize module_index,usize root,bool allowed,ref v::Vec<u8> flags,ref mut usize work){
    let found=switch(contains(c,p,module_index,root,flags,work)){case .ok(b):{b}case .err(code):{return fail(code,module_index,0,0);}};if(!found){return fail(0,0,0,0);}let n=syntax::type_at(p,root);
    let valid=switch(direct(c,p,module_index,root,flags,work)){case .ok(b):{b}case .err(code):{return fail(code,module_index,n.start,n.end);}};return fail(if(allowed && valid){0}else{9},module_index,n.start,n.end);
}
pub fn catalog::Problem validate_current(ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog c){
    if(!needed(sources)){return fail(0,0,0,0);}if(v::len<syntax::Parsed>(sources)!=catalog::module_count(c)){return fail(12,0,0,0);}var usize work=0;var flags=v::empty<u8>();var usize i=0;
    while(i<=catalog::declaration_count(c)){if(!charge(ref mut work,1)){return fail(10,0,0,0);}if(!v::push<u8>(ref mut flags,0).is_ok){return fail(11,0,0,0);}i+=1;}
    var bool changed=true;while(changed){changed=false;i=1;while(i<=catalog::declaration_count(c)){if(!charge(ref mut work,1)){return fail(10,0,0,0);}let d=catalog::declaration_at(c,i);
        if((d.kind==4 || d.kind==6 || d.kind==11) && *v::at<u8>(ref flags,i)==0){let found=switch(nominal_contains(c,sources,d,ref flags,ref mut work)){case .ok(b):{b}case .err(code):{return fail(code,d.module_index,d.source_start,d.source_end);}};if(found){let slot=v::at_mut<u8>(ref mut flags,i);*slot=1;changed=true;}}i+=1;
    }}
    var usize m=0;while(m<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,m);i=0;
        while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(!charge(ref mut work,1)){return fail(10,m,d.start,d.end);}let ordinary=d.kind==3 && d.generic_parameters==0;
            let result=use(c,p,m,d.result_type,ordinary,ref flags,ref mut work);if(result.code!=0){return result;}
            var usize j=0;while(j<d.entry_count){let e=syntax::entry_at(p,d.entry_start+j);let checked=use(c,p,m,e.type_root,ordinary && (e.kind==1 || e.kind==2),ref flags,ref mut work);if(checked.code!=0){return checked;}j+=1;}
            j=0;while(j<d.field_count){let f=syntax::field_at(p,d.field_start+j);let found=switch(contains(c,p,m,f.type_root,ref flags,ref mut work)){case .ok(b):{b}case .err(code):{return fail(code,m,f.start,f.end);}};
                if(found){let n=syntax::type_at(p,f.type_root);if(d.kind!=4 || d.generic_parameters!=0 || closure::repr_c(p,i) || f.kind!=0 || n.kind!=17 || !closure::signature(p,f.type_root)){return fail(9,m,n.start,n.end);}}j+=1;
            }i+=1;
        }
        i=1;while(i<=syntax::expression_count(p)){if(!charge(ref mut work,1)){return fail(10,m,0,0);}let e=syntax::expression_at(p,i);if(e.kind==20){var usize link=e.right;while(link!=0){if(link>syntax::type_count(p)){return fail(9,m,e.start,e.end);}let n=syntax::type_at(p,link);let found=switch(contains(c,p,m,n.left,ref flags,ref mut work)){case .ok(b):{b}case .err(code):{return fail(code,m,n.start,n.end);}};if(found){return fail(4502,m,n.start,n.end);}if(n.right>=link){return fail(9,m,n.start,n.end);}link=n.right;}}i+=1;}
        m+=1;
    }return fail(0,0,0,0);
}
