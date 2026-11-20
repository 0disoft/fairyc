module generic_source_binding_types;
import syntax as syntax;
import project_path as budget;
import std::bytes as bytes;

// An untyped binding still obscures an outer annotation.
struct Binding { usize start; usize root; }

fn bool contains(usize start,usize end,syntax::Expression use){return start<=use.start && use.end<=end;}

fn Result<Binding,u32> select(ref syntax::Parsed p,Binding prior,ref u8[] name,usize start,usize end,usize root,ref mut usize work){
    if(start<prior.start || end-start!=name.len){return .ok(prior);}
    if(!budget::charge(work,name.len+1)){return .err(10);}
    return .ok(if(bytes::equal(syntax::source_view(p)[start..end],name)){Binding{.start,.root}}else{prior});
}

fn Result<Binding,u32> entry(ref syntax::Parsed p,Binding prior,ref u8[] name,usize index,ref mut usize work){
    if(!budget::charge(work,1)){return .err(10);}
    let row=syntax::entry_at(p,index);
    return select(p,prior,name,row.name_start,row.name_end,row.type_root,work);
}

fn Result<Binding,u32> local(ref syntax::Parsed p,Binding prior,ref u8[] name,syntax::Statement s,syntax::Expression use,ref mut usize work){
    if(s.kind!=2 || s.end>use.start){return .ok(prior);}
    return entry(p,prior,name,s.entry-1,work);
}

fn Result<Binding,u32> pattern(ref syntax::Parsed p,Binding prior,ref u8[] name,usize id,ref mut usize work){
    let n=syntax::expression_at(p,id);
    if(n.kind==16 && n.left!=0){
        let binding=syntax::expression_at(p,n.left);
        return select(p,prior,name,binding.start,binding.end,0,work);
    }
    var Binding selected=prior;
    var usize link=if(n.kind==32){n.right}else{0};
    while(link!=0){
        if(!budget::charge(work,1)){return .err(10);}
        let field=syntax::expression_at(p,link);
        let binding=syntax::expression_at(p,field.left);
        selected=select(p,selected,name,binding.start,binding.end,0,work)?;
        link=field.right;
    }
    return .ok(selected);
}

// Direct children of enclosing blocks are the only ordinary locals in scope.
// Within those blocks, the latest preceding declaration shadows earlier ones.
pub fn Result<usize,u32> written_root(ref syntax::Parsed p,usize owner,usize id,ref mut usize work){
    let d=syntax::declaration_at(p,owner-1);
    let usize body=if(d.kind==6){d.cleanup_body}else{d.body};
    if(body==0){return .ok(0);}
    let use=syntax::expression_at(p,id);
    let name=syntax::source_view(p)[use.start..use.end];
    let outer=syntax::statement_at(p,body);
    var usize start=outer.start;
    var usize end=outer.end;
    var usize first=d.entry_start;
    var usize count=d.entry_count;
    var bool closure=false;
    var Binding selected=Binding{.start=0,.root=0};
    var usize i=1;
    // Written closures have their own environment. Only their explicitly
    // declared captures and parameters enter that environment.
    while(i<=syntax::expression_count(p)){
        if(!budget::charge(work,1)){return .err(10);}
        let n=syntax::expression_at(p,i);i+=1;
        if(n.start<start || n.end>end || !contains(n.start,n.end,use)){continue;}
        if(n.kind==34){
            let b=syntax::statement_at(p,n.left);
            if(!contains(b.start,b.end,use)){continue;}
            let header=syntax::expression_at(p,n.right);
            start=b.start;end=b.end;first=header.left-1;count=header.right/129+header.right%129;closure=true;
            if(selected.start<start){selected=Binding{.start=0,.root=0};}
        }else if(n.kind==18 && n.left!=0){
            let b=syntax::statement_at(p,n.right);
            if(contains(b.start,b.end,use)){selected=pattern(p,selected,name,n.left,work)?;}
        }
    }
    i=first;
    while(i<first+count){
        if(!budget::charge(work,1)){return .err(10);}
        let row=syntax::entry_at(p,i);
        if(!closure && row.kind!=1){break;}
        selected=select(p,selected,name,row.name_start,row.name_end,row.type_root,work)?;
        i+=1;
    }
    i=1;
    while(i<=syntax::statement_count(p)){
        if(!budget::charge(work,1)){return .err(10);}
        let s=syntax::statement_at(p,i);i+=1;
        if(s.start<start || s.end>end || !contains(s.start,s.end,use)){continue;}
        if(s.kind==1){
            var usize link=s.first;
            while(link!=0){
                if(!budget::charge(work,1)){return .err(10);}
                let row=syntax::statement_at(p,link);
                selected=local(p,selected,name,syntax::statement_at(p,row.first),use,work)?;
                link=row.second;
            }
        }else if(s.kind==14){
            let header=syntax::statement_at(p,s.second);
            if(header.first!=0){selected=local(p,selected,name,syntax::statement_at(p,header.first),use,work)?;}
        }else if(s.kind==17 || s.kind==18){
            let b=syntax::statement_at(p,s.first);
            if(!contains(b.start,b.end,use)){continue;}
            let target=syntax::expression_at(p,s.target);
            selected=select(p,selected,name,target.start,target.end,if(s.kind==18){s.second}else{0},work)?;
        }
    }
    return .ok(selected.root);
}
