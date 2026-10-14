module module_project_public_type_order;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_types as identity;
import module_types as types;
import module_signatures as signatures;
import module_state_layout as budget;
import std::vec as v;

// Detached interface numbering starts with current signature types. Executable
// IR retains its original type IDs; this mapping confers no source authority.
pub resource Order{v::Vec<types::Type> types;v::Vec<usize> old_to_new;v::Vec<usize> new_to_old;cleanup{}}
pub fn usize count(ref Order p){return v::len<types::Type>(ref p.types);}
pub fn ref v::Vec<types::Type> records(ref Order p){return ref p.types;}
pub fn types::Type at(ref Order p,usize id){assert(id>0 && id<=count(p));return *v::at<types::Type>(ref p.types,id-1);}
pub fn usize mapped(ref Order p,usize id){return if(id==0 || id>count(p)){0}else{*v::at<usize>(ref p.old_to_new,id-1)};}
pub fn usize original(ref Order p,usize id){assert(id>0 && id<=count(p));return *v::at<usize>(ref p.new_to_old,id-1);}
fn Result<types::Type,u32> translated(types::Type row,ref v::Vec<usize> map){
    var out=row;if(out.first!=0){if(out.first>v::len<usize>(map)){return .err(9);}out.first=*v::at<usize>(map,out.first-1);if(out.first==0){return .err(9);}}
    if(out.kind==6){if(out.second==0 || out.second>v::len<usize>(map)){return .err(9);}out.second=*v::at<usize>(map,out.second-1);if(out.second==0){return .err(9);}}return .ok(out);
}
fn Result<usize,u32> find(ref clean::Program p,ref v::Vec<usize> slots,types::Type row,ref mut usize work){var usize slot=identity::bucket(row);var usize probes=0;
    while(probes<8192){if(!budget::charge(work,1)){return .err(10);}let id=*v::at<usize>(slots,slot);if(id==0){return .err(9);}if(identity::same(clean::type_at(p,id),row)){return .ok(id);}slot=(slot+1)&8191;probes+=1;}return .err(10);
}
pub fn Result<Order,u32> build(ref clean::Program retained,ref signatures::Program current,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let count=clean::type_count(retained);let prefix=signatures::type_count(current);if(count==0 || count>4096 || prefix>count){return .err(9);}
    var bool unchanged=true;var usize i=1;while(i<=count){if(!budget::charge(work,1)){return .err(10);}if(!identity::valid(retained,i)){return .err(9);}if(i<=prefix && !identity::same(clean::type_at(retained,i),signatures::type_at(current,i))){unchanged=false;}i+=1;}
    var v::Vec<types::Type> types=v::empty<types::Type>();var v::Vec<usize> old_to_new=v::empty<usize>();var v::Vec<usize> new_to_old=v::empty<usize>();i=1;
    // Preserve already matching cold-owner numbering without an index allocation.
    while(i<=count){if(!budget::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut old_to_new,if(unchanged){i}else{0}).is_ok){return .err(11);}
        if(unchanged){if(!v::push<types::Type>(ref mut types,clean::type_at(retained,i)).is_ok || !v::push<usize>(ref mut new_to_old,i).is_ok){return .err(11);}}i+=1;}
    if(unchanged){return .ok(Order{.types,.old_to_new,.new_to_old});}
    var v::Vec<usize> slots=v::empty<usize>();i=0;while(i<8192){if(!budget::charge(work,1)){return .err(10);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(11);}i+=1;}i=1;
    while(i<=count){let row=clean::type_at(retained,i);var usize slot=identity::bucket(row);var usize probes=0;
        while(*v::at<usize>(ref slots,slot)!=0){if(probes>=8192 || !budget::charge(work,1)){return .err(10);}if(identity::same(row,clean::type_at(retained,*v::at<usize>(ref slots,slot)))){return .err(9);}slot=(slot+1)&8191;probes+=1;}
        if(!budget::charge(work,1)){return .err(10);}let cell=v::at_mut<usize>(ref mut slots,slot);*cell=i;i+=1;}
    i=1;while(i<=prefix){if(!budget::charge(work,1)){return .err(10);}let fresh=signatures::type_at(current,i);let row=switch(translated(fresh,ref new_to_old)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let old=switch(find(retained,ref slots,row,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(*v::at<usize>(ref old_to_new,old-1)!=0){return .err(9);}
        if(!v::push<types::Type>(ref mut types,fresh).is_ok || !v::push<usize>(ref mut new_to_old,old).is_ok){return .err(11);}let cell=v::at_mut<usize>(ref mut old_to_new,old-1);*cell=i;i+=1;}
    i=1;while(i<=count){if(!budget::charge(work,1)){return .err(10);}if(*v::at<usize>(ref old_to_new,i-1)==0){let row=switch(translated(clean::type_at(retained,i),ref old_to_new)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(!v::push<types::Type>(ref mut types,row).is_ok || !v::push<usize>(ref mut new_to_old,i).is_ok){return .err(11);}let cell=v::at_mut<usize>(ref mut old_to_new,i-1);*cell=v::len<types::Type>(ref types);}i+=1;}
    return .ok(Order{.types,.old_to_new,.new_to_old});
}
