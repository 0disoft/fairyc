module sdk_reachability;
import syntax as syntax;
import module_signatures as signatures;
import module_catalog as catalog;
import sdk_authority as sdk;
import std::bytes as bytes;
import std::vec as v;

pub struct Row {usize module_index;usize declaration;usize start;usize end;bool function;bool needed;usize first;}
struct Use {usize expression;usize next;}
struct Cached {usize module_index;usize expression;usize target;}
pub resource Plan {v::Vec<Row> rows;v::Vec<usize> bases;cleanup{}}
fn catalog::Problem fail(u32 code,usize module_index,usize start,usize end){return catalog::Problem{.code,.module_index,.start,.end};}
pub fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
pub fn usize base(ref Plan p,usize module_index){return *v::at<usize>(ref p.bases,module_index);}
pub fn usize stop(ref Plan p,usize module_index){return *v::at<usize>(ref p.bases,module_index+1);}
pub fn bool needed(ref Plan p,usize module_index,usize declaration){return v::at<Row>(ref p.rows,base(p,module_index)+declaration).needed;}
// Source declarations are disjoint and ordered, except the enclosing module.
// A binary search avoids scanning every declaration for each expression.
fn usize owner_rows(ref v::Vec<Row> rows,ref v::Vec<usize> bases,usize module_index,usize start,usize end,ref mut usize work){
    var usize low=*v::at<usize>(bases,module_index);var usize high=*v::at<usize>(bases,module_index+1);
    while(low<high){if(!charge(work,1)){return 0;}let usize mid=low+(high-low)/2;if(v::at<Row>(rows,mid).start<=start){low=mid+1;}else{high=mid;}}
    if(low==*v::at<usize>(bases,module_index)){return 0;}let r=v::at<Row>(rows,low-1);return if(r.start<=start && end<=r.end){low}else{0};
}
pub fn usize owner(ref Plan p,usize module_index,usize start,usize end,ref mut usize work){return owner_rows(ref p.rows,ref p.bases,module_index,start,end,work);}
pub fn bool keep(ref Plan p,usize module_index,usize start,usize end,ref mut usize work){let usize id=owner(p,module_index,start,end,work);return id==0 || !v::at<Row>(ref p.rows,id-1).function || v::at<Row>(ref p.rows,id-1).needed;}
// Cache both resolved declarations and misses by exact caller/module spelling.
// Collisions compare original bytes; aliases never become declaration identity.
fn usize lookup(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref v::Vec<Row> rows,ref v::Vec<usize> bases,ref mut v::Vec<Cached> cache,usize module_index,usize expression,ref mut usize work,ref mut u32 code){
    let file=v::at<syntax::Parsed>(sources,module_index);let name=syntax::expression_text(file,expression);
    var usize hash=module_index;var usize i=0;while(i<name.len){if(!charge(work,1)){*code=10;return 0;}hash=(hash*131+(name[i] as usize))%1048573;i+=1;}
    var usize slot=hash%4096;var usize probes=0;
    while(probes<4096){if(!charge(work,1)){*code=10;return 0;}let c=*v::at<Cached>(cache,slot);
        if(c.expression==0){
            if(!charge(work,v::len<Row>(rows)+v::len<syntax::Parsed>(sources)+1)){*code=10;return 0;}
            let found=signatures::declaration_lookup(metadata,module_index,name);var usize target=0;
            // Retain private targets; the ordinary binder owns privacy rejection.
            if(found.item!=0 && (found.code==0 || found.code==6)){let d=signatures::declaration_at(metadata,found.item);if(d.kind==3){
                if(d.module_index>=v::len<syntax::Parsed>(sources) || d.declaration==0 || d.declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,d.module_index))){*code=9;return 0;}
                let usize at=*v::at<usize>(bases,d.module_index)+d.declaration-1;let r=v::at<Row>(rows,at);
                if(!r.function || r.start!=d.source_start || r.end!=d.source_end){*code=9;return 0;}target=at+1;
            }}
            {let ref mut Cached c=v::at_mut<Cached>(cache,slot);c.module_index=module_index;c.expression=expression;c.target=target;}return target;
        }
        if(c.module_index==module_index){if(!charge(work,name.len+1)){*code=10;return 0;}if(bytes::equal(name,syntax::expression_text(file,c.expression))){return c.target;}}
        slot=(slot+1)%4096;probes+=1;
    }*code=10;return 0;
}
pub fn Result<Plan,catalog::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    if(signatures::module_count(metadata)!=v::len<syntax::Parsed>(sources)){return .err(fail(9,0,0,0));}
    var v::Vec<Row> rows=v::empty<Row>();var v::Vec<usize> bases=v::empty<usize>();var v::Vec<Use> uses=v::empty<Use>();var v::Vec<usize> queue=v::empty<usize>();
    var usize module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){
        let file=v::at<syntax::Parsed>(sources,module_index);if(!signatures::source_matches(metadata,module_index,file)){return .err(fail(9,module_index,0,0));}
        if(syntax::declaration_count(file)==0){return .err(fail(9,module_index,0,0));}let first=syntax::declaration_at(file,0);
        let bool bundled=sdk::authentic(syntax::source_view(file)[first.name_start..first.name_end],syntax::source_view(file));
        if(!v::push<usize>(ref mut bases,v::len<Row>(ref rows)).is_ok){return .err(fail(11,module_index,0,0));}
        var usize d=0;while(d<syntax::declaration_count(file)){
            if(!charge(work,1) || v::len<Row>(ref rows)>=4096){return .err(fail(10,module_index,0,0));}let item=syntax::declaration_at(file,d);
            let bool needed=!bundled || item.kind!=3;let usize at=v::len<Row>(ref rows);
            if(!v::push<Row>(ref mut rows,Row{.module_index,.declaration=d,.start=item.start,.end=item.end,.function=item.kind==3,.needed,.first=0}).is_ok){return .err(fail(11,module_index,0,0));}
            if(needed && item.kind!=1 && item.kind!=2 && !v::push<usize>(ref mut queue,at).is_ok){return .err(fail(11,module_index,0,0));}d+=1;
        }module_index+=1;
    }
    if(!v::push<usize>(ref mut bases,v::len<Row>(ref rows)).is_ok){return .err(fail(11,0,0,0));}
    module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){
        let file=v::at<syntax::Parsed>(sources,module_index);var usize i=1;while(i<=syntax::expression_count(file)){
            if(!charge(work,1)){return .err(fail(10,module_index,0,0));}let e=syntax::expression_at(file,i);
            if(e.kind==2){let usize id=owner_rows(ref rows,ref bases,module_index,e.start,e.end,work);if(*work>=1048576){return .err(fail(10,module_index,e.start,e.end));}
                if(id!=0){if(v::len<Use>(ref uses)>=16384 || !v::push<Use>(ref mut uses,Use{.expression=i,.next=v::at<Row>(ref rows,id-1).first}).is_ok){return .err(fail(11,module_index,e.start,e.end));}
                    let usize count=v::len<Use>(ref uses);{let ref mut Row target=v::at_mut<Row>(ref mut rows,id-1);target.first=count;}
                }
            }i+=1;
        }module_index+=1;
    }
    var v::Vec<Cached> cache=v::empty<Cached>();var usize slot=0;while(slot<4096){if(!charge(work,1)){return .err(fail(10,0,0,0));}if(!v::push<Cached>(ref mut cache,Cached{.module_index=0,.expression=0,.target=0}).is_ok){return .err(fail(11,0,0,0));}slot+=1;}
    var usize cursor=0;while(cursor<v::len<usize>(ref queue)){
        let row=*v::at<Row>(ref rows,*v::at<usize>(ref queue,cursor));let file=v::at<syntax::Parsed>(sources,row.module_index);var usize id=row.first;
        while(id!=0){let use=*v::at<Use>(ref uses,id-1);var u32 code=0;
            let usize target=lookup(sources,metadata,ref rows,ref bases,ref mut cache,row.module_index,use.expression,work,ref mut code);
            if(code!=0){return .err(fail(code,row.module_index,row.start,row.end));}
            if(target!=0){let usize at=target-1;
                if(!v::at<Row>(ref rows,at).needed){{let ref mut Row target=v::at_mut<Row>(ref mut rows,at);target.needed=true;}if(!v::push<usize>(ref mut queue,at).is_ok){return .err(fail(11,row.module_index,row.start,row.end));}}
            }id=use.next;
        }cursor+=1;
    }
    return .ok(Plan{.rows,.bases});
}
