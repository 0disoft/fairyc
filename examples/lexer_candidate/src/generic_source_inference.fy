module generic_source_inference;
import syntax as syntax;
import static_operation_declarations as scalars;
import project_path as budget;
import generic_source_binding_types as bindings;
import generic_source_catalog as catalog;
import generic_source_arguments as paths;
import module_signatures as signatures;
import std::bytes as bytes;
import std::vec as v;

pub struct Arguments { usize count; u32[8] types; }
struct Tagged { usize first; usize second; }
struct Constraints { usize count; u32[8] fixed; bool[8] integers; }
struct EvidenceType { usize owner; usize root; Arguments args; }
struct Target { usize module_index; usize declaration; }
pub struct Site { usize module_index; usize owner; usize call; usize expected; }

fn EvidenceType written_type(usize owner,usize root){
    return EvidenceType{.owner,.root,.args=Arguments{.count=0,.types=[0;8]}};
}

// Concrete scalar tags end at 38. Keep a distinct representation for each
// source binder until the selected caller instance supplies its argument.
pub fn u32 binder_base(){return 64;}
pub fn bool forwarded(u32 code){return code>=binder_base();}
pub fn usize forwarded_index(u32 code){return (code-binder_base()) as usize;}

fn bool covers(usize start,usize end,syntax::Expression site){
    return start<=site.start && site.end<=end;
}

// A result constraint follows only value-producing children. In particular,
// it cannot reach the condition, the switch subject, or a case pattern.
fn Result<usize,u32> value_child(ref syntax::Parsed p,syntax::Expression n,syntax::Expression site,ref mut usize work){
    if(n.kind==3){return .ok(n.left);}
    if(n.kind==12 || n.kind==31){return .ok(syntax::statement_at(p,n.left).expression);}
    if(n.kind==13){
        let branches=syntax::expression_at(p,n.right);
        let first=syntax::statement_at(p,branches.left);
        return .ok(if(covers(first.start,first.end,site)){first.expression}else{branches.right});
    }
    if(n.kind==15){
        var usize link=n.right;
        while(link!=0){
            if(!budget::charge(work,1)){return .err(10);}
            let row=syntax::expression_at(p,link);
            let arm=syntax::expression_at(p,row.left);
            let body=syntax::statement_at(p,arm.right);
            if(covers(body.start,body.end,site)){return .ok(body.expression);}
            link=row.right;
        }
    }
    return .ok(0);
}

// Derive the same written result context as the source request walk.
pub fn Result<usize,u32> expected_root(ref syntax::Parsed p, syntax::Declaration owner, usize call, ref mut usize work) {
    let site=syntax::expression_at(p,call);
    var usize i=1;
    while(i<=syntax::statement_count(p)){
        if(!budget::charge(work,1)){return .err(10);}
        let s=syntax::statement_at(p,i);i+=1;
        if(s.start<owner.start || s.end>owner.end || s.start>site.start || s.end<site.end){continue;}
        var usize root=s.expression;var usize expected=0;
        if(s.kind==3){expected=owner.result_type;}
        else if(s.kind==2){let e=syntax::entry_at(p,s.entry-1);root=e.expression;expected=e.type_root;}
        if(expected==0){continue;}
        while(root>call){
            if(!budget::charge(work,1)){return .err(10);}
            let n=syntax::expression_at(p,root);
            if(!covers(n.start,n.end,site)){break;}
            if(n.kind==6){
                expected=constructor_context(p,n,expected,work)?;
                if(expected==0){break;}
                root=syntax::expression_at(p,n.right).left;
            }else{root=value_child(p,n,site,work)?;}
        }
        if(root==call && expected!=0){return .ok(expected);}
    }
    return .ok(0);
}

pub fn ref u8[] spelling(u32 code) {
    if (code==32) { return b"i8"; }
    if (code==33) { return b"i16"; }
    if (code==34) { return b"isize"; }
    if (code==35) { return b"u16"; }
    if (code==36) { return b"u64"; }
    if (code==37) { return b"f32"; }
    if (code==38) { return b"f64"; }
    if (code==1) { return b"i32"; }
    if (code==2) { return b"i64"; }
    if (code==3) { return b"u8"; }
    if (code==4) { return b"u32"; }
    if (code==5) { return b"usize"; }
    assert(code==6);
    return b"bool";
}

