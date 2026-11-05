module project_pack_query;
import project_pack_node as nodes;
import project_pack_relations as relation_rows;
import project_pack_graph as graph;
import project_pack_frontier as frontier;
import project_pack_context as context;
import project_pack_options as options;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub struct Policy{u32 direction;usize mask;usize records;usize relations;usize depth;usize selection;}
pub struct Usage{usize records;usize relations;usize depth;usize selection;}
pub resource Selection{bytes::Buffer subject;bytes::Buffer facts;bytes::Buffer completeness;bytes::Buffer relations;bytes::Buffer omissions;Usage usage;cleanup{}}
pub fn ref u8[] subject(ref Selection p){return bytes::view(ref p.subject);}
pub fn ref u8[] facts(ref Selection p){return bytes::view(ref p.facts);}
pub fn ref u8[] completeness(ref Selection p){return bytes::view(ref p.completeness);}
pub fn ref u8[] relations(ref Selection p){return bytes::view(ref p.relations);}
pub fn ref u8[] omissions(ref Selection p){return bytes::view(ref p.omissions);}
pub fn Usage usage(ref Selection p){return p.usage;}
// Ownership assembly for the sibling around selector. Pack encoders run their
// own current-map selector and never accept this detached value from callers.
pub fn Selection assemble(bytes::Buffer subject,bytes::Buffer facts,bytes::Buffer completeness,bytes::Buffer relations,bytes::Buffer omissions,Usage usage){return Selection{.subject,.facts,.completeness,.relations,.omissions,.usage};}
pub fn Result<bool,u32> filter(ref mut bytes::Buffer out,usize mask,ref mut usize meter){
    var usize excluded=0;var usize i=0;while(i<10){if((mask & (1<<i))==0){excluded|=1<<i;}i+=1;}
    if(excluded==0){return .ok(true);}return omission(out,b"relation-filter",b"relations",excluded,0,false,meter);
}
pub fn Result<bool,u32> kinds(ref mut bytes::Buffer out,usize mask,ref mut usize meter){
    discard w::raw(out,b"[",meter)?;var bool first=true;var usize i=0;while(i<10){if((mask & (1<<i))!=0){if(!first){discard w::raw(out,b",",meter)?;}first=false;discard w::string(out,options::relation(i),meter)?;}i+=1;}return w::raw(out,b"]",meter);
}
pub fn Result<bool,u32> omission(ref mut bytes::Buffer out,ref u8[] code,ref u8[] section,usize mask,usize count,bool known,ref mut usize meter){
    if(bytes::len(out)>0){discard w::raw(out,b",",meter)?;}discard w::raw(out,b"{",meter)?;discard w::pair(out,b"code",code,meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"section",section,meter)?;
    if(mask!=0){discard w::raw(out,b",\"relations\":",meter)?;discard kinds(out,mask,meter)?;}
    if(count>0){discard w::raw(out,b",\"count\":",meter)?;discard w::number(out,count,meter)?;}
    discard w::raw(out,b",\"count_known\":",meter)?;discard w::boolean(out,known,meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> joins(ref mut bytes::Buffer out,Policy policy,usize unjoined,ref mut usize meter){
    if(unjoined>0){discard omission(out,b"dependency-map-not-joined",b"nodes",8,unjoined,true,meter)?;}
    if(policy.direction!=0 && (policy.mask & 8)!=0){discard omission(out,b"incoming-package-maps-not-joined",b"relations",8,0,false,meter)?;}return .ok(true);
}
// Relation count takes precedence over records, then selection bytes. Costs
// include one logical record separator, independently of final pack bytes.
pub fn u32 reserve(ref mut Usage used,Policy policy,usize cost,bool relation){
    if(relation && used.relations>=policy.relations){return 1;}if(used.records>=policy.records){return 2;}if(cost>policy.selection-used.selection){return 3;}
    used.records+=1;used.selection+=cost;if(relation){used.relations+=1;}return 0;
}
pub fn ref u8[] refusal(u32 code){let ref u8[] names=b"relation-budgetrecord-budgetselection-byte-budget";return if(code==1){names[..15]}else if(code==2){names[15..28]}else{names[28..49]};}
pub fn bool valid(Policy policy){return policy.direction<=2 && policy.mask>0 && policy.mask<1024 && policy.records>0 && policy.selection>0;}
pub fn Result<Selection,u32> show(ref current::Ready ready,ref u8[] symbol,Policy policy,ref mut usize meter){
    if(!valid(policy)){return .err(9);}let p=current::partition(ready);let root=graph::resolve(p,symbol,meter)?;let c=context::build(p,policy.mask,meter)?;let subject=nodes::encode(ready,root,meter)?;
    var mandatory=bytes::empty();discard w::raw(ref mut mandatory,b"{\"context\":",meter)?;discard w::raw(ref mut mandatory,context::json(ref c),meter)?;
    discard w::raw(ref mut mandatory,b",\"subject\":",meter)?;discard w::raw(ref mut mandatory,bytes::view(ref subject),meter)?;discard w::raw(ref mut mandatory,b"}",meter)?;
    var Usage used=Usage{.records=0,.relations=0,.depth=0,.selection=0};if(reserve(ref mut used,policy,bytes::len(ref mandatory)+1,false)!=0){return .err(3);}
    let g=graph::build(ready,meter)?;var root_list=v::empty<usize>();if(!v::push<usize>(ref mut root_list,root).is_ok){return .err(2);}let seen=v::empty<usize>();var iterator=frontier::start(ref g,ref root_list,policy.direction,meter)?;
    var usize total=0;var usize previous=graph::count(ref g);var usize rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen,ref mut previous,meter)?;while(rank<graph::count(ref g)){total+=1;rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen,ref mut previous,meter)?;}
    iterator=frontier::start(ref g,ref root_list,policy.direction,meter)?;previous=graph::count(ref g);var out=bytes::empty();var omitted=bytes::empty();discard filter(ref mut omitted,policy.mask,meter)?;var usize unjoined=0;
    rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen,ref mut previous,meter)?;while(rank<graph::count(ref g)){let r=graph::row(ref g,rank);let encoded=relation_rows::encode(p,r,meter)?;let stopped=reserve(ref mut used,policy,bytes::len(ref encoded)+1,true);
        if(stopped!=0){discard omission(ref mut omitted,refusal(stopped),b"relations",0,total-used.relations,true,meter)?;break;}
        if(used.relations>1){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,bytes::view(ref encoded),meter)?;if((r.kind==3 || r.kind==4) && graph::target(ref g,rank)==graph::nodes(ref g)){unjoined+=1;}
        rank=frontier::next(ref mut iterator,ref g,policy.mask,ref seen,ref mut previous,meter)?;
    }discard joins(ref mut omitted,policy,unjoined,meter)?;
    let identity=switch(bytes::copy(wire::string(validate::tree(p),validate::symbol(p,root),b"visible"))){case .ok(p):{p}case .err(_):{return .err(2);}};
    let complete=switch(bytes::copy(context::completeness(ref c))){case .ok(p):{p}case .err(_):{return .err(2);}};
    return .ok(Selection{.subject=identity,.facts=subject,.completeness=complete,.relations=out,.omissions=omitted,.usage=used});
}
