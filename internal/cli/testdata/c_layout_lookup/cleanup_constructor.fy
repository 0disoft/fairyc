// Test-only raw records, appended to a disposable copy of the owning module.
// No production factory or verifier is bypassed by product code.
pub fn Program c_layout_lookup_types(usize count){
    var v::Vec<types::Type> types=v::empty<types::Type>();var usize i=1;
    while(i<=count+2){assert(v::push<types::Type>(ref mut types,types::Type{.kind=if(i==1){1}else if(i==2 || i==count+2){4}else{2},.scalar=if(i==1){1}else{0},.module_index=0,.declaration=if(i==1){0}else{i-1},.first=0,.second=0}).is_ok);i+=1;}
    return Program{.types,.frames=v::empty<Frame>(),.glues=v::empty<Glue>(),.fields=v::empty<Field>(),.operands=v::empty<Operand>(),.drops=v::empty<Drop>(),.work=0,.project=true,.aggregates=false,.closures=false};
}