fn Result<usize,u32> ungroup_type(ref syntax::Parsed p, usize root, ref mut usize work) {
    var usize node=root;
    while (node!=0) {
        if (!budget::charge(work,1)) { return .err(10); }
        let n=syntax::type_at(p,node);
        if (n.kind!=5) { return .ok(node); }
        node=n.left;
    }
    return .ok(0);
}

fn Result<u32,u32> scalar_root(ref syntax::Parsed p, usize owner, usize root, ref mut usize work) {
    let name=ungroup_type(p,root,work)?;
    if (name==0 || syntax::type_at(p,name).kind!=1) { return .ok(0); }
    let text=syntax::type_text(p,name);
    if (!budget::charge(work,text.len+1)) { return .err(10); }
    let scalar=scalars::scalar(text);
    if(scalar!=0 || owner==0){return .ok(scalar);}
    let binder=binder_index(p,owner-1,name,syntax::generic_parameter_count(p,owner-1),work)?;
    if(binder==0){return .ok(0);}
    var u32 code=binder_base();
    var usize i=1;
    while(i<binder){code+=1;i+=1;}
    return .ok(code);
}

fn Result<usize,u32> binder_index(ref syntax::Parsed provider, usize declaration, usize formal_root,
    usize count, ref mut usize work) {
    let root=ungroup_type(provider,formal_root,work)?;
    if (root==0 || syntax::type_at(provider,root).kind!=1) { return .ok(0); }
    let formal=syntax::type_text(provider,root);
    var usize i=0;
    while (i<count) {
        let binder=syntax::generic_parameter_at(provider,declaration,i);
        if (!budget::charge(work,formal.len+1)) { return .err(10); }
        if (bytes::equal(formal,syntax::source_view(provider)[binder.start..binder.end])) {
            return .ok(i+1);
        }
        i+=1;
    }
    return .ok(0);
}

// The parser links type arguments in reverse source order.
fn Result<Tagged,u32> tagged(ref syntax::Parsed p,usize root,ref mut usize work){
    let empty=Tagged{.first=0,.second=0};
    if(root==0){return .ok(empty);}
    let t=syntax::type_at(p,root);
    if(t.kind!=6 || t.left==0 || t.right==0){return .ok(empty);}
    let name=syntax::type_text(p,t.left);
    if(!budget::charge(work,name.len+1)){return .err(10);}
    let last=syntax::type_at(p,t.right);
    if(last.kind!=7){return .err(9);}
    if(last.right==0 && bytes::equal(name,b"Option")){
        return .ok(Tagged{.first=last.left,.second=0});
    }
    if(last.right==0 || !bytes::equal(name,b"Result")){return .ok(empty);}
    let first=syntax::type_at(p,last.right);
    if(first.kind!=7){return .err(9);}
    return .ok(if(first.right==0){Tagged{.first=first.left,.second=last.left}}else{empty});
}

fn Result<usize,u32> constructor_payload(ref syntax::Parsed p,usize root,ref u8[] name,ref mut usize work){
    if(!budget::charge(work,name.len)){return .err(10);}
    let shape=tagged(p,root,work)?;
    if(shape.second==0){return .ok(if(bytes::equal(name,b"some")){shape.first}else{0});}
    if(bytes::equal(name,b"ok")){return .ok(shape.first);}
    return .ok(if(bytes::equal(name,b"err")){shape.second}else{0});
}

pub fn Result<usize,u32> constructor_context(ref syntax::Parsed p,syntax::Expression call,usize expected,ref mut usize work){
    if(expected==0 || call.left==0 || call.right==0){return .ok(0);}
    if(syntax::expression_at(p,call.left).kind!=28){return .ok(0);}
    let argument=syntax::expression_at(p,call.right);
    if(argument.kind!=7 || argument.left==0 || argument.right!=0){return .ok(0);}
    let root=ungroup_type(p,expected,work)?;
    return constructor_payload(p,root,syntax::expression_text(p,call.left),work);
}

fn Result<bool,u32> constrain_fixed(ref mut Constraints constraints,usize binder,u32 value){
    if(value==0){return .ok(true);}
    let prior=constraints.fixed[binder-1];
    if(prior!=0 && prior!=value){return .err(4502);}
    constraints.fixed[binder-1]=value;
    return .ok(true);
}

