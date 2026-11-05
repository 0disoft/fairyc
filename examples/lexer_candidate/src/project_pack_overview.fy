module project_pack_overview;
import project_pack_projection as projection;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub resource Overview{bytes::Buffer package;bytes::Buffer completeness;bytes::Buffer modules;bytes::Buffer omissions;usize records;usize selection;cleanup{}}
pub fn ref u8[] package(ref Overview p){return bytes::view(ref p.package);}
pub fn ref u8[] completeness(ref Overview p){return bytes::view(ref p.completeness);}
pub fn ref u8[] modules(ref Overview p){return bytes::view(ref p.modules);}
pub fn ref u8[] omissions(ref Overview p){return bytes::view(ref p.omissions);}
pub fn usize records(ref Overview p){return p.records;}
pub fn usize selection(ref Overview p){return p.selection;}

// Logical record limits are independent of the global work meter. Record-count
// refusal has precedence over byte refusal, as in the existing Go query owner.
fn bool record_limit(usize records,usize maximum){return records>=maximum;}
pub fn Result<Overview,u32> select(ref current::Ready ready,usize max_records,usize max_selection,ref mut usize meter){
    if(max_records==0 || max_selection==0){return .err(9);}let p=current::partition(ready);let t=validate::tree(p);
    let package=projection::package(p,meter)?;var usize used=bytes::len(ref package)+1;if(used>max_selection){return .err(3);}var usize records=1;
    let module_rows=model::rows(t,wire::field(t,validate::root(p),b"modules"),4096,meter)?;
    var completeness=bytes::empty();var modules=bytes::empty();var omissions=bytes::empty();var usize i=0;
    while(i<validate::completeness_count(p)){
        if(!work::charge(meter,1)){return .err(3);}let limited=record_limit(records,max_records);
        let row=if(limited){bytes::empty()}else{projection::completeness(p,i,meter)?};let cost=bytes::len(ref row)+1;
        if(limited || cost>max_selection-used){let ref u8[] code=if(limited){b"record-budget"}else{b"selection-byte-budget"};
            discard projection::omission(ref mut omissions,code,b"completeness",validate::completeness_count(p)-i,meter)?;discard w::raw(ref mut omissions,b",",meter)?;
            discard projection::omission(ref mut omissions,code,b"modules",v::len<usize>(ref module_rows),meter)?;
            return .ok(Overview{.package,.completeness,.modules,.omissions,.records,.selection=used});
        }
        if(i>0){discard w::raw(ref mut completeness,b",",meter)?;}discard w::raw(ref mut completeness,bytes::view(ref row),meter)?;records+=1;used+=cost;i+=1;
    }
    i=0;while(i<v::len<usize>(ref module_rows)){
        if(!work::charge(meter,1)){return .err(3);}let limited=record_limit(records,max_records);
        let row=if(limited){bytes::empty()}else{projection::module_record(p,*v::at<usize>(ref module_rows,i),meter)?};let cost=bytes::len(ref row)+1;
        if(limited || cost>max_selection-used){discard projection::omission(ref mut omissions,if(limited){b"record-budget"}else{b"selection-byte-budget"},b"modules",v::len<usize>(ref module_rows)-i,meter)?;break;}
        if(i>0){discard w::raw(ref mut modules,b",",meter)?;}discard w::raw(ref mut modules,bytes::view(ref row),meter)?;records+=1;used+=cost;i+=1;
    }
    return .ok(Overview{.package,.completeness,.modules,.omissions,.records,.selection=used});
}
