module generic_source_inference;
import syntax as syntax;
import static_operation_declarations as scalars;
import project_path as budget;
import std::bytes as bytes;

pub struct Arguments { usize count; u32[8] types; }
struct Constraint { u32 fixed; bool integer; }

// Derive a written scalar context for a call without changing parser IDs.
// Only a return or typed initializer and its transparent value wrappers
// provide context, as in the source request walk.
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
            if(n.kind==3){root=n.left;}
            else if(n.kind==12 || n.kind==31){root=syntax::statement_at(p,n.left).expression;}
            else{break;}
        }
        if(root==call){return .ok(expected);}
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

fn Result<usize,u32> named_root(ref syntax::Parsed p, usize root, ref mut usize work) {
    var usize node=root;
    while (node!=0) {
        if (!budget::charge(work,1)) { return .err(10); }
        let n=syntax::type_at(p,node);
        if (n.kind!=5) { return .ok(if (n.kind==1) { node } else { 0 }); }
        node=n.left;
    }
    return .ok(0);
}

fn Result<u32,u32> scalar_root(ref syntax::Parsed p, usize root, ref mut usize work) {
    let name=named_root(p,root,work)?;
    if (name==0) { return .ok(0); }
    let text=syntax::type_text(p,name);
    if (!budget::charge(work,text.len+1)) { return .err(10); }
    return .ok(scalars::scalar(text));
}

fn Result<Constraint,u32> evidence(ref syntax::Parsed p, usize root, ref mut usize work) {
    var usize id=root;
    while (id!=0) {
        if (!budget::charge(work,1)) { return .err(10); }
        let n=syntax::expression_at(p,id);
        if (n.kind==3) { id=n.left; continue; }
        if (n.kind==23) { return .ok(Constraint{.fixed=scalar_root(p,n.right,work)?,.integer=false}); }
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
    let root=named_root(provider,formal_root,work)?;
    if (root==0) { return .ok(0); }
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

// Side-effect-free request inference apart from charging the caller's work meter.
// The source planner keeps argument order and feeds these scalars into its usual key.
pub fn Result<Arguments,u32> solve(ref syntax::Parsed caller, usize call,
    ref syntax::Parsed provider, usize declaration, usize expected, ref mut usize work) {
    let target=syntax::declaration_at(provider,declaration);
    let count=syntax::generic_parameter_count(provider,declaration);
    if (count==0 || count>8) { return .err(4502); }
    var u32[8] fixed=[0;8];
    var bool[8] integers=[false;8];
    let result=scalar_root(caller,expected,work)?;
    if (result!=0) {
        let binder=binder_index(provider,declaration,target.result_type,count,work)?;
        if (binder!=0) { fixed[binder-1]=result; }
    }
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
        let actual=evidence(caller,link.left,work)?;
        if (actual.fixed!=0 || actual.integer) {
            let binder=binder_index(provider,declaration,entry.type_root,count,work)?;
            if (binder!=0) {
                let index=binder-1;
                if (fixed[index]!=0 && actual.fixed!=0 && fixed[index]!=actual.fixed) { return .err(4502); }
                if (actual.fixed!=0) { fixed[index]=actual.fixed; }
                integers[index]=integers[index] || actual.integer;
            }
        }
        argument=link.right;
    }
    if (argument!=0) { return .err(4502); }
    var usize i=0;
    while (i<count) {
        if (fixed[i]==0 && integers[i]) { fixed[i]=1; }
        if (fixed[i]==0) { return .err(4502); }
        i+=1;
    }
    return .ok(Arguments{.count,.types=fixed});
}