// Each scalar in a matching tagged shape can constrain a binder, including
// multiple occurrences of the same binder in a Result.
fn Result<bool,u32> constrain_type(ref syntax::Parsed caller,EvidenceType actual,
    ref syntax::Parsed provider,usize declaration,usize formal,ref mut Constraints constraints,ref mut usize work){
    if(actual.root==0 || formal==0){return .ok(true);}
    let binder=binder_index(provider,declaration,formal,constraints.count,work)?;
    if(binder!=0){
        let scalar=scalar_root(caller,actual.owner,actual.root,work)?;
        if(actual.args.count==0 || !forwarded(scalar)){return constrain_fixed(constraints,binder,scalar);}
        let index=forwarded_index(scalar);
        if(index>=actual.args.count){return .err(9);}
        return constrain_fixed(constraints,binder,actual.args.types[index]);
    }
    let pattern=tagged(provider,ungroup_type(provider,formal,work)?,work)?;
    if(pattern.first==0){return .ok(true);}
    let supplied=tagged(caller,ungroup_type(caller,actual.root,work)?,work)?;
    if(supplied.first==0 || (pattern.second==0)!=(supplied.second==0)){return .ok(true);}
    let first=EvidenceType{.owner=actual.owner,.root=supplied.first,.args=actual.args};
    let second=EvidenceType{.owner=actual.owner,.root=supplied.second,.args=actual.args};
    discard constrain_type(caller,first,provider,declaration,pattern.first,constraints,work)?;
    return constrain_type(caller,second,provider,declaration,pattern.second,constraints,work);
}

fn Result<Target,u32> nested_target(usize module_index,ref u8[] name,Option<ref catalog::Catalog> templates,
    Option<ref signatures::Program> metadata,ref mut usize work){
    let absent=Target{.module_index,.declaration=0};
    switch(templates){
        case .some(c):{
            let found=catalog::lookup(c,module_index,name,3,work)?;
            if(found==0){return .ok(absent);}
            let t=catalog::template_at(c,found-1);
            return .ok(Target{.module_index=t.module_index,.declaration=t.declaration+1});
        }
        case .none:{}
    };
    switch(metadata){
        case .some(m):{
            var usize probes=0;
            let found=signatures::declaration_lookup_measured(m,module_index,name,ref mut probes);
            if(!budget::charge(work,probes)){return .err(10);}
            if(found.code!=0){return .ok(absent);}
            let t=signatures::declaration_at(m,found.item);
            if(t.kind==3 && t.parameters!=0){return .ok(Target{.module_index=t.module_index,.declaration=t.declaration});}
        }
        case .none:{}
    };
    return .ok(absent);
}

fn Result<Arguments,u32> written_arguments(ref syntax::Parsed caller,usize owner,usize selector,usize count,ref mut usize work){
    if(count==0 || count>8 || syntax::specialization_argument_count(caller,selector)!=count){return .err(4502);}
    var Arguments args=Arguments{.count,.types=[0;8]};
    var usize i=0;
    while(i<count){
        args.types[i]=scalar_root(caller,owner,syntax::specialization_argument(caller,selector,i),work)?;
        i+=1;
    }
    return .ok(args);
}

// Recursion only follows argument expressions, whose parser IDs are strictly
// smaller than the enclosing call. It never traverses the callee's body.
fn Result<bool,u32> constrain_nested(ref syntax::Parsed caller,Site site,usize id,
    ref syntax::Parsed provider,usize declaration,usize formal,ref mut Constraints constraints,
    ref v::Vec<syntax::Parsed> sources,Option<ref catalog::Catalog> templates,Option<ref signatures::Program> metadata,ref mut usize work){
    let call=syntax::expression_at(caller,id);
    if(call.left==0 || call.left>=id){return .err(9);}
    let selector=syntax::expression_at(caller,call.left);
    let explicit=selector.kind==20;
    let name=if(explicit){syntax::expression_at(caller,selector.left)}else{selector};
    if(name.kind!=2 || name.token>=1048576){return .ok(true);}
    let spelling=paths::path(caller,name.start,name.end,work)?;
    let found=nested_target(site.module_index,bytes::view(ref spelling),templates,metadata,work)?;
    if(found.declaration==0){return .ok(true);}
    let source=v::at<syntax::Parsed>(sources,found.module_index);
    let d=syntax::declaration_at(source,found.declaration-1);
    let args=if(explicit){
        written_arguments(caller,site.owner,call.left,syntax::generic_parameter_count(source,found.declaration-1),work)?
    }else{
        let nested=Site{.module_index=site.module_index,.owner=site.owner,.call=id,.expected=0};
        solve(caller,nested,source,found.declaration-1,sources,templates,metadata,work)?
    };
    let actual=EvidenceType{.owner=found.declaration,.root=d.result_type,.args};
    return constrain_type(source,actual,provider,declaration,formal,constraints,work);
}

