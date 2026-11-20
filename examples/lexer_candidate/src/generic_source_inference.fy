module generic_source_inference;
import syntax as syntax;
import static_operation_declarations as scalars;
import project_path as budget;
import generic_source_binding_types as bindings;
import std::bytes as bytes;

pub struct Arguments { usize count; u32[8] types; }
struct Constraint { u32 fixed; bool integer; }
struct ArgumentEvidence { usize binder; Constraint actual; }
struct Tagged { usize first; usize second; }
struct ResultConstraints { usize count; u32[8] fixed; }

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

fn Result<Constraint,u32> evidence(ref syntax::Parsed p, usize owner, usize root, ref mut usize work) {
    var usize id=root;
    while (id!=0) {
        if (!budget::charge(work,1)) { return .err(10); }
        let n=syntax::expression_at(p,id);
        if (n.kind==3) { id=n.left; continue; }
        if (n.kind==23) { return .ok(Constraint{.fixed=scalar_root(p,owner,n.right,work)?,.integer=false}); }
        if (n.kind==2 && owner!=0) {
            let written=bindings::written_root(p,owner,id,work)?;
            return .ok(Constraint{.fixed=scalar_root(p,owner,written,work)?,.integer=false});
        }
        if (n.kind==1) {
            let raw=syntax::expression_text(p,id);
            let boolean=bytes::equal(raw,b"true") || bytes::equal(raw,b"false");
            return .ok(Constraint{.fixed=if (boolean) { 6 } else { 0 },.integer=syntax::literal_kind(p,id)==5});
        }
        break;
    }
    return .ok(Constraint{.fixed=0,.integer=false});
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

// Unify only matching tagged shapes. Each scalar occurrence can constrain
// a binder, including multiple occurrences of the same binder in a Result.
fn Result<bool,u32> constrain_result(ref syntax::Parsed caller,usize owner,usize actual,
    ref syntax::Parsed provider,usize declaration,usize formal,ref mut ResultConstraints constraints,ref mut usize work){
    if(actual==0 || formal==0){return .ok(true);}
    let binder=binder_index(provider,declaration,formal,constraints.count,work)?;
    if(binder!=0){
        let value=scalar_root(caller,owner,actual,work)?;
        if(value==0){return .ok(true);}
        let prior=constraints.fixed[binder-1];
        if(prior!=0 && prior!=value){return .err(4502);}
        constraints.fixed[binder-1]=value;
        return .ok(true);
    }
    let pattern=tagged(provider,ungroup_type(provider,formal,work)?,work)?;
    if(pattern.first==0){return .ok(true);}
    let supplied=tagged(caller,ungroup_type(caller,actual,work)?,work)?;
    if(supplied.first==0 || (pattern.second==0)!=(supplied.second==0)){return .ok(true);}
    discard constrain_result(caller,owner,supplied.first,provider,declaration,pattern.first,constraints,work)?;
    return constrain_result(caller,owner,supplied.second,provider,declaration,pattern.second,constraints,work);
}

fn Result<ArgumentEvidence,u32> argument_evidence(ref syntax::Parsed caller,usize owner,usize argument,
    ref syntax::Parsed provider,usize declaration,usize formal,usize count,ref mut usize work){
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
        let binder=binder_index(provider,declaration,expected,count,work)?;
        if(binder==0){break;}
        let actual=evidence(caller,owner,id,work)?;
        return .ok(ArgumentEvidence{.binder,.actual});
    }
    return .ok(ArgumentEvidence{.binder=0,.actual=Constraint{.fixed=0,.integer=false}});
}

// Side-effect-free request inference apart from charging the caller's work meter.
// The source planner keeps argument order and feeds these scalars into its usual key.
pub fn Result<Arguments,u32> solve(ref syntax::Parsed caller, usize owner, usize call,
    ref syntax::Parsed provider, usize declaration, usize expected, ref mut usize work) {
    let target=syntax::declaration_at(provider,declaration);
    let count=syntax::generic_parameter_count(provider,declaration);
    if (count==0 || count>8) { return .err(4502); }
    var ResultConstraints constraints=ResultConstraints{.count,.fixed=[0;8]};
    var bool[8] integers=[false;8];
    discard constrain_result(caller,owner,expected,provider,declaration,target.result_type,ref mut constraints,work)?;
    var usize parameter=target.entry_start+target.entry_count;
    var usize argument=syntax::expression_at(caller,call).right;
    // Both lists are consumed in reverse source order, in one pass each.
    while (parameter>target.entry_start) {
        parameter-=1;
        if (!budget::charge(work,1)) { return .err(10); }
        let entry=syntax::entry_at(provider,parameter);
        if (entry.kind!=1) { continue; }
        if (argument==0) { return .err(4502); }
        if (argument>=call) { return .err(9); }
        let link=syntax::expression_at(caller,argument);
        if (link.kind!=7 || link.left==0 || link.left>=argument || link.right>=argument) { return .err(9); }
        let evidence=argument_evidence(caller,owner,link.left,provider,declaration,entry.type_root,count,work)?;
        if (evidence.binder!=0) {
            let index=evidence.binder-1;
            let actual=evidence.actual;
            if (constraints.fixed[index]!=0 && actual.fixed!=0 && constraints.fixed[index]!=actual.fixed) { return .err(4502); }
            if (actual.fixed!=0) { constraints.fixed[index]=actual.fixed; }
            integers[index]=integers[index] || actual.integer;
        }
        argument=link.right;
    }
    if (argument!=0) { return .err(4502); }
    var usize i=0;
    while (i<count) {
        if (constraints.fixed[i]==0 && integers[i]) { constraints.fixed[i]=1; }
        if (constraints.fixed[i]==0) { return .err(4502); }
        i+=1;
    }
    return .ok(Arguments{.count,.types=constraints.fixed});
}
