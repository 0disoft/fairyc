// Raw malformed fixtures stay inside this temporary owner-module extension.
fn values::Field c_layout_lookup_field(usize id,usize owner,usize source,bool invalid){return values::Field{.id,.owner,.module_index=if(invalid){1}else{0},.declaration=1,.source_index=source};}
fn Node c_layout_lookup_node(usize source,u32 kind,usize type,usize left,usize right,usize target,values::Field field){
    return Node{.value=values::Expression{.constant=0,.constant_bits=0,.source_index=source,.binding=0,.kind,.start=0,.end=1,.left,.right,.target,.field,.place=values::no_place(),.builtin=values::Builtin{.kind=0,.element=0,.slice=false,.unsafe_context=false},.expected=type,.type,.state=1,.flexible=false,.minimum=false},.syntax=syntax::Expression{.kind,.start=0,.end=1,.token=0,.left,.right},.literal_kind=0,.text_start=0,.text_end=0,.cooked_start=0,.cooked_end=0};
}
pub fn Program c_layout_lookup_fixture(usize count,u32 mode){
    var v::Vec<Layout> layouts=v::empty<Layout>();var usize i=0;
    while(i<count){var usize type=count+1-i;if(mode==1 && i+1==count){type=count+1;}if(mode==2 && i+1==count){type=0;}if(mode==3 && i+1==count){type=count+3;}
        assert(v::push<Layout>(ref mut layouts,Layout{.value=interfaces::Shape{.type,.module_index=0,.declaration=if(type==0){0}else{type-1},.field_start=0,.field_count=if(type==2){3}else{0},.abilities=0,.repr_c=false,.start=0,.end=1,.cleanup_nonempty=false,.cleanup_effects=0},.name_start=0,.name_end=0}).is_ok);i+=1;
    }
    var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();i=0;
    while(i<3){assert(v::push<interfaces::Field>(ref mut fields,interfaces::Field{.kind=if(i==0){4}else{3},.type=if(i==0){1}else{0},.module_index=0,.declaration=1,.source_index=i+1,.start=0,.end=1,.name_start=0,.name_end=0,.payload_start=0,.payload_count=if(i==1){1}else{0}}).is_ok);i+=1;}
    let absent=mode==4;let invalid=mode==5;let usize type=if(absent){count+2}else if(invalid){0}else{2};let usize subject_type=if(invalid){2}else{type};
    let none=c_layout_lookup_field(0,0,0,false);var v::Vec<Node> nodes=v::empty<Node>();
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(1,1,1,0,0,0,none)).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(2,11,1,1,0,0,c_layout_lookup_field(1,2,1,false))).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(3,10,type,0,2,type,c_layout_lookup_field(2,type,2,mode==6))).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(4,2,subject_type,0,0,0,none)).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(5,15,0,4,0,subject_type,none)).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(6,16,type,0,0,0,c_layout_lookup_field(3,type,3,mode==6))).is_ok);
    assert(v::push<Node>(ref mut nodes,c_layout_lookup_node(7,18,0,6,0,0,none)).is_ok);
    var v::Vec<Body> bodies=v::empty<Body>();assert(v::push<Body>(ref mut bodies,Body{.body=0,.source_bytes=1,.start=0,.end=1,.inline_0=0,.inline_1=0,.inline_2=0,.inline_3=0,.inline_4=0,.inline_5=0,.inline_6=0,.inline_7=0,.nodes,.statements=v::empty<Statement>(),.locals=v::empty<values::Local>(),.calls=v::empty<values::Invocation>(),.arguments=v::empty<values::Argument>(),.regions=v::empty<regions::Scope>(),.placements=v::empty<regions::Placement>(),.resets=v::empty<regions::Exit>(),.text=bytes::empty()}).is_ok);
    return Program{.states=state::empty(),.sdk_sources=sdk_sources::empty(),.cleanup_ir=clean_ir::c_layout_lookup_types(count),.bodies,.layouts,.fields,.callables=v::empty<interfaces::Callable>(),.parameters=v::empty<usize>(),.names=bytes::empty(),.work=0};
}