fn Result<bool,u32> constrain_argument(ref syntax::Parsed caller,Site site,usize argument,
    ref syntax::Parsed provider,usize declaration,usize formal,ref mut Constraints constraints,
    ref v::Vec<syntax::Parsed> sources,Option<ref catalog::Catalog> templates,Option<ref signatures::Program> metadata,ref mut usize work){
    var usize id=argument;
    var usize expected=formal;
    while(id!=0 && expected!=0){
        if(!budget::charge(work,1)){return .err(10);}
        let n=syntax::expression_at(caller,id);
        if(n.kind==3){id=n.left;continue;}
        expected=ungroup_type(provider,expected,work)?;
        if(expected==0){break;}
        if(n.kind==6 && n.left!=0 && n.right!=0 && syntax::expression_at(caller,n.left).kind==28){
            let link=syntax::expression_at(caller,n.right);
            if(link.kind!=7 || link.right!=0 || link.left==0 || link.left>=n.right || n.right>=id){break;}
            expected=constructor_payload(provider,expected,syntax::expression_text(caller,n.left),work)?;
            id=link.left;
            continue;
        }
        if(n.kind==1){
            let binder=binder_index(provider,declaration,expected,constraints.count,work)?;
            if(binder==0){break;}
            let raw=syntax::expression_text(caller,id);
            if(bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){return constrain_fixed(constraints,binder,6);}
            if(syntax::literal_kind(caller,id)==5){constraints.integers[binder-1]=true;}
            break;
        }
        if(n.kind==6){return constrain_nested(caller,site,id,provider,declaration,expected,constraints,sources,templates,metadata,work);}
        if(n.kind==23){return constrain_type(caller,written_type(site.owner,n.right),provider,declaration,expected,constraints,work);}
        if(n.kind!=2 || site.owner==0){break;}
        let binder=binder_index(provider,declaration,expected,constraints.count,work)?;
        if(binder==0 && tagged(provider,expected,work)?.first==0){break;}
        let written=bindings::written_root(caller,site.owner,id,work)?;
        if(binder!=0){return constrain_fixed(constraints,binder,scalar_root(caller,site.owner,written,work)?);}
        return constrain_type(caller,written_type(site.owner,written),provider,declaration,expected,constraints,work);
    }
    return .ok(true);
}

// Side-effect-free request inference apart from charging the caller's work meter.
// The source planner keeps argument order and feeds these scalars into its usual key.
pub fn Result<Arguments,u32> solve(ref syntax::Parsed caller, Site site, ref syntax::Parsed provider, usize declaration,
    ref v::Vec<syntax::Parsed> sources,Option<ref catalog::Catalog> templates,Option<ref signatures::Program> metadata,ref mut usize work) {
    let target=syntax::declaration_at(provider,declaration);
    let count=syntax::generic_parameter_count(provider,declaration);
    if (count==0 || count>8) { return .err(4502); }
    var Constraints constraints=Constraints{.count,.fixed=[0;8],.integers=[false;8]};
    discard constrain_type(caller,written_type(site.owner,site.expected),provider,declaration,target.result_type,ref mut constraints,work)?;
    var usize parameter=target.entry_start+target.entry_count;
    var usize argument=syntax::expression_at(caller,site.call).right;
    // Both lists are consumed in reverse source order, in one pass each.
    while (parameter>target.entry_start) {
        parameter-=1;
        if (!budget::charge(work,1)) { return .err(10); }
        let entry=syntax::entry_at(provider,parameter);
        if (entry.kind!=1) { continue; }
        if (argument==0) { return .err(4502); }
        if (argument>=site.call) { return .err(9); }
        let link=syntax::expression_at(caller,argument);
        if (link.kind!=7 || link.left==0 || link.left>=argument || link.right>=argument) { return .err(9); }
        discard constrain_argument(caller,site,link.left,provider,declaration,entry.type_root,ref mut constraints,sources,templates,metadata,work)?;
        argument=link.right;
    }
    if (argument!=0) { return .err(4502); }
    var usize i=0;
    while (i<count) {
        if (constraints.fixed[i]==0 && constraints.integers[i]) { constraints.fixed[i]=1; }
        if (constraints.fixed[i]==0) { return .err(4502); }
        i+=1;
    }
    return .ok(Arguments{.count,.types=constraints.fixed});
}
